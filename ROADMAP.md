# Roadmap

Where lazyftp is going and in what order. Each release is meant to be short and shippable
rather than a grab-bag; connectivity and stability come first, the interface second, and
features on top of solid ground.

Items linked to an issue have one. Items without a link are planned but not filed yet.

**Current release:** v0.1.1

---

## v0.1.2 — FTP connectivity and stability

Plain FTP on port 21 does not reliably connect. This release makes the failure visible
before trying to fix it, and clears the crashes that make any failure worse.

Today `goftp.Config` is built with only `User`, `Password` and a hardcoded 10s `Timeout`,
leaving every connectivity knob at its default — including `Logger`, which is the one that
would explain what is actually happening on the wire.

- [ ] **FTP protocol log in the Log panel.** `goftp.Config.Logger` accepts an `io.Writer`;
      wiring it to the existing log panel gives the equivalent of FileZilla's message log.
      Passwords are not logged by the library. Highest value, lowest cost item here.
- [ ] Diagnose the port 21 failure against a real server using that log. Leading hypothesis
      is EPSV: goftp's own `DisableEPSV` documentation describes the exact symptom — *"EPSV
      connections neither complete nor downgrade to PASV successfully by themselves,
      resulting in hung connections."*
- [ ] Expose the `goftp.Config` knobs currently left at defaults: `DisableEPSV`,
      `ActiveTransfers` / `ActiveListenAddr`, `TLSConfig` / `TLSMode`, `ConnectionsPerHost`,
      configurable `Timeout`, `ServerLocation`.
- [ ] **Explicit protocol selector** (FTP / FTPS / SFTP) instead of inferring it from the
      port. Today SFTP on a non-standard port is impossible, and FTP is chosen by accident.
- [ ] **Asynchronous `Connect` and `List`.** They currently run synchronously inside the
      update loop, so a connection timeout freezes the whole interface with no feedback.
      Replace with commands plus a spinner, elapsed time, and `Esc` to cancel.
- [ ] **Fix the Windows navigation crash.** Going up a directory from a Windows local path
      panics, because parent-path resolution assumes `/` separators.
- [ ] **Restore the terminal on panic.** Without a recover handler, any crash leaves the
      terminal in raw mode and on the alternate screen.
- [ ] Guard against negative widths when the terminal is very narrow.
- [ ] Fix the visible-rows calculation in the Processes panel, where operator precedence
      makes the expression mean something other than intended.
- [ ] **CI**: build, vet and test on push and pull request.
- [ ] Fix the malformed `v.0.1.1` tag and publish releases with prebuilt binaries.
- [ ] Remove dead code — roughly 215 unreachable lines, including an entire unused hints
      module, unused client interface methods, and struct fields that are populated but
      never read.

The `client.Client` interface makes a fake straightforward, so the first tests land here.

## v0.2.0 — TUI overhaul

The interface has never had a design pass. Measured against the terminal-UI checklist it
fails on several counts, the worst of which is quantifiable: **at the conventional 80×24
floor the file panels show two entries.** Fixed vertical budgets leave the panel eight rows,
and six of those go to chrome.

- [ ] **An explicit degradation plan across terminal sizes.** Full layout above 120 columns;
      baseline at 80–120; a single switchable panel below 80 instead of two 26-column ones;
      a clear "terminal too small" message below 60×20 rather than a crash.
- [ ] **Connection bar becomes an overlay.** Five permanently bordered rows — a fifth of an
      80×24 screen — are spent on a form used once per session. Summon it with `Ctrl+L` and
      collapse it to a one-line status when connected: `● user@host:21 (FTP)`.
- [ ] **Semantic color tokens.** Eleven raw 256-palette codes are scattered across six
      files, with styles rebuilt on every row of every frame. One theme module fixes both
      the theming and the per-frame allocations.
- [ ] **Help screen on `?`**, and a footer trimmed to the four or five most useful keys for
      the focused panel. The current hint bar renders about 115 characters into 80 columns.
- [ ] **Truncate by display width, not byte length.** The transfer and log panels slice
      strings by bytes, which splits accented characters mid-rune and produces mojibake.
- [ ] **Separate the cursor from the selection mark.** A file that is both marked and
      selected loses its `✓` and becomes distinguishable by background color alone —
      invisible to colorblind users and in monochrome.
