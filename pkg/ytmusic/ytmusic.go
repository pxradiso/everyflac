package ytmusic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"everyflac-tui/pkg/models"
)

var (
	ytMusicTrackRegex    = regexp.MustCompile(`https?://music\.youtube\.com/watch\?v=([a-zA-Z0-9_-]+)`)
	ytMusicPlaylistRegex = regexp.MustCompile(`https?://music\.youtube\.com/playlist\?list=([a-zA-Z0-9_-]+)`)
	ytStandardTrackRegex = regexp.MustCompile(`https?://(?:www\.)?youtube\.com/watch\?v=([a-zA-Z0-9_-]+)`)
	ytShortRegex         = regexp.MustCompile(`https?://youtu\.be/([a-zA-Z0-9_-]+)`)
)

func IsYouTubeURL(raw string) bool {
	return ytMusicTrackRegex.MatchString(raw) ||
		ytMusicPlaylistRegex.MatchString(raw) ||
		ytStandardTrackRegex.MatchString(raw) ||
		ytShortRegex.MatchString(raw)
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

type ytdlpJSONEntry struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Track      string  `json:"track"`
	Album      string  `json:"album"`
	Creator    string  `json:"creator"`
	Uploader   string  `json:"uploader"`
	Channel    string  `json:"channel"`
	Duration   float64 `json:"duration"`
	Thumbnail  string  `json:"thumbnail"`
	Thumbnails []struct {
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"thumbnails"`
	WebpageURL    string           `json:"webpage_url"`
	Entries       []ytdlpJSONEntry `json:"entries"`
	PlaylistTitle string           `json:"playlist_title"`
}

func (c *Client) ResolveInput(input string) (*models.DownloadBatch, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input query")
	}

	target := input
	isSearch := false
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		isSearch = true
		target = fmt.Sprintf("ytsearch1:%s", input)
	}

	cmd := exec.Command(
		"yt-dlp",
		"--dump-single-json",
		"--flat-playlist",
		"--no-warnings",
		target,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve metadata via yt-dlp: %w", err)
	}

	var data ytdlpJSONEntry
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, fmt.Errorf("failed to parse yt-dlp JSON output: %w", err)
	}

	platform := models.PlatformYouTubeMusic
	if strings.Contains(input, "youtube.com/watch") || strings.Contains(input, "youtu.be/") {
		platform = models.PlatformYouTube
	} else if isSearch {
		platform = models.PlatformDirectSearch
	}

	if len(data.Entries) == 1 && (data.ID == "" || strings.HasPrefix(target, "ytsearch")) {
		entry := data.Entries[0]
		track := c.buildTrack(&entry, 1, 1, platform)
		batch := &models.DownloadBatch{
			Type:        "track",
			Name:        track.Title,
			Subtitle:    track.Artist,
			CoverArtURL: track.CoverArtURL,
			Platform:    platform,
			Tracks:      []*models.Track{track},
			Progress: []*models.TrackProgress{
				{Track: track, Status: models.StatusPending},
			},
		}
		return batch, nil
	}

	if len(data.Entries) > 1 {
		batchTitle := data.Title
		if batchTitle == "" {
			batchTitle = data.PlaylistTitle
		}
		if batchTitle == "" {
			batchTitle = "YouTube Music Collection"
		}

		batch := &models.DownloadBatch{
			Type:        "playlist",
			Name:        batchTitle,
			Subtitle:    data.Uploader,
			CoverArtURL: data.Thumbnail,
			Platform:    platform,
			Tracks:      make([]*models.Track, 0, len(data.Entries)),
			Progress:    make([]*models.TrackProgress, 0, len(data.Entries)),
		}

		for i, entry := range data.Entries {
			track := c.buildTrack(&entry, i+1, len(data.Entries), platform)
			if track.Album == "" || track.Album == "YouTube Music" {
				track.Album = batchTitle
			}
			batch.Tracks = append(batch.Tracks, track)
			batch.Progress = append(batch.Progress, &models.TrackProgress{
				Track:  track,
				Status: models.StatusPending,
			})
		}
		return batch, nil
	}

	track := c.buildTrack(&data, 1, 1, platform)
	batch := &models.DownloadBatch{
		Type:        "track",
		Name:        track.Title,
		Subtitle:    track.Artist,
		CoverArtURL: track.CoverArtURL,
		Platform:    platform,
		Tracks:      []*models.Track{track},
		Progress: []*models.TrackProgress{
			{Track: track, Status: models.StatusPending},
		},
	}
	return batch, nil
}

func (c *Client) buildTrack(e *ytdlpJSONEntry, trackNum, totalTracks int, platform models.SourcePlatform) *models.Track {
	title := e.Track
	if title == "" {
		title = e.Title
	}

	artist := e.Artist
	if artist == "" {
		artist = e.Creator
	}
	if artist == "" {
		artist = e.Channel
	}
	if artist == "" {
		artist = e.Uploader
	}
	artist = strings.TrimSuffix(artist, " - Topic")

	if artist == "" || artist == e.Uploader {
		if parts := strings.Split(title, " - "); len(parts) == 2 {
			artist = strings.TrimSpace(parts[0])
			title = strings.TrimSpace(parts[1])
		}
	}

	album := e.Album
	if album == "" {
		album = "Lossless Collection"
	}

	sourceURL := e.WebpageURL
	if sourceURL == "" && e.ID != "" {
		sourceURL = fmt.Sprintf("https://music.youtube.com/watch?v=%s", e.ID)
	}

	coverURL := getBestThumbnail(e.Thumbnail, e.Thumbnails, e.ID)

	return &models.Track{
		ID:          e.ID,
		Title:       title,
		Artist:      artist,
		Album:       album,
		DurationMs:  int(e.Duration * 1000),
		CoverArtURL: coverURL,
		TrackNumber: trackNum,
		TotalTracks: totalTracks,
		Platform:    platform,
		SourceURL:   sourceURL,
	}
}

func getBestThumbnail(topThumb string, thumbs []struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}, id string) string {
	best := ""
	maxWidth := 0
	for _, t := range thumbs {
		if t.Width >= maxWidth && t.URL != "" {
			maxWidth = t.Width
			best = t.URL
		}
	}
	if best != "" {
		return best
	}
	if topThumb != "" {
		return topThumb
	}
	if id != "" {
		return fmt.Sprintf("https://i.ytimg.com/vi/%s/maxresdefault.jpg", id)
	}
	return ""
}

func (c *Client) FetchCoverArt(coverURL string) ([]byte, error) {
	if coverURL == "" {
		return nil, nil
	}

	raw, err := c.downloadImage(coverURL)
	if (err != nil || len(raw) == 0) && strings.Contains(coverURL, "maxresdefault.jpg") {
		hqURL := strings.Replace(coverURL, "maxresdefault.jpg", "hqdefault.jpg", 1)
		raw, err = c.downloadImage(hqURL)
	}
	return raw, err
}

func (c *Client) downloadImage(imageURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cover art request returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
