package ui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/MawCeron/lazyftp/internal/config"
	"github.com/mattn/go-runewidth"
)

type barMode int

const (
	modeForm barMode = iota
	modeList
	modeHistory
	modeSave
	modeReplace
)

const favoritesVisible = 8

var (
	keyListUp      = key.NewBinding(key.WithKeys("up", "k"))
	keyListDown    = key.NewBinding(key.WithKeys("down", "j"))
	keyListDelete  = key.NewBinding(key.WithKeys("d", "delete"))
	keyListEdit    = key.NewBinding(key.WithKeys("e"))
	keyToggleStore = key.NewBinding(key.WithKeys("tab"))
	keyReplaceYes  = key.NewBinding(key.WithKeys("y"))
	keyReplaceNo   = key.NewBinding(key.WithKeys("n", "esc"))
)

// saveFavoriteMsg carries the password only so App can hand it to the keyring;
// it never reaches the config file.
type saveFavoriteMsg struct {
	Conn     config.Connection
	Pass     string
	Remember bool
}

type deleteFavoriteMsg struct{ Name string }

// SetFavorites replaces the list; App owns it, the bar only shows it.
func (c ConnectionBar) SetFavorites(f []config.Connection) ConnectionBar {
	c.favorites = f
	if n := len(c.items()); c.cursor >= n {
		c.cursor = max(0, n-1)
	}
	if len(c.items()) == 0 && c.mode == modeList {
		c.mode = modeForm
	}
	return c
}

// SetSSHHosts installs the servers read from ~/.ssh/config. They are listed
// after the favorites and can be filled from but never edited or deleted.
func (c ConnectionBar) SetSSHHosts(h []config.Connection) ConnectionBar {
	c.sshHosts = h
	return c
}

func (c ConnectionBar) favorite(name string) (config.Connection, bool) {
	for _, f := range c.favorites {
		if f.Name == name {
			return f, true
		}
	}
	return config.Connection{}, false
}

// freeName suggests the host, or host (2), (3)... so the default never
// overwrites anything.
func (c ConnectionBar) freeName(host string) string {
	name := host
	for n := 2; ; n++ {
		if _, taken := c.favorite(name); !taken {
			return name
		}
		name = fmt.Sprintf("%s (%d)", host, n)
	}
}

// items is what the list modes show: favorites, or the recent connections.
func (c ConnectionBar) items() []config.Connection {
	if c.mode == modeHistory {
		return c.recent
	}
	return append(append([]config.Connection(nil), c.favorites...), c.sshHosts...)
}

// SetRecent replaces the history the same way SetFavorites replaces favorites.
func (c ConnectionBar) SetRecent(r []config.Connection) ConnectionBar {
	c.recent = r
	if len(r) == 0 && c.mode == modeHistory {
		c.mode = modeForm
	}
	return c
}

func (c ConnectionBar) current() config.Connection {
	port, err := strconv.Atoi(c.inputs[fieldPort].Value())
	if err != nil || port <= 0 {
		port = c.protocol.DefaultPort()
	}
	return config.Connection{
		Name:     strings.TrimSpace(c.name.Value()),
		Host:     c.inputs[fieldHost].Value(),
		User:     c.inputs[fieldUser].Value(),
		Port:     port,
		Protocol: c.protocol.String(),

		IdentityFile: strings.TrimSpace(c.inputs[fieldKey].Value()),
	}
}

func (c ConnectionBar) fill(f config.Connection) ConnectionBar {
	c = c.blur()
	c.protocol = client.ParseProtocol(f.Protocol)
	c.inputs[fieldHost].SetValue(f.Host)
	c.inputs[fieldUser].SetValue(f.User)
	c.inputs[fieldPort].SetValue(strconv.Itoa(f.Port))
	pass, _ := config.Secret(f)
	c.inputs[fieldPass].SetValue(pass)
	c.inputs[fieldKey].SetValue(f.IdentityFile)
	c.mode = modeForm
	c.focused = fieldPass
	return c.showDefaultPort().focus()
}

func (c ConnectionBar) updateList(msg tea.KeyPressMsg) (ConnectionBar, tea.Cmd) {
	switch {
	case key.Matches(msg, keyEsc):
		c.mode = modeForm
	case key.Matches(msg, keyListUp):
		c.cursor = max(0, c.cursor-1)
	case key.Matches(msg, keyListDown):
		c.cursor = min(len(c.items())-1, c.cursor+1)
	case key.Matches(msg, keyListEdit):
		return c.fill(c.items()[c.cursor]), nil
	case key.Matches(msg, keySubmit):
		// An empty Pass would read as "type it here", so a connection that
		// still needs one goes to the form instead. SFTP can do without: the
		// agent and the keys are tried first.
		c = c.fill(c.items()[c.cursor])
		if c.inputs[fieldPass].Value() != "" || c.protocol == client.SFTP {
			return c, c.connect()
		}
		return c, nil
	case key.Matches(msg, keyListDelete) && c.mode == modeList && c.cursor < len(c.favorites):
		name := c.favorites[c.cursor].Name
		return c, func() tea.Msg { return deleteFavoriteMsg{Name: name} }
	}
	return c, nil
}

