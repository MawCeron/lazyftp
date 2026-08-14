package shared

import (
	"bytes"
	"strings"
	"sync"
)

// LineBuffer is an io.Writer that accumulates complete lines until they are
// drained. Writes are safe from any goroutine.
//
// It exists because the Bubble Tea message channel is unbuffered: sending a
// message while the update loop is busy blocks until the loop is free again.
// Library logging happens during blocking calls made from inside that loop, so
// the lines are parked here and collected once the loop can accept them.
type LineBuffer struct {
	mu      sync.Mutex
	partial []byte
	lines   []string
}

func (b *LineBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.partial = append(b.partial, p...)
	for {
		i := bytes.IndexByte(b.partial, '\n')
		if i < 0 {
			break
		}
		if line := strings.TrimRight(string(b.partial[:i]), "\r"); line != "" {
			b.lines = append(b.lines, line)
		}
		b.partial = b.partial[i+1:]
	}

	return len(p), nil
}

// Drain returns the lines written since the last call and forgets them.
func (b *LineBuffer) Drain() []string {
	b.mu.Lock()
	defer b.mu.Unlock()

	lines := b.lines
	b.lines = nil
	return lines
}
