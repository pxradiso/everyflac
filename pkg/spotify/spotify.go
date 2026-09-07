package spotify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"everyflac-tui/pkg/models"
)

var (
	spotifyURLRegex = regexp.MustCompile(`https?://open\.spotify\.com/(?:intl-[a-zA-Z-]+/)?(track|album|playlist)/([a-zA-Z0-9]+)`)
	spotifyURIRegex = regexp.MustCompile(`spotify:(track|album|playlist):([a-zA-Z0-9]+)`)
	nextDataRegex   = regexp.MustCompile(`<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)
)

type ParsedLink struct {
	Type string
	ID   string
	URL  string
}

func IsSpotifyURL(raw string) bool {
	return spotifyURLRegex.MatchString(raw) || spotifyURIRegex.MatchString(raw)
}

func ParseLink(raw string) (*ParsedLink, error) {
	raw = strings.TrimSpace(raw)

	if match := spotifyURLRegex.FindStringSubmatch(raw); len(match) == 3 {
		return &ParsedLink{
			Type: match[1],
			ID:   match[2],
			URL:  fmt.Sprintf("https://open.spotify.com/%s/%s", match[1], match[2]),
		}, nil
	}

	if match := spotifyURIRegex.FindStringSubmatch(raw); len(match) == 3 {
		return &ParsedLink{
			Type: match[1],
			ID:   match[2],
			URL:  fmt.Sprintf("https://open.spotify.com/%s/%s", match[1], match[2]),
		}, nil
	}

	return nil, fmt.Errorf("invalid Spotify link format: %s", raw)
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

type nextDataPayload struct {
	Props struct {
		PageProps struct {
			State struct {
				Data struct {
					Entity struct {
						Type           string `json:"type"`
						Name           string `json:"name"`
						Title          string `json:"title"`
						Duration       int    `json:"duration"`
						RelatedEntity  string `json:"relatedEntityUri"`
						VisualIdentity struct {
							Image []struct {
								URL       string `json:"url"`
								MaxHeight int    `json:"maxHeight"`
								MaxWidth  int    `json:"maxWidth"`
							} `json:"image"`
						} `json:"visualIdentity"`
						CoverArt struct {
							Sources []struct {
								URL    string `json:"url"`
								Width  int    `json:"width"`
								Height int    `json:"height"`
							} `json:"sources"`
						} `json:"coverArt"`
						ReleaseDate *struct {
							IsoString string `json:"isoString"`
						} `json:"releaseDate"`
						Artists []struct {
							Name string `json:"name"`
							URI  string `json:"uri"`
						} `json:"artists"`
						TrackList []struct {
							URI        string `json:"uri"`
							UID        string `json:"uid"`
							Title      string `json:"title"`
							Subtitle   string `json:"subtitle"`
							Duration   int    `json:"duration"`
							IsPlayable bool   `json:"isPlayable"`
							Artists    []struct {
								Name string `json:"name"`
								URI  string `json:"uri"`
							} `json:"artists"`
						} `json:"trackList"`
					} `json:"entity"`
				} `json:"data"`
			} `json:"state"`
		} `json:"pageProps"`
	} `json:"props"`
}

func (c *Client) FetchBatch(link *ParsedLink) (*models.DownloadBatch, error) {
	embedURL := fmt.Sprintf("https://open.spotify.com/embed/%s/%s", link.Type, link.ID)

	req, err := http.NewRequest("GET", embedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Spotify embed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify embed returned HTTP %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Spotify response: %w", err)
	}

	body := string(bodyBytes)
	matches := nextDataRegex.FindStringSubmatch(body)
	if len(matches) < 2 {
		return nil, fmt.Errorf("unable to locate Spotify metadata in page")
	}

	var payload nextDataPayload
	if err := json.Unmarshal([]byte(matches[1]), &payload); err != nil {
		return nil, fmt.Errorf("failed to decode Spotify payload: %w", err)
	}

	entity := payload.Props.PageProps.State.Data.Entity
	title := entity.Name
	if title == "" {
		title = entity.Title
	}

	subtitle := ""
	if len(entity.Artists) > 0 {
		names := make([]string, 0, len(entity.Artists))
		for _, a := range entity.Artists {
			names = append(names, a.Name)
		}
		subtitle = strings.Join(names, ", ")
	}

	coverURL := ""
	bestW := 0
	for _, img := range entity.VisualIdentity.Image {
		if img.URL != "" && img.MaxWidth >= bestW {
			bestW = img.MaxWidth
			coverURL = img.URL
		}
	}
	if coverURL == "" && len(entity.CoverArt.Sources) > 0 {
		coverURL = entity.CoverArt.Sources[len(entity.CoverArt.Sources)-1].URL
	}
	if coverURL == "" {
		coverURL = c.fetchOembedCover(link.URL)
	}
	if strings.HasPrefix(coverURL, "http:") {
		coverURL = "https:" + coverURL[5:]
	}

	releaseYear := ""
	if entity.ReleaseDate != nil && len(entity.ReleaseDate.IsoString) >= 4 {
		releaseYear = entity.ReleaseDate.IsoString[:4]
	}

	batch := &models.DownloadBatch{
		Type:        link.Type,
		Name:        title,
		Subtitle:    subtitle,
		CoverArtURL: coverURL,
		Platform:    models.PlatformSpotify,
		Tracks:      make([]*models.Track, 0),
		Progress:    make([]*models.TrackProgress, 0),
	}

	if link.Type == "track" {
		trackID := link.ID
		track := &models.Track{
			ID:          trackID,
			Title:       title,
			Artist:      subtitle,
			Album:       title,
			DurationMs:  entity.Duration,
			CoverArtURL: coverURL,
			TrackNumber: 1,
			TotalTracks: 1,
			Year:        releaseYear,
			Platform:    models.PlatformSpotify,
			SpotifyURL:  fmt.Sprintf("https://open.spotify.com/track/%s", trackID),
			SourceURL:   fmt.Sprintf("https://open.spotify.com/track/%s", trackID),
		}
		batch.Tracks = append(batch.Tracks, track)
		batch.Progress = append(batch.Progress, &models.TrackProgress{
			Track:  track,
			Status: models.StatusPending,
		})
		return batch, nil
	}

	for idx, item := range entity.TrackList {
		tID := extractTrackID(item.URI)
		tArtist := item.Subtitle
		if len(item.Artists) > 0 {
			aList := make([]string, 0, len(item.Artists))
			for _, a := range item.Artists {
				aList = append(aList, a.Name)
			}
			tArtist = strings.Join(aList, ", ")
		}

		albumName := title
		if link.Type == "playlist" {
			albumName = "Spotify Playlist: " + title
		}

		track := &models.Track{
			ID:          tID,
			Title:       item.Title,
			Artist:      tArtist,
			Album:       albumName,
			DurationMs:  item.Duration,
			CoverArtURL: coverURL,
			TrackNumber: idx + 1,
			TotalTracks: len(entity.TrackList),
			Year:        releaseYear,
			Platform:    models.PlatformSpotify,
			SpotifyURL:  fmt.Sprintf("https://open.spotify.com/track/%s", tID),
			SourceURL:   fmt.Sprintf("https://open.spotify.com/track/%s", tID),
		}

		batch.Tracks = append(batch.Tracks, track)
		batch.Progress = append(batch.Progress, &models.TrackProgress{
			Track:  track,
			Status: models.StatusPending,
		})
	}

	return batch, nil
}

func extractTrackID(uri string) string {
	parts := strings.Split(uri, ":")
	if len(parts) >= 3 && parts[1] == "track" {
		return parts[2]
	}
	return uri
}

func (c *Client) fetchOembedCover(spotifyURL string) string {
	oembedURL := fmt.Sprintf("https://open.spotify.com/oembed?url=%s", url.QueryEscape(spotifyURL))
	req, err := http.NewRequest("GET", oembedURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := c.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()

	var data struct {
		ThumbnailURL string `json:"thumbnail_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
		return data.ThumbnailURL
	}
	return ""
}
