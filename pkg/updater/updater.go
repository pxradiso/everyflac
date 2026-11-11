// Package updater performs a lightweight, non-blocking check for new everyFLAC
// releases. It never modifies the installation; it only reports availability.
package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// Repo is the GitHub repository that publishes everyFLAC.
	Repo = "pxradiso/everyflac"

	checkInterval = 24 * time.Hour
	httpTimeout   = 3 * time.Second
)

// Result describes the outcome of an update check.
type Result struct {
	Current   string
	Latest    string
	Available bool
}

type cacheFile struct {
	CheckedAt int64  `json:"checked_at"`
	Latest    string `json:"latest"`
}

// Check reports whether a version newer than current is available.
//
// The result is cached for 24 hours so the TUI does not hit GitHub on every
// launch. Set EVERYFLAC_NO_UPDATE_CHECK=1 to disable the check completely.
func Check(current string) (Result, error) {
	return check(current, false)
}

// CheckFresh is like Check but ignores the cache. Use it for explicit,
// user-requested checks.
func CheckFresh(current string) (Result, error) {
	return check(current, true)
}

func check(current string, force bool) (Result, error) {
	current = normalize(current)
	res := Result{Current: current}

	if disabled() {
		return res, nil
	}

	if !force {
		if c, ok := readCache(); ok && time.Since(time.Unix(c.CheckedAt, 0)) < checkInterval {
			res.Latest = normalize(c.Latest)
			res.Available = isNewer(res.Latest, current)
			return res, nil
		}
	}

	latest, err := fetchLatest()
	if err != nil {
		return res, err
	}
	latest = normalize(latest)
	res.Latest = latest
	res.Available = isNewer(latest, current)
	writeCache(latest)
	return res, nil
}

func disabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("EVERYFLAC_NO_UPDATE_CHECK")))
	return v != "" && v != "0" && v != "false" && v != "no"
}

func fetchLatest() (string, error) {
	client := &http.Client{Timeout: httpTimeout}

	// Prefer the repository VERSION file: no API rate limits.
	rawURL := "https://raw.githubusercontent.com/" + Repo + "/main/VERSION"
	if v, err := fetchText(client, rawURL); err == nil {
		if v = normalize(v); v != "" {
			return v, nil
		}
	}

	// Fall back to the latest GitHub release tag.
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+Repo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "everyflac-update-check")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github: %s", resp.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.TagName == "" {
		return "", fmt.Errorf("github: empty tag")
	}
	return normalize(payload.TagName), nil
}

func fetchText(client *http.Client, url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "everyflac-update-check")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func cachePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "everyflac", "update-check.json")
}

func readCache() (cacheFile, bool) {
	p := cachePath()
	if p == "" {
		return cacheFile{}, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return cacheFile{}, false
	}
	var c cacheFile
	if err := json.Unmarshal(data, &c); err != nil || c.Latest == "" {
		return cacheFile{}, false
	}
	return c, true
}

func writeCache(latest string) {
	p := cachePath()
	if p == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	data, err := json.Marshal(cacheFile{CheckedAt: time.Now().Unix(), Latest: latest})
	if err != nil {
		return
	}
	_ = os.WriteFile(p, data, 0o644)
}

func normalize(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return v
}

func isNewer(latest, current string) bool {
	l := parseVersion(latest)
	c := parseVersion(current)
	for i := 0; i < len(l) || i < len(c); i++ {
		var a, b int
		if i < len(l) {
			a = l[i]
		}
		if i < len(c) {
			b = c[i]
		}
		if a != b {
			return a > b
		}
	}
	return false
}

func parseVersion(v string) []int {
	v = normalize(v)
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, _ := strconv.Atoi(strings.TrimSpace(p))
		out = append(out, n)
	}
	return out
}
