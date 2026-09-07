package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type SourceMode string

const (
	SourceModeSpotify SourceMode = "spotify"
	SourceModeYouTube SourceMode = "youtube"
)

type AppConfig struct {
	OutputDir          string   `json:"output_dir"`
	CreateArtistFolder bool     `json:"create_artist_folder"`
	CreateAlbumFolder  bool     `json:"create_album_folder"`
	PreferredQuality   string   `json:"preferred_quality"`
	EmbedLyrics        bool     `json:"embed_lyrics"`
	SourceMode         string   `json:"source_mode"`
	QobuzMirrors       []string `json:"qobuz_mirrors"`
	TidalMirrors       []string `json:"tidal_mirrors"`
}

func DefaultConfig() *AppConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return &AppConfig{
		OutputDir:          filepath.Join(homeDir, "Music"),
		CreateArtistFolder: false,
		CreateAlbumFolder:  false,
		PreferredQuality:   "FLAC_24BIT",
		EmbedLyrics:        true,
		SourceMode:         "spotify",
		QobuzMirrors: []string{
			"https://qobuz.anandserver.cfd",
			"https://mono.scavengerfurs.net",
		},
		TidalMirrors: []string{
			"https://tidal.anandserver.cfd",
		},
	}
}

func ConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "everyflac", "config.json")
}

func Load() (*AppConfig, error) {
	cfgPath := ConfigPath()
	if data, err := os.ReadFile(cfgPath); err == nil {
		cfg := DefaultConfig()
		if err := json.Unmarshal(data, cfg); err == nil {
			if cfg.OutputDir == "" {
				homeDir, _ := os.UserHomeDir()
				cfg.OutputDir = filepath.Join(homeDir, "Music")
			}
			if cfg.SourceMode != "youtube" {
				cfg.SourceMode = "spotify"
			}
			return cfg, nil
		}
	}

	cfg := DefaultConfig()
	_ = Save(cfg)
	return cfg, nil
}

func Save(cfg *AppConfig) error {
	cfgPath := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cfgPath, data, 0644)
}

func ExpandPath(path string) string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
