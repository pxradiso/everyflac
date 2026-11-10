package tui

import (
	"fmt"
	"strings"

	"everyflac-tui/pkg/models"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.showHelp {
		return m.renderHelpModal()
	}

	w := m.width
	if w <= 0 {
		w = 100
	}
	h := m.height
	if h <= 0 {
		h = 34
	}

	pageW := max(40, w-2)
	accent := accentFor(m.cfg.SourceMode)
	panelH := max(6, h-14)

	content := strings.Join([]string{
		m.renderHeader(pageW, accent),
		"",
		m.renderEngineRow(pageW, accent),
		"",
		m.renderInput(pageW, accent),
		"",
		m.renderTabs(accent),
		"",
		m.renderContentPanel(pageW, panelH, accent),
		"",
		m.renderFooter(pageW),
	}, "\n")

	return lipgloss.NewStyle().Padding(0, 1).Render(content)
}

func (m Model) renderHeader(w int, accent lipgloss.Color) string {
	bar := accentStyle(accent).Render(brandBar())
	left := bar + " " + BrandStyle.Render("everyFLAC") + " " + VersionStyle.Render(Version)
	if m.updateAvailable && w >= 64 {
		left += "  " + accentStyle(accent).Render("update v"+m.latestVersion)
	}

	status := "ready"
	statusStyle := DimStyle
	if m.isResolving {
		status = "resolving"
		statusStyle = StatusWarn
	}
	if m.isDownloading {
		status = "downloading"
		statusStyle = StatusBusy
	}

	right := statusStyle.Render(status)
	if w >= 68 {
		dir := shortenPath(m.cfg.OutputDir, clamp(w/3, 16, 30))
		right = DimStyle.Render(dir) + "   " + right
	}

	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		right = statusStyle.Render(status)
		gap = w - lipgloss.Width(left) - lipgloss.Width(right)
	}
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) renderEngineRow(w int, accent lipgloss.Color) string {
	isYT := m.cfg.SourceMode == "youtube"
	active := lipgloss.NewStyle().Bold(true).Foreground(accent)

	spotify := DimStyle.Render("spotify")
	youtube := DimStyle.Render("youtube music")
	if isYT {
		youtube = active.Render("[ youtube music ]")
	} else {
		spotify = active.Render("[ spotify ]")
	}

	left := SectionStyle.Render("engine") + "   " + spotify + "   " + youtube

	hint := ""
	if w >= 86 {
		hint = DimStyle.Render("ctrl+1 / ctrl+2")
	}
	return joinLR(left, hint, w)
}

