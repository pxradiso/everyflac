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

	waveBars    = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	waveBarsWin = []rune{' ', ' ', '_', '_', '-', '-', '=', '=', '#'}

	asciiBanner = []string{
		" ▄███▄ █   █ ▄███▄ ████▄ █   █ ████ █    ▄███▄ ▄███▄ ",
		" █▄▄▄▄ █   █ █▄▄▄▄ █▄▄▄█  █▄█  █▄▄  █    █▄▄▄█ █     ",
		" █▀▀▀▀ ▀█▄█▀ █▀▀▀▀ █▀▀▄    █   █    █    █   █ █     ",
		" ▀███▀   ▀   ▀███▀ █  ▀█   █   █    ████ █   █ ▀███▀ ",
	}
)

// GetASCIIBanner is used by the headless CLI and the help output. The TUI
// itself uses a compact text header instead.
func GetASCIIBanner(frame int, isYouTube bool, animated bool) string {
	period := 30

	base := Accent
	soft := AccentSoft
	if isYouTube {
		base = Rose
		soft = lipgloss.Color("#d9a7a7")
	}

	var sb strings.Builder
	for y, line := range asciiBanner {
		for x, r := range line {
			if r == ' ' {
				sb.WriteRune(' ')
				continue
			}

			pos := (x / 3) % period
			if animated {
				pos = (x + y*2 - frame*2) % period
				if pos < 0 {
					pos += period
				}
			}

			c := base
			bold := false
			switch {
			case pos < 4:
				c = soft
				bold = true
			case pos < 14:
				c = base
			default:
				c = TextMuted
			}

			st := lipgloss.NewStyle().Foreground(c)
			if bold {
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
		idleHeights := []int{1, 2, 3, 2, 1, 2, 3, 2, 1, 2}
		offset := (frame / 3) % len(idleHeights)
		for i := 0; i < count; i++ {
			sb.WriteRune(bars[idleHeights[(i+offset)%len(idleHeights)]])
		}
		return style.Render(sb.String())
	}

	pat := eqPatterns[frame%len(eqPatterns)]
	for i := 0; i < count; i++ {
		sb.WriteRune(bars[pat[i%len(pat)]])
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

	fillColor := Accent
	if isYouTube {
		fillColor = Rose
	}

	fillChar, emptyChar := "━", "─"
	if windowsConsole {
		fillChar, emptyChar = "=", "-"
	}
	fill := lipgloss.NewStyle().Foreground(fillColor).Render(strings.Repeat(fillChar, filledLen))
	empty := lipgloss.NewStyle().Foreground(Line).Render(strings.Repeat(emptyChar, width-filledLen))
	return fill + empty
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
