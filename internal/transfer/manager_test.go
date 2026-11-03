package transfer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
)

// noopModel is the minimal tea.Model a headless Program needs to run so that
// runDirDownload's p.Send calls -- gated behind m.program() being non-nil --
// have somewhere to land instead of blocking.
type noopModel struct{}

func (noopModel) Init() tea.Cmd                       { return nil }
func (noopModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return noopModel{}, nil }
func (noopModel) View() tea.View                      { return tea.NewView("") }

// startHeadlessProgram runs a real, minimal tea.Program in the background so
// production code that only checks "is there a program to Send to" behaves as
// it would for real, without a terminal.
func startHeadlessProgram(t *testing.T) *tea.Program {
	t.Helper()
	p := tea.NewProgram(noopModel{},
		tea.WithInput(strings.NewReader("")),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
		tea.WithoutSignalHandler(),
		tea.WithoutSignals(),
	)
	go p.Run()
	t.Cleanup(func() {
		p.Quit()
		p.Wait()
	})
	return p
}

// dirClient is a stub Client backing a small in-memory remote directory tree,
// for exercising runDirDownload's recursion without a real server.
type dirClient struct {
	tree     map[string][]model.FileInfo
	fail     map[string]error // remote paths whose Download should fail
	mkdirErr map[string]error // remote paths whose Mkdir should fail

	mkdirCalls []string
	downloaded []string
	uploaded   []string
}

func (c *dirClient) Connect(string, string, string, int) error { return nil }
func (c *dirClient) Disconnect() error                         { return nil }
func (c *dirClient) List(path string) ([]model.FileInfo, error) {
	return c.tree[path], nil
}
func (c *dirClient) Upload(_ context.Context, localPath, remotePath string, _ func(int64)) error {
	c.uploaded = append(c.uploaded, path.Join(remotePath, filepath.Base(localPath)))
	return nil
}
func (c *dirClient) Download(_ context.Context, remotePath, _ string, _ func(int64)) error {
	c.downloaded = append(c.downloaded, remotePath)
	if err, ok := c.fail[remotePath]; ok {
		return err
	}
	return nil
}
func (c *dirClient) Mkdir(path string) error {
	c.mkdirCalls = append(c.mkdirCalls, path)
	return c.mkdirErr[path]
}
func (c *dirClient) Rename(string, string) error { return nil }
func (c *dirClient) Delete(string, bool) error   { return nil }

// Transfers run as bare goroutines, outside Bubble Tea's panic handling. An
// unrecovered panic in one of them ends the process with the terminal still in
// raw mode, so reaching the end of this test is the whole assertion.
func TestAPanickingTransferDoesNotTakeTheProcessDown(t *testing.T) {
	m := NewManager(nil, func() *tea.Program { return nil })

	m.guard(Job{File: model.FileInfo{Name: "report.pdf"}}, func(Job) {
		panic("the server hung up mid-write")
	})
}

// recorder is a tea.Model that hands every message to a channel, so a test can
// see what a transfer sent to the program.
type recorder struct{ msgs chan tea.Msg }

func (recorder) Init() tea.Cmd { return nil }
func (r recorder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	r.msgs <- msg
	return r, nil
}
func (recorder) View() tea.View { return tea.NewView("") }

type panickingClient struct{ dirClient }

func (panickingClient) Upload(context.Context, string, string, func(int64)) error {
	panic("the server hung up")
}

