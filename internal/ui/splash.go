package ui

import (
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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

const splashText = ` ████████  █████  ███    ███  █████   ██████   ██████  ███████ ██   ██
    ██    ██   ██ ████  ████ ██   ██ ██       ██    ██ ██      ██   ██
    ██    ███████ ██ ████ ██ ███████ ██   ███ ██    ██ ███████ ███████
    ██    ██   ██ ██  ██  ██ ██   ██ ██    ██ ██    ██      ██ ██   ██
    ██    ██   ██ ██      ██ ██   ██  ██████   ██████  ███████ ██   ██`

const splashGhost = ` ●●●●●●●
●●○●●●○●●
●●●●●●●●●
●●●●v●●●●
 ●●●●●●● `

var (
	bgGrid string
	bgW    int
	bgH    int
)

func renderBackground(width, height int) string {
	if width != bgW || height != bgH || bgGrid == "" {
		r := rand.New(rand.NewSource(42))
		var lines []string
		for y := 0; y < height; y++ {
			row := make([]byte, width)
			for x := 0; x < width; x++ {
				if r.Intn(2) == 0 {
					row[x] = '0'
				} else {
					row[x] = '1'
				}
			}
			lines = append(lines, string(row))
		}
		bgGrid = strings.Join(lines, "\n")
		bgW = width
		bgH = height
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2a2a")).Render(bgGrid)
}

func overlayOnBackground(view string, width, height int) string {
	bg := renderBackground(width, height)
	baseLines := strings.Split(bg, "\n")
	boxLines := strings.Split(view, "\n")
	for len(baseLines) < height {
		baseLines = append(baseLines, "")
	}
	bl := len(baseLines)
	btl := len(boxLines)
	if btl > bl {
		btl = bl
		boxLines = boxLines[:bl]
	}
	startRow := (bl - btl) / 2
	if startRow < 0 {
		startRow = 0
	}
	for i := 0; i < btl && startRow+i < bl; i++ {
		boxLine := boxLines[i]
		bw := lipgloss.Width(boxLine)
		col := (width - bw) / 2
		if col < 0 {
			col = 0
		}
		baseLine := baseLines[startRow+i]
		baseLineW := lipgloss.Width(baseLine)
		if baseLineW < width {
			baseLine += strings.Repeat(" ", width-baseLineW)
			baseLineW = width
		}
		leftPart := ansi.Cut(baseLine, 0, col)
		rightPart := ansi.Cut(baseLine, col+bw, baseLineW)
		baseLines[startRow+i] = leftPart + boxLine + rightPart
	}
	return strings.Join(baseLines, "\n")
}

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
