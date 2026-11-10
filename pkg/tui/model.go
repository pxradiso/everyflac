package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"everyflac-tui/pkg/config"
	"everyflac-tui/pkg/downloader"
	"everyflac-tui/pkg/models"
	"everyflac-tui/pkg/spotify"
	"everyflac-tui/pkg/updater"
	"everyflac-tui/pkg/ytmusic"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type batchResolvedMsg struct {
	batch *models.DownloadBatch
	err   error
}

type trackProgressMsg struct {
	prog *models.TrackProgress
}

type trackCompletedMsg struct {
	prog *models.TrackProgress
	err  error
}

type batchCompletedMsg struct{}

type updateCheckMsg struct {
	available bool
	latest    string
}

type animTickMsg time.Time

type Model struct {
	cfg           *config.AppConfig
	spotifyClient *spotify.Client
	ytClient      *ytmusic.Client
	downloader    *downloader.Downloader

	textInput textinput.Model
	dirInput  textinput.Model
	spinner   spinner.Model
	progress  progress.Model

	activeTab     int
	width         int
	height        int
	isResolving   bool
	isDownloading bool
	isSettingDir  bool
	showHelp      bool
	animFrame     int
	visualFX      bool

	updateAvailable bool
	latestVersion   string

	currentBatch *models.DownloadBatch
	library      []models.DownloadedFile

	subChan chan tea.Msg
}

func NewModel(cfg *config.AppConfig) Model {
	prompt := "› "
	if runtime.GOOS == "windows" {
		prompt = "> "
	}

	if cfg.SourceMode != "youtube" {
		cfg.SourceMode = "spotify"
	}
	accent := accentFor(cfg.SourceMode)

	ti := textinput.New()
	ti.Prompt = prompt
	ti.Placeholder = "paste a link or search a song"
	ti.PromptStyle = lipgloss.NewStyle().Foreground(accent)
	ti.TextStyle = lipgloss.NewStyle().Foreground(TextPrimary)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(TextFaint)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(BgBase).Background(accent)
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 60

	di := textinput.New()
	di.Placeholder = "/path/to/Music"
	di.Prompt = prompt
	di.PromptStyle = lipgloss.NewStyle().Foreground(accent)
	di.TextStyle = lipgloss.NewStyle().Foreground(TextPrimary)
	di.PlaceholderStyle = lipgloss.NewStyle().Foreground(TextFaint)
	di.Cursor.Style = lipgloss.NewStyle().Foreground(BgBase).Background(accent)
	di.CharLimit = 500
	di.Width = 60

	sp := spinner.New()
	sp.Spinner = spinner.Pulse
	sp.Style = StatusBusy

	prog := progress.New(
		progress.WithGradient(string(Accent), string(AccentSoft)),
		progress.WithoutPercentage(),
	)
	prog.Width = 50

	return Model{
		cfg:           cfg,
		spotifyClient: spotify.NewClient(),
		ytClient:      ytmusic.NewClient(),
		downloader:    downloader.NewDownloader(cfg),
		textInput:     ti,
		dirInput:      di,
		spinner:       sp,
		progress:      prog,
		activeTab:     0,
		isSettingDir:  false,
		showHelp:      false,
		animFrame:     0,
		visualFX:      true,
		currentBatch:  nil,
		library:       make([]models.DownloadedFile, 0),
		subChan:       make(chan tea.Msg, 500),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		m.animTickCmd(),
		m.listenProgressUpdates(),
		m.checkForUpdate(),
	)
}

// checkForUpdate runs a lightweight background check. It is intentionally
// non-blocking and never installs anything by itself.
func (m Model) checkForUpdate() tea.Cmd {
	return func() tea.Msg {
		res, err := updater.Check(Version)
		if err != nil || !res.Available {
			return updateCheckMsg{}
		}
		return updateCheckMsg{available: true, latest: res.Latest}
	}
}

func (m Model) animTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

