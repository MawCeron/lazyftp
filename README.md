<div align="center">

# lazyftp

A simple, keyboard-driven TUI FTP, FTPS and SFTP client inspired by
[lazygit](https://github.com/jesseduffield/lazygit).

[![Release](https://img.shields.io/github/v/release/MawCeron/lazyftp?style=for-the-badge)](https://github.com/MawCeron/lazyftp/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go)](https://go.dev)
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
- Dual-pane layout — local and remote side by side
- Real-time transfer progress with direction indicators
- Multiple file selection and batch transfers
- Keyboard-driven navigation (vim-style + arrow keys)
- Context-aware hints bar
- Transfer and connection log

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
| `--version` | Print the version and exit |

### Connecting

Fill in the connection bar at the top:

| Field | Description |
|-------|-------------|
| Proto | `FTP`, `FTPS` or `SFTP` — cycle with `←` / `→` |
| Host | Server hostname or IP |
| User | Username |
| Pass | Password |
| Port | Leave empty for the protocol's default: `21` for FTP and FTPS, `22` for SFTP |

Press `Enter` to connect, `Esc` to give up on an attempt that is taking too long.

FTPS certificates are verified, so a server with a self-signed certificate is refused.

### Transferring files

1. Navigate to the file or directory you want to transfer
2. Optionally mark multiple files with `x`
3. Press `t` to transfer

If you are in the **local panel**, the file will be uploaded to the current remote path. If you are
in the **remote panel**, it will be downloaded to the current local path.

---

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `Ctrl+L` | Focus connection bar |
| `Tab` | Switch between local and remote panels |
| `Esc` | Exit connection bar, or abandon a connection attempt |
| `q` / `Q` | Quit |

### Connection bar

| Key | Action |
|-----|--------|
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `←` / `→` | Change protocol (on the Proto field) |
| `Enter` | Connect |
| `Esc` | Close, or abandon an attempt in progress |

### Panels

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Enter` / `Space` | Enter directory |
| `-` / `Backspace` | Go up one level |
| `x` | Mark / unmark file or directory |
| `t` | Transfer (upload or download depending on active panel) |

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
| v0.2.0 | TUI overhaul — responsive layout, theming, help screen |
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
