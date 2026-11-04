# Roadmap

Where lazyftp is going and in what order. Each release is meant to be short and shippable
rather than a grab-bag; connectivity and stability come first, the interface second, and
features on top of solid ground.

Every item is tracked by an issue. Progress lives in the
[milestones](https://github.com/MawCeron/lazyftp/milestones).

What has already shipped is in the [changelog](CHANGELOG.md).

---

## v0.2.0 — TUI overhaul

The interface has never had a design pass. Measured against the terminal-UI checklist it
fails on several counts, the worst of which is quantifiable: **at the conventional 80×24
floor the file panels show two entries.** Fixed vertical budgets leave the panel eight rows,
and six of those go to chrome.

- [#23](https://github.com/MawCeron/lazyftp/issues/23) **An explicit degradation plan across terminal sizes**, down to a "terminal too
  small" message instead of a broken render.
- [#24](https://github.com/MawCeron/lazyftp/issues/24) **Connection bar becomes an overlay.** Five permanently bordered rows — a fifth of
  an 80×24 screen — are spent on a form used once per session.
- [#25](https://github.com/MawCeron/lazyftp/issues/25) **Semantic color tokens.** Eleven raw palette codes across six files, with styles
  rebuilt on every row of every frame.
- [#26](https://github.com/MawCeron/lazyftp/issues/26) **Help screen on `?`**, and a footer trimmed to the keys that fit in 80 columns.
- [#27](https://github.com/MawCeron/lazyftp/issues/27) **Truncate by display width, not byte length** — accented filenames currently
  split mid-character.
- [#28](https://github.com/MawCeron/lazyftp/issues/28) **Separate the cursor from the selection mark.** A file that is both marked and
  selected is distinguishable by background color alone.
- [#29](https://github.com/MawCeron/lazyftp/issues/29) **Size and date columns.** Both are already collected for every file and never shown.
- [#31](https://github.com/MawCeron/lazyftp/issues/31) **Fuzzy filter on `/`** with a match counter.
- [#30](https://github.com/MawCeron/lazyftp/issues/30) **Sort by name, size or date** with a column indicator.
- [#32](https://github.com/MawCeron/lazyftp/issues/32) Make the Log panel focusable and scrollable.
- [#33](https://github.com/MawCeron/lazyftp/issues/33) Jump to a typed path.
- [#52](https://github.com/MawCeron/lazyftp/issues/52) Mark entries present on only one side, by name — comparing sizes or timestamps is
  not reliable over plain FTP.

## v0.3.0 — File operations

These four share one inline-input pattern — `Enter` confirms, `Esc` cancels, the panel
refreshes — so the pattern gets built once on top of the reworked interface.

- [#7](https://github.com/MawCeron/lazyftp/issues/7) Rename files and directories
- [#8](https://github.com/MawCeron/lazyftp/issues/8) Delete files and directories
- [#9](https://github.com/MawCeron/lazyftp/issues/9) Create directories
- [#5](https://github.com/MawCeron/lazyftp/issues/5) Toggle hidden files
- [#35](https://github.com/MawCeron/lazyftp/issues/35) **Recursive directory download.** Recursion only exists for uploads today; this is
  the symmetric half of [#1](https://github.com/MawCeron/lazyftp/issues/1).

Requires extending the client interface with rename and remove operations.

## v0.4.0 — Connections and authentication

- [#36](https://github.com/MawCeron/lazyftp/issues/36) **Configuration layer** — the shared foundation for saved connections, history,
  bookmarks and themes. Credentials policy: **no plaintext passwords**.
- [#4](https://github.com/MawCeron/lazyftp/issues/4) Save and manage favorite connections
- [#3](https://github.com/MawCeron/lazyftp/issues/3) Connection history with quick reconnect
- [#37](https://github.com/MawCeron/lazyftp/issues/37) **SSH key and ssh-agent authentication.** Password auth is the only option today,
  so key-only servers cannot be reached at all.
- [#38](https://github.com/MawCeron/lazyftp/issues/38) **Host key verification**, removing the current accept-anything placeholder.
- [#53](https://github.com/MawCeron/lazyftp/issues/53) **Read connections from `~/.ssh/config`** — servers the user already maintains for
  `ssh` and `scp`, available with nothing to configure. Suggested by a user in #4.
- [#39](https://github.com/MawCeron/lazyftp/issues/39) Connect from the command line.
- [#41](https://github.com/MawCeron/lazyftp/issues/41) Recover a dropped SFTP session. FTP already survives idle timeouts through goftp's
  connection pool; SFTP holds a single session with no equivalent.
- [#42](https://github.com/MawCeron/lazyftp/issues/42) Loadable community themes, on top of the tokens from #25.
- [#80](https://github.com/MawCeron/lazyftp/issues/80) Disconnect without quitting, and [#89](https://github.com/MawCeron/lazyftp/issues/89) log
  the old connection closing when connecting elsewhere.
- [#90](https://github.com/MawCeron/lazyftp/issues/90), [#91](https://github.com/MawCeron/lazyftp/issues/91) and
  [#92](https://github.com/MawCeron/lazyftp/issues/92) Log lines that confirm what worked and say what was
  attempted, and `--verbose` output for SFTP.
- [#93](https://github.com/MawCeron/lazyftp/issues/93) Keep the cursor on the same file when a panel reloads, and
  [#95](https://github.com/MawCeron/lazyftp/issues/95) a scrollable help screen.

## v0.4.2 — Security fixes

An audit of v0.4.1 found that a hostile server could make a download write outside its directory,
inject FTP commands, redirect the data connection and draw on the user's terminal. This release
closes those and updates the dependencies that had known vulnerabilities.

- [#98](https://github.com/MawCeron/lazyftp/issues/98) Remote names that would escape the download directory are skipped.
- [#99](https://github.com/MawCeron/lazyftp/issues/99) Control characters in server text are replaced before they are drawn or logged.
- [#100](https://github.com/MawCeron/lazyftp/issues/100) File names that could inject an FTP command are refused.
- [#101](https://github.com/MawCeron/lazyftp/issues/101) Dependencies updated, `govulncheck` in CI, weekly Dependabot.
- [#102](https://github.com/MawCeron/lazyftp/issues/102) Passive FTP connects to the host that was logged in to. This needed a change goftp
  does not have, so lazyftp moved to a fork of it.
- [#105](https://github.com/MawCeron/lazyftp/issues/105), [#106](https://github.com/MawCeron/lazyftp/issues/106) A saved password no longer outlives "remember" or the server it
  was for.
- [#107](https://github.com/MawCeron/lazyftp/issues/107) The log file is created readable only by the user.
- [#112](https://github.com/MawCeron/lazyftp/issues/112) A warning when the protocol is plain FTP.
- [#113](https://github.com/MawCeron/lazyftp/issues/113) A security policy.

## v0.4.3 — Hardening

- [#114](https://github.com/MawCeron/lazyftp/issues/114) Bidirectional text controls in file names could disguise a file's extension.
- Dependency updates, among them Bubble Tea 2.1.1, which carries a security fix for terminal size
  reports.

## v0.5.0 — Transfer queue and permissions

- [#6](https://github.com/MawCeron/lazyftp/issues/6) Remember pending transfers across restarts, offering them back as new transfers.
  Not a resumable queue: resuming is goftp's job, unblocked in v0.1.2 by #54.
- [#10](https://github.com/MawCeron/lazyftp/issues/10) Change permissions on remote files, through the inline input from v0.3.0. SFTP
  only: goftp has no chmod, and `SITE CHMOD` is outside the FTP standard.
- [#44](https://github.com/MawCeron/lazyftp/issues/44) **Cancellable transfers and a concurrency limit.** Transfers spawn unbounded
  goroutines with no way to abort them.
- [#45](https://github.com/MawCeron/lazyftp/issues/45) **Overwrite protection** — existing files are clobbered silently today.
- [#46](https://github.com/MawCeron/lazyftp/issues/46) Transfer speed and estimated time.
- [#47](https://github.com/MawCeron/lazyftp/issues/47) Confirm before quitting with transfers in flight.
- [#87](https://github.com/MawCeron/lazyftp/issues/87) Completion logs that show direction, size and duration.
- [#88](https://github.com/MawCeron/lazyftp/issues/88) Visibility into the transfer queue's pending work.
- [#94](https://github.com/MawCeron/lazyftp/issues/94) Open the remote panel at the server's login directory, not at `/`.
- [#103](https://github.com/MawCeron/lazyftp/issues/103) **Atomic writes.** A failed transfer truncates the file it was replacing; writing to
  a temporary sibling and renaming it keeps the old one until the new one is whole. It belongs with
  #45 and #44, which decide what a transfer may overwrite and how it ends.
- [#104](https://github.com/MawCeron/lazyftp/issues/104) Limits on the depth, entries and size of a recursive transfer, and a free-space
  check before a download.

## v0.6.0 — Interoperability

- [#108](https://github.com/MawCeron/lazyftp/issues/108) Implicit FTPS, which port 990 already promises and only explicit TLS delivers.
- [#109](https://github.com/MawCeron/lazyftp/issues/109) The rest of `ssh_config`: known_hosts paths, `ProxyJump` with one hop, and
  `ServerLocation`.
- [#110](https://github.com/MawCeron/lazyftp/issues/110) `keyboard-interactive` authentication for SFTP, for servers with PAM, one-time
  passwords or two-factor login.
- [#111](https://github.com/MawCeron/lazyftp/issues/111) Negotiate UTF-8 over FTP, so non-ASCII names do not depend on the server default.

## Someday

Ideas worth keeping but outside what lazyftp is for, or needing groundwork the project does
not have. Nothing here blocks a release.

- [#51](https://github.com/MawCeron/lazyftp/issues/51) One-way directory mirror. Deciding what "differs" needs guarantees plain FTP does
  not give — no checksums, and timestamps unreliable enough that goftp ships a setting to
  compensate. A mirror that gets it wrong silently skips a changed file. rsync and lftp
  already solve this from the same terminal.

Editing and previewing remote files were considered and dropped: lazyftp moves files, and the
editor and pager the user already has are better at reading and changing them.

## Toward v1.0

After v0.6.0 the goal is to stabilize the configuration format and the key bindings so they
can be committed to, and tag v1.0.0.