func TestAPanickingTransferMarksItsOwnRowAsFailed(t *testing.T) {
	r := recorder{msgs: make(chan tea.Msg, 16)}
	p := tea.NewProgram(r,
		tea.WithInput(strings.NewReader("")),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
		tea.WithoutSignalHandler(),
		tea.WithoutSignals(),
	)
	go p.Run()
	t.Cleanup(func() {
		p.Quit()
		p.Wait()
	})
	m := NewManager(&panickingClient{}, func() *tea.Program { return p })

	m.guard(Job{File: model.FileInfo{Name: "report.pdf"}, Direction: Upload}, m.run)

	var started, failed int64
	timeout := time.After(5 * time.Second)
	for failed == 0 {
		select {
		case msg := <-r.msgs:
			switch msg := msg.(type) {
			case shared.TransferStartMsg:
				started = msg.Transfer.ID
			case shared.TransferErrorMsg:
				failed = msg.ID
			}
		case <-timeout:
			t.Fatal("no TransferErrorMsg arrived for the panicking transfer")
		}
	}
	if started == 0 || failed != started {
		t.Errorf("error was attributed to row %d, want the started row %d", failed, started)
	}
}

// startRecordingProgram runs a headless program and hands back every message
// sent to it.
func startRecordingProgram(t *testing.T) (*tea.Program, <-chan tea.Msg) {
	t.Helper()
	r := recorder{msgs: make(chan tea.Msg, 64)}
	p := tea.NewProgram(r,
		tea.WithInput(strings.NewReader("")),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
		tea.WithoutSignalHandler(),
		tea.WithoutSignals(),
	)
	go p.Run()
	t.Cleanup(func() {
		p.Quit()
		p.Wait()
	})
	return p, r.msgs
}

// waitFor returns the first message for which match is true.
func waitFor(t *testing.T, msgs <-chan tea.Msg, match func(tea.Msg) bool) tea.Msg {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case msg := <-msgs:
			if match(msg) {
				return msg
			}
		case <-timeout:
			t.Fatal("the message never arrived")
		}
	}
}

func waitForIdle(t *testing.T, m *Manager) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for m.Active() != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("%d job(s) still active", m.Active())
		}
		time.Sleep(time.Millisecond)
	}
}

func isStart(msg tea.Msg) bool     { _, ok := msg.(shared.TransferStartMsg); return ok }
func isDone(msg tea.Msg) bool      { _, ok := msg.(shared.TransferDoneMsg); return ok }
func isCancelled(msg tea.Msg) bool { _, ok := msg.(shared.TransferCancelledMsg); return ok }
func endsRow(msg tea.Msg) bool {
	_, failed := msg.(shared.TransferErrorMsg)
	return failed || isCancelled(msg)
}

// ctxClient holds every transfer until its context is cancelled or release is
// closed, except the files named in pass, which finish at once, call
// afterPass and, on download, leave the file behind. It records the name of
// every file it was asked to transfer.
type ctxClient struct {
	dirClient
	pass      map[string]bool
	release   chan struct{}
	afterPass func()

	mu        sync.Mutex
	requested []string
}

func (c *ctxClient) hold(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.release:
		return nil
	}
}

func (c *ctxClient) record(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requested = append(c.requested, name)
}

func (c *ctxClient) asked() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.requested...)
}

func (c *ctxClient) finish() error {
	if c.afterPass != nil {
		c.afterPass()
	}
	return nil
}

func (c *ctxClient) Upload(ctx context.Context, localPath, _ string, _ func(int64)) error {
	name := filepath.Base(localPath)
	c.record(name)
	if c.pass[name] {
		return c.finish()
	}
	return c.hold(ctx)
}

func (c *ctxClient) Download(ctx context.Context, remotePath, localPath string, _ func(int64)) error {
	name := path.Base(remotePath)
	c.record(name)
	if c.pass[name] {
		os.WriteFile(filepath.Join(localPath, name), []byte("done"), 0o644)
		return c.finish()
	}
	return c.hold(ctx)
}

