package models

import "time"

type SourcePlatform string

const (
	PlatformSpotify      SourcePlatform = "Spotify"
	PlatformYouTubeMusic SourcePlatform = "YouTube Music"
	PlatformYouTube      SourcePlatform = "YouTube"
	PlatformDirectSearch SourcePlatform = "Search Query"
)

type Track struct {
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	Artist          string         `json:"artist"`
	Album           string         `json:"album"`
	DurationMs      int            `json:"duration_ms"`
	CoverArtURL     string         `json:"cover_art_url"`
	TrackNumber     int            `json:"track_number"`
	TotalTracks     int            `json:"total_tracks"`
	Year            string         `json:"year"`
	ISRC            string         `json:"isrc"`
	Platform        SourcePlatform `json:"platform"`
	SourceURL       string         `json:"source_url"`
	SpotifyURL      string         `json:"spotify_url,omitempty"`
	YouTubeURL      string         `json:"youtube_url,omitempty"`
	YouTubeMusicURL string         `json:"youtube_music_url,omitempty"`
	AudioQuality    string         `json:"audio_quality"`
}

type DownloadBatch struct {
	Type        string           `json:"type"`
	Name        string           `json:"name"`
	Subtitle    string           `json:"subtitle"`
	CoverArtURL string           `json:"cover_art_url"`
	Platform    SourcePlatform   `json:"platform"`
	Tracks      []*Track         `json:"tracks"`
	Progress    []*TrackProgress `json:"progress"`
}

type DownloadStatus string

const (
	StatusPending     DownloadStatus = "PENDING"
	StatusResolving   DownloadStatus = "RESOLVING"
	StatusDownloading DownloadStatus = "DOWNLOADING"
	StatusTagging     DownloadStatus = "TAGGING"
	StatusComplete    DownloadStatus = "COMPLETE"
	StatusFailed      DownloadStatus = "FAILED"
	StatusSkipped     DownloadStatus = "SKIPPED"
)

type TrackProgress struct {
	Track            *Track         `json:"track"`
	Status           DownloadStatus `json:"status"`
	DownloadedBytes  int64          `json:"downloaded_bytes"`
	TotalBytes       int64          `json:"total_bytes"`
	SpeedBytesPerSec int64          `json:"speed_bytes_per_sec"`
	ProgressPercent  float64        `json:"progress_percent"`
	FilePath         string         `json:"file_path"`
	QualityLabel     string         `json:"quality_label"`
	SourceUsed       string         `json:"source_used"`
	ErrorMessage     string         `json:"error_message"`
	StartedAt        time.Time      `json:"started_at"`
	CompletedAt      time.Time      `json:"completed_at"`
}

type DownloadedFile struct {
	FilePath     string         `json:"file_path"`
	Title        string         `json:"title"`
	Artist       string         `json:"artist"`
	Album        string         `json:"album"`
	FileSize     int64          `json:"file_size"`
	QualityLabel string         `json:"quality_label"`
	Platform     SourcePlatform `json:"platform"`
	CompletedAt  time.Time      `json:"completed_at"`
}

type ResolvedSources struct {
	ISRC         string `json:"isrc"`
	QobuzID      string `json:"qobuz_id"`
	QobuzURL     string `json:"qobuz_url"`
	TidalID      string `json:"tidal_id"`
	TidalURL     string `json:"tidal_url"`
	DeezerID     string `json:"deezer_id"`
	DeezerURL    string `json:"deezer_url"`
	AmazonASIN   string `json:"amazon_asin"`
	AmazonURL    string `json:"amazon_url"`
	AppleURL     string `json:"apple_url"`
	YouTubeURL   string `json:"youtube_url"`
	YouTubeMusic string `json:"youtube_music"`
	SoundCloud   string `json:"sound_cloud"`
	CoverArtURL  string `json:"cover_art_url"`
}

type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}
