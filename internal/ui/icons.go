package ui

// On by default (lazyftp targets terminal power users, most of whom already
// run a patched font); --no-nerd-fonts falls back to plain Unicode for
// terminals without one. Codepoints are Font Awesome glyphs, whose range is
// unchanged between Nerd Fonts v2 and v3 — no version pinning needed.
var nerdFonts = true

func SetNerdFonts(enabled bool) {
	nerdFonts = enabled
}

func iconMark() string {
	if nerdFonts {
		return "" // nf-fa-check
	}
	return "✓"
}

func iconDone() string {
	if nerdFonts {
		return "" // nf-fa-check
	}
	return "✔"
}

func iconError() string {
	if nerdFonts {
		return "" // nf-fa-times
	}
	return "✗"
}

func iconUpload() string {
	if nerdFonts {
		return "" // nf-fa-arrow_up
	}
	return "↑"
}

func iconDownload() string {
	if nerdFonts {
		return "" // nf-fa-arrow_down
	}
	return "↓"
}
