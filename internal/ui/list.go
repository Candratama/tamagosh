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
	Info      string // success / informational toast (rendered green)
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
	visible := m.Visible()

	type styledLine struct {
		text  string
		plain string
	}
	var rows []styledLine

	if len(visible) == 0 {
		raw := "(no connections — press 'n' to add)"
		rows = append(rows, styledLine{text: StyleHelp.Render(raw), plain: raw})
	}
	for i, c := range visible {
		raw := fmt.Sprintf("  %-12s %-18s :%d", c.Name, c.Host, c.Port)
		var styled string
		if i == m.Cursor {
			styled = StyleSelected.Render("▸ " + strings.TrimLeft(raw, " "))
		} else {
			styled = StyleNormal.Render(raw)
		}
		rows = append(rows, styledLine{text: styled, plain: raw})
	}

	var footer styledLine
	if m.Filtering {
		raw := fmt.Sprintf("/%s_", m.Filter)
		footer = styledLine{text: StyleHelp.Render(raw), plain: raw}
	} else {
		raw := "[n]ew [e]dit [d]el [f]sftp [K]eygen [/]find [q]uit"
		footer = styledLine{text: StyleHelp.Render(raw), plain: raw}
	}

	title := "Connection List"

	// widest content line drives the box's natural width.
	widestRow := 0
	for _, r := range rows {
		if w := lipgloss.Width(r.plain); w > widestRow {
			widestRow = w
		}
	}
	contentW := widestRow
	if w := lipgloss.Width(footer.plain); w > contentW {
		contentW = w
	}
	if w := lipgloss.Width(title); w > contentW {
		contentW = w
	}
	// breathing room on both sides so rows don't hug edges
	const sidePad = 6
	targetW := contentW + sidePad*2

	// Uniform left shift for the row block — same pad for every row, so
	// columns stay vertically aligned and the whole block sits centered.
	rowShift := (targetW - widestRow) / 2

	var b strings.Builder
	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(gbYellow)).
		Render(title)
	b.WriteString(lipgloss.PlaceHorizontal(targetW, lipgloss.Center, titleStyled))
	b.WriteString("\n\n")

	shift := strings.Repeat(" ", rowShift)
	for _, r := range rows {
		// pad row block uniformly on the left; right side fills with spaces
		// up to targetW so the box width is stable.
		line := lipgloss.PlaceHorizontal(targetW, lipgloss.Left, shift+r.text)
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(lipgloss.PlaceHorizontal(targetW, lipgloss.Center, footer.text))

	if m.Info != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.PlaceHorizontal(targetW, lipgloss.Center, StyleSuccess.Render(m.Info)))
	}
	if m.Err != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.PlaceHorizontal(targetW, lipgloss.Center, StyleError.Render(m.Err)))
	}
	box := StyleBorder.Render(b.String())
	return lipgloss.JoinVertical(lipgloss.Center, renderHeader(), "", box)
}
