# lazyftp

A simple TUI FTP/SFTP client inspired by lazygit.

## Keybindings

| Key | Action |
|-----|--------|
| `Ctrl+L` | Focus connection bar |
| `Tab` | Switch between local/remote panels |
| `Esc` | Exit connection bar |
| `q` / `Q` | Quit |
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Enter` / `Space` | Enter directory |
| `-` / `Backspace` | Go up one level |
| `x` | Mark/unmark file |
| `t` | Transfer (upload if local panel, download if remote) |

## Connection

- Port `22` → SFTP
- Any other port → FTP

## Build

```bash
go build -o lazyftp .
```
