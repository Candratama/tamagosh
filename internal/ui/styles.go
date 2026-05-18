package ui

import "github.com/charmbracelet/lipgloss"

// gruvbox material dark hard palette
const (
	gbFg     = "#d4be98"
	gbFgMute = "#928374"
	gbBgSel  = "#45403d"
	gbRed    = "#ea6962"
	gbOrange = "#e78a4e"
	gbYellow = "#d8a657"
	gbGreen  = "#a9b665"
	gbAqua   = "#89b482"
	gbBlue   = "#7daea3"
	gbPurple = "#d3869b"
	gbBorder = "#7daea3"
)

var (
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(gbYellow)).
			Padding(0, 1)

	StyleSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbFg)).
			Background(lipgloss.Color(gbBgSel)).
			Bold(true)

	StyleNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbFg))

	StyleHelp = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbFgMute))

	StyleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbRed)).
			Bold(true)

	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(gbBlue)).
			Padding(0, 1)

	StylePaneActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(gbGreen)).
			Padding(0, 1)

	StylePaneInactive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(gbFgMute)).
				Padding(0, 1)

	StyleKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbAqua)).
			Bold(true)

	StyleKeyBracket = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbFgMute))

	StyleKeyLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbFg))

	StyleSection = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbOrange)).
			Bold(true)

	StyleConfirm = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(gbRed)).
			Padding(0, 1)
)
