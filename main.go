package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MawCeron/lazyftp/internal/config"
	"github.com/MawCeron/lazyftp/internal/sshconfig"
	"github.com/MawCeron/lazyftp/internal/ui"
)

// Set by the linker from the tag being built. A build from source keeps "dev".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	verbose := flag.Bool("verbose", false, "log the FTP control dialogue to the Log panel; for SFTP, one line per request instead")
	logFile := flag.String("log-file", "", "also write the log to this file, appending to it")
	noNerdFonts := flag.Bool("no-nerd-fonts", false, "use plain Unicode symbols instead of Nerd Font icons")
	highlightDiff := flag.Bool("highlight-diff", false, "mark files present on only one side, comparing Local and Remote by name")
	identity := flag.String("identity", "", "SFTP private key `path` to try first")
	flag.StringVar(identity, "i", "", "shorthand for --identity")
	protocolFlag := flag.String("protocol", "", "`ftp`, ftps or sftp; by default the destination's scheme, else its port decides")
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Println("lazyftp", version)
		return
	}

	ui.SetNerdFonts(!*noNerdFonts)

	if flag.NArg() > 1 {
		usageError("expected at most one destination, got %d (flags go before it)", flag.NArg())
	}
	forced, err := config.ParseProtocolFlag(*protocolFlag)
	if err != nil {
		usageError("%v", err)
	}

	var cfg, history config.Config
	var sshHosts []config.Connection
	var cfgErr, historyErr, sshErr error
	cfgPath, cfgPathErr := config.Path()
	if cfgPathErr == nil {
		cfg, cfgErr = config.Load(cfgPath)
	}
	historyPath, historyPathErr := config.HistoryPath()
	if historyPathErr == nil {
		history, historyErr = config.Load(historyPath)
	}
	sshPath, sshPathErr := sshconfig.DefaultPath()
	if sshPathErr == nil {
		sshHosts, sshErr = sshconfig.Load(sshPath)
	}

	var target *config.Connection
	if flag.NArg() == 1 {
		c, err := config.Resolve(flag.Arg(0), forced, cfg.Connections, sshHosts)
		if err != nil {
			usageError("%v", err)
		}
		if *identity != "" {
			c.IdentityFile = *identity
		}
		target = &c
	}

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
	app := ui.NewApp(func() *tea.Program { return p }, *verbose, logWriter, version, *highlightDiff)
	if cfgPathErr == nil {
		app = app.WithConfig(cfgPath, cfg, cfgErr)
	}
	app = applyTheme(app, cfg.Theme)
	if historyPathErr == nil {
		app = app.WithHistory(historyPath, history, historyErr)
	}
	if sshPathErr == nil {
		app = app.WithSSHHosts(sshHosts, sshErr)
	}
	if target != nil {
		app = app.WithTarget(*target)
	}
	p = tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: lazyftp [flags] [destination]

destination is a saved connection (a favorite or a Host from ~/.ssh/config) or
[ftp|ftps|sftp://][user@]host[:port]. Without a scheme or --protocol the port
decides: 22 is SFTP, 990 is FTPS, anything else FTP. Passwords are never taken
from the command line; lazyftp asks for them.

flags:
`)
	flag.PrintDefaults()
}

func usageError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "lazyftp: "+format+"\n\n", args...)
	usage()
	os.Exit(2)
}

// themeNoticeWait is how long a theme that failed to load stays on the console
// before the interface takes the screen over.
const themeNoticeWait = 3 * time.Second

// applyTheme loads the theme named in config.toml. No name is the built-in
// palette. A name that cannot be loaded is the same palette too, but the reason
// goes to the console first, where it can be read, and only then does the
// interface start: once it does, a startup dialog covers the Log. The Log keeps
// the line as well.
func applyTheme(app ui.App, name string) ui.App {
	if name == "" {
		return app
	}
	th, err := config.LoadTheme(name)
	if err != nil {
		warnTheme(os.Stderr, err, themeNoticeWait)
	}
	return app.WithTheme(th, err)
}

func warnTheme(w io.Writer, err error, wait time.Duration) {
	fmt.Fprintf(w, "lazyftp: %v\nlazyftp: starting with the default theme in %s (Ctrl+C to quit)\n", err, wait)
	time.Sleep(wait)
}
