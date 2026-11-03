package shared

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

// goftp decides whether an upload can be resumed by asserting the source to an
// io.Seeker. This is that assertion, and it is the whole reason the wrapper has
// a Seek method.
func TestProgressReaderCanBeResumed(t *testing.T) {
	var r io.Reader = &ProgressReader{Reader: bytes.NewReader(nil)}

	if _, ok := r.(io.Seeker); !ok {
		t.Fatal("a wrapped upload cannot seek, so goftp will not resume it")
	}
}

func TestSeekMovesProgressToTheNewOffset(t *testing.T) {
	var reported int64
	r := &ProgressReader{
		Reader:   bytes.NewReader([]byte("0123456789")),
		Total:    10,
		Callback: func(n int64) { reported = n },
	}

	if _, err := io.ReadAll(io.LimitReader(r, 8)); err != nil {
		t.Fatalf("reading: %v", err)
	}
	if r.Current != 8 {
		t.Fatalf("after reading 8 bytes progress is %d, want 8", r.Current)
	}

	// What a resume looks like: the server holds 3 bytes, so the source goes
	// back to 3 and carries on from there.
	if _, err := r.Seek(3, io.SeekStart); err != nil {
		t.Fatalf("seeking: %v", err)
	}
	if r.Current != 3 {
		t.Errorf("progress is %d after seeking to 3, want 3", r.Current)
	}
	if reported != 3 {
		t.Errorf("progress reported as %d after seeking to 3, want 3", reported)
	}

	rest, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading the rest: %v", err)
	}
	if string(rest) != "3456789" {
		t.Errorf("read %q after the resume, want \"3456789\"", rest)
	}

	// The count has to land on the file's size. Carrying on from 8 would have
	// ended at 15 for a file of 10 bytes.
	if r.Current != 10 {
		t.Errorf("progress ended at %d for a 10 byte file, want 10", r.Current)
	}
}

func TestAFailedSeekLeavesProgressAlone(t *testing.T) {
	r := &ProgressReader{Reader: bytes.NewReader([]byte("0123456789"))}
	if _, err := io.ReadAll(io.LimitReader(r, 5)); err != nil {
		t.Fatalf("reading: %v", err)
	}

	if _, err := r.Seek(-1, io.SeekStart); err == nil {
		t.Fatal("seeking before the start was accepted")
	}
	if r.Current != 5 {
		t.Errorf("a failed seek moved progress to %d, want it left at 5", r.Current)
	}
}

func TestAProgressReaderStopsOnceItsContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &ProgressReader{Reader: strings.NewReader("abcdef"), Ctx: ctx}
	buf := make([]byte, 3)

	if n, err := r.Read(buf); n != 3 || err != nil {
		t.Fatalf("read before cancelling = %d, %v; want 3, nil", n, err)
	}
	cancel()

	n, err := r.Read(buf)
	if n != 0 || !errors.Is(err, context.Canceled) {
		t.Errorf("read after cancelling = %d, %v; want 0, context.Canceled", n, err)
	}
}

func TestAProgressWriterStopsOnceItsContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var dst bytes.Buffer
	w := &ProgressWriter{Writer: &dst, Ctx: ctx}

	if _, err := w.Write([]byte("abc")); err != nil {
		t.Fatalf("write before cancelling: %v", err)
	}
	cancel()

	n, err := w.Write([]byte("def"))
	if n != 0 || !errors.Is(err, context.Canceled) {
		t.Errorf("write after cancelling = %d, %v; want 0, context.Canceled", n, err)
	}
	if dst.String() != "abc" {
		t.Errorf("destination holds %q, want only the first write", dst.String())
	}
}

func TestProgressWrappersWithoutAContextKeepWorking(t *testing.T) {
	var dst bytes.Buffer
	r := &ProgressReader{Reader: strings.NewReader("abcdef")}
	w := &ProgressWriter{Writer: &dst}

	if _, err := io.Copy(w, r); err != nil || dst.String() != "abcdef" {
		t.Errorf("copy = %q, %v; want abcdef, nil", dst.String(), err)
	}
}
