# Changelog

All notable changes to lazyftp are recorded here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

### Changed

- The local panel opens in the directory lazyftp was launched from instead of the home
  directory. ([#55](https://github.com/MawCeron/lazyftp/issues/55))
- Connecting and listing a remote directory no longer block the interface.
  ([#15](https://github.com/MawCeron/lazyftp/issues/15))
- Leaving the port field empty uses the selected protocol's default — 21 for FTP and FTPS, 22
  for SFTP — instead of always 22.
  ([#14](https://github.com/MawCeron/lazyftp/issues/14))
- The FTP client moved from `jlaffaye/ftp` to `secsy/goftp`, for connections through NAT.

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

[Unreleased]: https://github.com/MawCeron/lazyftp/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/MawCeron/lazyftp/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/MawCeron/lazyftp/releases/tag/v0.1.0
