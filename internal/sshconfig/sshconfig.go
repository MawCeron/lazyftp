package sshconfig

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MawCeron/lazyftp/internal/config"
	"github.com/kevinburke/ssh_config"
)

const maxIncludeDepth = 5

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

// Load offers every concrete Host alias in an ssh_config file as an SFTP
// connection. Values are resolved by the parser, so wildcard blocks and
// first-value-wins apply as they do for ssh itself. A missing file is not an
// error, and the file is only ever read.
func Load(path string) ([]config.Connection, error) {
	cfg, err := decodeFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var out []config.Connection
	seen := map[string]bool{}
	for _, alias := range aliases(cfg, 0) {
		if seen[alias] {
			continue
		}
		seen[alias] = true

		get := func(key string) string {
			v, _ := cfg.Get(alias, key)
			return v
		}
		host := get("HostName")
		if host == "" {
			host = alias
		}
		port, err := strconv.Atoi(get("Port"))
		if err != nil || port <= 0 {
			port = 22
		}
		out = append(out, config.Connection{
			Name:         alias,
			Host:         host,
			User:         get("User"),
			Port:         port,
			Protocol:     "SFTP",
			IdentityFile: get("IdentityFile"),
		})
	}
	return out, nil
}

func decodeFile(path string) (*ssh_config.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ssh_config.Decode(f)
}

// aliases lists the Host names that stand for one server, descending into
// Include. The parser resolves values through includes but does not list what
// they define, so their files are read again here.
// ponytail: Include arguments are split on whitespace, so quoted paths with spaces are missed.
func aliases(cfg *ssh_config.Config, depth int) []string {
	var out []string
	for _, h := range cfg.Hosts {
		for _, p := range h.Patterns {
			if name := p.String(); !strings.ContainsAny(name, "*?!") {
				out = append(out, name)
			}
		}
		if depth >= maxIncludeDepth {
			continue
		}
		for _, n := range h.Nodes {
			inc, ok := n.(*ssh_config.Include)
			if !ok {
				continue
			}
			for _, file := range includedFiles(inc.String()) {
				if sub, err := decodeFile(file); err == nil {
					out = append(out, aliases(sub, depth+1)...)
				}
			}
		}
	}
	return out
}

// includedFiles resolves an "Include a b" line the way ssh does: relative
// paths live under ~/.ssh, and globs expand.
func includedFiles(line string) []string {
	if i := strings.Index(line, "#"); i >= 0 {
		line = line[:i]
	}
	fields := strings.Fields(strings.Replace(line, "=", " ", 1))
	if len(fields) < 2 {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var files []string
	for _, arg := range fields[1:] {
		switch {
		case strings.HasPrefix(arg, "~/"):
			arg = filepath.Join(home, arg[2:])
		case !filepath.IsAbs(arg):
			arg = filepath.Join(home, ".ssh", arg)
		}
		matches, _ := filepath.Glob(arg)
		files = append(files, matches...)
	}
	return files
}
