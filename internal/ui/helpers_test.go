package ui

import (
	"strings"
	"testing"
	"unicode"
)

// The bug this guards against: Height(N) on the box is N's own total row
// count including the border, not the content area underneath it -- get
// that wrong and every panel renders one row short of what it was given.
func TestBorderWithTitleRendersExactlyTheRequestedHeight(t *testing.T) {
	for _, height := range []int{6, 8, 15, 24} {
		out := borderWithTitle("a\nb\nc", "Title", 30, height, colorAccent)
		if got := len(strings.Split(out, "\n")); got != height {
			t.Errorf("borderWithTitle(height=%d) rendered %d lines, want %d", height, got, height)
		}
	}
}

func TestFormatSize(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024*1024*1024*3 + 1024*1024*512, "3.5 GB"},
	}

	for _, c := range cases {
		if got := formatSize(c.n); got != c.want {
			t.Errorf("formatSize(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestTruncateHead(t *testing.T) {
	cases := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{name: "fits", s: "/var/www", width: 20, want: "/var/www"},
		{name: "ascii cut", s: "/var/www/html/public", width: 10, want: "…ml/public"},
		{name: "accented not split", s: "/home/José/año-2026", width: 10, want: "…/año-2026"},
		{name: "cjk counts double width", s: "/home/文档/相册", width: 8, want: "…档/相册"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := truncateHead(c.s, c.width)
			if got != c.want {
				t.Errorf("truncateHead(%q, %d) = %q, want %q", c.s, c.width, got, c.want)
			}
		})
	}
}

// What a server sends is text to show, never instructions to the terminal.
func TestPrintableLeavesNoControlSequenceStanding(t *testing.T) {
	hostile := map[string]string{
		"ESC":        "a\x1bb",
		"BEL":        "a\x07b",
		"CSI":        "a\x1b[2Jb",
		"OSC 52":     "a\x1b]52;c;ZXZpbA==\x07b",
		"8-bit CSI":  "a\u009b2Jb",
		"DEL":        "a\x7fb",
		"CR":         "a\rb",
		"NUL":        "a\x00b",
		"line break": "a\nb",
	}
	for what, in := range hostile {
		out := printable(in)
		for _, r := range out {
			if unicode.IsControl(r) {
				t.Errorf("%s: %q survived as %q", what, in, out)
				break
			}
		}
		if !strings.HasPrefix(out, "a") || !strings.HasSuffix(out, "b") {
			t.Errorf("%s: the text around the control character was lost: %q", what, out)
		}
	}
}

func TestPrintableKeepsOrdinaryText(t *testing.T) {
	for _, s := range []string{"report.pdf", "my file (1).txt", "résumé.pdf", "日本語", "emoji 🎉", ""} {
		if got := printable(s); got != s {
			t.Errorf("printable(%q) = %q, want it unchanged", s, got)
		}
	}
}

// hostileBytes are what a server-supplied name must never put on screen: the
// introducers of an OSC and a CSI sequence, BEL, an 8-bit CSI, NUL and CR.
// Lipgloss emits CSI sequences of its own for color, but never these.
var hostileBytes = []string{"\x1b]", "\x07", "\u009b", "\x1b[2J", "\x00", "\r"}

func requireNoHostileBytes(t *testing.T, what, out string) {
	t.Helper()
	for _, bad := range hostileBytes {
		if strings.Contains(out, bad) {
			t.Errorf("%s carries %q", what, bad)
		}
	}
}
