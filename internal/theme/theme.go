// Package theme reads color themes from TOML files: a palette for dark
// terminals and one for light ones, each naming any of the semantic tokens the
// interface draws with. It only parses and validates; applying a theme is the
// interface's business.
package theme

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// Colors holds one palette as hex strings. Every token is optional: one left
// empty keeps the built-in color, so a theme can change only what it cares about.
type Colors struct {
	Primary     string `toml:"primary"`
	Emphasis    string `toml:"emphasis"`
	Muted       string `toml:"muted"`
	Border      string `toml:"border"`
	Accent      string `toml:"accent"`
	Success     string `toml:"success"`
	Error       string `toml:"error"`
	Directory   string `toml:"directory"`
	Marked      string `toml:"marked"`
	BarBg       string `toml:"bar_bg"`
	DiffOnly    string `toml:"diff_only"`
	SizeDiffers string `toml:"size_differs"`
}

type Theme struct {
	Name  string `toml:"name"`
	Dark  Colors `toml:"dark"`
	Light Colors `toml:"light"`
}

var hexColor = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// Parse reads a theme and checks it all: an unknown key is an error, since it
// is almost always a typo that would otherwise quietly do nothing, and so is a
// color that is not #RGB or #RRGGBB.
func Parse(data []byte) (Theme, error) {
	var t Theme
	meta, err := toml.Decode(string(data), &t)
	if err != nil {
		return Theme{}, fmt.Errorf("not valid TOML: %w", err)
	}
	if un := meta.Undecoded(); len(un) > 0 {
		keys := make([]string, len(un))
		for i, k := range un {
			keys[i] = k.String()
		}
		return Theme{}, fmt.Errorf("unknown key(s): %s", strings.Join(keys, ", "))
	}

	if t.Dark.empty() && t.Light.empty() {
		return Theme{}, fmt.Errorf("defines no colors: expected a [dark] or a [light] table")
	}
	if err := t.Dark.validate("dark"); err != nil {
		return Theme{}, err
	}
	if err := t.Light.validate("light"); err != nil {
		return Theme{}, err
	}
	return t, nil
}

// Load reads and parses a theme file, naming the file in any error.
func Load(path string) (Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, err
	}
	t, err := Parse(data)
	if err != nil {
		return Theme{}, fmt.Errorf("%s: %w", path, err)
	}
	return t, nil
}

func (c Colors) entries() []struct{ key, value string } {
	return []struct{ key, value string }{
		{"primary", c.Primary}, {"emphasis", c.Emphasis}, {"muted", c.Muted},
		{"border", c.Border}, {"accent", c.Accent}, {"success", c.Success},
		{"error", c.Error}, {"directory", c.Directory}, {"marked", c.Marked},
		{"bar_bg", c.BarBg}, {"diff_only", c.DiffOnly}, {"size_differs", c.SizeDiffers},
	}
}

func (c Colors) empty() bool {
	for _, e := range c.entries() {
		if e.value != "" {
			return false
		}
	}
	return true
}

func (c Colors) validate(table string) error {
	for _, e := range c.entries() {
		if e.value != "" && !hexColor.MatchString(e.value) {
			return fmt.Errorf("[%s] %s = %q is not a color: use #RGB or #RRGGBB", table, e.key, e.value)
		}
	}
	return nil
}
