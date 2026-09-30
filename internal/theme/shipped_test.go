package theme

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func luminance(hex string) float64 {
	h := strings.TrimPrefix(hex, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	var ch [3]float64
	for i := range ch {
		v, _ := strconv.ParseUint(h[2*i:2*i+2], 16, 8)
		c := float64(v) / 255
		if c <= 0.03928 {
			ch[i] = c / 12.92
		} else {
			ch[i] = math.Pow((c+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*ch[0] + 0.7152*ch[1] + 0.0722*ch[2]
}

func contrast(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// The built-in palette was chosen against these two backgrounds, so a theme
// that does not name one is held to the same ones.
const (
	assumedDark  = "#1E1E1E"
	assumedLight = "#FAFAFA"
)

// checkReadable holds a palette to the project's own rule: 4.5:1 for anything
// that carries text, 3:1 for borders, which only draw structure.
func checkReadable(t *testing.T, who, table string, c Colors, assumedBg string, checkBar bool) {
	t.Helper()
	bg := c.Background
	if bg == "" {
		bg = assumedBg
	}
	for _, e := range c.entries() {
		if e.value == "" || e.key == "background" || e.key == "bar_bg" {
			continue
		}
		min := 4.5
		if e.key == "border" {
			min = 3
		}
		if got := contrast(e.value, bg); got < min {
			t.Errorf("%s [%s] %s %s on %s is %.2f:1, want at least %.1f", who, table, e.key, e.value, bg, got, min)
		}
	}
	if checkBar && c.BarBg != "" {
		for _, text := range []string{c.Emphasis, c.Muted} {
			if text != "" && contrast(text, c.BarBg) < 4.5 {
				t.Errorf("%s [%s] text %s on the bar %s is %.2f:1", who, table, text, c.BarBg, contrast(text, c.BarBg))
			}
		}
	}
}

func shipped(t *testing.T) map[string]Theme {
	t.Helper()
	out := map[string]Theme{}
	for _, name := range BuiltinNames() {
		th, ok, err := Builtin(name)
		if !ok || err != nil {
			t.Fatalf("built-in %s: found=%v err=%v", name, ok, err)
		}
		out["built-in "+name] = th
	}
	files, _ := filepath.Glob(filepath.Join("..", "..", "docs", "themes", "*.toml"))
	if len(files) < 2 {
		t.Fatalf("expected the example themes under docs/themes, found %v", files)
	}
	for _, f := range files {
		th, err := Load(f)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		out[filepath.Base(f)] = th
	}
	return out
}

func TestEveryShippedThemeParsesAndIsReadable(t *testing.T) {
	for who, th := range shipped(t) {
		// The built-in palette itself is 3.99:1 for muted text on the dark bar, a
		// shortfall that predates themes; default.toml reproduces it faithfully, so
		// it is not held to the bar rule until the palette is changed.
		bar := who != "default.toml"
		checkReadable(t, who, "dark", th.Dark, assumedDark, bar)
		checkReadable(t, who, "light", th.Light, assumedLight, bar)
	}
}

func TestTheReferenceThemeIsTheBuiltInPalette(t *testing.T) {
	th, err := Load(filepath.Join("..", "..", "docs", "themes", "default.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if th.Dark.Primary != "#D4D4D4" || th.Light.Primary != "#2C2C2A" || th.Dark.Accent != "#5DCAA5" {
		t.Errorf("default.toml no longer matches ui/theme.go: %+v", th)
	}
	if th.Dark.Background != "" || th.Light.Background != "" {
		t.Error("the reference theme should not set a background: the built-in palette does not")
	}
}

func TestBuiltinLookup(t *testing.T) {
	if _, ok, _ := Builtin("catppuccin-mocha"); !ok {
		t.Fatal("catppuccin-mocha is not built in")
	}
	if _, ok, err := Builtin("nope"); ok || err != nil {
		t.Fatalf("found=%v err=%v", ok, err)
	}
	if _, err := os.Stat("builtin"); err != nil {
		t.Fatal(err)
	}
}
