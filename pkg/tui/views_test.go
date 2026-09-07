package tui

import (
	"fmt"
	"testing"

	"everyflac-tui/pkg/config"
	"everyflac-tui/pkg/models"
)

func TestQueueViewRender(t *testing.T) {
	cfg := &config.AppConfig{
		OutputDir:  "/home/pxradise/Music",
		SourceMode: "spotify",
	}
	m := NewModel(cfg)

	batch := &models.DownloadBatch{
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
				ProgressPercent: 100.0,
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
	m.currentBatch = batch

	view := m.View()
	fmt.Println("--- Rendered View with Active Batch ---")
	fmt.Println(view)
	fmt.Println("-----------------------------------------------")
}