func (c ConnectionBar) updateSave(msg tea.KeyPressMsg) (ConnectionBar, tea.Cmd) {
	switch {
	case key.Matches(msg, keyEsc):
		c.mode = modeForm
		return c.focus(), nil
	case key.Matches(msg, keyToggleStore):
		c.remember = !c.remember
		return c, nil
	case key.Matches(msg, keySubmit):
		conn := c.current()
		if conn.Name == "" {
			return c, nil
		}
		c.pending = saveFavoriteMsg{Conn: conn, Pass: c.inputs[fieldPass].Value(), Remember: c.remember}
		if _, taken := c.favorite(conn.Name); taken {
			c.mode = modeReplace
			return c, nil
		}
		return c.send()
	}
	var cmd tea.Cmd
	c.name, cmd = c.name.Update(msg)
	return c, cmd
}

func (c ConnectionBar) send() (ConnectionBar, tea.Cmd) {
	out := c.pending
	c.mode = modeForm
	return c.focus(), func() tea.Msg { return out }
}

func (c ConnectionBar) updateReplace(msg tea.KeyPressMsg) (ConnectionBar, tea.Cmd) {
	switch {
	case key.Matches(msg, keyReplaceYes):
		return c.send()
	case key.Matches(msg, keyReplaceNo):
		c.mode = modeSave
	}
	return c, nil
}

func describe(f config.Connection) string {
	return fmt.Sprintf("%s://%s@%s:%d", strings.ToLower(f.Protocol), f.User, f.Host, f.Port)
}

func (c ConnectionBar) replaceView(maxWidth int) string {
	width := dialogWidth(maxWidth)
	inner := borderInteriorWidth(width)
	old, _ := c.favorite(c.pending.Conn.Name)

	label := lipgloss.NewStyle().Foreground(colorMuted)
	line := func(l string, f config.Connection) string {
		return label.Render(l) + runewidth.Truncate(describe(f), inner-len(l), "...")
	}
	body := fmt.Sprintf("A favorite named %q already exists.", c.pending.Conn.Name) + "\n\n" +
		line("Current: ", old) + "\n" + line("New:     ", c.pending.Conn)
	return borderWithTitle(body, "Replace Favorite", width, lipgloss.Height(body)+2, colorAccent)
}

func dialogWidth(maxWidth int) int {
	return max(20, min(56, maxWidth-2))
}

func (c ConnectionBar) listView(maxWidth int) string {
	width := dialogWidth(maxWidth)
	inner := borderInteriorWidth(width)

	items := c.items()
	start := max(0, min(c.cursor-favoritesVisible+1, len(items)-favoritesVisible))
	end := min(len(items), start+favoritesVisible)

	rows := make([]string, 0, favoritesVisible+3)
	for i := start; i < end; i++ {
		f := items[i]
		line := describe(f)
		if f.Name != "" {
			line = f.Name + "  " + line
		}
		tag := ""
		if c.mode == modeList && i >= len(c.favorites) {
			tag = "ssh "
		}
		line = runewidth.Truncate(line, inner-2-len(tag), "...")
		if i == c.cursor {
			line = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("> " + tag + line)
		} else {
			line = "  " + lipgloss.NewStyle().Foreground(colorMuted).Render(tag) + line
		}
		rows = append(rows, line)
	}

	title := "Favorites"
	if c.mode == modeHistory {
		title = "Recent"
	}
	body := strings.Join(rows, "\n")
	return borderWithTitle(body, title, width, lipgloss.Height(body)+2, colorAccent)
}

func (c ConnectionBar) saveView(maxWidth int) string {
	width := dialogWidth(maxWidth)

	labelStyle := lipgloss.NewStyle().Foreground(colorEmphasis).Bold(true).Width(10)
	box := "[ ]"
	if c.remember {
		box = iconChecked()
	}
	pass := "no password typed"
	if c.inputs[fieldPass].Value() != "" {
		pass = "remember password " + box
	}

	body := labelStyle.Render("Name:") + " " + c.name.View() + "\n\n" + pass
	return borderWithTitle(body, "Save Favorite", width, lipgloss.Height(body)+2, colorAccent)
}
