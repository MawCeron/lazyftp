package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MawCeron/lazyftp/internal/ui"
)

// Set by the linker from the tag being built. A build from source keeps "dev".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	verbose := flag.Bool("verbose", false, "log the FTP control dialogue to the Log panel")
	logFile := flag.String("log-file", "", "also write the log to this file, appending to it")
	noNerdFonts := flag.Bool("no-nerd-fonts", false, "use plain Unicode symbols instead of Nerd Font icons")
	flag.Parse()

	if *showVersion {
		fmt.Println("lazyftp", version)
		return
	}

	ui.SetNerdFonts(!*noNerdFonts)

	// A typed nil would satisfy the io.Writer interface and be written to.
	var logWriter io.Writer
	if *logFile != "" {
		// --log-file takes a value, so a flag written after it becomes the
		// filename and silently disables itself.
		if strings.HasPrefix(*logFile, "-") {
			fmt.Fprintf(os.Stderr, "error: --log-file needs a filename, got %q\n", *logFile)
			os.Exit(1)
		}

		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot write to %s: %v\n", *logFile, err)
			os.Exit(1)
		}
		defer f.Close()

		// The file is appended to, so runs would otherwise run together.
		fmt.Fprintf(f, "\n%s ---- lazyftp started ----\n", time.Now().Format(time.RFC3339))
		logWriter = f
	}

	var p *tea.Program
	app := ui.NewApp(func() *tea.Program { return p }, *verbose, logWriter, version)
	p = tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
