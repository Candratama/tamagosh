package ui

import "github.com/charmbracelet/lipgloss"

// gruvbox material dark hard palette — minimalist subset
const (
	gbFg     = "#d4be98" // primary text
	gbFgMute = "#7c6f64" // secondary / hints / inactive borders
	gbBgSel  = "#45403d" // cursor row background
	gbRed    = "#ea6962" // error / destructive
	gbOrange = "#e78a4e" // section headers
	gbYellow = "#d8a657" // accent (titles, focused)
	gbGreen  = "#a9b665" // active pane / success
	gbAqua   = "#89b482" // interactive keys
	gbBlue   = "#7daea3"
	gbPurple = "#d3869b"
	gbBorder = "#504945" // unified subtle border
)

var (
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(gbYellow))

	StyleSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbFg)).
			Background(lipgloss.Color(gbBgSel)).
			Bold(true)

	StyleNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbFg))

	StyleHelp = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbFgMute))

	StyleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbRed))

	StyleSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbGreen))

	// Unified subtle border — used for all box chrome.
	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(gbBorder)).
			Padding(1, 2)

	// Active pane: same shape as inactive, brighter border to draw the eye.
	StylePaneActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(gbGreen)).
			Padding(0, 1)

	StylePaneInactive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(gbBorder)).
				Padding(0, 1)

	StyleKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbFg)).
			Bold(true)

	StyleKeyBracket = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbFgMute))

	StyleKeyLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbFgMute))

	StyleSection = lipgloss.NewStyle().
			Foreground(lipgloss.Color(gbOrange)).
			Bold(true)

	// Confirm: same border treatment, red foreground accent on the title only.
	StyleConfirm = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(gbRed)).
			Padding(1, 2)
)
