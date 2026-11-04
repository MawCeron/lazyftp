package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestABrokenThemeIsExplainedOnTheConsoleBeforeTheDefaultLoads(t *testing.T) {
	var out bytes.Buffer
	start := time.Now()
	warnTheme(&out, errors.New(`theme "x": [dark] accent = "green" is not a color`), 50*time.Millisecond)

	for _, want := range []string{`theme "x"`, "not a color", "default theme", "50ms"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("console output is missing %q:\n%s", want, out.String())
		}
	}
	if time.Since(start) < 50*time.Millisecond {
		t.Error("it did not wait, so the message would be gone before anyone could read it")
	}
}

func TestTheLogFileIsNotReadableByOtherUsers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits do not carry the same meaning on Windows")
	}

	t.Run("a new log file is created 0600", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "lazyftp.log")

		f, err := openLogFile(path)
		if err != nil {
			t.Fatalf("openLogFile: %v", err)
		}
		f.Close()

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("new log file is %04o, want 0600", perm)
		}
	})

	t.Run("an existing world-readable log is tightened", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "lazyftp.log")
		if err := os.WriteFile(path, []byte("previous run\n"), 0o600); err != nil {
			t.Fatalf("seed the log: %v", err)
		}
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatalf("make the log world-readable: %v", err)
		}

		f, err := openLogFile(path)
		if err != nil {
			t.Fatalf("openLogFile: %v", err)
		}
		f.Close()

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("existing log file is %04o, want 0600", perm)
		}
	})
}
