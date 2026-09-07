package downloader

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"everyflac-tui/pkg/config"
	"everyflac-tui/pkg/models"
	"everyflac-tui/pkg/resolver"
	"everyflac-tui/pkg/tagger"
	"everyflac-tui/pkg/ytmusic"
)

var (
	percentRegex = regexp.MustCompile(`([0-9.]+)%`)
	speedRegex   = regexp.MustCompile(`at\s+([0-9.]+[a-zA-Z]+/s)`)
	sizeRegex    = regexp.MustCompile(`of\s+~?([0-9.]+[a-zA-Z]+)`)
)

type ProgressCallback func(p *models.TrackProgress)

type Downloader struct {
	cfg           *config.AppConfig
	resolver      *resolver.Resolver
	ytClient      *ytmusic.Client
	httpClient    *http.Client
	failedMirrors sync.Map
}

func deadMirrorsCachePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "everyflac_dead_mirrors.json")
	}
	dir := filepath.Join(configDir, "everyflac")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "dead_mirrors.json")
}

func NewDownloader(cfg *config.AppConfig) *Downloader {
	d := &Downloader{
		cfg:      cfg,
		resolver: resolver.NewResolver(),
		ytClient: ytmusic.NewClient(),
		httpClient: &http.Client{
			Timeout: 1200 * time.Millisecond,
		},
	}
	d.loadDeadMirrors()
	return d
}

func (d *Downloader) loadDeadMirrors() {
	if data, err := os.ReadFile(deadMirrorsCachePath()); err == nil {
		var cache map[string]int64
		if json.Unmarshal(data, &cache) == nil {
			now := time.Now().Unix()
			for u, exp := range cache {
				if exp > now {
					d.failedMirrors.Store(u, time.Unix(exp, 0))
				}
			}
		}
	}
}

func (d *Downloader) saveDeadMirrors() {
	cache := make(map[string]int64)
	d.failedMirrors.Range(func(k, v any) bool {
		if t, ok := v.(time.Time); ok && time.Now().Before(t) {
			cache[k.(string)] = t.Unix()
		}
		return true
	})
	if data, err := json.Marshal(cache); err == nil {
		_ = os.WriteFile(deadMirrorsCachePath(), data, 0644)
	}
}

func (d *Downloader) isMirrorDead(url string) bool {
	if exp, ok := d.failedMirrors.Load(url); ok {
		if time.Now().Before(exp.(time.Time)) {
			return true
		}
		d.failedMirrors.Delete(url)
	}
	return false
}

func (d *Downloader) markMirrorDead(url string) {
	d.failedMirrors.Store(url, time.Now().Add(24*time.Hour))
	d.saveDeadMirrors()
}

