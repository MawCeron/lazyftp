package ui

import "testing"

func TestNerdFontsToggleFallsBackToPlainUnicode(t *testing.T) {
	t.Cleanup(func() { SetNerdFonts(true) })

	SetNerdFonts(true)
	nf := []string{iconMark(), iconDone(), iconError(), iconUpload(), iconDownload()}

	SetNerdFonts(false)
	plain := []string{iconMark(), iconDone(), iconError(), iconUpload(), iconDownload()}

	for i := range nf {
		if nf[i] == plain[i] {
			t.Errorf("icon %d: Nerd Font and plain fallback are both %q, want distinct glyphs", i, nf[i])
		}
		if plain[i] == "" || nf[i] == "" {
			t.Errorf("icon %d: got empty glyph (nf=%q, plain=%q)", i, nf[i], plain[i])
		}
	}
}
