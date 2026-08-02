package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/MawCeron/lazyftp/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	verbose := flag.Bool("verbose", false, "log the FTP control dialogue to the Log panel")
	flag.Parse()

	var p *tea.Program
	app := ui.NewApp(func() *tea.Program { return p }, *verbose)
	p = tea.NewProgram(
		app,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
