package resolver

import (
	"bytes"
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
	idRegex   = regexp.MustCompile(`/([0-9]+)(?:/|\?|$)`)
	asinRegex = regexp.MustCompile(`/(B[0-9A-Z]{9})(?:/|\?|$)`)
)

type Resolver struct {
	httpClient *http.Client
}

func NewResolver() *Resolver {
	return &Resolver{
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

type zarzResolveResponse struct {
	Success  bool   `json:"success"`
	ISRC     string `json:"isrc"`
	SongURLs struct {
		Spotify      string `json:"Spotify"`
		Deezer       string `json:"Deezer"`
		Tidal        string `json:"Tidal"`
		AppleMusic   string `json:"AppleMusic"`
		AmazonMusic  string `json:"AmazonMusic"`
		Qobuz        string `json:"Qobuz"`
		YouTube      string `json:"YouTube"`
		YouTubeMusic string `json:"YouTubeMusic"`
		SoundCloud   string `json:"SoundCloud"`
	} `json:"songUrls"`
}

type deezerSearchResponse struct {
	Data []struct {
		ID       int64  `json:"id"`
		Title    string `json:"title"`
		Link     string `json:"link"`
		Duration int    `json:"duration"`
		Artist   struct {
			Name string `json:"name"`
		} `json:"artist"`
		Album struct {
			Title string `json:"title"`
			Cover string `json:"cover_xl"`
		} `json:"album"`
	} `json:"data"`
}

type deezerTrackDetail struct {
	ID   int64  `json:"id"`
	ISRC string `json:"isrc"`
}

func (r *Resolver) Resolve(track *models.Track) (*models.ResolvedSources, error) {
	sources := &models.ResolvedSources{}

	if track.SpotifyURL != "" {
		if err := r.resolveZarz(track.SpotifyURL, sources); err == nil && (sources.ISRC != "" || sources.QobuzID != "" || sources.TidalID != "") {
			if track.ISRC == "" && sources.ISRC != "" {
				track.ISRC = sources.ISRC
			}
			return sources, nil
		}
	}

	if sources.ISRC == "" || sources.DeezerID == "" {
		_ = r.resolveDeezer(track, sources)
	}

	if sources.ISRC != "" && sources.QobuzID == "" {
		track.ISRC = sources.ISRC
	}

	return sources, nil
}

func (r *Resolver) resolveZarz(spotifyURL string, sources *models.ResolvedSources) error {
	payload, _ := json.Marshal(map[string]string{"url": spotifyURL})
	req, err := http.NewRequest("POST", "https://api.zarz.moe/v1/resolve", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("zarz resolver returned %d", resp.StatusCode)
	}

	var data zarzResolveResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	sources.ISRC = data.ISRC
	sources.QobuzURL = data.SongURLs.Qobuz
	sources.TidalURL = data.SongURLs.Tidal
	sources.DeezerURL = data.SongURLs.Deezer
	sources.AmazonURL = data.SongURLs.AmazonMusic
	sources.AppleURL = data.SongURLs.AppleMusic
	sources.YouTubeURL = data.SongURLs.YouTube
	sources.YouTubeMusic = data.SongURLs.YouTubeMusic
	sources.SoundCloud = data.SongURLs.SoundCloud

	if sources.QobuzURL != "" {
		sources.QobuzID = extractID(sources.QobuzURL)
	}
	if sources.TidalURL != "" {
		sources.TidalID = extractID(sources.TidalURL)
	}
	if sources.DeezerURL != "" {
		sources.DeezerID = extractID(sources.DeezerURL)
	}
	if sources.AmazonURL != "" {
		sources.AmazonASIN = extractASIN(sources.AmazonURL)
	}

	return nil
}

func (r *Resolver) resolveDeezer(track *models.Track, sources *models.ResolvedSources) error {
	query := fmt.Sprintf(`artist:"%s" track:"%s"`, cleanQuery(track.Artist), cleanQuery(track.Title))
	searchURL := fmt.Sprintf("https://api.deezer.com/search?q=%s&limit=5", url.QueryEscape(query))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result deezerSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if len(result.Data) == 0 {
		return nil
	}

	best := result.Data[0]
	sources.DeezerID = fmt.Sprintf("%d", best.ID)
	sources.DeezerURL = best.Link

	trackDetailURL := fmt.Sprintf("https://api.deezer.com/track/%d", best.ID)
	reqDetail, _ := http.NewRequest("GET", trackDetailURL, nil)
	reqDetail.Header.Set("User-Agent", "Mozilla/5.0")
	if respDetail, err := r.httpClient.Do(reqDetail); err == nil {
		defer respDetail.Body.Close()
		var detail deezerTrackDetail
		if json.NewDecoder(respDetail.Body).Decode(&detail) == nil && detail.ISRC != "" {
			sources.ISRC = detail.ISRC
		}
	}

	return nil
}

func cleanQuery(s string) string {
	s = strings.ReplaceAll(s, `"`, "")
	s = strings.ReplaceAll(s, `'`, "")
	return strings.TrimSpace(s)
}

func extractID(rawURL string) string {
	if match := idRegex.FindStringSubmatch(rawURL); len(match) > 1 {
		return match[1]
	}
	parts := strings.Split(strings.TrimRight(rawURL, "/"), "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if regexp.MustCompile(`^[0-9]+$`).MatchString(last) {
			return last
		}
	}
	return ""
}

func extractASIN(rawURL string) string {
	if match := asinRegex.FindStringSubmatch(rawURL); len(match) > 1 {
		return match[1]
	}
	if u, err := url.Parse(rawURL); err == nil {
		if asin := u.Query().Get("trackAsin"); asin != "" {
			return asin
		}
	}
	return ""
}

func (r *Resolver) FetchCoverArt(coverURL string) ([]byte, error) {
	if coverURL == "" {
		return nil, nil
	}
	req, err := http.NewRequest("GET", coverURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch cover art, status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
