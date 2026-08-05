package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/MawCeron/lazyftp/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	verbose := flag.Bool("verbose", false, "log the FTP control dialogue to the Log panel")
	logFile := flag.String("log-file", "", "also write the log to this file, appending to it")
	flag.Parse()

	// A typed nil would satisfy the io.Writer interface and be written to.
	var logWriter io.Writer
	if *logFile != "" {
		// --log-file takes a value, so the flag written after it is taken as the
		// filename: --log-file --verbose writes to a file called "--verbose" and
		// quietly drops the verbose logging that was asked for.
		if strings.HasPrefix(*logFile, "-") {
			fmt.Fprintf(os.Stderr, "error: --log-file needs a filename, got %q\n", *logFile)
			os.Exit(1)
		}

		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			// Reported now rather than discovered as an empty file afterwards.
			fmt.Fprintf(os.Stderr, "error: cannot write to %s: %v\n", *logFile, err)
			os.Exit(1)
		}
		defer f.Close()

		// The file is appended to, so runs would otherwise run together, and a
		// session that logs nothing would be indistinguishable from a flag that
		// did not work.
		fmt.Fprintf(f, "\n%s ---- lazyftp started ----\n", time.Now().Format(time.RFC3339))
		logWriter = f
	}

	var p *tea.Program
	app := ui.NewApp(func() *tea.Program { return p }, *verbose, logWriter)
	p = tea.NewProgram(
		app,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
