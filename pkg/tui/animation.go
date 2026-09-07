package tui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	eqPatterns = [][]int{
		{1, 2, 4, 6, 5, 3, 2, 4, 6, 7, 5, 2, 1, 3, 5, 4, 2},
		{2, 4, 6, 7, 6, 4, 1, 3, 5, 6, 5, 3, 2, 5, 6, 5, 1},
		{3, 6, 7, 5, 3, 2, 3, 5, 7, 6, 4, 2, 4, 6, 7, 3, 2},
		{5, 7, 6, 3, 2, 4, 5, 7, 5, 3, 2, 4, 6, 7, 6, 2, 4},
		{6, 5, 4, 2, 4, 6, 7, 5, 2, 1, 3, 5, 7, 5, 3, 4, 5},
		{7, 5, 2, 1, 5, 7, 6, 3, 2, 4, 5, 7, 5, 2, 4, 6, 7},
		{5, 3, 1, 3, 5, 6, 4, 2, 4, 6, 7, 5, 3, 1, 5, 7, 5},
		{4, 2, 3, 5, 7, 5, 2, 4, 6, 7, 5, 3, 1, 3, 5, 6, 4},
	}

	waveBars    = []rune{' ', ' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	waveBarsWin = []rune{' ', ' ', '▂', '▃', '▄', '▄', '█', '█', '█'}

	spinners    = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnersWin = []string{"|", "/", "-", "\\"}

	pulseDots    = []string{"·", "•", "●", "•"}
	pulseDotsWin = []string{".", "o", "O", "o"}

	JadePalette = []string{
		"#73daca", "#2ac3de", "#7aa2f7", "#9ece6a", "#73daca",
	}

	TokyoPalette = []string{
		"#7aa2f7", "#bb9af7", "#73daca", "#2ac3de",
	}

	CoralPalette = []string{
		"#f7768e", "#ff9e64", "#e0af68", "#f7768e",
	}

	asciiBanner = []string{
		" ▄███▄ █   █ ▄███▄ ████▄ █   █ ████ █    ▄███▄ ▄███▄ ",
		" █▄▄▄▄ █   █ █▄▄▄▄ █▄▄▄█  █▄█  █▄▄  █    █▄▄▄█ █     ",
		" █▀▀▀▀ ▀█▄█▀ █▀▀▀▀ █▀▀▄    █   █    █    █   █ █     ",
		" ▀███▀   ▀   ▀███▀ █  ▀█   █   █    ████ █   █ ▀███▀ ",
	}
)

func GetASCIIBanner(frame int, isYouTube bool, animated bool) string {
	period := 30

	var colors []lipgloss.Color
	if !isYouTube {
		colors = []lipgloss.Color{
			lipgloss.Color("#c8fdf0"),
			lipgloss.Color("#73daca"),
			lipgloss.Color("#9ece6a"),
			lipgloss.Color("#2ac3de"),
			lipgloss.Color("#7aa2f7"),
			lipgloss.Color("#41a6b5"),
		}
	} else {
		colors = []lipgloss.Color{
			lipgloss.Color("#fff0f3"),
			lipgloss.Color("#ff9e64"),
			lipgloss.Color("#f7768e"),
			lipgloss.Color("#e0af68"),
			lipgloss.Color("#bb9af7"),
			lipgloss.Color("#b55a6d"),
		}
	}

	var sb strings.Builder
	for y, line := range asciiBanner {
		for x, r := range line {
			if r == ' ' {
				sb.WriteRune(' ')
				continue
			}

			var pos int
			if animated {

				pos = (x + y*2 - frame*2) % period
				if pos < 0 {
					pos += period
				}
			} else {

				pos = (x / 2) % period
			}

			var c lipgloss.Color
			isBold := false

			switch {
			case pos < 3:
				c = colors[0]
				isBold = true
			case pos < 7:
				c = colors[1]
				isBold = true
			case pos < 12:
				c = colors[2]
			case pos < 17:
				c = colors[3]
			case pos < 23:
				c = colors[4]
			default:
				c = colors[5]
			}

			st := lipgloss.NewStyle().Foreground(c)
			if isBold {
				st = st.Bold(true)
			}
			sb.WriteString(st.Render(string(r)))
		}
		if y < len(asciiBanner)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func RenderGradient(text string, frame int, palette []string) string {
	if len(palette) == 0 {
		return text
	}
	var sb strings.Builder
	charIdx := 0
	for _, r := range text {
		if r == ' ' || r == '\t' || r == '\n' {
			sb.WriteRune(r)
			continue
		}
		colorHex := palette[(charIdx+frame)%len(palette)]
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex)).Bold(true)
		sb.WriteString(style.Render(string(r)))
		charIdx++
	}
	return sb.String()
}

