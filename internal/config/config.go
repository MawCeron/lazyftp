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
	Name     string `toml:"name"`
	Host     string `toml:"host"`
	User     string `toml:"user"`
	Port     int    `toml:"port"`
	Protocol string `toml:"protocol"`

	// A path, not a secret: the key stays where it is.
	IdentityFile string `toml:"identity_file,omitempty"`
}

type Config struct {
	Connections []Connection `toml:"connections"`
}

// Path is os.UserConfigDir per platform: $XDG_CONFIG_HOME or ~/.config on
// Linux, %AppData% on Windows, ~/Library/Application Support on macOS.
func Path() (string, error) { return inConfigDir("config.toml") }

// HistoryPath is a file of its own: the app rewrites it after every
// connection, and config.toml is the one the user edits by hand.
func HistoryPath() (string, error) { return inConfigDir("history.toml") }

func inConfigDir(name string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lazyftp", name), nil
}

// Push puts c first, dropping an earlier copy of the same server and whatever
// falls past max. The name is ignored: history entries are servers, not favorites.
func Push(list []Connection, c Connection, max int) []Connection {
	c.Name = ""
	out := []Connection{c}
	for _, x := range list {
		x.Name = ""
		if x != c && len(out) < max {
			out = append(out, x)
		}
	}
	return out
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
