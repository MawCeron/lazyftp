package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const full = `
name = "example"

[dark]
primary = "#D4D4D4"
accent = "#5DCAA5"
bar_bg = "#282828"

[light]
primary = "#2C2C2A"
accent = "#0F6E56"
`

func TestParseReadsBothPalettes(t *testing.T) {
	th, err := Parse([]byte(full))
	if err != nil {
		t.Fatal(err)
	}
	if th.Name != "example" || th.Dark.Accent != "#5DCAA5" || th.Dark.BarBg != "#282828" || th.Light.Primary != "#2C2C2A" {
		t.Fatalf("%+v", th)
	}
	if th.Dark.Muted != "" {
		t.Error("a token the file leaves out should stay empty, meaning the built-in color")
	}
}

func TestAThemeMayDefineOnlyOnePalette(t *testing.T) {
	if _, err := Parse([]byte("[dark]\naccent = \"#fff\"\n")); err != nil {
		t.Fatalf("dark only: %v", err)
	}
	if _, err := Parse([]byte("[light]\naccent = \"#FFFFFF\"\n")); err != nil {
		t.Fatalf("light only: %v", err)
	}
}

func TestParseRejectsWhatWouldQuietlyDoNothing(t *testing.T) {
	cases := map[string]string{
		"not toml":       "[dark\naccent =",
		"no colors":      `name = "empty"`,
		"empty tables":   "[dark]\n[light]\n",
		"unknown token":  "[dark]\nacent = \"#fff\"\n",
		"unknown table":  "[darker]\naccent = \"#fff\"\n",
		"name not hex":   "[dark]\naccent = \"green\"\n",
		"hash missing":   "[dark]\naccent = \"5DCAA5\"\n",
		"wrong length":   "[light]\nerror = \"#12345\"\n",
		"not hex digits": "[dark]\nerror = \"#GGGGGG\"\n",
	}
	for name, src := range cases {
		if _, err := Parse([]byte(src)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestErrorsNameTheTableAndTheKey(t *testing.T) {
	_, err := Parse([]byte("[light]\nborder = \"blue\"\n"))
	if err == nil || !strings.Contains(err.Error(), "[light] border") || !strings.Contains(err.Error(), "blue") {
		t.Fatalf("error %v", err)
	}
	_, err = Parse([]byte("[dark]\nacent = \"#fff\"\n"))
	if err == nil || !strings.Contains(err.Error(), "dark.acent") {
		t.Fatalf("typo not named: %v", err)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.toml")
	os.WriteFile(good, []byte(full), 0o644)
	if th, err := Load(good); err != nil || th.Name != "example" {
		t.Fatalf("%+v, %v", th, err)
	}

	bad := filepath.Join(dir, "bad.toml")
	os.WriteFile(bad, []byte("[dark]\naccent = \"x\"\n"), 0o644)
	if _, err := Load(bad); err == nil || !strings.Contains(err.Error(), "bad.toml") {
		t.Fatalf("the error should name the file: %v", err)
	}

	if _, err := Load(filepath.Join(dir, "missing.toml")); !os.IsNotExist(err) {
		t.Fatalf("a missing file should stay recognisable as one: %v", err)
	}
}
