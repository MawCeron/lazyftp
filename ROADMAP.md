# Roadmap

Where lazyftp is going and in what order. Each release is meant to be short and shippable
rather than a grab-bag; connectivity and stability come first, the interface second, and
features on top of solid ground.

Every item is tracked by an issue. Progress lives in the
[milestones](https://github.com/MawCeron/lazyftp/milestones).

**Current release:** v0.1.1

---

## v0.1.2 — FTP connectivity and stability

Plain FTP on port 21 does not reliably connect. This release makes the failure visible
before trying to fix it, and clears the crashes that make any failure worse.

Today `goftp.Config` is built with only `User`, `Password` and a hardcoded 10s `Timeout`,
leaving every connectivity knob at its default — including `Logger`, which is the one that
would explain what is actually happening on the wire.

- [ ] [#12](https://github.com/MawCeron/lazyftp/issues/12) **FTP protocol log in the Log panel.** The equivalent of FileZilla's message
      log. Passwords are not logged by the library. Highest value, lowest cost item here.
- [ ] [#11](https://github.com/MawCeron/lazyftp/issues/11) **Diagnose the port 21 failure** against a real server using that log. Leading
      hypothesis is EPSV: goftp's own `DisableEPSV` documentation describes the exact
      symptom — *"EPSV connections neither complete nor downgrade to PASV successfully by
      themselves, resulting in hung connections."*
- [ ] [#13](https://github.com/MawCeron/lazyftp/issues/13) Expose the `goftp.Config` knobs currently left at defaults.
- [ ] [#14](https://github.com/MawCeron/lazyftp/issues/14) **Explicit protocol selector** (FTP / FTPS / SFTP) instead of inferring it from
      the port. Today SFTP on a non-standard port is impossible.
- [ ] [#15](https://github.com/MawCeron/lazyftp/issues/15) **Asynchronous `Connect` and `List`.** They run synchronously inside the update
      loop today, so a connection timeout freezes the interface with no feedback.
- [ ] [#16](https://github.com/MawCeron/lazyftp/issues/16) **Fix the Windows navigation crash.** Going up from a Windows local path panics.
- [ ] [#17](https://github.com/MawCeron/lazyftp/issues/17) **Recover from panics in transfer goroutines.** They run outside Bubble Tea's own
      panic handling, so a failure there takes the process down with the terminal still in
      raw mode.
- [ ] [#18](https://github.com/MawCeron/lazyftp/issues/18) Guard against negative widths on very narrow terminals.
- [ ] [#19](https://github.com/MawCeron/lazyftp/issues/19) Fix the visible-rows calculation in the Processes panel.
- [ ] [#20](https://github.com/MawCeron/lazyftp/issues/20) **CI**: build, vet and test on push and pull request.
- [ ] [#21](https://github.com/MawCeron/lazyftp/issues/21) Fix the malformed `v.0.1.1` tag and publish releases with prebuilt binaries.
- [ ] [#22](https://github.com/MawCeron/lazyftp/issues/22) Remove roughly 145 lines of dead code.
- [ ] [#54](https://github.com/MawCeron/lazyftp/issues/54) **The progress wrapper disables upload resume.** goftp resumes interrupted
      transfers on its own, but only when the source is seekable — which the wrapper is not, so
      a large upload that drops starts again from zero.
- [ ] [#55](https://github.com/MawCeron/lazyftp/issues/55) Open the local panel in the working directory instead of the home directory.

The `client.Client` interface makes a fake straightforward, so the first tests land here.

## v0.2.0 — TUI overhaul

The interface has never had a design pass. Measured against the terminal-UI checklist it
fails on several counts, the worst of which is quantifiable: **at the conventional 80×24
floor the file panels show two entries.** Fixed vertical budgets leave the panel eight rows,
and six of those go to chrome.

- [ ] [#23](https://github.com/MawCeron/lazyftp/issues/23) **An explicit degradation plan across terminal sizes**, down to a "terminal too
      small" message instead of a broken render.
- [ ] [#24](https://github.com/MawCeron/lazyftp/issues/24) **Connection bar becomes an overlay.** Five permanently bordered rows — a fifth of
      an 80×24 screen — are spent on a form used once per session.
- [ ] [#25](https://github.com/MawCeron/lazyftp/issues/25) **Semantic color tokens.** Eleven raw palette codes across six files, with styles
      rebuilt on every row of every frame.
- [ ] [#26](https://github.com/MawCeron/lazyftp/issues/26) **Help screen on `?`**, and a footer trimmed to the keys that fit in 80 columns.
- [ ] [#27](https://github.com/MawCeron/lazyftp/issues/27) **Truncate by display width, not byte length** — accented filenames currently
      split mid-character.
- [ ] [#28](https://github.com/MawCeron/lazyftp/issues/28) **Separate the cursor from the selection mark.** A file that is both marked and
      selected is distinguishable by background color alone.
- [ ] [#29](https://github.com/MawCeron/lazyftp/issues/29) **Size and date columns.** Both are already collected for every file and never shown.
- [ ] [#31](https://github.com/MawCeron/lazyftp/issues/31) **Fuzzy filter on `/`** with a match counter.
- [ ] [#30](https://github.com/MawCeron/lazyftp/issues/30) **Sort by name, size or date** with a column indicator.
- [ ] [#32](https://github.com/MawCeron/lazyftp/issues/32) Make the Log panel focusable and scrollable.
- [ ] [#33](https://github.com/MawCeron/lazyftp/issues/33) Jump to a typed path.
- [ ] [#34](https://github.com/MawCeron/lazyftp/issues/34) Copy the current path to the clipboard, surviving SSH and tmux.
- [ ] [#52](https://github.com/MawCeron/lazyftp/issues/52) Mark entries present on only one side, by name — comparing sizes or timestamps is
      not reliable over plain FTP.

## v0.3.0 — File operations

These four share one inline-input pattern — `Enter` confirms, `Esc` cancels, the panel
refreshes — so the pattern gets built once on top of the reworked interface.

- [ ] [#7](https://github.com/MawCeron/lazyftp/issues/7) Rename files and directories
- [ ] [#8](https://github.com/MawCeron/lazyftp/issues/8) Delete files and directories
- [ ] [#9](https://github.com/MawCeron/lazyftp/issues/9) Create directories
- [ ] [#5](https://github.com/MawCeron/lazyftp/issues/5) Toggle hidden files
- [ ] [#35](https://github.com/MawCeron/lazyftp/issues/35) **Recursive directory download.** Recursion only exists for uploads today; this is
      the symmetric half of [#1](https://github.com/MawCeron/lazyftp/issues/1).

Requires extending the client interface with rename and remove operations.

## v0.4.0 — Connections and authentication

- [ ] [#36](https://github.com/MawCeron/lazyftp/issues/36) **Configuration layer** — the shared foundation for saved connections, history,
      bookmarks and themes. Credentials policy: **no plaintext passwords**.
- [ ] [#4](https://github.com/MawCeron/lazyftp/issues/4) Save and manage favorite connections
- [ ] [#3](https://github.com/MawCeron/lazyftp/issues/3) Connection history with quick reconnect
- [ ] [#37](https://github.com/MawCeron/lazyftp/issues/37) **SSH key and ssh-agent authentication.** Password auth is the only option today,
      so key-only servers cannot be reached at all.
- [ ] [#38](https://github.com/MawCeron/lazyftp/issues/38) **Host key verification**, removing the current accept-anything placeholder.
- [ ] [#53](https://github.com/MawCeron/lazyftp/issues/53) **Read connections from `~/.ssh/config`** — servers the user already maintains for
      `ssh` and `scp`, available with nothing to configure. Suggested by a user in #4.
- [ ] [#39](https://github.com/MawCeron/lazyftp/issues/39) Connect from the command line.
- [ ] [#41](https://github.com/MawCeron/lazyftp/issues/41) Recover a dropped SFTP session. FTP already survives idle timeouts through goftp's
      connection pool; SFTP holds a single session with no equivalent.
- [ ] [#42](https://github.com/MawCeron/lazyftp/issues/42) Loadable community themes, on top of the tokens from #25.

## v0.5.0 — Transfer queue and permissions

- [ ] [#6](https://github.com/MawCeron/lazyftp/issues/6) Remember pending transfers across restarts, offering them back as new transfers.
      Not a resumable queue: resuming is goftp's job once #54 stops preventing it.
- [ ] [#10](https://github.com/MawCeron/lazyftp/issues/10) File permissions management
- [ ] [#43](https://github.com/MawCeron/lazyftp/issues/43) **Transfer IDs.** Progress is matched by filename, so identically named files in
      different directories overwrite each other. Prerequisite for the queue.
- [ ] [#44](https://github.com/MawCeron/lazyftp/issues/44) **Cancellable transfers and a concurrency limit.** Transfers spawn unbounded
      goroutines with no way to abort them.
- [ ] [#45](https://github.com/MawCeron/lazyftp/issues/45) **Overwrite protection** — existing files are clobbered silently today.
- [ ] [#46](https://github.com/MawCeron/lazyftp/issues/46) Transfer speed and estimated time.
- [ ] [#47](https://github.com/MawCeron/lazyftp/issues/47) Confirm before quitting with transfers in flight.

## v0.6.0 — Productivity

- [ ] [#48](https://github.com/MawCeron/lazyftp/issues/48) **Edit a remote file in place** with `$EDITOR`, uploading again on save.
- [ ] [#49](https://github.com/MawCeron/lazyftp/issues/49) Preview file contents, with binary detection.

## Someday

Ideas worth keeping but outside what lazyftp is for, or needing groundwork the project does
not have. Nothing here blocks a release.

- [ ] [#51](https://github.com/MawCeron/lazyftp/issues/51) One-way directory mirror. Deciding what "differs" needs guarantees plain FTP does
      not give — no checksums, and timestamps unreliable enough that goftp ships a setting to
      compensate. A mirror that gets it wrong silently skips a changed file. rsync and lftp
      already solve this from the same terminal.

## Toward v1.0

After v0.6.0 the goal is to stabilize the configuration format and the key bindings so they
can be committed to, and tag v1.0.0.
