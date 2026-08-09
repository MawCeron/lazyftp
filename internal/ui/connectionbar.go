package ui

import (
	"strconv"
	"strings"

	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	focused connField
}

func NewConnectionBar() ConnectionBar {
	host := textinput.New()
	host.Placeholder = "Host"
	host.Width = 20

	user := textinput.New()
	user.Placeholder = "User"
	user.Width = 15

	pass := textinput.New()
	pass.Placeholder = "Pass"
	pass.EchoMode = textinput.EchoPassword
	pass.Width = 15

	port := textinput.New()
	port.Width = 5

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
	case tea.KeyMsg:
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
			case "right", " ":
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
	borderColor := lipgloss.Color("240")
	if active {
		borderColor = lipgloss.Color("39")
	}

	protocol := c.protocol.String()
	if c.focused == fieldProtocol {
		protocol = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
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
