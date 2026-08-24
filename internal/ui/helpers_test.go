package ui

import (
	"strings"
	"testing"
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
