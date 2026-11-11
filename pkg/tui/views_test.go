package tui

import (
	"strings"
	"testing"

	"everyflac-tui/pkg/config"
	"everyflac-tui/pkg/models"
	"github.com/charmbracelet/lipgloss"
)

func testModel(width, height int) Model {
	cfg := &config.AppConfig{OutputDir: "/home/pxradise/Music", SourceMode: "spotify"}
	m := NewModel(cfg)
	m.width = width
	m.height = height
	return m
}

func populatedModel(width, height int) Model {
	m := testModel(width, height)
	m.currentBatch = &models.DownloadBatch{
		Type:     "album",
		Platform: models.PlatformSpotify,
		Name:     "Random Access Memories",
		Subtitle: "Daft Punk",
		Tracks: []*models.Track{
			{TrackNumber: 1, Title: "Give Life Back to Music", Artist: "Daft Punk"},
			{TrackNumber: 2, Title: "The Game of Love", Artist: "Daft Punk"},
			{TrackNumber: 3, Title: "Giorgio by Moroder", Artist: "Daft Punk"},
		},
		Progress: []*models.TrackProgress{
			{
				Track:           &models.Track{TrackNumber: 1, Title: "Give Life Back to Music", Artist: "Daft Punk"},
				Status:          models.StatusComplete,
				ProgressPercent: 100,
				QualityLabel:    "24-bit / 48kHz FLAC",
			},
			{
				Track:            &models.Track{TrackNumber: 2, Title: "The Game of Love", Artist: "Daft Punk"},
				Status:           models.StatusDownloading,
				ProgressPercent:  54.2,
				SpeedBytesPerSec: 2500000,
			},
			{
				Track:           &models.Track{TrackNumber: 3, Title: "Giorgio by Moroder", Artist: "Daft Punk"},
				Status:          models.StatusPending,
				ProgressPercent: 0,
			},
		},
	}
	m.library = []models.DownloadedFile{
		{
			Title:        "Give Life Back to Music",
			Artist:       "Daft Punk",
			QualityLabel: "24-bit / 48kHz",
			FileSize:     42 * 1024 * 1024,
			FilePath:     "/home/pxradise/Music/Give Life Back to Music.flac",
		},
	}
	return m
}

func assertFits(t *testing.T, view string, width, height int) {
	t.Helper()

	lines := strings.Split(view, "\n")
	if len(lines) > height {
		t.Errorf("view has %d lines, want at most %d", len(lines), height)
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w > width {
			t.Errorf("line %d is %d cells wide, want at most %d", i+1, w, width)
		}
	}
}

func TestViewFitsTerminal(t *testing.T) {
	cases := []struct {
		name   string
		width  int
		height int
	}{
		{"wide", 120, 36},
		{"narrow", 72, 28},
		{"minimum", 50, 20},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := populatedModel(tc.width, tc.height)
			for tab := 0; tab < 2; tab++ {
				m.activeTab = tab
				assertFits(t, m.View(), tc.width, tc.height)
			}

			m.showHelp = true
			assertFits(t, m.View(), tc.width, tc.height)
		})
	}
}

func TestQueueViewContent(t *testing.T) {
	m := populatedModel(120, 36)
	view := m.View()

	for _, want := range []string{"queue · 3", "Random Access Memories", "Give Life Back to Music", "done", "54%", "queued"} {
		if !strings.Contains(view, want) {
			t.Errorf("queue view is missing %q", want)
		}
	}
}

func TestLibraryContent(t *testing.T) {
	m := populatedModel(120, 36)

	m.activeTab = 1
	library := m.View()
	for _, want := range []string{"library · 1", "Give Life Back to Music", "24-bit / 48kHz"} {
		if !strings.Contains(library, want) {
			t.Errorf("library view is missing %q", want)
		}
	}
}

func TestWindowsFallbacksFitWidth(t *testing.T) {
	old := windowsConsole
	windowsConsole = true
	defer func() { windowsConsole = old }()

	for _, width := range []int{1, 2, 3, 4, 6, 10} {
		got := truncateWidth("a very long track title", width)
		if lipgloss.Width(got) > width {
			t.Errorf("truncateWidth(%d) = %q, width %d exceeds %d", width, got, lipgloss.Width(got), width)
		}

		path := shortenPath("/home/user/Music/Some Album", width)
		if lipgloss.Width(path) > width {
			t.Errorf("shortenPath(%d) = %q, width %d exceeds %d", width, path, lipgloss.Width(path), width)
		}
	}

	if !strings.Contains(testModel(100, 34).View(), "| everyFLAC") {
		t.Error("windows header should use the ASCII brand bar")
	}
}

func TestUpdateNotice(t *testing.T) {
	m := testModel(120, 34)
	if strings.Contains(m.View(), "update v") {
		t.Fatal("update notice should not appear before a check succeeds")
	}

	m.updateAvailable = true
	m.latestVersion = "1.2.1"
	if !strings.Contains(m.View(), "update v1.2.1") {
		t.Error("header should show the available update")
	}

	m.showHelp = true
	if !strings.Contains(m.View(), "update v1.2.1 available") {
		t.Error("help panel should show the available update")
	}
	assertFits(t, m.View(), 120, 34)

	// Narrow terminals must not overflow when the notice is present.
	m.showHelp = false
	m.width = 50
	m.height = 20
	assertFits(t, m.View(), 50, 20)

	// A successful check with no newer release is shown as up to date.
	up := testModel(120, 34)
	up.updateChecked = true
	up.showHelp = true
	if !strings.Contains(up.View(), "up to date") {
		t.Error("help panel should show the up-to-date state after a successful check")
	}
}

func TestEngineKeyHints(t *testing.T) {
	if windowsConsole {
		t.Skip("Windows uses alt+1/2 and f1/f2")
	}

	if got := engineKeys(); got != "ctrl+1 / ctrl+2" {
		t.Fatalf("engineKeys() = %q", got)
	}

	m := testModel(120, 34)
	if !strings.Contains(m.View(), "ctrl+1 / ctrl+2") {
		t.Error("engine row should show the Ctrl shortcut on Linux")
	}

	m.showHelp = true
	if !strings.Contains(m.View(), "ctrl+1 / ctrl+2") {
		t.Error("help panel should show the Ctrl shortcut on Linux")
	}
}

func TestEmptyState(t *testing.T) {
	m := testModel(100, 34)
	if !strings.Contains(m.View(), "nothing queued") {
		t.Error("empty queue view should explain that nothing is queued")
	}

	m.showHelp = true
	if !strings.Contains(m.View(), "keyboard shortcuts") {
		t.Error("help view should list keyboard shortcuts")
	}
}
