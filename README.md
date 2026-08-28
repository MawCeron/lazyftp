<div align="center">

# lazyftp

A simple, keyboard-driven TUI FTP, FTPS and SFTP client inspired by
[lazygit](https://github.com/jesseduffield/lazygit).

[![Release](https://img.shields.io/github/v/release/MawCeron/lazyftp?style=for-the-badge)](https://github.com/MawCeron/lazyftp/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go)](https://go.dev)
[![Build](https://img.shields.io/github/actions/workflow/status/MawCeron/lazyftp/ci.yml?style=for-the-badge)](https://github.com/MawCeron/lazyftp/actions)
[![License](https://img.shields.io/github/license/MawCeron/lazyftp?style=for-the-badge)](LICENSE)
[![Stars](https://img.shields.io/github/stars/MawCeron/lazyftp?style=for-the-badge)](https://github.com/MawCeron/lazyftp/stargazers)

![lazyftp screenshot](lazyftp_screenshot.png)

</div>

---

## About

lazyftp brings a familiar TUI experience to file transfers. If you live in the terminal and find
yourself constantly switching to a GUI client just to move files around — this is for you.

Dual-pane local/remote navigation, real-time transfer progress, FTP, FTPS and SFTP support, all
from the keyboard.

### Built with

[![Bubbletea](https://img.shields.io/badge/bubbletea-gray?style=for-the-badge)](https://github.com/charmbracelet/bubbletea)
[![Lipgloss](https://img.shields.io/badge/lipgloss-gray?style=for-the-badge)](https://github.com/charmbracelet/lipgloss)

---

## Features

- FTP, FTPS and SFTP support
- Dual-pane layout — local and remote side by side, responsive down to an 80x24 terminal
- File size and modification date, sortable by name, size or date
- Fuzzy filtering and jump-to-path, so a deep or specific file is a few keystrokes away
- `--highlight-diff` marks entries that differ between Local and Remote
- Real-time transfer progress with direction indicators
- Multiple file selection and batch transfers, with direction-independent `U`/`D` shortcuts
- Keyboard-driven navigation (vim-style + arrow keys)
- A help screen (`?`) and a context-aware hints bar
- Scrollable transfer and connection log, and a scrollable process list

---

## Installation

### Download a binary

Grab the archive for your platform from the
[latest release](https://github.com/MawCeron/lazyftp/releases/latest), unpack it and put `lazyftp`
somewhere on your `PATH`. Linux, macOS and Windows, on both x86-64 and arm64. No Go toolchain
needed.

### From source

```bash
git clone https://github.com/MawCeron/lazyftp.git
cd lazyftp
go build -o lazyftp .
```

### With go install

```bash
go install github.com/MawCeron/lazyftp@latest
```

---

## Usage

```bash
lazyftp
```

The local panel opens in the directory you ran it from.

| Flag | What it does |
|------|--------------|
| `--verbose` | Show the FTP control dialogue in the Log panel |
| `--log-file <path>` | Write the log to a file as well, appending to it |
| `--no-nerd-fonts` | Use plain Unicode symbols instead of Nerd Font icons |
| `--highlight-diff` | Mark files that differ between Local and Remote (by name, and by size when both share a name) |
| `--version` | Print the version and exit |

### Connecting

Press `Ctrl+L` to open the connection dialog:

| Field | Description |
|-------|-------------|
| Proto | `FTP`, `FTPS` or `SFTP` — cycle with `←` / `→` |
| Host | Server hostname or IP |
| User | Username |
| Pass | Password |
| Port | Leave empty for the protocol's default: `21` for FTP and FTPS, `22` for SFTP |

Press `Enter` to connect, `Esc` to close the dialog or give up on an attempt that is taking too
long. Once connected, the status line shows the protocol, user, host and connection state.

FTPS certificates are verified, so a server with a self-signed certificate is refused.

### Transferring files

1. Navigate to the file or directory you want to transfer
2. Optionally mark multiple files with `Space`
3. Press `t` to transfer, or `U`/`D` to upload/download whichever side has marked files
   regardless of which panel has focus

If you are in the **local panel**, the file will be uploaded to the current remote path. If you are
in the **remote panel**, it will be downloaded to the current local path.

---

## Keybindings

The full reference — every binding, grouped by context — is also one keystroke away in the app:
press `?`.

### Global

| Key | Action |
|-----|--------|
| `Ctrl+L` | Open the connection dialog |
| `?` | Help screen |
| `Tab` | Switch panel within the current group (Local/Remote, or Log/Processes) |
| `Shift+Tab` | Switch between the Local/Remote group and the Log/Processes group |
| `U` / `D` | Upload / download whichever side has marked files, regardless of focus |
| `q` / `Q` | Quit |

### File panels (Local, Remote)

| Key | Action |
|-----|--------|
| `j` / `↓`, `k` / `↑` | Move down / up |
| `l` / `Enter` | Open directory |
| `h` / `-` / `Backspace` | Go up one level |
| `Space` | Mark / unmark file or directory |
| `t` | Transfer (upload or download depending on active panel) |
| `r` | Refresh the current directory listing |
| `s` / `S` | Cycle sort column / reverse sort direction |
| `:` | Jump to a path by typing it — `Enter` to go, `Esc` to cancel |
| `/` | Fuzzy-filter the listing — `Esc` to clear |

### Log & Processes

| Key | Action |
|-----|--------|
| `↑`/`k`, `↓`/`j` | Scroll up / down |
| `b`/`pgup`, `f`/`pgdn` | Page up / down |
| `Tab` | Switch between Log and Processes |

### Connection dialog

| Key | Action |
|-----|--------|
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `←` / `→` | Change protocol (on the Proto field) |
| `Enter` | Connect |
| `Esc` | Close, or abandon an attempt in progress |

---

## Troubleshooting

**A connection fails and you want to know why.** Both flags together put the whole exchange in a
file you can attach to an issue. Passwords are masked.

```bash
lazyftp --verbose --log-file lazyftp.log
```

**FTPS is refused and the credentials are right.** The server most likely does not offer TLS.
Connect over `FTP` instead.

---

## Project structure

```
lazyftp/
├── .github/
│   └── workflows/     CI on Linux and Windows, plus the release build
├── docs/              Contributor documentation
├── internal/
│   ├── client/        FTP, FTPS and SFTP behind one interface
│   ├── model/         FileInfo — one entry in a listing, local or remote
│   ├── shared/        Messages and progress wrappers used across packages
│   ├── transfer/      Uploads and downloads, running in the background
│   └── ui/            The Bubble Tea model, the panels and every keystroke
├── CHANGELOG.md
├── LICENSE
├── ROADMAP.md
├── go.mod
└── main.go
```

---

## Roadmap

| Release | Focus |
|---------|-------|
| v0.1.2 | FTP connectivity and stability |
| v0.2.0 | TUI overhaul — responsive layout, sort/filter/jump, help screen |
| v0.3.0 | File operations — rename, delete, create directories |
| v0.4.0 | Connections and authentication — favorites, history, SSH keys |
| v0.5.0 | Transfer queue and permissions |

See [ROADMAP.md](ROADMAP.md) for what each release contains and why, or the
[milestones](https://github.com/MawCeron/lazyftp/milestones) for progress.

---

## Documentation

| Resource | What it covers |
|----------|----------------|
| [CHANGELOG.md](CHANGELOG.md) | What changed in each release |
| [ROADMAP.md](ROADMAP.md) | What each release is for, and why the issues are ordered as they are |
| [docs/architecture.md](docs/architecture.md) | Where things live, how a keystroke becomes a transfer, the rules that are easy to break |
| [docs/style.md](docs/style.md) | What a patch is expected to look like — comments, naming, errors, tests |
| [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) | Picking up an issue, branching, commits, and what is in scope |

---

## Contributing

Pull requests are welcome — see [CONTRIBUTING.md](docs/CONTRIBUTING.md).

For anything larger than a fix, open an issue before writing code.

<a href="https://github.com/MawCeron/lazyftp/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=MawCeron/lazyftp" alt="Contributors" />
</a>

---

## License

Distributed under the MIT License. See [LICENSE](LICENSE) for more information.
