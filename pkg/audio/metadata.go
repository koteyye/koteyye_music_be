package audio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"strconv"
)

// Metadata represents audio file metadata
type Metadata struct {
	Duration float64 `json:"duration"`
	Format   string  `json:"format_name"`
	BitRate  int     `json:"bit_rate"`
}

// FFProbeOutput represents the structure of ffprobe JSON output
type FFProbeOutput struct {
	Format struct {
		Duration string `json:"duration"`
		FormatName string `json:"format_name"`
		BitRate string `json:"bit_rate"`
	} `json:"format"`
}

// ExtractMetadata extracts duration and other metadata from audio file using ffprobe
func ExtractMetadata(audioFile multipart.File) (*Metadata, error) {
	// Create temporary file
	tempFile, err := os.CreateTemp("", "audio_metadata_*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy uploaded file to temp file
	_, err = io.Copy(tempFile, audioFile)
	if err != nil {
		return nil, fmt.Errorf("failed to copy audio file: %w", err)
	}

	// Reset file pointers
	tempFile.Seek(0, 0)
	audioFile.Seek(0, 0)

	// Run ffprobe to get metadata
	cmd := exec.Command("ffprobe", 
		"-v", "quiet",           // Suppress verbose output
		"-print_format", "json", // Output in JSON format
		"-show_format",          // Show format information
		tempFile.Name())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w (stderr: %s)", err, stderr.String())
	}

	// Parse ffprobe output
	var output FFProbeOutput
	err = json.Unmarshal(stdout.Bytes(), &output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	// Parse duration
	duration, err := strconv.ParseFloat(output.Format.Duration, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration: %w", err)
	}

	// Parse bit rate (optional)
	bitRate := 0
	if output.Format.BitRate != "" {
		bitRateInt, err := strconv.Atoi(output.Format.BitRate)
		if err == nil {
			bitRate = bitRateInt
		}
	}

	return &Metadata{
		Duration: duration,
		Format:   output.Format.FormatName,
		BitRate:  bitRate,
	}, nil
}

// GetDurationSeconds returns duration in seconds as integer
func (m *Metadata) GetDurationSeconds() int {
	return int(m.Duration)
}

// IsValidAudioFormat checks if the detected format is a supported audio format
func (m *Metadata) IsValidAudioFormat() bool {
	validFormats := []string{
		"mp3", "wav", "m4a", "aac", "flac", "ogg", "wma",
	}
	
	for _, format := range validFormats {
		if m.Format == format {
			return true
		}
	}
	
	return false
}

// FormatDuration returns a human-readable duration string (MM:SS)
func (m *Metadata) FormatDuration() string {
	totalSeconds := int(m.Duration)
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}