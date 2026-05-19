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
	FieldAuthMethod
	FieldPassword
	FieldKeyPath
	FieldPassphrase
)

type FormField struct {
	Label  string
	Value  string
	Secret bool
}

type FormSecret struct {
	Password   string
	Passphrase string
}

type FormModel struct {
	Fields   []*FormField
	Focus    int
	IsEdit   bool
	Original string
	Err      string

	// Auth holds "password" or "key". Rebuilds Visible when changed.
	Auth string
	// Visible holds the field indices currently shown (depends on Auth).
	Visible []int
}

type FormSubmitMsg struct {
	IsEdit   bool
	Original string
	Conn     config.Connection
	Secret   FormSecret
}

type FormCancelMsg struct{}

func NewFormModel(c config.Connection, isEdit bool) FormModel {
	port := ""
	if c.Port != 0 {
		port = strconv.Itoa(c.Port)
	} else if !isEdit {
		port = "22"
	}
	auth := c.AuthMethod
	if auth == "" {
		auth = "password"
	}
	m := FormModel{
		Fields: []*FormField{
			FieldName:       {Label: "Name", Value: c.Name},
			FieldHost:       {Label: "Host", Value: c.Host},
			FieldPort:       {Label: "Port", Value: port},
			FieldUser:       {Label: "User", Value: c.User},
			FieldAuthMethod: {Label: "Auth", Value: auth},
			FieldPassword:   {Label: "Password", Value: "", Secret: true},
			FieldKeyPath:    {Label: "Key Path", Value: c.KeyPath},
			FieldPassphrase: {Label: "Passphrase", Value: "", Secret: true},
		},
		IsEdit:   isEdit,
		Original: c.Name,
		Auth:     auth,
	}
	m.rebuildVisible()
	return m
}

func (m *FormModel) rebuildVisible() {
	base := []int{FieldName, FieldHost, FieldPort, FieldUser, FieldAuthMethod}
	if m.Auth == "key" {
		m.Visible = append(base, FieldKeyPath, FieldPassphrase)
	} else {
		m.Visible = append(base, FieldPassword)
	}
	if m.Focus >= len(m.Visible) {
		m.Focus = 0
	}
}

func (m *FormModel) SetAuth(a string) {
	if a != "password" && a != "key" {
		return
	}
	m.Auth = a
	m.Fields[FieldAuthMethod].Value = a
	m.rebuildVisible()
}

func (m FormModel) HasField(idx int) bool {
	for _, v := range m.Visible {
		if v == idx {
			return true
		}
	}
	return false
}

func (m FormModel) Init() tea.Cmd { return nil }

func (m FormModel) Build() (config.Connection, FormSecret, error) {
	name := strings.TrimSpace(m.Fields[FieldName].Value)
	host := strings.TrimSpace(m.Fields[FieldHost].Value)
	portStr := strings.TrimSpace(m.Fields[FieldPort].Value)
	user := strings.TrimSpace(m.Fields[FieldUser].Value)
	if name == "" {
		return config.Connection{}, FormSecret{}, fmt.Errorf("name required")
	}
	if strings.Contains(name, ":") {
		// ':' would collide with the passphrase namespace suffix in the secret store.
		return config.Connection{}, FormSecret{}, fmt.Errorf("name cannot contain ':'")
	}
	if host == "" {
		return config.Connection{}, FormSecret{}, fmt.Errorf("host required")
	}
	if user == "" {
		return config.Connection{}, FormSecret{}, fmt.Errorf("user required")
	}
	if portStr == "" {
		portStr = "22"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return config.Connection{}, FormSecret{}, fmt.Errorf("port must be 1-65535")
	}
	conn := config.Connection{
		Name:       name,
		Host:       host,
		Port:       port,
		User:       user,
		PassKey:    "ssh/" + name,
		AuthMethod: m.Auth,
	}
	sec := FormSecret{}
	if m.Auth == "key" {
		conn.KeyPath = strings.TrimSpace(m.Fields[FieldKeyPath].Value)
		if conn.KeyPath == "" {
			return config.Connection{}, FormSecret{}, fmt.Errorf("key path required")
		}
		sec.Passphrase = m.Fields[FieldPassphrase].Value
	} else {
		sec.Password = m.Fields[FieldPassword].Value
	}
	return conn, sec, nil
}

func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	cur := m.Visible[m.Focus]

	// AuthMethod field handles its own keys (toggle on space/left/right/enter)
	if cur == FieldAuthMethod {
		switch k.Type {
		case tea.KeyEsc:
			return m, func() tea.Msg { return FormCancelMsg{} }
		case tea.KeyTab, tea.KeyDown:
			m.Focus = (m.Focus + 1) % len(m.Visible)
		case tea.KeyShiftTab, tea.KeyUp:
			m.Focus = (m.Focus - 1 + len(m.Visible)) % len(m.Visible)
		case tea.KeySpace, tea.KeyLeft, tea.KeyRight, tea.KeyEnter:
			if m.Auth == "password" {
				m.SetAuth("key")
			} else {
				m.SetAuth("password")
			}
		}
		return m, nil
	}

	switch k.Type {
	case tea.KeyEsc:
		return m, func() tea.Msg { return FormCancelMsg{} }
	case tea.KeyTab, tea.KeyDown:
		m.Focus = (m.Focus + 1) % len(m.Visible)
	case tea.KeyShiftTab, tea.KeyUp:
		m.Focus = (m.Focus - 1 + len(m.Visible)) % len(m.Visible)
	case tea.KeyBackspace:
		f := m.Fields[cur]
		if len(f.Value) > 0 {
			f.Value = f.Value[:len(f.Value)-1]
		}
	case tea.KeyEnter:
		c, sec, err := m.Build()
		if err != nil {
			m.Err = err.Error()
			return m, nil
		}
		if !m.IsEdit {
			if c.AuthMethod == "password" && sec.Password == "" {
				m.Err = "password required for new connection"
				return m, nil
			}
		}
		return m, func() tea.Msg {
			return FormSubmitMsg{IsEdit: m.IsEdit, Original: m.Original, Conn: c, Secret: sec}
		}
	case tea.KeyRunes:
		m.Fields[cur].Value += string(k.Runes)
	case tea.KeySpace:
		m.Fields[cur].Value += " "
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
	for i, idx := range m.Visible {
		f := m.Fields[idx]
		var val string
		if idx == FieldAuthMethod {
			if m.Auth == "password" {
				val = "(•) Password  ( ) SSH Key"
			} else {
				val = "( ) Password  (•) SSH Key"
			}
		} else if f.Secret {
			val = strings.Repeat("*", len(f.Value))
		} else {
			val = f.Value
		}
		line := fmt.Sprintf("%s%-10s : %s", leftPad, f.Label, val)
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
		Render("[Enter] save   [Esc] cancel   [Tab] next   [Space] toggle Auth")
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
	return StyleBorder.Render(b.String())
}
