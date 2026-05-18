package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Candratama/tamagosh/internal/config"
)

const (
	FieldName = iota
	FieldHost
	FieldPort
	FieldUser
	FieldPassword
)

type FormField struct {
	Label  string
	Value  string
	Secret bool
}

type FormModel struct {
	Fields   []*FormField
	Focus    int
	IsEdit   bool
	Original string
	Err      string
}

type FormSubmitMsg struct {
	IsEdit   bool
	Original string
	Conn     config.Connection
	Password string
}

type FormCancelMsg struct{}

func NewFormModel(c config.Connection, isEdit bool) FormModel {
	port := ""
	if c.Port != 0 {
		port = strconv.Itoa(c.Port)
	} else if !isEdit {
		port = "22"
	}
	return FormModel{
		Fields: []*FormField{
			{Label: "Name", Value: c.Name},
			{Label: "Host", Value: c.Host},
			{Label: "Port", Value: port},
			{Label: "User", Value: c.User},
			{Label: "Password", Value: "", Secret: true},
		},
		IsEdit:   isEdit,
		Original: c.Name,
	}
}

func (m FormModel) Init() tea.Cmd { return nil }

func (m FormModel) Build() (config.Connection, error) {
	name := strings.TrimSpace(m.Fields[FieldName].Value)
	host := strings.TrimSpace(m.Fields[FieldHost].Value)
	portStr := strings.TrimSpace(m.Fields[FieldPort].Value)
	user := strings.TrimSpace(m.Fields[FieldUser].Value)

	if name == "" {
		return config.Connection{}, fmt.Errorf("name required")
	}
	if host == "" {
		return config.Connection{}, fmt.Errorf("host required")
	}
	if user == "" {
		return config.Connection{}, fmt.Errorf("user required")
	}
	if portStr == "" {
		portStr = "22"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return config.Connection{}, fmt.Errorf("port must be 1-65535")
	}
	return config.Connection{
		Name:    name,
		Host:    host,
		Port:    port,
		User:    user,
		PassKey: "ssh/" + name,
	}, nil
}

func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.Type {
	case tea.KeyEsc:
		return m, func() tea.Msg { return FormCancelMsg{} }
	case tea.KeyTab, tea.KeyDown:
		m.Focus = (m.Focus + 1) % len(m.Fields)
	case tea.KeyShiftTab, tea.KeyUp:
		m.Focus = (m.Focus - 1 + len(m.Fields)) % len(m.Fields)
	case tea.KeyBackspace:
		f := m.Fields[m.Focus]
		if len(f.Value) > 0 {
			f.Value = f.Value[:len(f.Value)-1]
		}
	case tea.KeyEnter:
		c, err := m.Build()
		if err != nil {
			m.Err = err.Error()
			return m, nil
		}
		pwd := m.Fields[FieldPassword].Value
		if !m.IsEdit && pwd == "" {
			m.Err = "password required for new connection"
			return m, nil
		}
		return m, func() tea.Msg {
			return FormSubmitMsg{IsEdit: m.IsEdit, Original: m.Original, Conn: c, Password: pwd}
		}
	case tea.KeyRunes:
		m.Fields[m.Focus].Value += string(k.Runes)
	case tea.KeySpace:
		m.Fields[m.Focus].Value += " "
	}
	return m, nil
}

func (m FormModel) View() string {
	const innerW = 48

	var b strings.Builder
	title := "New Connection"
	if m.IsEdit {
		title = "Edit " + m.Original
	}
	titleLine := lipgloss.NewStyle().
		Width(innerW).
		Align(lipgloss.Center).
		Bold(true).
		Foreground(lipgloss.Color(gbYellow)).
		Render(title)
	b.WriteString(titleLine)
	b.WriteString("\n\n")

	const leftPad = "          "
	for i, f := range m.Fields {
		val := f.Value
		if f.Secret {
			val = strings.Repeat("*", len(f.Value))
		}
		line := fmt.Sprintf("%s%-9s : %s", leftPad, f.Label, val)
		if i == m.Focus {
			line = StyleSelected.Render(line + "_")
		} else {
			line = StyleNormal.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	hint := lipgloss.NewStyle().
		Width(innerW).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color(gbFgMute)).
		Render("[Enter] save   [Esc] cancel   [Tab] next")
	b.WriteString(hint)
	if m.Err != "" {
		b.WriteString("\n")
		errLine := lipgloss.NewStyle().
			Width(innerW).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color(gbRed)).
			Bold(true).
			Render(m.Err)
		b.WriteString(errLine)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(gbAqua)).
		Padding(1, 2).
		Render(b.String())
}
