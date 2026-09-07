package tui

import (
	"fmt"
	"runtime"
	"strings"

	"everyflac-tui/pkg/models"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.showHelp {
		return m.renderHelpModal()
	}

	var s strings.Builder

	s.WriteString(m.renderHeader())
	s.WriteString("\n")

	s.WriteString(m.renderSourceSelector())
	s.WriteString("\n")

	if m.isSettingDir {
		s.WriteString(m.renderDirBox())
	} else {
		s.WriteString(m.renderInputBox())
	}
	s.WriteString("\n")

	s.WriteString(m.renderTabBar())
	s.WriteString("\n")

	switch m.activeTab {
	case 0:
		s.WriteString(m.renderQueueTab())
	case 1:
		s.WriteString(m.renderLibraryTab())
	case 2:
		s.WriteString(m.renderLogsTab())
	}

	s.WriteString("\n")
	s.WriteString(m.renderFooter())

	return s.String()
}

func (m Model) renderHeader() string {
	var title string
	if m.visualFX {
		palette := JadePalette
		if m.cfg.SourceMode == "youtube" {
			palette = CoralPalette
		}
		title = RenderGradient("⚡ everyFLAC by pxradise", m.animFrame/3, palette)
	} else {
		title = TitleStyle.Render("⚡ everyFLAC by pxradise")
	}

	isBusy := m.isDownloading || m.isResolving
	eqColor := EqualizerSpotifyStyle
	if m.cfg.SourceMode == "youtube" {
		eqColor = EqualizerYTStyle
	}
	eqWave := GetEqualizer(m.animFrame, 12, isBusy, eqColor)

	qualityBadge := BadgeHifiStyle.Render("UNIVERSAL LOSSLESS FLAC")
	dirBadge := SubtextStyle.Render(fmt.Sprintf("📂 %s", truncate(m.cfg.OutputDir, 30)))

	if m.width > 0 && m.width < 95 {
		return lipgloss.JoinHorizontal(lipgloss.Center, title, " ", eqWave, "   ", dirBadge)
	}

	headerLeft := lipgloss.JoinHorizontal(lipgloss.Center, title, " ", eqWave, " ", qualityBadge)

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		headerLeft,
		"   ",
		dirBadge,
	)
}

func (m Model) renderSourceSelector() string {
	isSpotify := m.cfg.SourceMode != "youtube"

	var spBadge, ytBadge string
	if isSpotify {
		spBadge = BadgeSpotifyActive.Render("● SPOTIFY [F1]")
		ytBadge = BadgeYTInactive.Render("○ YOUTUBE MUSIC [F2]")
	} else {
		spBadge = BadgeSpotifyInactive.Render("○ SPOTIFY [F1]")
		ytBadge = BadgeYTActive.Render("● YOUTUBE MUSIC [F2]")
	}

	hint := SubtextStyle.Render(" (Press [F1/F2] or [Ctrl+T] to toggle)")

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		LabelStyle.Render("Engine: "),
		spBadge,
		" ",
		ytBadge,
		hint,
	)
}

func (m Model) renderDirBox() string {
	title := LabelStyle.Render("📂 Set Custom Destination Folder (Saved permanently to config):")
	box := BoxActiveStyle.Copy().
		Width(m.dirInput.Width + 6).
		Render(strings.TrimRight(m.dirInput.View(), "\r\n"))

	hint := SubtextStyle.Render("Press [Enter] to save, [Esc] to cancel")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		box,
		hint,
	)
}