func TestCancellingAJobStopsItsTransfer(t *testing.T) {
	p, msgs := startRecordingProgram(t)
	m := NewManager(&ctxClient{}, func() *tea.Program { return p })

	id := m.Enqueue([]Job{{File: model.FileInfo{Name: "report.pdf"}, Direction: Upload}})[0]
	started := waitFor(t, msgs, isStart).(shared.TransferStartMsg)

	if !m.Cancel(id) {
		t.Fatal("Cancel did not find the running job")
	}

	cancelled, ok := waitFor(t, msgs, endsRow).(shared.TransferCancelledMsg)
	if !ok {
		t.Fatal("the cancelled transfer was reported as an error")
	}
	if cancelled.ID != started.Transfer.ID {
		t.Errorf("cancelled row %d, want the started row %d", cancelled.ID, started.Transfer.ID)
	}
	waitForIdle(t, m)
}

func TestCancellingOneJobLeavesTheOthersRunning(t *testing.T) {
	p, msgs := startRecordingProgram(t)
	c := &ctxClient{release: make(chan struct{})}
	m := NewManager(c, func() *tea.Program { return p })

	ids := m.Enqueue([]Job{
		{File: model.FileInfo{Name: "first.bin"}, Direction: Upload},
		{File: model.FileInfo{Name: "second.bin"}, Direction: Upload},
	})
	waitFor(t, msgs, isStart)
	waitFor(t, msgs, isStart)

	m.Cancel(ids[0])
	waitFor(t, msgs, isCancelled)

	// The second job only finishes as Done if the cancel did not reach it.
	close(c.release)
	waitFor(t, msgs, isDone)
	waitForIdle(t, m)
}

// partialClient leaves a half-written file behind and then holds, as a
// transfer cut mid-way does, and records what it is asked to delete.
type partialClient struct {
	ctxClient
	written chan struct{}
	dmu     sync.Mutex
	deleted []string
}

func (c *partialClient) Download(ctx context.Context, remotePath, localPath string, _ func(int64)) error {
	os.WriteFile(filepath.Join(localPath, path.Base(remotePath)), []byte("half"), 0o644)
	close(c.written)
	<-ctx.Done()
	return ctx.Err()
}

func (c *partialClient) Delete(p string, _ bool) error {
	c.dmu.Lock()
	defer c.dmu.Unlock()
	c.deleted = append(c.deleted, p)
	return nil
}

func TestACancelledDownloadLeavesNoPartialFile(t *testing.T) {
	p, msgs := startRecordingProgram(t)
	c := &partialClient{written: make(chan struct{})}
	m := NewManager(c, func() *tea.Program { return p })
	local := t.TempDir()

	id := m.Enqueue([]Job{{
		File:       model.FileInfo{Name: "big.iso"},
		LocalPath:  local,
		RemotePath: "/pub",
		Direction:  Download,
	}})[0]
	select {
	case <-c.written:
	case <-time.After(5 * time.Second):
		t.Fatal("the transfer never wrote its partial file")
	}

	m.Cancel(id)
	waitFor(t, msgs, isCancelled)

	if _, err := os.Stat(filepath.Join(local, "big.iso")); !os.IsNotExist(err) {
		t.Errorf("the partial download is still there: %v", err)
	}
}

func TestACancelledUploadDeletesTheRemotePartial(t *testing.T) {
	p, msgs := startRecordingProgram(t)
	c := &partialClient{}
	m := NewManager(c, func() *tea.Program { return p })

	id := m.Enqueue([]Job{{
		File:       model.FileInfo{Name: "big.iso"},
		LocalPath:  t.TempDir(),
		RemotePath: "/incoming",
		Direction:  Upload,
	}})[0]
	waitFor(t, msgs, isStart)

	m.Cancel(id)
	waitFor(t, msgs, isCancelled)

	c.dmu.Lock()
	defer c.dmu.Unlock()
	if len(c.deleted) != 1 || c.deleted[0] != "/incoming/big.iso" {
		t.Errorf("deleted %v, want [/incoming/big.iso]", c.deleted)
	}
}

func isLog(text string) func(tea.Msg) bool {
	return func(msg tea.Msg) bool {
		l, ok := msg.(shared.LogMsg)
		return ok && l.Message == text
	}
}

