package ui

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
)

// TestPreview writes one frame per screen to assets/frames/*.ans, which
// assets/screens.sh turns into the SVGs the README embeds. Skipped unless
// PREVIEW=1, so an ordinary `go test ./...` never writes anything.
//
// These frames are the real render() output, byte for byte what the
// terminal is handed -- not a mock-up. Lipgloss v2 resolves colors at
// Render() time regardless of whether stdout is a TTY, so no color-profile
// override is needed here the way lipgloss v1 required one.
func TestPreview(t *testing.T) {
	if os.Getenv("PREVIEW") == "" {
		t.Skip("set PREVIEW=1 to write assets/frames")
	}

	// Anchored to this source file, not the working directory: a compiled
	// test binary run from the repo root would otherwise write two levels
	// above it.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the source file to write beside")
	}
	out := filepath.Join(filepath.Dir(thisFile), "..", "..", "assets", "frames")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	write := func(name string, a App) {
		t.Helper()
		path := filepath.Join(out, name+".ans")
		if err := os.WriteFile(path, []byte(a.render()+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", path)
	}

	// A fixed reference time, not time.Now(): dates in the frame would
	// otherwise drift a little further from "now" on every regeneration.
	modTime := time.Date(2026, 3, 12, 9, 41, 0, 0, time.UTC)

	localFiles := []model.FileInfo{
		{Name: "content", Type: model.FileTypeDir, ModTime: modTime},
		{Name: "static", Type: model.FileTypeDir, ModTime: modTime},
		{Name: "templates", Type: model.FileTypeDir, ModTime: modTime},
		{Name: "config.toml", Size: 612, ModTime: modTime},
		{Name: "deploy.sh", Size: 340, ModTime: modTime},
		{Name: "go.mod", Size: 58, ModTime: modTime},
		{Name: "LICENSE", Size: 1072, ModTime: modTime},
		{Name: "README.md", Size: 2380, ModTime: modTime},
		{Name: "site.tar.gz", Size: 4718592, ModTime: modTime},
	}

	remoteModTime := modTime.Add(3 * time.Minute)
	remoteFiles := []model.FileInfo{
		{Name: "content", Type: model.FileTypeDir, ModTime: modTime},
		{Name: "static", Type: model.FileTypeDir, ModTime: modTime},
		{Name: "templates", Type: model.FileTypeDir, ModTime: modTime},
		{Name: ".htaccess", ModTime: modTime},
		{Name: "config.toml", Size: 612, ModTime: modTime},
		{Name: "deploy.sh", Size: 340, ModTime: modTime},
		{Name: "index.html", Size: 8140, ModTime: modTime},
		{Name: "site.tar.gz", Size: 4718592, ModTime: remoteModTime},
	}

	a := NewApp(nil, false, nil, "0.2.0", false)
	a.width, a.height = 140, 34
	a.focus = focusLocal
	a.connected = true
	a.connUser = "deploy"
	a.connAddr = "203.0.113.10:21"
	a.connProtocol = client.FTP
	a.local, _ = a.local.WithFiles(localFiles, "/home/dev/blog")
	a.remote, _ = a.remote.WithFiles(remoteFiles, "/var/www/blog")
	_, panelH, bottomH := a.heights()
	a.local = a.local.SetSize(a.panelWidth(), panelH)
	a.remote = a.remote.SetSize(a.panelWidth(), panelH)
	a.log = a.log.SetSize(a.panelWidth(), a.bottomPanelHeight(bottomH))
	a.processes = a.processes.SetSize(a.panelWidth(), a.bottomPanelHeight(bottomH))

	// Built directly rather than through Add, which stamps time.Now(): a
	// frame regenerated tomorrow must come out byte-identical to one
	// regenerated today, or the CI diff check could never pass twice.
	logTime := modTime.Add(3 * time.Minute)
	a.log.entries = []LogEntry{
		{Time: logTime, Message: "Connecting to 203.0.113.10:21 over FTP", Level: LogInfo},
		{Time: logTime, Message: "Connected to 203.0.113.10:21", Level: LogSuccess},
		{Time: logTime, Message: "Complete transfer: index.html", Level: LogSuccess},
		{Time: logTime, Message: "Complete transfer: site.tar.gz", Level: LogSuccess},
	}
	a.log.viewport.SetContent(a.log.renderEntries())

	a.processes = a.processes.AddTransfer(shared.Transfer{
		Filename: "index.html", Total: 8140, Current: 8140,
		Direction: shared.DirectionUpload, Status: shared.StatusDone,
	})
	a.processes = a.processes.AddTransfer(shared.Transfer{
		Filename: "site.tar.gz", Total: 4718592, Current: 4718592,
		Direction: shared.DirectionUpload, Status: shared.StatusDone,
	})

	write("main", a)
}
