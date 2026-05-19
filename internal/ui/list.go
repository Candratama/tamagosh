package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Candratama/tamagosh/internal/config"
)

type ConnectMsg struct{ Conn config.Connection }
type OpenSftpMsg struct{ Conn config.Connection }
type NewFormMsg struct{}
type EditFormMsg struct{ Conn config.Connection }
type DeleteMsg struct{ Conn config.Connection }

type ListModel struct {
	Store     *config.Store
	Cursor    int
	Filter    string
	Filtering bool
	Err       string
}

func NewListModel(s *config.Store) ListModel {
	return ListModel{Store: s}
}

func (m ListModel) Init() tea.Cmd { return nil }

func (m ListModel) Visible() []config.Connection {
	if m.Filter == "" {
		return m.Store.Connections
	}
	q := strings.ToLower(m.Filter)
	out := []config.Connection{}
	for _, c := range m.Store.Connections {
		if strings.Contains(strings.ToLower(c.Name), q) ||
			strings.Contains(strings.ToLower(c.Host), q) {
			out = append(out, c)
		}
	}
	return out
}

func (m ListModel) Selected() config.Connection {
	v := m.Visible()
	if len(v) == 0 {
		return config.Connection{}
	}
	if m.Cursor >= len(v) {
		return v[len(v)-1]
	}
	return v[m.Cursor]
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if mm, ok := msg.(tea.MouseMsg); ok {
		switch mm.Button {
		case tea.MouseButtonWheelUp:
			if m.Cursor > 0 {
				m.Cursor--
			}
		case tea.MouseButtonWheelDown:
			if m.Cursor < len(m.Visible())-1 {
				m.Cursor++
			}
		}
		return m, nil
	}
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.Filtering {
		switch k.Type {
		case tea.KeyEsc:
			m.Filtering = false
			m.Filter = ""
			m.Cursor = 0
		case tea.KeyEnter:
			m.Filtering = false
		case tea.KeyBackspace:
			if len(m.Filter) > 0 {
				m.Filter = m.Filter[:len(m.Filter)-1]
			}
		case tea.KeyRunes:
			m.Filter += string(k.Runes)
			m.Cursor = 0
		}
		return m, nil
	}

	switch k.Type {
	case tea.KeyUp:
		if m.Cursor > 0 {
			m.Cursor--
		}
	case tea.KeyDown:
		if m.Cursor < len(m.Visible())-1 {
			m.Cursor++
		}
	case tea.KeyEnter:
		sel := m.Selected()
		if sel.Name == "" {
			return m, nil
		}
		return m, func() tea.Msg { return ConnectMsg{Conn: sel} }
	case tea.KeyRunes:
		switch string(k.Runes) {
		case "/":
			m.Filtering = true
			m.Filter = ""
		case "n":
			return m, func() tea.Msg { return NewFormMsg{} }
		case "e":
			sel := m.Selected()
			if sel.Name == "" {
				return m, nil
			}
			return m, func() tea.Msg { return EditFormMsg{Conn: sel} }
		case "d":
			sel := m.Selected()
			if sel.Name == "" {
				return m, nil
			}
			return m, func() tea.Msg { return DeleteMsg{Conn: sel} }
		case "f":
			sel := m.Selected()
			if sel.Name == "" {
				return m, nil
			}
			return m, func() tea.Msg { return OpenSftpMsg{Conn: sel} }
		case "K":
			return m, func() tea.Msg { return KeygenStartMsg{} }
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ListModel) View() string {
	var b strings.Builder
	visible := m.Visible()
	if len(visible) == 0 {
		b.WriteString(StyleHelp.Render("  (no connections — press 'n' to add)"))
		b.WriteString("\n")
	}
	for i, c := range visible {
		line := fmt.Sprintf("  %-12s %-18s :%d", c.Name, c.Host, c.Port)
		if i == m.Cursor {
			line = StyleSelected.Render("▸ " + strings.TrimLeft(line, " "))
		} else {
			line = StyleNormal.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if m.Filtering {
		b.WriteString(StyleHelp.Render(fmt.Sprintf("  /%s_", m.Filter)))
	} else {
		b.WriteString(StyleHelp.Render("  [n]ew [e]dit [d]el [f]sftp [K]eygen [/]find [q]uit"))
	}
	if m.Err != "" {
		b.WriteString("\n")
		b.WriteString(StyleError.Render("  " + m.Err))
	}
	box := StyleBorder.Render(b.String())
	return lipgloss.JoinVertical(lipgloss.Center, renderHeader(), "", box)
}
