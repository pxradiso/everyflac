package tui

import (
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// windowsConsole is true on Windows, where legacy conhost fonts do not always
// render the decorative glyphs used by the rest of the UI. Keep the layout
// identical, but fall back to ASCII for those accents.
var windowsConsole = runtime.GOOS == "windows"

func ellipsis() string {
	if windowsConsole {
		return "..."
	}
	return "…"
}

func tabSeparator() string {
	if windowsConsole {
		return " - "
	}
	return " · "
}

func brandBar() string {
	if windowsConsole {
		return "|"
	}
	return "▍"
}

// Version is assigned by main at startup.
var Version = "1.3.0"

// A deliberately small palette: neutral surfaces, one accent, three semantic
// colors. Everything else is text at different levels of contrast.
var (
	BgBase = lipgloss.Color("#0e1116")

	Line       = lipgloss.Color("#232936")
	LineStrong = lipgloss.Color("#333b49")

	TextPrimary   = lipgloss.Color("#d7dde7")
	TextSecondary = lipgloss.Color("#98a2b3")
	TextMuted     = lipgloss.Color("#6b7484")
	TextFaint     = lipgloss.Color("#444c5a")

	Accent     = lipgloss.Color("#7fa8c9")
	AccentSoft = lipgloss.Color("#a8c6dd")
	Rose       = lipgloss.Color("#c08a8a")

	Success = lipgloss.Color("#8fbf8f")
	Warning = lipgloss.Color("#d0b070")
	Danger  = lipgloss.Color("#cf8080")
)

var (
	BrandStyle   = lipgloss.NewStyle().Bold(true).Foreground(TextPrimary)
	VersionStyle = lipgloss.NewStyle().Foreground(TextMuted)
	SectionStyle = lipgloss.NewStyle().Foreground(TextMuted)
	TextStyle    = lipgloss.NewStyle().Foreground(TextPrimary)
	MutedStyle   = lipgloss.NewStyle().Foreground(TextSecondary)
	DimStyle     = lipgloss.NewStyle().Foreground(TextMuted)
	FaintStyle   = lipgloss.NewStyle().Foreground(TextFaint)
	KeyStyle     = lipgloss.NewStyle().Foreground(TextSecondary)

	TrackTitleStyle = lipgloss.NewStyle().Foreground(TextPrimary)
	ArtistStyle     = lipgloss.NewStyle().Foreground(TextSecondary)

	StatusSuccess = lipgloss.NewStyle().Foreground(Success)
	StatusFailed  = lipgloss.NewStyle().Foreground(Danger)
	StatusBusy    = lipgloss.NewStyle().Foreground(Accent)
	StatusWarn    = lipgloss.NewStyle().Foreground(Warning)
)

func accentFor(sourceMode string) lipgloss.Color {
	if sourceMode == "youtube" {
		return Rose
	}
	return Accent
}

func accentStyle(c lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(c)
}

// renderPanel draws a box with a title embedded in the top border. Body lines
// are clipped and padded so the panel always occupies exactly width x height
// cells, which keeps the layout stable while content changes.
func renderPanel(title, body string, width, height int, accent lipgloss.Color) string {
	if width < 12 {
		width = 12
	}
	if height < 3 {
		height = 3
	}

	innerW := width - 4
	bodyH := height - 2

	border := lipgloss.NewStyle().Foreground(Line)
	titleStyle := lipgloss.NewStyle().Foreground(accent)

	label := ""
	if title != "" {
		label = " " + title + " "
	}
	fill := width - 3 - lipgloss.Width(label)
	if fill < 0 {
		label = ""
		fill = width - 3
	}

	var b strings.Builder
	b.WriteString(border.Render("┌─"))
	b.WriteString(titleStyle.Render(label))
	b.WriteString(border.Render(strings.Repeat("─", fill) + "┐"))
	b.WriteString("\n")

	lines := strings.Split(body, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > bodyH {
		lines = lines[:bodyH]
	}
	for len(lines) < bodyH {
		lines = append(lines, "")
	}

	for _, line := range lines {
		b.WriteString(border.Render("│"))
		b.WriteString(" ")
		b.WriteString(fitWidth(line, innerW))
		b.WriteString(" ")
		b.WriteString(border.Render("│"))
		b.WriteString("\n")
	}

	b.WriteString(border.Render("└" + strings.Repeat("─", width-2) + "┘"))
	return b.String()
}

// fitWidth truncates ANSI-styled text to width cells and pads it with spaces.
func fitWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = lipgloss.NewStyle().MaxWidth(width).Render(s)
	if pad := width - lipgloss.Width(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

// cell renders a plain-text table cell with a fixed width.
func cell(s string, width int, style lipgloss.Style) string {
	s = truncateWidth(s, width)
	if pad := width - lipgloss.Width(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return style.Render(s)
}

// cellRight is cell with the text aligned to the right edge.
func cellRight(s string, width int, style lipgloss.Style) string {
	s = truncateWidth(s, width)
	if pad := width - lipgloss.Width(s); pad > 0 {
		s = strings.Repeat(" ", pad) + s
	}
	return style.Render(s)
}

// truncateWidth shortens a plain string to width cells, adding an ellipsis.
func truncateWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	ell := ellipsis()
	ellW := lipgloss.Width(ell)
	if width <= ellW {
		if windowsConsole {
			return ell[:width]
		}
		return ell
	}

	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if used+rw > width-ellW {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	return b.String() + ell
}

// centerBlock centers a small group of lines inside the panel body.
func centerBlock(lines []string, width, height int) string {
	if height <= 0 {
		return strings.Join(lines, "\n")
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	top := (height - len(lines)) / 2
	out := make([]string, 0, height)
	for i := 0; i < top; i++ {
		out = append(out, "")
	}
	for _, line := range lines {
		left := (width - lipgloss.Width(line)) / 2
		if left < 0 {
			left = 0
		}
		out = append(out, strings.Repeat(" ", left)+line)
	}
	for len(out) < height {
		out = append(out, "")
	}
	return strings.Join(out, "\n")
}

// renderKeyHints lays out "key action" pairs, dropping the trailing ones when
// the terminal is too narrow.
func renderKeyHints(pairs [][2]string, width int) string {
	var parts []string
	used := 0
	for i, p := range pairs {
		part := KeyStyle.Render(p[0]) + " " + DimStyle.Render(p[1])
		add := lipgloss.Width(part)
		if i > 0 {
			add += 2
		}
		if used+add > width {
			break
		}
		parts = append(parts, part)
		used += add
	}
	return strings.Join(parts, "  ")
}

// joinLR places left and right on the same row, right-aligned.
func joinLR(left, right string, width int) string {
	if right == "" {
		return left
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