func GetEqualizer(frame int, count int, isPlaying bool, style lipgloss.Style) string {
	if count <= 0 {
		count = 12
	}

	bars := waveBars
	if runtime.GOOS == "windows" {
		bars = waveBarsWin
	}

	var sb strings.Builder
	if !isPlaying {

		idleHeights := []int{1, 2, 3, 4, 3, 2, 1, 2, 3, 2, 1, 2}
		offset := (frame / 2) % len(idleHeights)
		for i := 0; i < count; i++ {
			h := idleHeights[(i+offset)%len(idleHeights)]
			sb.WriteRune(bars[h])
		}
		return style.Render(sb.String())
	}

	pat := eqPatterns[frame%len(eqPatterns)]
	for i := 0; i < count; i++ {
		h := pat[i%len(pat)]
		sb.WriteRune(bars[h])
	}

	return style.Render(sb.String())
}

func GetProgressBar(percent float64, width int, isYouTube bool) string {
	if width <= 0 {
		width = 20
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filledLen := int((percent / 100.0) * float64(width))
	if filledLen > width {
		filledLen = width
	}

	fillColor := JadeMint
	if isYouTube {
		fillColor = TokyoRed
	}

	fillStyle := lipgloss.NewStyle().Foreground(fillColor)
	emptyStyle := lipgloss.NewStyle().Foreground(BorderDark)

	emptyChar := "░"
	if runtime.GOOS == "windows" {
		emptyChar = "─"
	}

	var sb strings.Builder
	for i := 0; i < width; i++ {
		if i < filledLen {
			sb.WriteString(fillStyle.Render("█"))
		} else {
			sb.WriteString(emptyStyle.Render(emptyChar))
		}
	}
	return sb.String()
}

func GetSpinner(frame int) string {
	if runtime.GOOS == "windows" {
		return spinnersWin[frame%len(spinnersWin)]
	}
	return spinners[frame%len(spinners)]
}

func GetPulseDot(frame int) string {
	if runtime.GOOS == "windows" {
		return pulseDotsWin[(frame/2)%len(pulseDotsWin)]
	}
	return pulseDots[(frame/2)%len(pulseDots)]
}

func GetDeck(frame int, sourceMode string, outputDir string, animated bool) string {
	isYT := sourceMode == "youtube"

	banner := GetASCIIBanner(frame, isYT, animated)
	accentColor := JadeMint
	modeTag := "SPOTIFY"
	if isYT {
		accentColor = TokyoRed
		modeTag = "YOUTUBE MUSIC"
	}

	wave := GetEqualizer(frame, 16, false, lipgloss.NewStyle().Foreground(accentColor))
	statusLine := fmt.Sprintf("  %s  %s  •  %s", wave, SubtextStyle.Render("Engine: "+modeTag+" [Ready]"), SubtextStyle.Render("Hi-Fi 24-bit FLAC"))

	p1 := TextNormalStyle("  • ") + SubtextStyle.Render("Paste Spotify/YouTube link or search song to download in lossless FLAC")
	p2 := TextNormalStyle("  • ") + SubtextStyle.Render(fmt.Sprintf("Auto-tagged with high-res cover art & metadata  •  Saved to: %s", truncate(outputDir, 28)))

	content := []string{
		banner,
		statusLine,
		p1,
		p2,
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BorderDark).
		Padding(0, 1)

	return cardStyle.Render(strings.Join(content, "\n"))
}

func TextNormalStyle(s string) string {
	return lipgloss.NewStyle().Foreground(TextNormal).Render(s)
}

func FormatSpeedWithUnit(bytesPerSec int64) string {
	if bytesPerSec <= 0 {
		return ""
	}
	mb := float64(bytesPerSec) / (1024 * 1024)
	if mb >= 1.0 {
		return fmt.Sprintf("%.1f MB/s", mb)
	}
	kb := float64(bytesPerSec) / 1024
	return fmt.Sprintf("%.0f KB/s", kb)
}
