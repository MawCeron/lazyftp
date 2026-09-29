package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Connection holds everything about a server except its secret. There is no
// password field on purpose: secrets live in the keyring, see Secret.
type Connection struct {
	Host     string `toml:"host"`
	User     string `toml:"user"`
	Port     int    `toml:"port"`
	Protocol string `toml:"protocol"`
}

type Config struct {
	Connections []Connection `toml:"connections"`
}

// Path is os.UserConfigDir per platform: $XDG_CONFIG_HOME or ~/.config on
// Linux, %AppData% on Windows, ~/Library/Application Support on macOS.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lazyftp", "config.toml"), nil
}

// Load never fails the caller: a missing file is the first run, and a
// malformed one returns the defaults together with the parse error to log.
func Load(path string) (Config, error) {
	var c Config
	_, err := toml.DecodeFile(path, &c)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	return c, nil
}

// Save writes through a temporary file so a crash never leaves half a config.
func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".config-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := toml.NewEncoder(f).Encode(c); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
