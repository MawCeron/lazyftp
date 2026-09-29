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
	modeSave
)

const favoritesVisible = 8

var (
	keyListUp      = key.NewBinding(key.WithKeys("up", "k"))
	keyListDown    = key.NewBinding(key.WithKeys("down", "j"))
	keyListDelete  = key.NewBinding(key.WithKeys("d", "delete"))
	keyToggleStore = key.NewBinding(key.WithKeys("tab"))
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
	if c.cursor >= len(f) {
		c.cursor = max(0, len(f)-1)
	}
	if len(f) == 0 && c.mode == modeList {
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
		c.cursor = min(len(c.favorites)-1, c.cursor+1)
	case key.Matches(msg, keySubmit):
		return c.fill(c.favorites[c.cursor]), nil
	case key.Matches(msg, keyListDelete):
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
		out := saveFavoriteMsg{Conn: conn, Pass: c.inputs[fieldPass].Value(), Remember: c.remember}
		c.mode = modeForm
		return c.focus(), func() tea.Msg { return out }
	}
	var cmd tea.Cmd
	c.name, cmd = c.name.Update(msg)
	return c, cmd
}

func dialogWidth(maxWidth int) int {
	return max(20, min(56, maxWidth-2))
}

func (c ConnectionBar) listView(maxWidth int) string {
	width := dialogWidth(maxWidth)
	inner := borderInteriorWidth(width)

	start := max(0, min(c.cursor-favoritesVisible+1, len(c.favorites)-favoritesVisible))
	end := min(len(c.favorites), start+favoritesVisible)

	rows := make([]string, 0, favoritesVisible+3)
	for i := start; i < end; i++ {
		f := c.favorites[i]
		line := fmt.Sprintf("%s  %s://%s@%s:%d", f.Name, strings.ToLower(f.Protocol), f.User, f.Host, f.Port)
		line = runewidth.Truncate(line, inner-2, "...")
		if i == c.cursor {
			line = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("> " + line)
		} else {
			line = "  " + line
		}
		rows = append(rows, line)
	}

	hint := lipgloss.NewStyle().Foreground(colorMuted).Render("Enter fill · d delete · Esc back")
	body := strings.Join(rows, "\n") + "\n\n" + hint
	return borderWithTitle(body, "Favorites", width, lipgloss.Height(body)+2, colorAccent)
}

func (c ConnectionBar) saveView(maxWidth int) string {
	width := dialogWidth(maxWidth)

	labelStyle := lipgloss.NewStyle().Foreground(colorEmphasis).Bold(true).Width(10)
	box := "[ ]"
	if c.remember {
		box = "[x]"
	}
	pass := "no password typed"
	if c.inputs[fieldPass].Value() != "" {
		pass = "remember password " + box
	}

	hint := lipgloss.NewStyle().Foreground(colorMuted).Render("Enter save · Tab password · Esc back")
	body := labelStyle.Render("Name:") + " " + c.name.View() + "\n\n" + pass + "\n\n" + hint
	return borderWithTitle(body, "Save Favorite", width, lipgloss.Height(body)+2, colorAccent)
}
