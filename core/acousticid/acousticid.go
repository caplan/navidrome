package acousticid

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/caplan/navidrome/conf"
	"github.com/caplan/navidrome/log"
	"github.com/caplan/navidrome/model"
)

const (
	// DefaultBatchSize is the number of files to process per run.
	DefaultBatchSize = 50
	// DefaultSchedule is how often the background processor runs.
	DefaultSchedule = "@every 5m"
)

// fpcalcResult is the JSON output of fpcalc -json.
type fpcalcResult struct {
	Duration    float64 `json:"duration"`
	Fingerprint string  `json:"fingerprint"`
}

// Calculator computes acoustic fingerprints for media files.
type Calculator struct {
	ds model.DataStore
}

func NewCalculator(ds model.DataStore) *Calculator {
	return &Calculator{ds: ds}
}

// ProcessBatch finds media files without acoustic IDs and calculates them.
// Returns the number of files processed.
func (c *Calculator) ProcessBatch(ctx context.Context, batchSize int) (int, error) {
	mfs, err := c.ds.MediaFile(ctx).GetWithoutAcousticID(batchSize)
	if err != nil {
		return 0, fmt.Errorf("querying files without acoustic ID: %w", err)
	}

	if len(mfs) == 0 {
		return 0, nil
	}

	log.Debug(ctx, "Processing acoustic IDs", "count", len(mfs))
	processed := 0
	for _, mf := range mfs {
		if ctx.Err() != nil {
			break
		}

		filePath := mf.AbsolutePath()

		fingerprint, err := Calculate(filePath)
		if err != nil {
			log.Warn(ctx, "Failed to calculate acoustic ID", "path", filePath, "id", mf.ID, err)
			continue
		}

		if err := c.ds.MediaFile(ctx).SetAcousticID(mf.ID, fingerprint); err != nil {
			log.Error(ctx, "Failed to store acoustic ID", "id", mf.ID, err)
			continue
		}

		// Write to file metadata (best effort)
		writeAcousticIDToFile(filePath, fingerprint)

		processed++
		log.Debug(ctx, "Calculated acoustic ID", "id", mf.ID, "title", mf.Title)
	}

	if processed > 0 {
		log.Info(ctx, "Acoustic ID batch complete", "processed", processed, "total", len(mfs))
	}
	return processed, nil
}

// RunUntilDone processes all files without acoustic IDs in batches.
func (c *Calculator) RunUntilDone(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n, err := c.ProcessBatch(ctx, DefaultBatchSize)
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		// Small pause between batches to avoid overloading
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// Calculate computes the Chromaprint acoustic fingerprint for the given audio file.
// It requires fpcalc (from Chromaprint) to be installed on the system.
func Calculate(filePath string) (string, error) {
	fpcalcPath := conf.Server.FpcalcPath
	if fpcalcPath == "" {
		fpcalcPath = "fpcalc"
	}

	cmd := exec.Command(fpcalcPath, "-json", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("running fpcalc: %w", err)
	}

	var result fpcalcResult
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("parsing fpcalc output: %w", err)
	}

	if result.Fingerprint == "" {
		return "", fmt.Errorf("fpcalc returned empty fingerprint for %s", filePath)
	}

	return result.Fingerprint, nil
}

// writeAcousticIDToFile writes the acoustic ID to the file's metadata tag.
// This is best-effort; errors are logged but not returned.
func writeAcousticIDToFile(filePath, fingerprint string) {
	ffmpegPath := conf.Server.FFmpegPath
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	// Use ffmpeg to write the ACOUSTID_FINGERPRINT tag
	// We don't use the temp-file-and-rename approach here since the acoustic ID
	// is also stored in the database. Just log failures.
	cmd := exec.Command(ffmpegPath,
		"-y", "-i", filePath,
		"-map", "0", "-c", "copy",
		"-metadata", fmt.Sprintf("ACOUSTID_FINGERPRINT=%s", fingerprint),
		filePath+".tmp",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Error("Failed to write acoustic ID to file metadata", "path", filePath, "output", string(output), err)
		return
	}

	if err := exec.Command("mv", filePath+".tmp", filePath).Run(); err != nil {
		log.Error("Failed to replace file after acoustic ID write", "path", filePath, err)
	}
}
