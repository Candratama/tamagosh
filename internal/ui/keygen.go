package ui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Candratama/tamagosh/internal/keygen"
)

// validKeyName rejects path separators, traversal, NUL bytes, and hidden-file
// patterns. Positive allowlist: must start with [a-zA-Z0-9_], then any of
// [a-zA-Z0-9._-]. Catches `.`, `..`, `.hidden`, `foo/bar`, `foo\x00bar`.
var validKeyName = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._-]*$`)

type KeygenStartMsg struct{}
type KeygenCancelMsg struct{}
type KeygenDoneMsg struct {
	Path string
	Err  error
}

const (
	KgFieldName = iota
	KgFieldPassphrase
)

type KeygenModel struct {
	ConfigDir       string
	NameField       *FormField
	PassphraseField *FormField
	Focus           int
	Err             string
}

func NewKeygenModel(configDir string) KeygenModel {
	return KeygenModel{
		ConfigDir:       configDir,
		NameField:       &FormField{Label: "Name", Value: "id_ed25519"},
		PassphraseField: &FormField{Label: "Passphrase", Value: "", Secret: true},
	}
}

func (m KeygenModel) TargetPath() string {
	return filepath.Join(m.ConfigDir, "keys", strings.TrimSpace(m.NameField.Value))
}

func (m KeygenModel) Generate() error {
	name := strings.TrimSpace(m.NameField.Value)
	if name == "" {
		return fmt.Errorf("name required")
	}
	if !validKeyName.MatchString(name) {
		return fmt.Errorf("name must match [a-zA-Z0-9_][a-zA-Z0-9._-]* (no path separators, no '.'/'..')")
	}
	return keygen.Generate(m.TargetPath(), m.PassphraseField.Value)
}

func (m KeygenModel) Init() tea.Cmd { return nil }

func (m KeygenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	fields := []*FormField{m.NameField, m.PassphraseField}
	switch k.Type {
	case tea.KeyEsc:
		return m, func() tea.Msg { return KeygenCancelMsg{} }
	case tea.KeyTab, tea.KeyDown:
		m.Focus = (m.Focus + 1) % len(fields)
	case tea.KeyShiftTab, tea.KeyUp:
		m.Focus = (m.Focus - 1 + len(fields)) % len(fields)
	case tea.KeyBackspace:
		f := fields[m.Focus]
		if len(f.Value) > 0 {
			f.Value = f.Value[:len(f.Value)-1]
		}
	case tea.KeyRunes:
		fields[m.Focus].Value += string(k.Runes)
	case tea.KeySpace:
		fields[m.Focus].Value += " "
	case tea.KeyEnter:
		err := m.Generate()
		return m, func() tea.Msg { return KeygenDoneMsg{Path: m.TargetPath(), Err: err} }
	}
	return m, nil
}

func (m KeygenModel) View() string {
	const innerW = 48
	var b strings.Builder
	title := lipgloss.NewStyle().
		Width(innerW).
		Align(lipgloss.Center).
		Bold(true).
		Foreground(lipgloss.Color(gbYellow)).
		Render("Generate SSH Key (ed25519)")
	b.WriteString(title)
	b.WriteString("\n\n")
	fields := []*FormField{m.NameField, m.PassphraseField}
	const leftPad = "          "
	for i, f := range fields {
		val := f.Value
		if f.Secret {
			val = strings.Repeat("*", len(f.Value))
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
		Render("[Enter] generate   [Esc] cancel   [Tab] next")
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