- [ ] **Size and date columns.** Both are already collected for every file and never shown.
      Numerics right-aligned, dates fixed-width ISO-8601.
- [ ] **Fuzzy filter on `/`** with a `12/340` match counter. Filtering is explicitly
      disabled today even though the dependencies for it are already present.
- [ ] **Sort by name, size or date** with a `▲`/`▼` column indicator.
- [ ] Make the Log panel focusable and scrollable — only the tail is reachable today.
- [ ] Jump to a typed path (`g`).
- [ ] Copy the current path to the clipboard via OSC 52, so it survives SSH and tmux.

## v0.3.0 — File operations

These four share one inline-input pattern — `Enter` confirms, `Esc` cancels, the panel
refreshes — so the pattern gets built once on top of the reworked interface.

- [ ] [#7](https://github.com/MawCeron/lazyftp/issues/7) Rename files and directories (`r`)
- [ ] [#8](https://github.com/MawCeron/lazyftp/issues/8) Delete files and directories (`d`, with confirmation)
- [ ] [#9](https://github.com/MawCeron/lazyftp/issues/9) Create directories (`Ctrl+N`)
- [ ] [#5](https://github.com/MawCeron/lazyftp/issues/5) Toggle hidden files (`Ctrl+H`)
- [ ] **Recursive directory download.** Recursion only exists for uploads; downloading a
      directory falls through and fails. This is the symmetric half of
      [#1](https://github.com/MawCeron/lazyftp/issues/1).

Requires extending the client interface with rename and remove operations.

## v0.4.0 — Connections and authentication

- [ ] **Configuration layer** under the platform config directory — the shared foundation
      for saved connections and history. Credentials policy: **no plaintext passwords**;
      store host, user, port and protocol only, and delegate secrets to SSH keys or the OS
      keyring.
- [ ] [#4](https://github.com/MawCeron/lazyftp/issues/4) Save and manage favorite connections
- [ ] [#3](https://github.com/MawCeron/lazyftp/issues/3) Connection history with quick reconnect
- [ ] **SSH key and ssh-agent authentication.** Password auth is the only option today.
- [ ] **Host key verification against `known_hosts`**, removing the current
      `InsecureIgnoreHostKey` placeholder.
- [ ] Command-line arguments: `lazyftp user@host:21` and `lazyftp -p <favorite>`.
- [ ] Local directory bookmarks, persisted alongside the rest of the configuration.
- [ ] Automatic reconnection with backoff, showing a degraded view rather than a blank one.
- [ ] Loadable community themes (Catppuccin, Gruvbox, Nord). The semantic tokens arrive in
      v0.2.0; here they gain a theme file, so adding a theme is a file and not a code change.

## v0.5.0 — Transfer queue and permissions

- [ ] [#6](https://github.com/MawCeron/lazyftp/issues/6) Persistent transfer queue
- [ ] [#10](https://github.com/MawCeron/lazyftp/issues/10) File permissions management
- [ ] **Transfer IDs.** Progress is matched by filename, so identically named files in
      different directories overwrite each other in the Processes panel. Prerequisite for
      the queue.
- [ ] **Cancellable transfers and a concurrency limit.** Transfers spawn unbounded
      goroutines with no way to abort them.
- [ ] **Overwrite protection** — existing files are currently clobbered silently.
- [ ] Transfer speed and ETA.
- [ ] Confirm before quitting with transfers in flight.

## v0.6.0 — Productivity

- [ ] **Edit a remote file with `$EDITOR`** (`e`): fetch to a temporary file, open the
      editor, upload again on save if it changed.
- [ ] Preview files with `v`, with binary detection so nothing garbles the terminal.

## v0.7.0 — Advanced

- [ ] **Multiple simultaneous connections.** The most invasive change on this roadmap: the
      app holds a single client and a single transfer manager today, and this means a
      collection of sessions with tabs and per-session transfer routing. Deliberately last,
      since it wants the queue and the configuration layer settled first.
- [ ] One-way directory sync / mirror, with a preview of the changes before running them.
- [ ] Compare both panels, highlighting what differs — local only, remote only, or a
      different size or date.

## Toward v1.0

After v0.7.0 the goal is to stabilize the configuration format and the key bindings so they
can be committed to, and tag v1.0.0.
