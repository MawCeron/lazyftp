package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type connField int

const (
	fieldHost connField = iota
	fieldUser
	fieldPass
	fieldPort
	fieldCount
)

type ConnectionBar struct {
	inputs  [fieldCount]textinput.Model
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
	port.Placeholder = "22"
	port.Width = 5

	bar := ConnectionBar{
		inputs:  [fieldCount]textinput.Model{host, user, pass, port},
		focused: fieldHost,
	}
	bar.inputs[fieldHost].Focus()

	return bar
}

func (c ConnectionBar) Update(msg tea.Msg) (ConnectionBar, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			c.inputs[c.focused].Blur()
			c.focused = (c.focused + 1) % fieldCount
			c.inputs[c.focused].Focus()
			return c, nil

		case "shift+tab":
			c.inputs[c.focused].Blur()
			if c.focused == 0 {
				c.focused = fieldCount - 1
			} else {
				c.focused--
			}
			c.inputs[c.focused].Focus()
			return c, nil

		case "enter":
			return c, func() tea.Msg {
				return ConnectMsg{
					Host: c.inputs[fieldHost].Value(),
					User: c.inputs[fieldUser].Value(),
					Pass: c.inputs[fieldPass].Value(),
					Port: c.inputs[fieldPort].Value(),
				}
			}
		}
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

	fields := []string{
		"Host: " + c.inputs[fieldHost].View(),
		"User: " + c.inputs[fieldUser].View(),
		"Pass: " + c.inputs[fieldPass].View(),
		"Port: " + c.inputs[fieldPort].View(),
	}

	body := strings.Join(fields, "  ")
	return borderWithTitle(body, "Connection", width, 3, borderColor)
}

type ConnectMsg struct {
	Host string
	User string
	Pass string
	Port string
}
