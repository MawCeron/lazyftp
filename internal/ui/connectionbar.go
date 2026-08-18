package ui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MawCeron/lazyftp/internal/client"
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
	fieldCount
)

type ConnectionBar struct {
	protocol client.Protocol
	inputs   [fieldCount]textinput.Model
	focused  connField
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

	port := textinput.New()
	port.Prompt = ""
	port.SetWidth(8)

	bar := ConnectionBar{
		inputs: [fieldCount]textinput.Model{
			fieldHost: host,
			fieldUser: user,
			fieldPass: pass,
			fieldPort: port,
		},
		focused: fieldProtocol,
	}
	return bar.showDefaultPort()
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
		switch msg.String() {
		case "tab":
			c = c.blur()
			c.focused = (c.focused + 1) % fieldCount
			return c.focus(), nil

		case "shift+tab":
			c = c.blur()
			if c.focused == 0 {
				c.focused = fieldCount - 1
			} else {
				c.focused--
			}
			return c.focus(), nil

		case "enter":
			return c, func() tea.Msg {
				return ConnectMsg{
					Protocol: c.protocol,
					Host:     c.inputs[fieldHost].Value(),
					User:     c.inputs[fieldUser].Value(),
					Pass:     c.inputs[fieldPass].Value(),
					Port:     c.inputs[fieldPort].Value(),
				}
			}
		}

		if c.focused == fieldProtocol {
			switch msg.String() {
			case "left":
				c.protocol = c.protocol.Prev()
			case "right", "space":
				c.protocol = c.protocol.Next()
			}
			return c.showDefaultPort(), nil
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
	width := 56
	if width > maxWidth-2 {
		width = maxWidth - 2
	}
	if width < 20 {
		width = 20
	}

	labelStyle := lipgloss.NewStyle().Foreground(colorEmphasis).Bold(true).Width(10)
	arrowStyle := lipgloss.NewStyle().Foreground(colorMuted)

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
	}

	hint := lipgloss.NewStyle().Foreground(colorMuted).Render("Enter connect · Esc cancel")
	body := strings.Join(fields, "\n") + "\n\n\n" + hint

	// Exactly as tall as the content needs: this is a fixed-size dialog, not
	// a panel truncating to fit whatever space is left.
	height := lipgloss.Height(body) + 4
	return borderWithTitle(body, "Connection", width, height, colorAccent)
}

type ConnectMsg struct {
	Protocol client.Protocol
	Host     string
	User     string
	Pass     string
	Port     string
}