func (m Model) renderInputBox() string {
	var inputTitle string
	modeName := "SPOTIFY"
	boxBorder := BoxActiveStyle
	if m.cfg.SourceMode == "youtube" {
		modeName = "YOUTUBE MUSIC"
		boxBorder = BoxYTStyle
	}

	if m.isResolving {
		dot := GetPulseDot(m.animFrame)
		inputTitle = StatusResolving.Render(fmt.Sprintf("%s Resolving metadata [%s]...", dot, modeName))
	} else if m.isDownloading {
		wave := GetEqualizer(m.animFrame, 8, true, EqualizerStyle)
		inputTitle = StatusDownloading.Render(fmt.Sprintf("%s Downloading & encoding lossless FLAC streams...", wave))
	} else {
		inputTitle = LabelStyle.Render(fmt.Sprintf("Paste %s link or search any song:", modeName))
	}

	inputBox := boxBorder.Copy().
		Width(m.textInput.Width + 6).
		Render(strings.TrimRight(m.textInput.View(), "\r\n"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		inputTitle,
		inputBox,
	)
}

func (m Model) renderTabBar() string {
	tabs := []string{"[1] Active Queue", "[2] Downloaded Library", "[3] Engine Logs"}
	var renderedTabs []string

	for i, t := range tabs {
		count := ""
		if i == 0 && m.currentBatch != nil {
			count = fmt.Sprintf(" (%d)", len(m.currentBatch.Tracks))
		} else if i == 1 {
			count = fmt.Sprintf(" (%d)", len(m.library))
		} else if i == 2 {
			count = fmt.Sprintf(" (%d)", len(m.logs))
		}

		label := t + count
		if m.activeTab == i {
			activeStyle := TabActiveStyle
			if m.cfg.SourceMode == "youtube" {
				activeStyle = activeStyle.Copy().Foreground(TokyoRed)
			}
			renderedTabs = append(renderedTabs, activeStyle.Render("● "+label))
		} else {
			renderedTabs = append(renderedTabs, TabInactiveStyle.Render("○ "+label))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Center, renderedTabs...)
	hint := SubtextStyle.Render(" [Tab / Shift+Tab to cycle]")
	return lipgloss.JoinHorizontal(lipgloss.Center, tabBar, "  ", hint)
}

func (m Model) renderQueueTab() string {
	if m.currentBatch == nil || len(m.currentBatch.Progress) == 0 {
		return GetDeck(m.animFrame, m.cfg.SourceMode, m.cfg.OutputDir, m.visualFX)
	}

	var s strings.Builder

	platformBadge := BadgeSpotifyActive.Render("SPOTIFY")
	if m.currentBatch.Platform == models.PlatformYouTubeMusic {
		platformBadge = BadgeYTActive.Render("YOUTUBE MUSIC")
	} else if m.currentBatch.Platform == models.PlatformYouTube {
		platformBadge = BadgeYTActive.Render("YOUTUBE")
	} else if m.currentBatch.Platform == models.PlatformDirectSearch {
		platformBadge = BadgeHifiStyle.Render("SEARCH")
	}

	header := fmt.Sprintf("  %s %s • %s", platformBadge, TrackTitleStyle.Render(truncate(m.currentBatch.Name, 32)), ArtistStyle.Render(truncate(m.currentBatch.Subtitle, 24)))
	s.WriteString(header + "\n")

	maxTracks := 10
	if m.height > 0 {
		maxTracks = max(3, m.height-16)
	}

	for i, p := range m.currentBatch.Progress {
		if i >= maxTracks {
			remaining := len(m.currentBatch.Progress) - i
			s.WriteString(SubtextStyle.Render(fmt.Sprintf("    ... and %d more track(s) queued\n", remaining)))
			break
		}

		s.WriteString(m.renderTrackRow(p))
		s.WriteString("\n")
	}

	return s.String()
}

func (m Model) renderTrackRow(p *models.TrackProgress) string {
	isYT := m.cfg.SourceMode == "youtube"

	var statusBadge string
	switch p.Status {
	case models.StatusPending:
		pendingIcon := "•"
		if runtime.GOOS == "windows" {
			pendingIcon = "o"
		}
		statusBadge = SubtextStyle.Render(pendingIcon + " Queued")
	case models.StatusResolving:
		dot := GetPulseDot(m.animFrame)
		statusBadge = StatusResolving.Render(fmt.Sprintf("%s Resolving", dot))
	case models.StatusDownloading:
		eq := GetEqualizer(m.animFrame, 4, true, EqualizerStyle)
		if isYT {
			eq = GetEqualizer(m.animFrame, 4, true, EqualizerYTStyle)
		}
		statusBadge = StatusDownloading.Render(fmt.Sprintf("%s %4.1f%%", eq, p.ProgressPercent))
	case models.StatusTagging:
		spinner := GetSpinner(m.animFrame)
		statusBadge = StatusResolving.Render(fmt.Sprintf("%s Tagging", spinner))
	case models.StatusComplete:
		check := "✔"
		if runtime.GOOS == "windows" {
			check = "√"
		}
		statusBadge = StatusSuccess.Render(check + " Lossless FLAC")
	case models.StatusFailed:
		failMark := "✖"
		if runtime.GOOS == "windows" {
			failMark = "×"
		}
		statusBadge = StatusFailed.Render(failMark + " Failed")
	default:
		statusBadge = SubtextStyle.Render(string(p.Status))
	}

	num := fmt.Sprintf("%02d.", p.Track.TrackNumber)
	title := TrackTitleStyle.Render(truncate(p.Track.Title, 28))
	artist := ArtistStyle.Render(truncate(p.Track.Artist, 20))

	speedStr := ""
	if p.SpeedBytesPerSec > 0 {
		speedStr = " " + FormatSpeedWithUnit(p.SpeedBytesPerSec)
	}

	progressView := GetProgressBar(p.ProgressPercent, 20, isYT)

	quality := ""
	if p.QualityLabel != "" {
		quality = SubtextStyle.Render(fmt.Sprintf(" [%s]", p.QualityLabel))
	}

	return fmt.Sprintf("  %-3s %-30s %-22s %-18s %s%s%s",
		num, title, artist, statusBadge, progressView, speedStr, quality)
}

func (m Model) renderLibraryTab() string {
	if len(m.library) == 0 {
		return SubtextStyle.Render("  No downloaded tracks yet in this session.\n  Downloaded FLAC files will appear here with audio specs & file sizes.")
	}

	var s strings.Builder
	s.WriteString(LabelStyle.Render(fmt.Sprintf("  Session Library — %d FLAC files completed:\n\n", len(m.library))))

	maxItems := 5
	if m.height > 0 {
		maxItems = max(2, (m.height-16)/2)
	}

	start := 0
	if len(m.library) > maxItems {
		start = len(m.library) - maxItems
	}

	for i := len(m.library) - 1; i >= start; i-- {
		item := m.library[i]
		sizeMB := float64(item.FileSize) / (1024 * 1024)
		check := "✔"
		if runtime.GOOS == "windows" {
			check = "√"
		}
		s.WriteString(fmt.Sprintf("  %s %s — %s\n     %s • %.1f MB • %s\n\n",
			StatusSuccess.Render(check),
			TrackTitleStyle.Render(truncate(item.Title, 28)),
			ArtistStyle.Render(truncate(item.Artist, 20)),
			BadgeHifiStyle.Render(item.QualityLabel),
			sizeMB,
			SubtextStyle.Render(truncate(item.FilePath, 35)),
		))
	}

	return s.String()
}

func (m Model) renderLogsTab() string {
	if len(m.logs) == 0 {
		return SubtextStyle.Render("  Engine logs are empty.")
	}

	var s strings.Builder
	maxLogs := 10
	if m.height > 0 {
		maxLogs = max(3, m.height-15)
	}

	start := 0
	if len(m.logs) > maxLogs {
		start = len(m.logs) - maxLogs
	}

	for _, l := range m.logs[start:] {
		timeStr := l.Timestamp.Format("15:04:05")
		var lvl string
		switch l.Level {
		case "INFO":
			lvl = SubtextStyle.Render("[INFO]")
		case "SUCCESS":
			lvl = StatusSuccess.Render("[OK]")
		case "WARN":
			lvl = StatusResolving.Render("[WARN]")
		case "ERROR":
			lvl = StatusFailed.Render("[ERR]")
		}

		s.WriteString(fmt.Sprintf("  %s %s %s\n", SubtextStyle.Render(timeStr), lvl, truncate(l.Message, max(30, m.width-25))))
	}

	return s.String()
}

func (m Model) renderFooter() string {
	var keys []string
	if m.width > 0 && m.width < 95 {
		keys = []string{
			"[Enter] Download",
			"[Tab] Tabs",
			"[F1/F2] Engine",
			"[Ctrl+D] Folder",
			"[A] FX",
			"[?] Help",
			"[Q] Quit",
		}
	} else {
		keys = []string{
			"[Enter] Download",
			"[Tab] Tabs",
			"[Ctrl+L] Logs",
			"[F1/F2] Engine",
			"[Ctrl+D] Folder",
			"[A] FX",
			"[O] Open",
			"[?] Help",
			"[Q] Quit",
		}
	}

	return SubtextStyle.Render("  " + strings.Join(keys, "  •  "))
}

func (m Model) renderHelpModal() string {
	content := []string{
		TitleStyle.Render("⚡ everyFLAC by pxradise — Keyboard Shortcuts"),
		"",
		HintKeyStyle.Render(" [Enter] ") + "     Start resolving & downloading input query",
		HintKeyStyle.Render(" [Tab] ") + "       Cycle tabs: Queue -> Library -> Engine Logs",
		HintKeyStyle.Render(" [Shift+Tab] ") + " Cycle tabs in reverse order",
		HintKeyStyle.Render(" [Ctrl+L] ") + "    Jump directly to Engine Logs anytime",
		HintKeyStyle.Render(" [1, 2, 3] ") + "   Jump directly to Queue (1), Library (2), Logs (3)",
		HintKeyStyle.Render(" [F1] ") + "        Switch directly to Spotify Engine",
		HintKeyStyle.Render(" [F2] ") + "        Switch directly to YouTube Music Engine",
		HintKeyStyle.Render(" [Ctrl+T] ") + "    Toggle between Spotify and YouTube Music",
		HintKeyStyle.Render(" [Ctrl+D] ") + "    Change custom destination folder (persisted)",
		HintKeyStyle.Render(" [O] ") + "         Open destination folder in system file manager",
		HintKeyStyle.Render(" [I] or [/] ") + "    Focus search / URL input box",
		HintKeyStyle.Render(" [Esc] ") + "       Unfocus input box / Dismiss dialogs",
		HintKeyStyle.Render(" [Q] ") + "         Quit application",
		"",
		SubtextStyle.Render("Press [Esc] or [?] to close this help window"),
	}

	card := ModalBoxStyle.Render(strings.Join(content, "\n"))
	return "\n\n" + lipgloss.NewStyle().PaddingLeft(4).Render(card)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
