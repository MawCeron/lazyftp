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

const (
	fieldProtocol connField = iota
	fieldHost
	fieldUser
	fieldPass
	fieldPort
	fieldCount
)

type ConnectionBar struct {
	protocol client.Protocol
	inputs   [fieldCount]textinput.Model
	focused  connField
}

func NewConnectionBar() ConnectionBar {
	host := textinput.New()
	host.Placeholder = "Host"
	host.SetWidth(20)

	user := textinput.New()
	user.Placeholder = "User"
	user.SetWidth(15)

	pass := textinput.New()
	pass.Placeholder = "Pass"
	pass.EchoMode = textinput.EchoPassword
	pass.SetWidth(15)

	port := textinput.New()
	port.SetWidth(5)

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

func (c ConnectionBar) View(width int, active bool) string {
	borderColor := colorMuted
	if active {
		borderColor = colorAccent
	}

	protocol := c.protocol.String()
	if c.focused == fieldProtocol {
		protocol = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Render(protocol)
	}

	fields := []string{
		"Proto: " + protocol,
		"Host: " + c.inputs[fieldHost].View(),
		"User: " + c.inputs[fieldUser].View(),
		"Pass: " + c.inputs[fieldPass].View(),
		"Port: " + c.inputs[fieldPort].View(),
	}

	body := strings.Join(fields, "  ")
	return borderWithTitle(body, "Connection", width, 3, borderColor)
}

type ConnectMsg struct {
	Protocol client.Protocol
	Host     string
	User     string
	Pass     string
	Port     string
}