func (d *Downloader) DownloadTrack(
	track *models.Track,
	batch *models.DownloadBatch,
	onProgress ProgressCallback,
) (*models.TrackProgress, error) {
	var prog *models.TrackProgress
	if batch != nil {
		for _, bp := range batch.Progress {
			if bp != nil && bp.Track != nil && (bp.Track.ID == track.ID || (bp.Track.Title == track.Title && bp.Track.Artist == track.Artist)) {
				prog = bp
				break
			}
		}
	}
	if prog == nil {
		prog = &models.TrackProgress{
			Track: track,
		}
	}
	prog.Status = models.StatusResolving
	prog.ProgressPercent = 0.0
	prog.StartedAt = time.Now()
	if onProgress != nil {
		onProgress(prog)
	}

	tempDir := filepath.Join(os.TempDir(), "everyflac_work")
	_ = os.MkdirAll(tempDir, 0755)

	tempBase := filepath.Join(tempDir, fmt.Sprintf("ef_%d_%s", time.Now().UnixNano(), tagger.SanitizeFilename(track.ID)))
	tempFlac := tempBase + ".flac"
	defer func() {
		_ = os.Remove(tempFlac)
		for _, ext := range []string{".opus", ".webm", ".m4a", ".ogg", ".mp3", ".part", ".ytdl"} {
			_ = os.Remove(tempBase + ext)
		}
	}()

	var downloadedSource string
	var downloadErr error
	var coverBytes []byte

	coverChan := make(chan []byte, 1)
	fetchCoverAsync := func(url string, isSpotify bool) {
		if url == "" {
			return
		}
		go func() {
			var b []byte
			if isSpotify {
				b, _ = d.resolver.FetchCoverArt(url)
			} else {
				b, _ = d.ytClient.FetchCoverArt(url)
			}
			coverChan <- b
		}()
	}

	if track.CoverArtURL != "" {
		fetchCoverAsync(track.CoverArtURL, track.Platform == models.PlatformSpotify)
	}

	if track.Platform == models.PlatformSpotify {

		sources, err := d.resolver.Resolve(track)
		if err != nil || sources == nil {
			sources = &models.ResolvedSources{}
		}

		if track.CoverArtURL == "" && sources.CoverArtURL != "" {
			track.CoverArtURL = sources.CoverArtURL
			fetchCoverAsync(track.CoverArtURL, true)
		}

		if d.cfg.SourceMode != "youtube" {

			if sources.TidalID != "" {
				for _, mirror := range d.cfg.TidalMirrors {
					if d.isMirrorDead(mirror) {
						continue
					}
					prog.SourceUsed = "Tidal HiFi"
					prog.Status = models.StatusDownloading
					if onProgress != nil {
						onProgress(prog)
					}

					tidalURL := fmt.Sprintf("%s/api/track/%s", mirror, sources.TidalID)
					err := d.streamToFile(tidalURL, nil, tempFlac, prog, onProgress)
					if err == nil && tagger.IsFLAC(tempFlac) {
						downloadedSource = "Tidal HiFi"
						break
					}
					_ = os.Remove(tempFlac)
					d.markMirrorDead(mirror)
				}
			}

			if downloadedSource == "" && sources.DeezerID != "" && !d.isMirrorDead("https://deezer.anandserver.cfd") {
				prog.SourceUsed = "Deezer"
				prog.Status = models.StatusDownloading
				if onProgress != nil {
					onProgress(prog)
				}

				deezerURL := fmt.Sprintf("https://deezer.anandserver.cfd/api/stream/%s", sources.DeezerID)
				headers := map[string]string{"X-API-Key": "ak_8e3f1a7c2b5d9e4f0a6c3b8d1e5f2a9c7b4d0e6f"}
				err := d.streamToFile(deezerURL, headers, tempFlac, prog, onProgress)
				if err == nil && tagger.IsFLAC(tempFlac) {
					downloadedSource = "Deezer FLAC"
				} else {
					_ = os.Remove(tempFlac)
					d.markMirrorDead("https://deezer.anandserver.cfd")
				}
			}
		}

		if downloadedSource == "" {
			prog.SourceUsed = "everyFLAC Lossless Engine"
			prog.Status = models.StatusDownloading
			if onProgress != nil {
				onProgress(prog)
			}

			err := d.downloadSpotifyLosslessPipeline(track, sources, tempBase, prog, onProgress)
			if err == nil && tagger.IsFLAC(tempFlac) {
				downloadedSource = "everyFLAC Engine"
			} else {
				downloadErr = err
			}
		}

	} else {

		prog.SourceUsed = string(track.Platform)
		prog.Status = models.StatusDownloading
		if onProgress != nil {
			onProgress(prog)
		}

		err := d.downloadDirectLosslessPipeline(track, tempBase, prog, onProgress)
		if err == nil && tagger.IsFLAC(tempFlac) {
			downloadedSource = string(track.Platform)
		} else {
			downloadErr = err
		}
	}

	select {
	case b := <-coverChan:
		coverBytes = b
	case <-time.After(1 * time.Second):

	}

	if downloadedSource == "" {
		prog.Status = models.StatusFailed
		if downloadErr != nil {
			prog.ErrorMessage = downloadErr.Error()
		} else {
			prog.ErrorMessage = "Audio stream could not be retrieved"
		}
		if onProgress != nil {
			onProgress(prog)
		}
		return prog, errors.New(prog.ErrorMessage)
	}

	prog.Status = models.StatusTagging
	prog.ProgressPercent = 99.0
	if onProgress != nil {
		onProgress(prog)
	}

	details, tagErr := tagger.TagFLACFile(tempFlac, track, coverBytes)
	if tagErr != nil {

		prog.ErrorMessage = fmt.Sprintf("tagging warning: %v", tagErr)
	}
	if details != nil {
		prog.QualityLabel = details.QualityLabel
	} else {
		prog.QualityLabel = "FLAC 24-bit / 48.0 kHz"
	}

	finalDir := config.ExpandPath(d.cfg.OutputDir)
	if d.cfg.CreateArtistFolder && track.Artist != "" {
		finalDir = filepath.Join(finalDir, tagger.SanitizeFilename(track.Artist))
	}
	if d.cfg.CreateAlbumFolder && track.Album != "" {
		finalDir = filepath.Join(finalDir, tagger.SanitizeFilename(track.Album))
	}
	if err := os.MkdirAll(finalDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination folder: %w", err)
	}

	var filename string
	if d.cfg.CreateArtistFolder || d.cfg.CreateAlbumFolder {
		if track.TrackNumber > 0 {
			filename = fmt.Sprintf("%02d. %s - %s.flac", track.TrackNumber, track.Artist, track.Title)
		} else {
			filename = fmt.Sprintf("%s - %s.flac", track.Artist, track.Title)
		}
	} else {

		if track.Artist != "" && track.Title != "" {
			filename = fmt.Sprintf("%s - %s.flac", track.Artist, track.Title)
		} else if track.Title != "" {
			filename = fmt.Sprintf("%s.flac", track.Title)
		} else {
			filename = "track.flac"
		}
	}
	filename = tagger.SanitizeFilename(filename)
	if !strings.HasSuffix(filename, ".flac") {
		filename += ".flac"
	}

	destPath := filepath.Join(finalDir, filename)
	_ = os.Remove(destPath)

	if err := copyFile(tempFlac, destPath); err != nil {
		return nil, fmt.Errorf("failed to save final FLAC file: %w", err)
	}

	prog.Status = models.StatusComplete
	prog.ProgressPercent = 100.0
	prog.FilePath = destPath
	prog.CompletedAt = time.Now()
	if onProgress != nil {
		onProgress(prog)
	}

	return prog, nil
}