func isLogWithPrefix(prefix string) func(tea.Msg) bool {
	return func(msg tea.Msg) bool {
		l, ok := msg.(shared.LogMsg)
		return ok && strings.HasPrefix(l.Message, prefix)
	}
}

// collectUntil returns every message up to and including the first that
// matches. A transfer's goroutine sends in order, so a message it sends last
// arrives after everything it sent before.
func collectUntil(t *testing.T, msgs <-chan tea.Msg, match func(tea.Msg) bool) []tea.Msg {
	t.Helper()
	var got []tea.Msg
	timeout := time.After(5 * time.Second)
	for {
		select {
		case msg := <-msgs:
			got = append(got, msg)
			if match(msg) {
				return got
			}
		case <-timeout:
			t.Fatal("the message never arrived")
		}
	}
}

func count(msgs []tea.Msg, match func(tea.Msg) bool) int {
	n := 0
	for _, msg := range msgs {
		if match(msg) {
			n++
		}
	}
	return n
}

func TestCancellingADirectoryStopsItsRemainingFiles(t *testing.T) {
	p, msgs := startRecordingProgram(t)
	c := &ctxClient{dirClient: dirClient{tree: map[string][]model.FileInfo{
		"/pub/photos": {{Name: "a.jpg"}, {Name: "b.jpg"}, {Name: "c.jpg"}},
	}}}
	m := NewManager(c, func() *tea.Program { return p })

	id := m.Enqueue([]Job{{
		File:       model.FileInfo{Name: "photos", Type: model.FileTypeDir},
		LocalPath:  t.TempDir(),
		RemotePath: "/pub",
		Direction:  Download,
	}})[0]
	waitFor(t, msgs, isStart)
	m.Cancel(id)

	got := collectUntil(t, msgs, isLog("Cancelled: photos"))
	waitForIdle(t, m)

	if n := count(got, isStart); n != 0 {
		t.Errorf("%d more file(s) started after the cancellation", n)
	}
	if n := count(got, isLogWithPrefix("Cancelled transfer:")); n != 1 {
		t.Errorf("%d files were visited after the cancellation, want only the one in flight", n)
	}
	if asked := c.asked(); len(asked) != 1 || asked[0] != "a.jpg" {
		t.Errorf("client was asked for %v, want [a.jpg]", asked)
	}
}

// Cancelling between two files leaves no transfer to fail, so the walk itself
// has to notice. The first file's transfer cancels the job as it finishes.
func TestCancellingBetweenFilesStopsTheDirectory(t *testing.T) {
	p, msgs := startRecordingProgram(t)
	ids := make(chan int64, 1)
	local := t.TempDir()
	c := &ctxClient{
		pass: map[string]bool{"a.jpg": true},
		dirClient: dirClient{tree: map[string][]model.FileInfo{
			"/pub/photos": {{Name: "a.jpg"}, {Name: "b.jpg"}, {Name: "c.jpg"}},
		}},
	}
	m := NewManager(c, func() *tea.Program { return p })
	c.afterPass = func() { m.Cancel(<-ids) }

	ids <- m.Enqueue([]Job{{
		File:       model.FileInfo{Name: "photos", Type: model.FileTypeDir},
		LocalPath:  local,
		RemotePath: "/pub",
		Direction:  Download,
	}})[0]

	got := collectUntil(t, msgs, isLog("Cancelled: photos"))
	waitForIdle(t, m)

	if count(got, isDone) != 1 || count(got, isCancelled) != 0 {
		t.Errorf("want the first file done and nothing cancelled, got %d done and %d cancelled",
			count(got, isDone), count(got, isCancelled))
	}
	if n := count(got, isLogWithPrefix("Cancelled transfer:")); n != 0 {
		t.Errorf("%d file(s) were visited after the cancellation", n)
	}
	if asked := c.asked(); len(asked) != 1 || asked[0] != "a.jpg" {
		t.Errorf("client was asked for %v, want [a.jpg]", asked)
	}
	if _, err := os.Stat(filepath.Join(local, "photos", "a.jpg")); err != nil {
		t.Errorf("the file that finished before the cancellation is gone: %v", err)
	}
}

