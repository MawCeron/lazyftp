package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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

// LoadTheme reads the named theme: the user's file if there is one, else the
// theme of that name that ships with lazyftp. Every failure, including a name
// that matches neither, comes back as an error whose text can go straight to
// the Log.
func LoadTheme(name string) (*theme.Theme, error) {
	path, err := ThemePath(name)
	if err != nil {
		return nil, err
	}
	t, err := theme.Load(path)
	switch {
	case err == nil:
		return &t, nil
	case !os.IsNotExist(err):
		return nil, fmt.Errorf("theme %q: %w", name, err)
	}

	t, found, err := theme.Builtin(name)
	if err != nil {
		return nil, fmt.Errorf("built-in theme %q: %w", name, err)
	}
	if !found {
		return nil, fmt.Errorf("theme %q not found: expected %s, and no built-in has that name (built-in: %s)",
			name, path, strings.Join(theme.BuiltinNames(), ", "))
	}
	return &t, nil
}
