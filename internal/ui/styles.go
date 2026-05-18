package ui

import "github.com/charmbracelet/lipgloss"

var (
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Padding(0, 1)

	StyleSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)

	StyleNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	StyleHelp = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	StyleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

	StylePaneActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("212")).
			Padding(0, 1)

	StylePaneInactive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	StyleKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color("51")).
			Bold(true)

	StyleKeyBracket = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	StyleKeyLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	StyleSection = lipgloss.NewStyle().
			Foreground(lipgloss.Color("213")).
			Bold(true)

	StyleConfirm = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(0, 1)
)
