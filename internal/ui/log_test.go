package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogWritesEveryEntryToTheFile(t *testing.T) {
	var file bytes.Buffer
	log := NewLogPanel(&file)

	log = log.Add("Connecting to example.org:21 over FTP", LogInfo)
	log = log.Add("Error connecting: i/o timeout", LogError)

	lines := strings.Split(strings.TrimSpace(file.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("wrote %d lines, want 2:\n%s", len(lines), file.String())
	}

	if !strings.Contains(lines[0], "INFO") || !strings.HasSuffix(lines[0], "Connecting to example.org:21 over FTP") {
		t.Errorf("first line is %q", lines[0])
	}
	if !strings.Contains(lines[1], "ERROR") {
		t.Errorf("second line does not record the level: %q", lines[1])
	}

	// A timestamp has to be there for the file to be worth keeping.
	if !strings.HasPrefix(lines[0], "20") {
		t.Errorf("first line is not timestamped: %q", lines[0])
	}
}

// The panel keeps the last hundred entries. The file is written on the way in,
// so it keeps the whole session — which is the point of having one.
func TestTheFileKeepsMoreThanThePanelShows(t *testing.T) {
	var file bytes.Buffer
	log := NewLogPanel(&file)

	for i := 0; i < log.maxSize+10; i++ {
		log = log.Add("line", LogInfo)
	}

	if len(log.entries) != log.maxSize {
		t.Errorf("the panel holds %d entries, want %d", len(log.entries), log.maxSize)
	}

	written := strings.Count(file.String(), "\n")
	if written != log.maxSize+10 {
		t.Errorf("wrote %d lines, want %d", written, log.maxSize+10)
	}
}

// The panel used to signal level by color alone, invisible with NO_COLOR or
// to colorblind users. Every level must also carry a text label.
func TestLogEntryLevelIsReadableWithoutColor(t *testing.T) {
	cases := []struct {
		level LogLevel
		want  string
	}{
		{LogInfo, "INFO"},
		{LogSuccess, "OK"},
		{LogError, "ERROR"},
	}

	for _, c := range cases {
		entry := LogEntry{Message: "test", Level: c.level}
		out := renderLogEntry(entry, 80)
		if !strings.Contains(out, c.want) {
			t.Errorf("renderLogEntry(%v) = %q, want it to contain %q", c.level, out, c.want)
		}
	}
}

func TestLogWithoutAFileDoesNotPanic(t *testing.T) {
	log := NewLogPanel(nil)
	log = log.Add("nothing to write to", LogInfo)

	if len(log.entries) != 1 {
		t.Errorf("the panel holds %d entries, want 1", len(log.entries))
	}
}
