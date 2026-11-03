package client

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// A cancelled transfer reads as a dead session to nothing: if SFTPClient.do
// took it for one, it would reopen the session and start the transfer over.
func TestACancelledSFTPTransferIsNotRetried(t *testing.T) {
	s := startServer(t)
	c, _ := s.connect(t, nil)
	src := filepath.Join(t.TempDir(), "big.bin")
	os.WriteFile(src, bytes.Repeat([]byte("x"), 1<<20), 0o644)
	sessions := s.sessions()

	ctx, cancel := context.WithCancel(context.Background())
	var calls int
	err := c.Upload(ctx, src, "/", func(int64) { calls++; cancel() })
	if !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled upload returned %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Errorf("progress was reported %d times, want 1: the upload went on or started over", calls)
	}

	if err := c.Upload(context.Background(), src, "/", func(int64) {}); err != nil {
		t.Fatalf("seeding the file to download: %v", err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	calls = 0
	err = c.Download(ctx, "/big.bin", t.TempDir(), func(int64) { calls++; cancel() })
	if !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled download returned %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Errorf("progress was reported %d times, want 1: the download went on or started over", calls)
	}

	if s.sessions() != sessions {
		t.Errorf("a cancelled transfer opened %d new session(s)", s.sessions()-sessions)
	}
}