func (d *Downloader) downloadSpotifyLosslessPipeline(
	track *models.Track,
	sources *models.ResolvedSources,
	tempBase string,
	prog *models.TrackProgress,
	onProgress ProgressCallback,
) error {
	ytdlpPath := findExecutable("yt-dlp", []string{
		"/home/pxradise/.local/bin/yt-dlp",
		"/usr/local/bin/yt-dlp",
		"/usr/bin/yt-dlp",
	})
	if ytdlpPath == "" {
		return errors.New("yt-dlp binary not found on system")
	}

	ffmpegPath := findExecutable("ffmpeg", []string{
		"/usr/bin/ffmpeg",
		"/usr/local/bin/ffmpeg",
	})
	if ffmpegPath == "" {
		return errors.New("ffmpeg binary not found on system")
	}

	var candidates []string
	if sources.YouTubeMusic != "" {
		candidates = append(candidates, sources.YouTubeMusic)
	}
	if sources.YouTubeURL != "" && sources.YouTubeURL != sources.YouTubeMusic {
		candidates = append(candidates, sources.YouTubeURL)
	}
	if track.ISRC != "" {
		candidates = append(candidates, fmt.Sprintf("ytsearch1:%s", track.ISRC))
	}
	if track.Artist != "" && track.Title != "" {
		candidates = append(candidates, fmt.Sprintf("ytsearch1:%s - %s", track.Artist, track.Title))
		candidates = append(candidates, fmt.Sprintf("ytsearch1:%s %s", track.Artist, track.Title))
	} else if track.Title != "" {
		candidates = append(candidates, fmt.Sprintf("ytsearch1:%s", track.Title))
	}

	outputPattern := tempBase + ".%(ext)s"
	expectedFlac := tempBase + ".flac"

	cleanupTemp := func() {
		_ = os.Remove(expectedFlac)
		for _, ext := range []string{".opus", ".webm", ".m4a", ".ogg", ".mp3", ".part", ".ytdl"} {
			_ = os.Remove(tempBase + ext)
		}
	}

	var lastErr error
	for _, target := range candidates {
		cleanupTemp()

		cmd := exec.Command(
			ytdlpPath,
			"-f", "ba/bestaudio",
			"--default-search", "ytsearch",
			"--extract-audio",
			"--audio-format", "flac",
			"--audio-quality", "0",
			"--no-playlist",
			"--concurrent-fragments", "5",
			"--socket-timeout", "5",
			"--no-cache-dir",
			"--newline",
			"--ffmpeg-location", filepath.Dir(ffmpegPath),
			"-o", outputPattern,
			target,
		)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			lastErr = err
			continue
		}
		cmd.Stderr = cmd.Stdout

		if err := cmd.Start(); err != nil {
			lastErr = err
			continue
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			d.parseYtDlpOutput(line, prog, onProgress)
		}

		_ = cmd.Wait()

		if _, err := os.Stat(expectedFlac); err == nil && tagger.IsFLAC(expectedFlac) {
			return nil
		}

		for _, ext := range []string{".opus", ".webm", ".m4a", ".ogg"} {
			cand := tempBase + ext
			if _, err := os.Stat(cand); err == nil {
				transCmd := exec.Command(ffmpegPath, "-y", "-i", cand, "-c:a", "flac", expectedFlac)
				if transCmd.Run() == nil && tagger.IsFLAC(expectedFlac) {
					_ = os.Remove(cand)
					return nil
				}
			}
		}
	}

	cleanupTemp()
	if lastErr != nil {
		return fmt.Errorf("audio conversion failed: %w", lastErr)
	}
	return errors.New("pipeline finished but no valid FLAC audio file was produced")
}

