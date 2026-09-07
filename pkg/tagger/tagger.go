package tagger

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"everyflac-tui/pkg/models"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	flac "github.com/go-flac/go-flac"
)

type AudioQualityDetails struct {
	SampleRate   int
	BitDepth     int
	ChannelCount int
	DurationSec  int
	QualityLabel string
}

func IsFLAC(filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	header := make([]byte, 4)
	if _, err := f.Read(header); err != nil {
		return false
	}
	return bytes.Equal(header, []byte("fLaC"))
}

func InspectFLAC(filePath string) (*AudioQualityDetails, error) {
	file, err := flac.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	si, err := file.GetStreamInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to read FLAC StreamInfo: %w", err)
	}

	durationSec := 0
	if si.SampleRate > 0 && si.SampleCount > 0 {
		durationSec = int(si.SampleCount / int64(si.SampleRate))
	}

	khz := float64(si.SampleRate) / 1000.0
	label := fmt.Sprintf("FLAC %d-bit / %.1f kHz", si.BitDepth, khz)

	return &AudioQualityDetails{
		SampleRate:   si.SampleRate,
		BitDepth:     si.BitDepth,
		ChannelCount: si.ChannelCount,
		DurationSec:  durationSec,
		QualityLabel: label,
	}, nil
}

func TagFLACFile(filePath string, track *models.Track, coverBytes []byte) (*AudioQualityDetails, error) {
	file, err := flac.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse FLAC file for tagging: %w", err)
	}

	vc := flacvorbis.New()
	vc.Vendor = "everyFLAC by pxradise Universal Lossless Engine"

	if track.Title != "" {
		_ = vc.Add(flacvorbis.FIELD_TITLE, track.Title)
	}
	if track.Artist != "" {
		_ = vc.Add(flacvorbis.FIELD_ARTIST, track.Artist)
		_ = vc.Add("ALBUMARTIST", track.Artist)
	}
	if track.Album != "" {
		_ = vc.Add(flacvorbis.FIELD_ALBUM, track.Album)
	}
	if track.TrackNumber > 0 {
		_ = vc.Add(flacvorbis.FIELD_TRACKNUMBER, strconv.Itoa(track.TrackNumber))
	}
	if track.TotalTracks > 0 {
		_ = vc.Add("TRACKTOTAL", strconv.Itoa(track.TotalTracks))
	}
	if track.Year != "" {
		_ = vc.Add(flacvorbis.FIELD_DATE, track.Year)
	}
	if track.ISRC != "" {
		_ = vc.Add(flacvorbis.FIELD_ISRC, track.ISRC)
	}
	_ = vc.Add("COMMENT", "Lossless FLAC downloaded by everyFLAC by pxradise")

	vcBlock := vc.Marshal()

	newMeta := make([]*flac.MetaDataBlock, 0, len(file.Meta)+2)
	for _, b := range file.Meta {
		if b.Type != flac.VorbisComment && b.Type != flac.Picture {
			newMeta = append(newMeta, b)
		}
	}

	newMeta = append(newMeta, &vcBlock)

	if len(coverBytes) > 0 {
		cleanBytes := EnsureJPEG(coverBytes)
		if len(cleanBytes) > 0 {
			mimeType := "image/jpeg"
			if len(cleanBytes) > 8 && bytes.Equal(cleanBytes[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
				mimeType = "image/png"
			}

			pic, err := flacpicture.NewFromImageData(
				flacpicture.PictureTypeFrontCover,
				"Front Cover",
				cleanBytes,
				mimeType,
			)
			if err == nil {
				picBlock := pic.Marshal()
				newMeta = append(newMeta, &picBlock)
			}
		}
	}

	file.Meta = newMeta

	if err := file.Save(filePath); err != nil {
		return nil, fmt.Errorf("failed to write tagged FLAC file: %w", err)
	}

	return InspectFLAC(filePath)
}

func EnsureJPEG(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}

	if len(data) > 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return data
	}

	if len(data) > 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
		return data
	}

	tmpIn := filepath.Join(os.TempDir(), fmt.Sprintf("everyflac_img_%d", time.Now().UnixNano()))
	tmpOut := tmpIn + ".jpg"
	defer func() {
		_ = os.Remove(tmpIn)
		_ = os.Remove(tmpOut)
	}()

	if err := os.WriteFile(tmpIn, data, 0644); err != nil {
		return data
	}

	cmd := exec.Command("ffmpeg", "-y", "-i", tmpIn, "-q:v", "1", tmpOut)
	if err := cmd.Run(); err == nil {
		if converted, err := os.ReadFile(tmpOut); err == nil && len(converted) > 0 {
			return converted
		}
	}

	return data
}

func SanitizeFilename(name string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	clean := name
	for _, char := range invalidChars {
		clean = strings.ReplaceAll(clean, char, "_")
	}
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return "track"
	}
	return clean
}
