package theme

import (
	"embed"
	"io/fs"
	"strings"
)

// The palettes that ship with lazyftp. A file of the same name in the user's
// themes directory takes their place.
//
//go:embed builtin/*.toml
var builtin embed.FS

// Builtin returns the shipped theme called name, and whether there is one.
func Builtin(name string) (Theme, bool, error) {
	data, err := builtin.ReadFile("builtin/" + name + ".toml")
	if err != nil {
		return Theme{}, false, nil
	}
	t, err := Parse(data)
	return t, true, err
}

func BuiltinNames() []string {
	entries, _ := fs.ReadDir(builtin, "builtin")
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = strings.TrimSuffix(e.Name(), ".toml")
	}
	return names
}
