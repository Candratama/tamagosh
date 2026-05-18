package ui

import (
	"github.com/charmbracelet/lipgloss"
)

const splashArt = `                                ÆÆÆÆÆÆÆ
                           ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
                        ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
                      ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
                    ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
                   ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
                  ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
                 ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
                ÆÆÆÆÆÆÆÆÆ     ÆÆÆÆÆÆÆÆÆÆÆ     ÆÆÆÆÆÆÆÆÆ
               ÆÆÆÆÆÆÆÆÆÆÆ   ÆÆÆÆÆÆÆÆÆÆÆÆÆ   ÆÆÆÆÆÆÆÆÆÆÆ
              ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
             ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
             ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
            ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
            ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
            ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
           ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
           ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
           ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
           ÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆÆ
           ÆÆÆÆÆÆÆ     ÆÆÆÆÆÆÆÆÆÆ     ÆÆÆÆÆÆÆÆÆÆ     ÆÆÆÆÆÆÆ
           ÆÆÆÆÆ        ÆÆÆÆÆÆÆÆ       ÆÆÆÆÆÆÆÆ        ÆÆÆÆÆ
           ÆÆÆÆ           ÆÆÆÆ           ÆÆÆÆ           ÆÆÆÆ`

const splashText = ` _____  _    __  __    _     ___    ___   ___  _  _
|_   _|/_\  |  \/  |  /_\   / __|  / _ \ / __|| || |
  | | / _ \ | |\/| | / _ \ | (_ | | (_) |\__ \| __ |
  |_|/_/ \_\|_|  |_|/_/ \_\ \___|  \___/ |___/|_||_|`

func renderSplash(width, height int) string {
	logoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(gbYellow)).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(gbOrange)).Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(gbFgMute))

	logo := logoStyle.Render(splashArt)
	text := textStyle.Render(splashText)
	hint := hintStyle.Render("press any key to continue")

	block := logo + "\n\n" + text + "\n\n" + hint
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, block)
}