func TestCancellingBetweenFilesStopsAnUploadedDirectory(t *testing.T) {
	p, msgs := startRecordingProgram(t)
	ids := make(chan int64, 1)
	local := t.TempDir()
	for _, name := range []string{"a.jpg", "b.jpg", "c.jpg"} {
		os.MkdirAll(filepath.Join(local, "photos"), 0o755)
		os.WriteFile(filepath.Join(local, "photos", name), []byte(name), 0o644)
	}
	c := &ctxClient{pass: map[string]bool{"a.jpg": true}}
	m := NewManager(c, func() *tea.Program { return p })
	c.afterPass = func() { m.Cancel(<-ids) }

	ids <- m.Enqueue([]Job{{
		File:       model.FileInfo{Name: "photos", Type: model.FileTypeDir},
		LocalPath:  local,
		RemotePath: "/pub",
		Direction:  Upload,
	}})[0]

	got := collectUntil(t, msgs, isLog("Cancelled: photos"))
	waitForIdle(t, m)

	if count(got, isDone) != 1 || count(got, isLogWithPrefix("Cancelled transfer:")) != 0 {
		t.Errorf("want one file done and none visited afterwards, got %d done and %d visited",
			count(got, isDone), count(got, isLogWithPrefix("Cancelled transfer:")))
	}
	if asked := c.asked(); len(asked) != 1 || asked[0] != "a.jpg" {
		t.Errorf("client was asked for %v, want [a.jpg]", asked)
	}
}

