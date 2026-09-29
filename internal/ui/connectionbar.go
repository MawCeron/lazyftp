package ui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/MawCeron/lazyftp/internal/config"
)

type connField int

// Order matches the visual layout top-to-bottom, so Tab follows reading
// order instead of skipping around it.
const (
	fieldProtocol connField = iota
	fieldHost
	fieldPort
	fieldUser
	fieldPass
	fieldKey
	fieldCount
)

type ConnectionBar struct {
	protocol client.Protocol
	inputs   [fieldCount]textinput.Model
	focused  connField

	mode      barMode
	favorites []config.Connection
	recent    []config.Connection
	sshHosts  []config.Connection
	cursor    int
	name      textinput.Model
	remember  bool
	pending   saveFavoriteMsg
}

func NewConnectionBar() ConnectionBar {
	host := textinput.New()
	host.Prompt = ""
	host.Placeholder = "Host"
	host.SetWidth(32)

	user := textinput.New()
	user.Prompt = ""
	user.Placeholder = "User"
	user.SetWidth(32)

	pass := textinput.New()
	pass.Prompt = ""
	pass.Placeholder = "Pass"
	pass.EchoMode = textinput.EchoPassword
	pass.SetWidth(32)

	keyFile := textinput.New()
	keyFile.Prompt = ""
	keyFile.Placeholder = "optional, SFTP"
	keyFile.SetWidth(32)

	port := textinput.New()
	port.Prompt = ""
	port.SetWidth(8)

	name := textinput.New()
	name.Prompt = ""
	name.Placeholder = "Name"
	name.SetWidth(32)

	bar := ConnectionBar{
		name: name,
		inputs: [fieldCount]textinput.Model{
			fieldHost: host,
			fieldUser: user,
			fieldPass: pass,
			fieldKey:  keyFile,
			fieldPort: port,
		},
		focused: fieldProtocol,
	}
	return bar.showDefaultPort()
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (c ConnectionBar) showDefaultPort() ConnectionBar {
	c.inputs[fieldPort].Placeholder = strconv.Itoa(c.protocol.DefaultPort())
	return c
}

// A zero-value textinput panics on Focus, so the protocol slot is never focused.

func (c ConnectionBar) focus() ConnectionBar {
	if c.focused != fieldProtocol {
		c.inputs[c.focused].Focus()
	}
	return c
}

func (c ConnectionBar) blur() ConnectionBar {
	if c.focused != fieldProtocol {
		c.inputs[c.focused].Blur()
	}
	return c
}

func (c ConnectionBar) Update(msg tea.Msg) (ConnectionBar, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch c.mode {
		case modeList, modeHistory:
			return c.updateList(msg)
		case modeSave:
			return c.updateSave(msg)
		case modeReplace:
			return c.updateReplace(msg)
		}

		switch {
		case key.Matches(msg, keyFavorites):
			if len(c.favorites)+len(c.sshHosts) > 0 {
				c.mode = modeList
				c.cursor = 0
			}
			return c, nil

		case key.Matches(msg, keyHistory):
			if len(c.recent) > 0 {
				c.mode = modeHistory
				c.cursor = 0
			}
			return c, nil

		case key.Matches(msg, keySaveFavorite):
			if c.inputs[fieldHost].Value() != "" {
				c = c.blur()
				c.mode = modeSave
				c.name.SetValue(c.freeName(c.inputs[fieldHost].Value()))
				c.name.Focus()
				c.remember = c.inputs[fieldPass].Value() != ""
			}
			return c, nil

		case key.Matches(msg, keyNextField):
			c = c.blur()
			c.focused = (c.focused + 1) % fieldCount
			return c.focus(), nil

		case key.Matches(msg, keyPrevField):
			c = c.blur()
			if c.focused == 0 {
				c.focused = fieldCount - 1
			} else {
				c.focused--
			}
			return c.focus(), nil

		case key.Matches(msg, keySubmit):
			return c, func() tea.Msg {
				return ConnectMsg{
					Protocol: c.protocol,
					Host:     c.inputs[fieldHost].Value(),
					User:     c.inputs[fieldUser].Value(),
					Pass:     c.inputs[fieldPass].Value(),
					Identity: c.inputs[fieldKey].Value(),
					Port:     c.inputs[fieldPort].Value(),
				}
			}
		}

		if c.focused == fieldProtocol {
			switch {
			case key.Matches(msg, keyProtocolPrev):
				c.protocol = c.protocol.Prev()
			case key.Matches(msg, keyProtocolNext):
				c.protocol = c.protocol.Next()
			}
			return c.showDefaultPort(), nil
		}

		// Port only accepts digits. Text is empty for non-printable keys
		// (backspace, arrows, ...), which must still reach the input.
		if c.focused == fieldPort && msg.Text != "" && !isDigits(msg.Text) {
			return c, nil
		}
	}

	if c.focused == fieldProtocol {
		return c, nil
	}

	var cmd tea.Cmd
	c.inputs[c.focused], cmd = c.inputs[c.focused].Update(msg)
	return c, cmd
}

// View renders the connection form as a self-contained floating dialog, no
// wider than maxWidth. It is always shown focused: the app only renders it
// at all while it holds focus.
func (c ConnectionBar) View(maxWidth int) string {
	switch c.mode {
	case modeList, modeHistory:
		return c.listView(maxWidth)
	case modeSave:
		return c.saveView(maxWidth)
	case modeReplace:
		return c.replaceView(maxWidth)
	}

	width := 56
	if width > maxWidth-2 {
		width = maxWidth - 2
	}
	if width < 20 {
		width = 20
	}

	labelStyle := lipgloss.NewStyle().Foreground(colorEmphasis).Bold(true).Width(10)
	arrowStyle := lipgloss.NewStyle().Foreground(colorBorder)

	protocol := c.protocol.String()
	if c.focused == fieldProtocol {
		protocol = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(protocol)
	}
	arrows := arrowStyle.Render("◂ ") + protocol + arrowStyle.Render(" ▸")

	row := func(label string, ti textinput.Model) string {
		return labelStyle.Render(label+":") + " " + ti.View()
	}

	fields := []string{
		labelStyle.Render("Protocol:") + " " + arrows,
		"",
		row("Host", c.inputs[fieldHost]),
		"",
		row("Port", c.inputs[fieldPort]),
		"",
		row("User", c.inputs[fieldUser]),
		"",
		row("Pass", c.inputs[fieldPass]),
		"",
		row("Key", c.inputs[fieldKey]),
	}

	hint := lipgloss.NewStyle().Foreground(colorMuted).Render("Enter connect · Esc cancel")
	more := lipgloss.NewStyle().Foreground(colorMuted).Render("^O favorites · ^R recent · ^S save")
	body := strings.Join(fields, "\n") + "\n\n\n" + hint + "\n" + more

	// Exactly as tall as the content needs: this is a fixed-size dialog, not
	// a panel truncating to fit whatever space is left.
	height := lipgloss.Height(body) + 2
	return borderWithTitle(body, "Connection", width, height, colorAccent)
}

type ConnectMsg struct {
	Protocol client.Protocol
	Host     string
	User     string
	Pass     string
	Identity string
	Port     string
}
