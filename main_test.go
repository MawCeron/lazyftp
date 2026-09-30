package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestABrokenThemeIsExplainedOnTheConsoleBeforeTheDefaultLoads(t *testing.T) {
	var out bytes.Buffer
	start := time.Now()
	warnTheme(&out, errors.New(`theme "x": [dark] accent = "green" is not a color`), 50*time.Millisecond)

	for _, want := range []string{`theme "x"`, "not a color", "default theme", "50ms"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("console output is missing %q:\n%s", want, out.String())
		}
	}
	if time.Since(start) < 50*time.Millisecond {
		t.Error("it did not wait, so the message would be gone before anyone could read it")
	}
}