func (m Model) renderInput(w int, accent lipgloss.Color) string {
	label := SectionStyle.Render("search")
	value := m.textInput.View()
	hint := ""

	if m.isSettingDir {
		label = SectionStyle.Render("destination")
		value = m.dirInput.View()
		hint = DimStyle.Render("enter save   esc cancel")
	}
	if m.isResolving {
		label = StatusWarn.Render("resolving metadata" + ellipsis())
	} else if m.isDownloading {
		label = StatusBusy.Render("downloading" + ellipsis())
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(accent).
		Padding(0, 1).
		Width(max(20, w-4)).
		Render(strings.TrimRight(value, "\r\n"))

	return joinLR(label, hint, w) + "\n" + box
}

func (m Model) renderTabs(accent lipgloss.Color) string {
	names := []string{"queue", "library"}
	counts := []int{0, len(m.library)}
	if m.currentBatch != nil {
		counts[0] = len(m.currentBatch.Tracks)
	}

	sep := "   "
	var labels, underline strings.Builder
	for i, name := range names {
		text := name
		if counts[i] > 0 {
			text = fmt.Sprintf("%s %d", name, counts[i])
		}

		if i > 0 {
			labels.WriteString(sep)
			underline.WriteString(strings.Repeat(" ", len(sep)))
		}

		if m.activeTab == i {
			labels.WriteString(lipgloss.NewStyle().Bold(true).Foreground(accent).Render(text))
			underline.WriteString(accentStyle(accent).Render(strings.Repeat("─", lipgloss.Width(text))))
		} else {
			labels.WriteString(DimStyle.Render(text))
			underline.WriteString(strings.Repeat(" ", lipgloss.Width(text)))
		}
	}

	return labels.String() + "\n" + underline.String()
}

func (m Model) renderContentPanel(w, h int, accent lipgloss.Color) string {
	innerW := w - 4
	innerH := h - 2
	title := "queue"
	body := ""

	switch m.activeTab {
	case 0:
		if m.currentBatch != nil && len(m.currentBatch.Tracks) > 0 {
			title = "queue" + tabSeparator() + fmt.Sprint(len(m.currentBatch.Tracks))
		}
		body = m.renderQueueBody(innerW, innerH, accent)
	case 1:
		if len(m.library) > 0 {
			title = "library" + tabSeparator() + fmt.Sprint(len(m.library))
		} else {
			title = "library"
		}
		body = m.renderLibraryBody(innerW, innerH)
	}

	return renderPanel(title, body, w, h, accent)
}

func (m Model) renderQueueBody(innerW, innerH int, accent lipgloss.Color) string {
	if m.currentBatch == nil || len(m.currentBatch.Progress) == 0 {
		return m.renderEmptyQueue(innerW, innerH)
	}

	platform := strings.ToLower(string(m.currentBatch.Platform))
	batchName := truncateWidth(m.currentBatch.Name, max(10, innerW/2))
	header := accentStyle(accent).Render(platform) + "   " + BrandStyle.Render(batchName)
	if m.currentBatch.Subtitle != "" {
		header += "   " + DimStyle.Render(truncateWidth(m.currentBatch.Subtitle, max(8, innerW/3)))
	}

	numW, titleW, artistW, progW, statusW := queueColumns(innerW)
	cols := cell("#", numW, DimStyle) + "  " +
		cell("title", titleW, DimStyle) + "  " +
		cell("artist", artistW, DimStyle) + "  " +
		cell("progress", progW, DimStyle) + "  " +
		cell("status", statusW, DimStyle)

	lines := []string{header, "", cols, ""}
	maxRows := max(1, innerH-4)

	for i, p := range m.currentBatch.Progress {
		if i >= maxRows {
			lines = append(lines, DimStyle.Render(ellipsis()+fmt.Sprintf(" %d more", len(m.currentBatch.Progress)-i)))
			break
		}
		lines = append(lines, m.renderTrackRow(p, numW, titleW, artistW, progW, statusW))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderTrackRow(p *models.TrackProgress, numW, titleW, artistW, progW, statusW int) string {
	if p == nil || p.Track == nil {
		return ""
	}

	num := cell(fmt.Sprintf("%02d", p.Track.TrackNumber), numW, DimStyle)
	title := cell(p.Track.Title, titleW, TrackTitleStyle)
	artist := cell(p.Track.Artist, artistW, ArtistStyle)
	bar := GetProgressBar(p.ProgressPercent, progW, m.cfg.SourceMode == "youtube")

	statusText, statusStyle := trackStatus(p)
	status := cell(statusText, statusW, statusStyle)

	return num + "  " + title + "  " + artist + "  " + bar + "  " + status
}

func trackStatus(p *models.TrackProgress) (string, lipgloss.Style) {
	switch p.Status {
	case models.StatusPending:
		return "queued", DimStyle
	case models.StatusResolving:
		return "resolving", StatusWarn
	case models.StatusDownloading:
		text := fmt.Sprintf("%.0f%%", p.ProgressPercent)
		if p.SpeedBytesPerSec > 0 {
			text += "  " + FormatSpeedWithUnit(p.SpeedBytesPerSec)
		}
		return text, StatusBusy
	case models.StatusTagging:
		return "tagging", StatusWarn
	case models.StatusComplete:
		return "done", StatusSuccess
	case models.StatusFailed:
		return "failed", StatusFailed
	default:
		return strings.ToLower(string(p.Status)), DimStyle
	}
}

func (m Model) renderEmptyQueue(innerW, innerH int) string {
	var lines []string
	if m.visualFX {
		wave := GetEqualizer(m.animFrame, 9, false, lipgloss.NewStyle().Foreground(LineStrong))
		lines = append(lines, wave, "")
	}
	lines = append(lines,
		TextStyle.Render("nothing queued"),
		"",
		DimStyle.Render("paste a link or type a song name, then press enter"),
	)
	return centerBlock(lines, innerW, innerH)
}

func (m Model) renderLibraryBody(innerW, innerH int) string {
	if len(m.library) == 0 {
		return centerBlock([]string{
			TextStyle.Render("no downloads yet"),
			"",
			DimStyle.Render("completed files will show up here"),
		}, innerW, innerH)
	}

	titleW := clamp(innerW*36/100, 14, 44)
	artistW := clamp(innerW*22/100, 10, 26)
	qualityW := clamp(innerW*20/100, 10, 24)
	sizeW := 9

	used := titleW + artistW + qualityW + sizeW + 6
	if used > innerW && titleW > 12 {
		titleW -= min(used-innerW, titleW-12)
	} else if used < innerW {
		titleW += innerW - used
	}

	cols := cell("title", titleW, DimStyle) + "  " +
		cell("artist", artistW, DimStyle) + "  " +
		cell("quality", qualityW, DimStyle) + "  " +
		cellRight("size", sizeW, DimStyle)

	lines := []string{cols, ""}
	maxRows := max(1, innerH-2)

	start := 0
	if len(m.library) > maxRows {
		start = len(m.library) - maxRows
	}

	for i := len(m.library) - 1; i >= start; i-- {
		item := m.library[i]
		size := fmt.Sprintf("%.1f MB", float64(item.FileSize)/(1024*1024))
		lines = append(lines,
			cell(item.Title, titleW, TrackTitleStyle)+"  "+
				cell(item.Artist, artistW, ArtistStyle)+"  "+
				cell(item.QualityLabel, qualityW, MutedStyle)+"  "+
				cellRight(size, sizeW, DimStyle),
		)
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderFooter(w int) string {
	var pairs [][2]string
	if w < 90 {
		pairs = [][2]string{
			{"enter", "download"},
			{"tab", "switch"},
			{"ctrl+1/2", "engine"},
			{"ctrl+a", "fx"},
			{"ctrl+c", "quit"},
			{"ctrl+h", "help"},
		}
	} else {
		pairs = [][2]string{
			{"enter", "download"},
			{"tab", "switch"},
			{"ctrl+1/2", "engine"},
			{"ctrl+a", "fx"},
			{"ctrl+h", "help"},
			{"ctrl+c", "quit"},
			{"ctrl+o", "open"},
			{"ctrl+d", "dir"},
		}
	}
	return renderKeyHints(pairs, w)
}

func (m Model) renderHelpModal() string {
	w := m.width
	if w <= 0 {
		w = 100
	}
	h := m.height
	if h <= 0 {
		h = 34
	}

	rows := [][2]string{
		{"enter", "download the current input"},
		{"tab", "next tab (queue, library)"},
		{"shift+tab", "previous tab"},
		{"ctrl+1/2", "spotify or youtube music"},
		{"ctrl+t", "toggle source engine"},
		{"ctrl+d", "change destination folder"},
		{"ctrl+o", "open destination folder"},
		{"ctrl+a", "toggle visual effects"},
		{"ctrl+h", "open this panel"},
		{"ctrl+c", "quit"},
		{"esc", "unfocus or close panel"},
		{"i, /", "focus the input"},
		{"a, o, q, ?", "same actions when unfocused"},
	}

	lines := []string{SectionStyle.Render("keyboard shortcuts"), ""}
	for _, row := range rows {
		lines = append(lines, KeyStyle.Render(fmt.Sprintf("%-13s", row[0]))+" "+DimStyle.Render(row[1]))
	}
	if m.updateAvailable {
		lines = append(lines, "",
			accentStyle(accentFor(m.cfg.SourceMode)).Render("update v"+m.latestVersion+" available"+tabSeparator()+"re-run the installer"),
		)
	}
	lines = append(lines,
		"",
		DimStyle.Render("esc closes"+tabSeparator()+"ctrl keys work while typing"),
	)

	modalW := clamp(w-10, 46, 74)
	if modalW > w-2 {
		modalW = max(30, w-2)
	}
	modalH := len(lines) + 2

	panel := renderPanel("keys", strings.Join(lines, "\n"), modalW, modalH, accentFor(m.cfg.SourceMode))
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, panel)
}

func queueColumns(innerW int) (numW, titleW, artistW, progW, statusW int) {
	numW = 3
	progW = clamp(innerW*18/100, 10, 22)
	titleW = clamp(innerW*34/100, 14, 40)
	artistW = clamp(innerW*22/100, 10, 26)
	statusW = innerW - numW - titleW - artistW - progW - 8

	for statusW < 9 && titleW > 12 {
		titleW--
		statusW++
	}
	for statusW < 9 && artistW > 9 {
		artistW--
		statusW++
	}
	if statusW < 0 {
		statusW = 0
	}
	return
}

func shortenPath(path string, width int) string {
	if lipgloss.Width(path) <= width {
		return path
	}
	ell := ellipsis()
	ellW := lipgloss.Width(ell)
	if width <= ellW {
		if windowsConsole {
			return ell[:width]
		}
		return ell
	}
	runes := []rune(path)
	if len(runes) <= width-ellW {
		return path
	}
	return ell + string(runes[len(runes)-(width-ellW):])
}