func (d *Downloader) downloadDirectLosslessPipeline(
	track *models.Track,
	tempBase string,
	prog *models.TrackProgress,
	onProgress ProgressCallback,
) error {
	ytdlpPath := findExecutable("yt-dlp", []string{
		"/home/pxradise/.local/bin/yt-dlp",
		"/usr/local/bin/yt-dlp",
		"/usr/bin/yt-dlp",
	})
	if ytdlpPath == "" {
		return errors.New("yt-dlp binary not found on system")
	}

	ffmpegPath := findExecutable("ffmpeg", []string{
		"/usr/bin/ffmpeg",
		"/usr/local/bin/ffmpeg",
	})
	if ffmpegPath == "" {
		return errors.New("ffmpeg binary not found on system")
	}

	target := track.SourceURL
	if target == "" {
		target = fmt.Sprintf("ytsearch1:%s - %s", track.Artist, track.Title)
	}

	outputPattern := tempBase + ".%(ext)s"
	expectedFlac := tempBase + ".flac"

	cmd := exec.Command(
		ytdlpPath,
		"-f", "ba/bestaudio",
		"--default-search", "ytsearch",
		"--extract-audio",
		"--audio-format", "flac",
		"--audio-quality", "0",
		"--no-playlist",
		"--concurrent-fragments", "5",
		"--socket-timeout", "5",
		"--no-cache-dir",
		"--newline",
		"--ffmpeg-location", filepath.Dir(ffmpegPath),
		"-o", outputPattern,
		target,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to open stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start audio pipeline: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		d.parseYtDlpOutput(line, prog, onProgress)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("audio conversion failed: %w", err)
	}

	if _, err := os.Stat(expectedFlac); err == nil && tagger.IsFLAC(expectedFlac) {
		return nil
	}

	for _, ext := range []string{".opus", ".webm", ".m4a", ".ogg"} {
		cand := tempBase + ext
		if _, err := os.Stat(cand); err == nil {
			transCmd := exec.Command(ffmpegPath, "-y", "-i", cand, "-c:a", "flac", expectedFlac)
			if transCmd.Run() == nil && tagger.IsFLAC(expectedFlac) {
				_ = os.Remove(cand)
				return nil
			}
		}
	}

	return errors.New("lossless FLAC conversion could not be verified")
}

