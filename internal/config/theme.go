package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/MawCeron/lazyftp/internal/theme"
)

// A theme name becomes a file name, so it may not reach outside the directory.
var themeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ThemePath is where the theme called name lives: a themes directory next to
// config.toml, so adding a theme is dropping a file there.
func ThemePath(name string) (string, error) {
	if !themeName.MatchString(name) {
		return "", fmt.Errorf("%q is not a valid theme name: use letters, digits, '.', '_' and '-'", name)
	}
	return inConfigDir(filepath.Join("themes", name+".toml"))
}

// LoadTheme reads the named theme. Every failure, including a file that does
// not exist, comes back as an error whose text can go straight to the Log.
func LoadTheme(name string) (*theme.Theme, error) {
	path, err := ThemePath(name)
	if err != nil {
		return nil, err
	}
	t, err := theme.Load(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("theme %q not found: expected %s", name, path)
	}
	if err != nil {
		return nil, fmt.Errorf("theme %q: %w", name, err)
	}
	return &t, nil
}