// Cancelling between two files leaves no transfer to fail, so the walk itself
// has to notice.
func TestACancelledDirectoryWalkStartsNothing(t *testing.T) {
	p, _ := startRecordingProgram(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	local := t.TempDir()
	os.MkdirAll(filepath.Join(local, "photos"), 0o755)
	os.WriteFile(filepath.Join(local, "photos", "a.jpg"), []byte("a"), 0o644)
	c := &dirClient{tree: map[string][]model.FileInfo{"/pub/photos": {{Name: "a.jpg"}}}}
	m := NewManager(c, func() *tea.Program { return p })
	dir := Job{
		File:       model.FileInfo{Name: "photos", Type: model.FileTypeDir},
		LocalPath:  local,
		RemotePath: "/pub",
		ctx:        cancelled,
	}

	dir.Direction = Download
	m.runDirDownload(dir)
	dir.Direction = Upload
	m.runDir(dir)

	if len(c.downloaded) != 0 || len(c.uploaded) != 0 {
		t.Errorf("a cancelled directory transferred %v and %v", c.downloaded, c.uploaded)
	}
}

// Truncating the destination is the first thing a transfer does, so one that
// starts after its job was cancelled would destroy a same-named file for nothing.
func TestAJobCancelledBeforeItStartsTouchesNothing(t *testing.T) {
	p, msgs := startRecordingProgram(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	c := &dirClient{}
	m := NewManager(c, func() *tea.Program { return p })

	m.run(Job{File: model.FileInfo{Name: "report.pdf"}, Direction: Upload, ctx: cancelled})
	m.run(Job{File: model.FileInfo{Name: "report.pdf"}, Direction: Download, ctx: cancelled})

	waitFor(t, msgs, isLog("Cancelled transfer: report.pdf"))
	if len(c.uploaded) != 0 || len(c.downloaded) != 0 {
		t.Errorf("a cancelled job still transferred %v and %v", c.uploaded, c.downloaded)
	}
}

func TestCancellingAJobThatIsGoneDoesNothing(t *testing.T) {
	p, msgs := startRecordingProgram(t)
	m := NewManager(&ctxClient{pass: map[string]bool{"fast.bin": true}}, func() *tea.Program { return p })

	if m.Cancel(12345) {
		t.Error("Cancel reported a job that never existed")
	}

	id := m.Enqueue([]Job{{File: model.FileInfo{Name: "fast.bin"}, Direction: Upload}})[0]
	collectUntil(t, msgs, isDone)
	waitForIdle(t, m)

	if m.Cancel(id) {
		t.Error("Cancel reported a job that had already finished")
	}
	select {
	case msg := <-msgs:
		t.Errorf("cancelling a job that is gone sent %T", msg)
	default:
	}
}

func TestCancellingTheSameJobTwiceCancelsItOnce(t *testing.T) {
	p, msgs := startRecordingProgram(t)
	m := NewManager(&ctxClient{}, func() *tea.Program { return p })

	id := m.Enqueue([]Job{{File: model.FileInfo{Name: "stuck.bin"}, Direction: Upload}})[0]
	waitFor(t, msgs, isStart)

	first, second := m.Cancel(id), m.Cancel(id)
	if !first || second {
		t.Errorf("Cancel returned %v then %v, want true then false", first, second)
	}
	waitForIdle(t, m)
}

func TestGuardRunsTheTransfer(t *testing.T) {
	m := NewManager(nil, func() *tea.Program { return nil })

	var got string
	m.guard(Job{File: model.FileInfo{Name: "report.pdf"}}, func(j Job) {
		got = j.File.Name
	})

	if got != "report.pdf" {
		t.Errorf("guard ran the transfer with %q, want \"report.pdf\"", got)
	}
}

// The download-side mirror of the existing upload directory recursion (#35):
// recreate the tree locally, walk the remote listing instead of the local
// filesystem, and reach files at any depth.
func TestRunDirDownloadRecreatesTheTreeLocally(t *testing.T) {
	tmp := t.TempDir()

	stub := &dirClient{
		tree: map[string][]model.FileInfo{
			"/remote/photos": {
				{Name: "a.jpg", Type: model.FileTypeFile, Size: 10},
				{Name: "nested", Type: model.FileTypeDir},
			},
			"/remote/photos/nested": {
				{Name: "b.jpg", Type: model.FileTypeFile, Size: 5},
			},
		},
	}
	prog := startHeadlessProgram(t)
	m := NewManager(stub, func() *tea.Program { return prog })

	m.runDirDownload(Job{
		File:       model.FileInfo{Name: "photos", Type: model.FileTypeDir},
		LocalPath:  tmp,
		RemotePath: "/remote",
		Direction:  Download,
	})

	for _, dir := range []string{"photos", filepath.Join("photos", "nested")} {
		if _, err := os.Stat(filepath.Join(tmp, dir)); err != nil {
			t.Errorf("local directory %s was not created: %v", dir, err)
		}
	}

	want := []string{"/remote/photos/a.jpg", "/remote/photos/nested/b.jpg"}
	if len(stub.downloaded) != len(want) {
		t.Fatalf("downloaded %v, want %v", stub.downloaded, want)
	}
	for i, path := range want {
		if stub.downloaded[i] != path {
			t.Errorf("downloaded[%d] = %q, want %q", i, stub.downloaded[i], path)
		}
	}
}

// A failure on one entry (a nested directory, here) must not stop its
// siblings at the same level from still being processed.
func TestRunDirDownloadContinuesPastAFailedEntry(t *testing.T) {
	tmp := t.TempDir()

	stub := &dirClient{
		tree: map[string][]model.FileInfo{
			"/remote/photos": {
				{Name: "broken.jpg", Type: model.FileTypeFile},
				{Name: "ok.jpg", Type: model.FileTypeFile},
			},
		},
		fail: map[string]error{
			"/remote/photos/broken.jpg": errors.New("connection reset"),
		},
	}
	prog := startHeadlessProgram(t)
	m := NewManager(stub, func() *tea.Program { return prog })

	m.runDirDownload(Job{
		File:       model.FileInfo{Name: "photos", Type: model.FileTypeDir},
		LocalPath:  tmp,
		RemotePath: "/remote",
		Direction:  Download,
	})

	want := []string{"/remote/photos/broken.jpg", "/remote/photos/ok.jpg"}
	if len(stub.downloaded) != len(want) {
		t.Fatalf("downloaded %v, want %v -- a failed entry must not abandon its siblings", stub.downloaded, want)
	}
}

func TestRunDirUploadsDirectoryTree(t *testing.T) {
	tmp := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmp, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "a.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "nested", "b.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	stub := &dirClient{tree: map[string][]model.FileInfo{}}
	prog := startHeadlessProgram(t)
	m := NewManager(stub, func() *tea.Program { return prog })

	m.runDir(Job{
		File:       model.FileInfo{Name: filepath.Base(tmp), Type: model.FileTypeDir},
		LocalPath:  filepath.Dir(tmp),
		RemotePath: "/remote",
		Direction:  Upload,
	})

	remoteRoot := path.Join("/remote", filepath.Base(tmp))
	wantMkdir := []string{remoteRoot, path.Join(remoteRoot, "nested")}
	if len(stub.mkdirCalls) != len(wantMkdir) {
		t.Fatalf("mkdirCalls = %v, want %v", stub.mkdirCalls, wantMkdir)
	}

	wantUploaded := []string{path.Join(remoteRoot, "a.jpg"), path.Join(remoteRoot, "nested", "b.jpg")}
	if len(stub.uploaded) != len(wantUploaded) {
		t.Fatalf("uploaded %v, want %v", stub.uploaded, wantUploaded)
	}
	for _, want := range wantUploaded {
		found := false
		for _, got := range stub.uploaded {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Errorf("uploaded %v, missing %q", stub.uploaded, want)
		}
	}
}

// Regression test for the bug this fix addresses: re-uploading a directory
// that already exists remotely must merge into it, the same overwrite
// semantics a single file upload already gets -- not abandon the whole
// subtree just because Mkdir refuses to recreate an existing directory.
func TestRunDirMergesIntoAnAlreadyExistingRemoteDirectory(t *testing.T) {
	tmp := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmp, "photos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "photos", "a.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	remoteRoot := "/remote/photos"
	stub := &dirClient{
		tree: map[string][]model.FileInfo{
			// "/remote" already lists "photos" as an existing directory --
			// what m.client.List("/remote") would return on a re-upload.
			"/remote": {{Name: "photos", Type: model.FileTypeDir}},
		},
		mkdirErr: map[string]error{
			remoteRoot: errors.New("550 Create directory operation failed"),
		},
	}
	prog := startHeadlessProgram(t)
	m := NewManager(stub, func() *tea.Program { return prog })

	m.runDir(Job{
		File:       model.FileInfo{Name: "photos", Type: model.FileTypeDir},
		LocalPath:  tmp,
		RemotePath: "/remote",
		Direction:  Upload,
	})

	want := path.Join(remoteRoot, "a.jpg")
	if len(stub.uploaded) != 1 || stub.uploaded[0] != want {
		t.Fatalf("uploaded %v, want [%q] -- Mkdir failing because the directory already exists must not abort the merge", stub.uploaded, want)
	}
}

// A Mkdir failure that ISN'T "already exists" (permission denied, disk
// full, connection dropped) must still abort instead of being silently
// treated as a pre-existing directory.
func TestRunDirAbortsOnARealMkdirError(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "a.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	remoteRoot := "/remote/photos"
	stub := &dirClient{
		tree: map[string][]model.FileInfo{"/remote": {}}, // "photos" is NOT listed
		mkdirErr: map[string]error{
			remoteRoot: errors.New("550 Permission denied"),
		},
	}
	prog := startHeadlessProgram(t)
	m := NewManager(stub, func() *tea.Program { return prog })

	m.runDir(Job{
		File:       model.FileInfo{Name: "photos", Type: model.FileTypeDir},
		LocalPath:  tmp,
		RemotePath: "/remote",
		Direction:  Upload,
	})

	if len(stub.uploaded) != 0 {
		t.Errorf("uploaded %v, want none -- a genuine Mkdir error must abort the transfer", stub.uploaded)
	}
}

// names lists a directory's entries, for checking that nothing else appeared.
func names(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func TestARemoteDirectoryCannotWriteOutsideTheDownloadDirectory(t *testing.T) {
	base := t.TempDir()
	dest := filepath.Join(base, "dl")
	os.Mkdir(dest, 0o755)

	hostile := []string{"..", "../..", "../../outside", "../escape.txt", "/tmp/absolute", "nested/deeper"}
	var entries []model.FileInfo
	for _, name := range hostile {
		typ := model.FileTypeDir
		if strings.HasSuffix(name, ".txt") || strings.HasPrefix(name, "/") {
			typ = model.FileTypeFile
		}
		entries = append(entries, model.FileInfo{Name: name, Type: typ})
	}
	entries = append(entries, model.FileInfo{Name: "ok.jpg", Type: model.FileTypeFile})
	stub := &dirClient{tree: map[string][]model.FileInfo{
		"/remote/photos": entries,
		"/outside":       {{Name: "inside.txt", Type: model.FileTypeFile}},
	}}
	prog, msgs := startRecordingProgram(t)
	m := NewManager(stub, func() *tea.Program { return prog })

	m.runDirDownload(Job{
		File:       model.FileInfo{Name: "photos", Type: model.FileTypeDir},
		LocalPath:  dest,
		RemotePath: "/remote",
		Direction:  Download,
	})

	if got := names(t, base); len(got) != 1 || got[0] != "dl" {
		t.Errorf("the download directory's parent holds %v, want only [dl]", got)
	}
	if got := names(t, dest); len(got) != 1 || got[0] != "photos" {
		t.Errorf("the download directory holds %v, want only [photos]", got)
	}
	if got := names(t, filepath.Join(dest, "photos")); len(got) != 0 {
		t.Errorf("the hostile entries created %v inside the downloaded directory", got)
	}
	if len(stub.downloaded) != 1 || stub.downloaded[0] != "/remote/photos/ok.jpg" {
		t.Errorf("downloaded %v, want only the well-named file", stub.downloaded)
	}

	logged := map[string]bool{}
	deadline := time.After(5 * time.Second)
	for len(logged) < len(hostile) {
		select {
		case msg := <-msgs:
			if l, ok := msg.(shared.LogMsg); ok && l.Level == shared.LogError {
				for _, name := range hostile {
					if strings.Contains(l.Message, fmt.Sprintf("%q", name)) {
						logged[name] = true
					}
				}
			}
		case <-deadline:
			t.Fatalf("only %v of the rejected names were logged", logged)
		}
	}
}

func TestARemoteFileNamedAsAPathIsNotDownloaded(t *testing.T) {
	stub := &dirClient{}
	prog := startHeadlessProgram(t)
	m := NewManager(stub, func() *tea.Program { return prog })

	for _, name := range []string{"../escape", "a/b", "..", "."} {
		m.run(Job{
			File:       model.FileInfo{Name: name, Type: model.FileTypeFile},
			LocalPath:  t.TempDir(),
			RemotePath: "/remote",
			Direction:  Download,
		})
	}

	if len(stub.downloaded) != 0 {
		t.Errorf("downloaded %v, want nothing", stub.downloaded)
	}
}

func TestOrdinaryRemoteNamesAreStillAccepted(t *testing.T) {
	for _, name := range []string{"report.pdf", "my file (1).txt", "résumé.pdf", ".hidden", "a..b", "...", "日本語"} {
		if !safeName(name) {
			t.Errorf("%q was rejected, but it is an ordinary file name", name)
		}
	}
	for _, name := range []string{"", ".", "..", "../x", "a/b", "/abs", "x/.."} {
		if safeName(name) {
			t.Errorf("%q was accepted", name)
		}
	}
}