func (d *Downloader) parseYtDlpOutput(line string, prog *models.TrackProgress, onProgress ProgressCallback) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	if strings.Contains(line, "[download]") {
		prog.Status = models.StatusDownloading

		if match := percentRegex.FindStringSubmatch(line); len(match) > 1 {
			if p, err := strconv.ParseFloat(match[1], 64); err == nil {
				prog.ProgressPercent = p
			}
		}

		if match := speedRegex.FindStringSubmatch(line); len(match) > 1 {
			prog.SpeedBytesPerSec = parseSpeed(match[1])
		}

		if match := sizeRegex.FindStringSubmatch(line); len(match) > 1 {
			prog.TotalBytes = parseBytes(match[1])
			if prog.TotalBytes > 0 && prog.ProgressPercent > 0 {
				prog.DownloadedBytes = int64(float64(prog.TotalBytes) * (prog.ProgressPercent / 100.0))
			}
		}

		if onProgress != nil {
			onProgress(prog)
		}
	}
}

func (d *Downloader) streamToFile(
	rawURL string,
	headers map[string]string,
	destFile string,
	prog *models.TrackProgress,
	onProgress ProgressCallback,
) error {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	totalBytes := resp.ContentLength
	prog.TotalBytes = totalBytes

	out, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer out.Close()

	buf := make([]byte, 64*1024)
	var downloaded int64
	var bytesSinceLastUpdate int64
	lastUpdate := time.Now()

	for {
		n, rErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := out.Write(buf[:n]); wErr != nil {
				return wErr
			}
			downloaded += int64(n)
			bytesSinceLastUpdate += int64(n)

			now := time.Now()
			elapsed := now.Sub(lastUpdate)
			if elapsed >= 120*time.Millisecond {
				speed := int64(float64(bytesSinceLastUpdate) / elapsed.Seconds())
				prog.DownloadedBytes = downloaded
				prog.SpeedBytesPerSec = speed
				if totalBytes > 0 {
					prog.ProgressPercent = (float64(downloaded) / float64(totalBytes)) * 100.0
				}
				if onProgress != nil {
					onProgress(prog)
				}
				lastUpdate = now
				bytesSinceLastUpdate = 0
			}
		}
		if rErr != nil {
			if rErr == io.EOF {
				break
			}
			return rErr
		}
	}

	prog.DownloadedBytes = downloaded
	prog.ProgressPercent = 100.0
	if onProgress != nil {
		onProgress(prog)
	}

	return nil
}

func parseSpeed(s string) int64 {
	s = strings.TrimSuffix(s, "/s")
	s = strings.TrimSpace(s)
	return parseBytes(s)
}

func parseBytes(s string) int64 {
	s = strings.ToUpper(strings.TrimSpace(s))
	multi := float64(1)

	if strings.HasSuffix(s, "GIB") || strings.HasSuffix(s, "GB") {
		multi = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GIB")
		s = strings.TrimSuffix(s, "GB")
	} else if strings.HasSuffix(s, "MIB") || strings.HasSuffix(s, "MB") {
		multi = 1024 * 1024
		s = strings.TrimSuffix(s, "MIB")
		s = strings.TrimSuffix(s, "MB")
	} else if strings.HasSuffix(s, "KIB") || strings.HasSuffix(s, "KB") {
		multi = 1024
		s = strings.TrimSuffix(s, "KIB")
		s = strings.TrimSuffix(s, "KB")
	} else if strings.HasSuffix(s, "B") {
		s = strings.TrimSuffix(s, "B")
	}

	val, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return int64(val * multi)
}

func findExecutable(name string, commonPaths []string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
