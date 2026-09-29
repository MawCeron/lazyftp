<div align="center">

# lazyftp

A simple, keyboard-driven TUI FTP, FTPS and SFTP client inspired by
[lazygit](https://github.com/jesseduffield/lazygit).

[![Release](https://img.shields.io/github/v/release/MawCeron/lazyftp?style=for-the-badge)](https://github.com/MawCeron/lazyftp/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go)](https://go.dev)
[![Build](https://img.shields.io/github/actions/workflow/status/MawCeron/lazyftp/ci.yml?style=for-the-badge)](https://github.com/MawCeron/lazyftp/actions)
[![License](https://img.shields.io/github/license/MawCeron/lazyftp?style=for-the-badge)](LICENSE)
[![Stars](https://img.shields.io/github/stars/MawCeron/lazyftp?style=for-the-badge)](https://github.com/MawCeron/lazyftp/stargazers)

<img src="assets/demo.gif" alt="lazyftp connecting over SFTP, then uploading a local file and downloading a remote one, both panels refreshing live" width="100%">

<sub>Recorded with <a href="https://github.com/charmbracelet/vhs">VHS</a> — see <a href="assets/demo.tape">assets/demo.tape</a>.</sub>

</div>

---

## About

lazyftp brings a familiar TUI experience to file transfers. If you live in the terminal and find
yourself constantly switching to a GUI client just to move files around — this is for you.

Dual-pane local/remote navigation, real-time transfer progress, FTP, FTPS and SFTP support, all
from the keyboard.

### Built with

[![Bubbletea](https://img.shields.io/badge/bubbletea-v2.0.8-gray?style=for-the-badge)](https://github.com/charmbracelet/bubbletea)
[![Bubbles](https://img.shields.io/badge/bubbles-v2.1.1-gray?style=for-the-badge)](https://github.com/charmbracelet/bubbles)
[![Lipgloss](https://img.shields.io/badge/lipgloss-v2.0.6-gray?style=for-the-badge)](https://github.com/charmbracelet/lipgloss)

---

## Features

- FTP, FTPS and SFTP support
- Dual-pane layout — local and remote side by side, responsive down to an 80x24 terminal
- File size and modification date, sortable by name, size or date, with the exact byte count and
  full timestamp a keystroke away when the panel is too narrow to show them
- Fuzzy filtering and jump-to-path, so a deep or specific file is a few keystrokes away
- Create, rename and delete files and directories, on either panel
- Toggle hidden (dot) file visibility per panel
- `--highlight-diff` marks entries that differ between Local and Remote
- Real-time transfer progress with direction indicators, including recursive directory
  upload/download
- Multiple file selection and batch transfers, with direction-independent `U`/`D` shortcuts
- Keyboard-driven navigation (vim-style + arrow keys)
- A help screen (`?`) and a context-aware hints bar
- Scrollable transfer and connection log, and a scrollable process list

---

## Installation

### Download a binary

Packages and archives for Linux, macOS and Windows are on the
[latest release](https://github.com/MawCeron/lazyftp/releases/latest), for both x86-64 and arm64.
No Go toolchain needed.

Debian, Ubuntu:

```bash
sudo dpkg -i lazyftp_*_linux_amd64.deb
```

Fedora, RHEL, openSUSE:

```bash
sudo rpm -i lazyftp_*_linux_amd64.rpm
```

Anywhere else, unpack the archive and put `lazyftp` somewhere on your `PATH`.

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
lazyftp                              # opens the connection dialog
lazyftp nas                          # a favorite, or a Host from ~/.ssh/config
lazyftp ana@nas.lan                  # user@host, FTP unless the port or scheme says otherwise
lazyftp ana@nas.lan:22               # port 22 means SFTP, 990 means FTPS
lazyftp sftp://ana@nas.lan:2222      # the scheme is explicit: ftp, ftps or sftp
lazyftp -i ~/.ssh/work sftp://nas    # a private key for this connection
```

The local panel opens in the directory you ran it from.

A destination connects at once when nothing more is needed: a stored password, or SFTP with an
agent or key. Otherwise the connection dialog opens filled in, on the Pass field. The password is
never accepted on the command line, so it stays out of your shell history: `user:pass@host` is
refused. A saved name wins over a host of the same name. Flags go before the destination, and an
invalid one prints the usage and exits with status 2.

| Flag | What it does |
|------|--------------|
| `--verbose` | Show the FTP control dialogue in the Log panel |
| `--log-file <path>` | Write the log to a file as well, appending to it |
| `--no-nerd-fonts` | Use plain Unicode symbols instead of Nerd Font icons |
| `--highlight-diff` | Mark files that differ between Local and Remote (by name, and by size when both share a name) |
| `-i`, `--identity <path>` | SFTP private key to try first |
| `--protocol <ftp\|ftps\|sftp>` | Protocol for the destination, when it has no scheme |
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
| Key | Optional, SFTP only: path to a private key (`~` works). Tried before the agent and the default keys. If it is encrypted, type its passphrase in Pass |

Press `Enter` to connect, `Esc` to close the dialog or give up on an attempt that is taking too
long. Once connected, the status line shows the protocol, user, host and connection state.

FTPS certificates are verified, so a server with a self-signed certificate is refused.

### Servers from `~/.ssh/config`

Every concrete `Host` entry in `~/.ssh/config` shows up in the `Ctrl+O` list after your favorites,
tagged `ssh`, with its `HostName`, `User`, `Port` and `IdentityFile` filled in. `Include` is
followed and wildcard blocks such as `Host *` apply the way `ssh` applies them. The file is only
read, never written, and a missing or broken one is ignored. These entries cannot be deleted from
lazyftp; `Ctrl+S` after filling one saves your own copy. `ProxyJump`, `ProxyCommand` and `Match`
are not supported.

### Recent connections

Every successful connection is remembered — protocol, host, port and user, never the password.
`Ctrl+R` in the dialog lists the last ten, most recent first; `Enter` connects and `e` fills the form to edit it first. They live
in `history.toml` next to `config.toml`. A server that is also a favorite gets its keyring
password filled in too.

### Favorites

`Ctrl+S` in the dialog saves the current host, user, port and protocol under a name. The
suggested name never collides with an existing one, and typing one that exists asks before
replacing it, showing the old favorite next to the new. `Ctrl+O`
lists your favorites: `Enter` connects, `e` fills the form to edit it first, `d` deletes one. Favorites live in `config.toml`
under your config directory (`~/.config/lazyftp/` on Linux).

The password is never written to that file. While saving, `Tab` toggles "remember password", which
stores it in the system keyring (Secret Service, Keychain or Credential Manager) so the favorite
connects without asking. `Enter` on a favorite with no stored password opens the form on the Pass
field, except for SFTP, which tries the agent and your keys first. Without a keyring the favorite is still saved and the password is asked
each time; the Log says so.

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
| `Ctrl+X` | Disconnect without quitting; refused while a transfer is running |
| `?` | Help screen |
| `Tab` | Switch panel within the current group (Local/Remote, or Log/Processes) |
| `Shift+Tab` | Switch between the Local/Remote group and the Log/Processes group |
| `U` / `D` | Upload / download whichever side has marked files, regardless of focus |
| `q` / `Q` | Quit |

### File panels (Local, Remote)

| Key | Action |
|-----|--------|
| `j` / `↓`, `k` / `↑` | Move down / up |
| `l` / `Enter` | Open directory, or show a file's exact size and full timestamp |
| `h` / `-` / `Backspace` | Go up one level |
| `Space` | Mark / unmark file or directory |
| `t` | Transfer (upload or download depending on active panel) |
| `r` | Refresh the current directory listing |
| `s` / `S` | Cycle sort column / reverse sort direction |
| `:` | Jump to a path by typing it — `Enter` to go, `Esc` to cancel |
| `/` | Fuzzy-filter the listing — `Esc` to clear |
| `Ctrl+H` | Toggle hidden (dot) file visibility |
| `Ctrl+N` | Create a directory — `Enter` to confirm, `Esc` to cancel |
| `F2` | Rename the selected file or directory — `Enter` to confirm, `Esc` to cancel |
| `d` | Delete the selected or marked files — `Enter` to confirm, `Esc` to cancel |

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
| `Ctrl+S` / `Ctrl+O` | Save the connection as a favorite / open the favorites list |
| `Ctrl+R` | Open the recent connections |
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