func (m Model) listenProgressUpdates() tea.Cmd {
	return func() tea.Msg {
		return <-m.subChan
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Some terminals encode Ctrl+1/Ctrl+2 as CSI-u sequences instead of a
	// control byte. Catch them here so the engine switch works while typing.
	if mode := extendedEngineKey(msg); mode != "" {
		m.setSourceMode(mode)
		return m, nil
	}

	switch msg := msg.(type) {
	case animTickMsg:
		m.animFrame++
		return m, m.animTickCmd()

	case updateCheckMsg:
		if msg.available {
			m.updateAvailable = true
			m.latestVersion = msg.latest
		}
		return m, nil

	case tea.KeyMsg:
		k := msg.String()

		if k == "ctrl+c" {
			return m, tea.Quit
		}

		if m.showHelp {
			if k == "esc" || k == "?" || k == "h" || k == "ctrl+h" || k == "enter" || k == "q" {
				m.showHelp = false
				return m, tea.ClearScreen
			}
			return m, nil
		}

		switch k {
		case "f1", "alt+1", "alt+s", "ctrl+1":
			m.setSourceMode("spotify")
			return m, nil
		case "f2", "alt+2", "alt+y", "ctrl+2", "ctrl+@":
			m.setSourceMode("youtube")
			return m, nil
		case "ctrl+t":
			m.toggleSourceMode()
			return m, nil
		case "ctrl+a":
			m.toggleVisualFX()
			return m, nil
		case "ctrl+o":
			m.openOutputDir()
			return m, nil
		case "ctrl+h":
			m.showHelp = true
			return m, tea.ClearScreen
		}

		if k == "ctrl+d" {
			if !m.isSettingDir {
				m.isSettingDir = true
				m.textInput.Blur()
				m.dirInput.SetValue(m.cfg.OutputDir)
				m.dirInput.Focus()
				return m, textinput.Blink
			}
		}

		if m.isSettingDir {
			switch k {
			case "enter":
				newDir := config.ExpandPath(strings.TrimSpace(m.dirInput.Value()))
				if newDir != "" {
					_ = os.MkdirAll(newDir, 0755)
					m.cfg.OutputDir = newDir
					_ = config.Save(m.cfg)
				}
				m.isSettingDir = false
				m.dirInput.Blur()
				m.textInput.Focus()
				return m, textinput.Blink

			case "esc":
				m.isSettingDir = false
				m.dirInput.Blur()
				m.textInput.Focus()
				return m, textinput.Blink
			}

			var cmd tea.Cmd
			m.dirInput, cmd = m.dirInput.Update(msg)
			return m, cmd
		}

		if msg.Type == tea.KeyTab || k == "tab" || k == "\t" {
			m.activeTab = (m.activeTab + 1) % 2
			if m.activeTab == 0 {
				m.textInput.Focus()
				return m, tea.Batch(textinput.Blink, tea.ClearScreen)
			}
			m.textInput.Blur()
			return m, tea.ClearScreen
		}

		if msg.Type == tea.KeyShiftTab || k == "shift+tab" || k == "backtab" {
			m.activeTab = (m.activeTab + 1) % 2
			if m.activeTab == 0 {
				m.textInput.Focus()
				return m, tea.Batch(textinput.Blink, tea.ClearScreen)
			}
			m.textInput.Blur()
			return m, tea.ClearScreen
		}

		switch k {
		case "left":
			if !m.textInput.Focused() {
				m.activeTab = (m.activeTab + 1) % 2
				if m.activeTab == 0 {
					m.textInput.Focus()
					return m, tea.Batch(textinput.Blink, tea.ClearScreen)
				}
				m.textInput.Blur()
				return m, tea.ClearScreen
			}

		case "right":
			if !m.textInput.Focused() {
				m.activeTab = (m.activeTab + 1) % 2
				if m.activeTab == 0 {
					m.textInput.Focus()
					return m, tea.Batch(textinput.Blink, tea.ClearScreen)
				}
				m.textInput.Blur()
				return m, tea.ClearScreen
			}

		case "esc":
			if m.textInput.Focused() {
				m.textInput.Blur()
			}
			return m, nil

		case "i", "/":
			if !m.textInput.Focused() {
				m.activeTab = 0
				m.textInput.Focus()
				return m, tea.Batch(textinput.Blink, tea.ClearScreen)
			}

		case "?", "h", "H":
			if !m.textInput.Focused() {
				m.showHelp = true
				return m, tea.ClearScreen
			}

		case "s", "S":
			if !m.textInput.Focused() {
				m.toggleSourceMode()
				return m, nil
			}

		case "1":
			if !m.textInput.Focused() {
				m.setSourceMode("spotify")
				return m, nil
			}

		case "2":
			if !m.textInput.Focused() {
				m.setSourceMode("youtube")
				return m, nil
			}

		case "a", "A":
			if !m.textInput.Focused() {
				m.toggleVisualFX()
				return m, nil
			}

		case "d", "D":
			if !m.textInput.Focused() {
				m.isSettingDir = true
				m.textInput.Blur()
				m.dirInput.SetValue(m.cfg.OutputDir)
				m.dirInput.Focus()
				return m, textinput.Blink
			}

		case "q":
			if !m.textInput.Focused() {
				return m, tea.Quit
			}

		case "o":
			if !m.textInput.Focused() {
				m.openOutputDir()
				return m, nil
			}

		case "enter":
			if m.textInput.Focused() {
				val := strings.TrimSpace(m.textInput.Value())
				if val != "" {

					if val == "/sp" || val == "/spotify" {
						m.setSourceMode("spotify")
						m.textInput.SetValue("")
						return m, nil
					}
					if val == "/yt" || val == "/youtube" {
						m.setSourceMode("youtube")
						m.textInput.SetValue("")
						return m, nil
					}

					if !m.isResolving && !m.isDownloading {
						m.isResolving = true
						return m, tea.Batch(m.resolveInput(val), m.spinner.Tick)
					}
				}
			} else {
				m.activeTab = 0
				m.textInput.Focus()
				return m, textinput.Blink
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = max(30, min(90, msg.Width-12))
		m.dirInput.Width = max(30, min(90, msg.Width-12))
		m.progress.Width = max(25, min(60, msg.Width-25))
		return m, tea.ClearScreen

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case batchResolvedMsg:
		m.isResolving = false
		if msg.err != nil {
			return m, nil
		}
		m.currentBatch = msg.batch
		m.isDownloading = true
		go m.startBatchDownload(msg.batch)

	case trackProgressMsg:
		cmds = append(cmds, m.listenProgressUpdates())
		if msg.prog != nil && m.currentBatch != nil {
			for i, p := range m.currentBatch.Progress {
				if p != nil && p.Track != nil && msg.prog.Track != nil {
					if p.Track.ID == msg.prog.Track.ID || (p.Track.Title == msg.prog.Track.Title && p.Track.Artist == msg.prog.Track.Artist) {
						m.currentBatch.Progress[i] = msg.prog
						break
					}
				}
			}
		}

	case trackCompletedMsg:
		cmds = append(cmds, m.listenProgressUpdates())
		if msg.prog != nil && m.currentBatch != nil {
			for i, p := range m.currentBatch.Progress {
				if p != nil && p.Track != nil && msg.prog.Track != nil {
					if p.Track.ID == msg.prog.Track.ID || (p.Track.Title == msg.prog.Track.Title && p.Track.Artist == msg.prog.Track.Artist) {
						if msg.err == nil {
							msg.prog.Status = models.StatusComplete
							msg.prog.ProgressPercent = 100.0
						} else {
							msg.prog.Status = models.StatusFailed
							msg.prog.ErrorMessage = msg.err.Error()
						}
						m.currentBatch.Progress[i] = msg.prog
						break
					}
				}
			}
		}
		if msg.err == nil && msg.prog != nil {
			m.library = append(m.library, models.DownloadedFile{
				FilePath:     msg.prog.FilePath,
				Title:        msg.prog.Track.Title,
				Artist:       msg.prog.Track.Artist,
				Album:        msg.prog.Track.Album,
				FileSize:     msg.prog.DownloadedBytes,
				QualityLabel: msg.prog.QualityLabel,
				Platform:     msg.prog.Track.Platform,
				CompletedAt:  time.Now(),
			})
		}

	case batchCompletedMsg:
		cmds = append(cmds, m.listenProgressUpdates())
		m.isDownloading = false
		m.textInput.SetValue("")
	}

	if m.textInput.Focused() {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// extendedEngineKey decodes the CSI-u / modifyOtherKeys sequences emitted by
// terminals for Ctrl+1 and Ctrl+2, which would otherwise be dropped by the
// input parser or arrive as a plain digit.
func extendedEngineKey(msg tea.Msg) string {
	s, ok := msg.(fmt.Stringer)
	if !ok {
		return ""
	}

	text := s.String()
	if !strings.HasPrefix(text, "?CSI[") || !strings.HasSuffix(text, "]?") {
		return ""
	}

	var seq strings.Builder
	for _, field := range strings.Fields(text[len("?CSI[") : len(text)-len("]?")]) {
		var b int
		if _, err := fmt.Sscanf(field, "%d", &b); err != nil || b < 0 || b > 255 {
			return ""
		}
		seq.WriteByte(byte(b))
	}

	switch seq.String() {
	case "49;5u", "27;5;49~":
		return "spotify"
	case "50;5u", "27;5;50~":
		return "youtube"
	default:
		return ""
	}
}

func (m *Model) toggleSourceMode() {
	if m.cfg.SourceMode == "youtube" {
		m.setSourceMode("spotify")
	} else {
		m.setSourceMode("youtube")
	}
}

func (m *Model) toggleVisualFX() {
	m.visualFX = !m.visualFX
}

func (m *Model) setSourceMode(mode string) {
	m.cfg.SourceMode = mode
	accent := accentFor(mode)
	m.textInput.PromptStyle = lipgloss.NewStyle().Foreground(accent)
	m.textInput.Cursor.Style = lipgloss.NewStyle().Foreground(BgBase).Background(accent)
	m.dirInput.PromptStyle = lipgloss.NewStyle().Foreground(accent)
	m.dirInput.Cursor.Style = lipgloss.NewStyle().Foreground(BgBase).Background(accent)
	_ = config.Save(m.cfg)
}

func (m *Model) resolveInput(raw string) tea.Cmd {
	return func() tea.Msg {
		raw = strings.TrimSpace(raw)
		mode := m.cfg.SourceMode

		if mode == "youtube" {
			if spotify.IsSpotifyURL(raw) {
				link, err := spotify.ParseLink(raw)
				if err == nil {
					spBatch, err := m.spotifyClient.FetchBatch(link)
					if err == nil && spBatch != nil {
						spBatch.Platform = models.PlatformYouTubeMusic
						for _, t := range spBatch.Tracks {
							t.Platform = models.PlatformYouTubeMusic
							t.SourceURL = fmt.Sprintf("ytsearch1:%s - %s", t.Artist, t.Title)
						}
						return batchResolvedMsg{batch: spBatch, err: nil}
					}
				}
			}
			batch, err := m.ytClient.ResolveInput(raw)
			return batchResolvedMsg{batch: batch, err: err}
		}

		if spotify.IsSpotifyURL(raw) {
			link, err := spotify.ParseLink(raw)
			if err != nil {
				return batchResolvedMsg{err: err}
			}
			batch, err := m.spotifyClient.FetchBatch(link)
			return batchResolvedMsg{batch: batch, err: err}
		}

		batch, err := m.ytClient.ResolveInput(raw)
		if err == nil && batch != nil {
			batch.Platform = models.PlatformSpotify
			for _, t := range batch.Tracks {
				t.Platform = models.PlatformSpotify
			}
		}
		return batchResolvedMsg{batch: batch, err: err}
	}
}

func (m *Model) startBatchDownload(batch *models.DownloadBatch) {
	for _, track := range batch.Tracks {
		prog, err := m.downloader.DownloadTrack(track, batch, func(p *models.TrackProgress) {
			m.subChan <- trackProgressMsg{prog: p}
		})

		m.subChan <- trackCompletedMsg{prog: prog, err: err}
	}
	m.subChan <- batchCompletedMsg{}
}

func (m *Model) openOutputDir() {
	dir := config.ExpandPath(m.cfg.OutputDir)
	_ = os.MkdirAll(dir, 0755)
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("explorer", dir).Start()
	case "darwin":
		_ = exec.Command("open", dir).Start()
	default:
		_ = exec.Command("xdg-open", dir).Start()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
