# Changelog

All notable changes to lazyftp are recorded here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.1] - 2026-08-31

Theme accuracy and connection-bar polish, plus fixes to how transfers and SFTP connections
recover from the unexpected.

### Added

- The interface now reacts live to the terminal's light/dark setting changing mid-session,
  instead of only resolving it once at startup.

### Fixed

- Both palettes are now truecolor hex, each value checked with WCAG contrast against a
  representative dark and light background; `colorMuted` (readable secondary text) and
  `colorBorder` (structural lines) no longer share one value tuned for neither role well.
  ([#76](https://github.com/MawCeron/lazyftp/issues/76))
- `colorMuted` on the light theme cleared WCAG AA by only 0.07:1 -- technically passing but too
  close to the line to read comfortably in practice. Darkened for a real margin.
- A terminal or multiplexer that never answers the background-color query (tmux without
  passthrough, some SSH paths) left the theme silently stuck on its dark default forever; it now
  falls back explicitly after 300ms, and a late real reply still overrides it.
- `q`/`Q` now quit from the connection bar's Protocol and Port fields -- neither can hold literal
  text, so both had nothing to lose the way Host/User/Pass do.
- The Port field now rejects anything but digits, instead of accepting arbitrary text that a
  later conversion to a number would silently fall back from.
- Processes rows were matched by filename alone, so re-transferring a file that shared a name
  with an earlier (possibly already-finished) row corrupted both progress bars. Each transfer
  now gets a unique ID, and a 0-byte transfer -- which could never trip the old "current reached
  total" threshold -- gets an explicit completion signal instead of sitting at "in progress"
  forever.
- An SFTP server that accepted the SSH handshake but never answered the SFTP subsystem request
  could hang the connection attempt forever; the connect timeout now covers that negotiation too.

## [0.2.0] - 2026-08-28

The TUI overhaul. The connection form no longer eats a fifth of the screen, panels are usable
down to the 80x24 floor, and file listings finally show more than a name: size, date, sort,
filter, and a diff view between both sides.

### Added

- The connection form is now an overlay summoned with `Ctrl+L`, instead of a permanently
  bordered panel. `Esc` dismisses it without disturbing panel state. Connected, a segmented
  status line shows protocol, user, host, connection state and, once anything is marked, a
  marked-file count -- visibility that previously meant opening Processes or counting checkmarks
  by eye. ([#24](https://github.com/MawCeron/lazyftp/issues/24),
  [#74](https://github.com/MawCeron/lazyftp/issues/74))
- The status line and footer render as solid, full-width bars instead of loose text on the
  terminal's own background, matching every other bordered element in the interface.
  ([#72](https://github.com/MawCeron/lazyftp/issues/72))
- The layout responds to terminal size instead of assuming a large window: the full two-panel
  view above 120 columns, a narrower baseline layout down to 80, a single Tab-switched panel
  below that, and a clear "terminal too small" message below 60x20 instead of a broken render.
  ([#23](https://github.com/MawCeron/lazyftp/issues/23))
- Panels show file size and modification date alongside the name, right-aligned and
  fixed-width, degrading gracefully as the panel narrows.
  ([#29](https://github.com/MawCeron/lazyftp/issues/29))
- Sort by name, size or modification date, reversible, with the active column and direction
  shown in the panel header. Directories still group first, and the choice is kept per panel
  for the session. ([#30](https://github.com/MawCeron/lazyftp/issues/30))
- Fuzzy filtering with `/`, with a live match counter (`12/340`) and `Esc` to clear -- thanks to
  [@OdaloV](https://github.com/OdaloV). ([#31](https://github.com/MawCeron/lazyftp/issues/31))
- Jump directly to a path by typing it with `:`, on either panel, using that panel's own path
  conventions. ([#33](https://github.com/MawCeron/lazyftp/issues/33))
- The Log and Processes panels can take focus and scroll through their full retained history,
  instead of only ever showing whatever fits on screen.
  ([#32](https://github.com/MawCeron/lazyftp/issues/32),
  [#75](https://github.com/MawCeron/lazyftp/issues/75))
- `--highlight-diff` marks, in both panels, entries that exist on only one side or that share a
  name but differ in size -- comparison by name and exact byte count only, nothing timestamp-based
  that plain FTP can't guarantee. Off by default. ([#52](https://github.com/MawCeron/lazyftp/issues/52))
- `U`/`D` upload or download whichever side has marked files, regardless of which panel has
  focus, alongside the existing focus-dependent `t`.
  ([#67](https://github.com/MawCeron/lazyftp/issues/67))
- A help screen (`?`) lists every keybinding grouped by context; the footer now shows only the
  handful most useful for the focused panel instead of eight hints fighting for 80 columns.
  ([#26](https://github.com/MawCeron/lazyftp/issues/26))
- Success and error log entries carry a label alongside their color, readable in monochrome or
  with `NO_COLOR` set. ([#64](https://github.com/MawCeron/lazyftp/issues/64))

### Changed

- Keybindings realigned to standard TUI convention: `Ctrl+C` now quits cleanly from anywhere,
  `Space` toggles a mark (replacing `x`), `h`/`l` are aliases for up/open instead of silently
  triggering the file list's own pagination, and `r` refreshes the focused panel.
  ([#66](https://github.com/MawCeron/lazyftp/issues/66))
- Migrated to Bubble Tea v2, Lipgloss v2 and Bubbles v2 -- the infrastructure the overlay
  connection dialog needed, since only v2's layer compositor can float a form over the panels
  without a full-view swap. No visible behavior change on its own.
  ([#65](https://github.com/MawCeron/lazyftp/issues/65))
- Colors moved from raw 256-palette codes scattered across six files to named semantic tokens,
  and per-row style objects are now built once instead of every frame. Groundwork for loadable
  themes; no visual change from this alone. ([#25](https://github.com/MawCeron/lazyftp/issues/25))

### Fixed

- A marked file that was also under the cursor lost its checkmark, because both shared one
  character slot -- marking was then only visible as a background color, invisible in
  monochrome or with `NO_COLOR` set. Cursor and mark now have their own columns.
  ([#28](https://github.com/MawCeron/lazyftp/issues/28))
- Filenames with accents or other multi-byte characters could be truncated mid-character in the
  Processes and Log panels, rendering a replacement character. Truncation and column alignment
  now go by display width everywhere, not byte length.
  ([#27](https://github.com/MawCeron/lazyftp/issues/27))
- A long path could wrap to two lines instead of truncating at the 80-column floor, misaligning
  a panel's bottom border against its neighbor -- content width was computed two cells short of
  what a bordered, padded box actually has available.
  ([#68](https://github.com/MawCeron/lazyftp/issues/68))
- The connection dialog let panel text behind it show through at its edges instead of owning a
  clean space; the panels now render blank while the dialog has focus.
  ([#69](https://github.com/MawCeron/lazyftp/issues/69))
- **Every panel rendered one row short of the height it was given**, which pushed the footer up
  off the true bottom of the terminal in normal view. `Height(N)` on a bordered box is the box's
  total row count, not the content area on top of its border.
  ([#73](https://github.com/MawCeron/lazyftp/issues/73))

## [0.1.2] - 2026-08-13

The release that makes plain FTP work. Connecting to an FTP server on port 21 was not possible,
and the surrounding crashes and silences made every failure harder to understand than it needed
to be.

### Added

- Protocol selector in the connection bar — FTP, FTPS or SFTP, cycled with `←` / `→`. SFTP on a
  non-standard port is now reachable. ([#14](https://github.com/MawCeron/lazyftp/issues/14))
- FTPS support over explicit TLS. Certificates are verified, so a server with a self-signed
  certificate is refused rather than silently accepted.
  ([#14](https://github.com/MawCeron/lazyftp/issues/14))
- `--verbose` writes the FTP control dialogue to the Log panel. Passwords are masked.
  ([#12](https://github.com/MawCeron/lazyftp/issues/12))
- `--log-file <path>` writes the log to a file as well, so it can be searched, kept or attached
  to a bug report. ([#56](https://github.com/MawCeron/lazyftp/issues/56))
- A connection attempt now shows a spinner and the elapsed time, and `Esc` hands the interface
  back without waiting for it to finish.
  ([#15](https://github.com/MawCeron/lazyftp/issues/15))
- Prebuilt binaries for Linux, macOS and Windows, on x86-64 and arm64. Installing no longer
  requires a Go toolchain. ([#21](https://github.com/MawCeron/lazyftp/issues/21))
- Continuous integration on Linux and Windows.
  ([#20](https://github.com/MawCeron/lazyftp/issues/20))
- `--version` prints the version and exits, so a downloaded binary can be identified.

### Changed

- The local panel opens in the directory lazyftp was launched from instead of the home
  directory. ([#55](https://github.com/MawCeron/lazyftp/issues/55))
- Connecting and listing a remote directory no longer block the interface.
  ([#15](https://github.com/MawCeron/lazyftp/issues/15))
- Leaving the port field empty uses the selected protocol's default — 21 for FTP and FTPS, 22
  for SFTP — instead of always 22.
  ([#14](https://github.com/MawCeron/lazyftp/issues/14))
- The FTP client moved from `jlaffaye/ftp` to `secsy/goftp`, for connections through NAT.

### Removed

- Dead code: an unreachable hints module, two client interface methods nothing called, and a size
  formatter with no callers. Nothing changes for users — thanks to
  [@OdaloV](https://github.com/OdaloV). ([#22](https://github.com/MawCeron/lazyftp/issues/22))

### Fixed

- **Plain FTP servers on port 21 could not be connected to.** An empty port field selected SFTP
  against a server that only speaks FTP, and the attempt then hung with no error.
  ([#11](https://github.com/MawCeron/lazyftp/issues/11))
- Connecting over FTP reported success before reaching the server, so a wrong host or a refused
  login surfaced later as an empty panel.
  ([#61](https://github.com/MawCeron/lazyftp/issues/61))
- Going up a directory from the local panel crashed on Windows.
  ([#16](https://github.com/MawCeron/lazyftp/issues/16))
- Rendering crashed on very narrow terminals — thanks to
  [@OdaloV](https://github.com/OdaloV). ([#18](https://github.com/MawCeron/lazyftp/issues/18))
- A failing transfer took the whole program down and left the terminal unusable.
  ([#17](https://github.com/MawCeron/lazyftp/issues/17))
- The Processes panel computed its visible rows incorrectly — thanks to
  [@IzzaldinSamir](https://github.com/IzzaldinSamir).
  ([#19](https://github.com/MawCeron/lazyftp/issues/19))
- Interrupted uploads started again from zero instead of resuming, even against servers that
  support it. ([#54](https://github.com/MawCeron/lazyftp/issues/54))

### Security

- SSH connections had no timeout at all, and the handshake had none even after one was added.
  A host accepting on port 22 without speaking SSH — an FTP server sharing the port, for
  instance — held the program indefinitely.
  ([#58](https://github.com/MawCeron/lazyftp/issues/58),
  [#60](https://github.com/MawCeron/lazyftp/issues/60))

## [0.1.1] - 2026-04-21

### Added

- Uploading a directory now uploads its contents recursively, creating the remote directories as
  it goes. ([#1](https://github.com/MawCeron/lazyftp/issues/1))

### Changed

- Interface messages and log output translated to English.

### Fixed

- The Processes and Log panels showed the oldest entries and never scrolled to the newest.
  ([#2](https://github.com/MawCeron/lazyftp/issues/2))
- A narrow window crashed the Processes panel while drawing a progress bar.

## [0.1.0] - 2026-04-13

First working release.

### Added

- Dual-pane layout, local and remote side by side.
- FTP and SFTP connections from a form at the top of the screen.
- Uploads and downloads with real-time progress and a direction indicator.
- Marking several files for a batch transfer.
- Keyboard navigation, vim-style keys and arrows.
- A hints bar reflecting the focused panel, and a panel logging transfers and connections.

[Unreleased]: https://github.com/MawCeron/lazyftp/compare/v0.2.0...HEAD
[0.2.1]: https://github.com/MawCeron/lazyftp/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/MawCeron/lazyftp/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/MawCeron/lazyftp/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/MawCeron/lazyftp/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/MawCeron/lazyftp/releases/tag/v0.1.0
