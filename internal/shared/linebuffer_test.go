package shared

import (
	"reflect"
	"sync"
	"testing"
)

func TestLineBufferSplitsAcrossWrites(t *testing.T) {
	var b LineBuffer

	b.Write([]byte("220 wel"))
	if got := b.Drain(); len(got) != 0 {
		t.Fatalf("incomplete line drained early: %q", got)
	}

	b.Write([]byte("come\n331 need password\n"))
	want := []string{"220 welcome", "331 need password"}
	if got := b.Drain(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLineBufferDrainForgets(t *testing.T) {
	var b LineBuffer

	b.Write([]byte("230 logged in\n"))
	b.Drain()

	if got := b.Drain(); len(got) != 0 {
		t.Fatalf("second drain returned %q, want nothing", got)
	}
}

func TestLineBufferTrimsCarriageReturnAndBlanks(t *testing.T) {
	var b LineBuffer

	b.Write([]byte("227 entering passive mode\r\n\n500 oops\r\n"))

	want := []string{"227 entering passive mode", "500 oops"}
	if got := b.Drain(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// The FTP client logs from its own goroutines while the update loop drains.
func TestLineBufferConcurrentWrites(t *testing.T) {
	var b LineBuffer
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Write([]byte("200 ok\n"))
		}()
	}
	wg.Wait()

	if got := b.Drain(); len(got) != 50 {
		t.Fatalf("got %d lines, want 50", len(got))
	}
}
