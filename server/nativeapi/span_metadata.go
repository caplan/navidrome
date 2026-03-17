package nativeapi

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/caplan/navidrome/conf"
	"github.com/caplan/navidrome/log"
)

// writeMetadataTag writes a custom metadata tag to an audio file using ffmpeg.
// It creates a temporary file and replaces the original on success.
func writeMetadataTag(filePath, tagName, tagValue string) error {
	ffmpegPath := conf.Server.FFmpegPath
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	dir := filepath.Dir(filePath)
	ext := filepath.Ext(filePath)
	tmpFile, err := os.CreateTemp(dir, "navidrome-span-*"+ext)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Use ffmpeg to copy the file and add/update the custom metadata tag.
	// -y overwrites the temp file, -i reads input, -map 0 copies all streams,
	// -c copy avoids re-encoding, -metadata sets the tag.
	args := []string{
		"-y",
		"-i", filePath,
		"-map", "0",
		"-c", "copy",
		"-metadata", fmt.Sprintf("%s=%s", tagName, tagValue),
		tmpPath,
	}

	cmd := exec.Command(ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tmpPath)
		log.Error("ffmpeg metadata write failed", "path", filePath, "output", string(output), err)
		return fmt.Errorf("ffmpeg metadata write: %w", err)
	}

	// Replace original file with the new one
	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replacing file: %w", err)
	}

	return nil
}
