package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/MawCeron/lazyftp/internal/ui"
)

func main() {
	var p *tea.Program
	app := ui.NewApp(func() *tea.Program { return p })
	p = tea.NewProgram(
		app,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
