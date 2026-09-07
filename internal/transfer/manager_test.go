package transfer

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MawCeron/lazyftp/internal/model"
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
func (c *dirClient) Upload(localPath, remotePath string, _ func(int64)) error {
	c.uploaded = append(c.uploaded, path.Join(remotePath, filepath.Base(localPath)))
	return nil
}
func (c *dirClient) Download(remotePath, _ string, _ func(int64)) error {
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

// Transfers run as bare goroutines, outside Bubble Tea's panic handling. An
// unrecovered panic in one of them ends the process with the terminal still in
// raw mode, so reaching the end of this test is the whole assertion.
func TestAPanickingTransferDoesNotTakeTheProcessDown(t *testing.T) {
	m := NewManager(nil, func() *tea.Program { return nil })

	m.guard(Job{File: model.FileInfo{Name: "report.pdf"}}, func(Job) {
		panic("the server hung up mid-write")
	})
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
