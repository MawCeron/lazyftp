package ui

// On by default (lazyftp targets terminal power users, most of whom already
// run a patched font); --no-nerd-fonts falls back to plain Unicode for
// terminals without one. Codepoints are Font Awesome glyphs, whose range is
// unchanged between Nerd Fonts v2 and v3 — no version pinning needed.
var nerdFonts = true

func SetNerdFonts(enabled bool) {
	nerdFonts = enabled
}

// pick is the glyph for the current font setting: the Nerd Font one, or its
// plain Unicode fallback.
func pick(nerd, plain string) string {
	if nerdFonts {
		return nerd
	}
	return plain
}

func iconMark() string { return pick("", "✓") } // nf-fa-check

// iconChecked is a ticked checkbox; the unticked one is "[ ]" either way.
func iconChecked() string { return pick("["+iconMark()+"]", "[x]") }

func iconDone() string { return pick("", "✔") } // nf-fa-check

func iconError() string { return pick("", "✗") } // nf-fa-times

func iconUpload() string { return pick("", "↑") } // nf-fa-arrow_up

func iconDownload() string { return pick("", "↓") } // nf-fa-arrow_down

func iconUnique() string { return pick("", "!") } // nf-fa-exclamation

func iconSizeDiffers() string { return pick("", "M") } // nf-fa-exchange
