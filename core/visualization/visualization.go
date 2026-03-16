package visualization

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/caplan/music-visualizer/go/songviz"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// Modes is the set of visualization types generated for each song.
var Modes = []string{"radio", "blocky", "ribbon", "heatmap"}

const (
	// DefaultBatchSize is the number of songs to process per scheduled run.
	DefaultBatchSize = 10
	// DefaultSchedule is how often the background processor runs.
	DefaultSchedule = "@every 5m"
	// svgSize is the pixel dimension passed to songviz.Render.
	svgSize = 800
)

// Generator creates visualization SVGs for songs.
type Generator struct {
	ds model.DataStore
}

func NewGenerator(ds model.DataStore) *Generator {
	return &Generator{ds: ds}
}

// ProcessBatch finds songs that need visualization and generates SVGs.
// Visualizations are keyed by a hash of the acoustic fingerprint, so
// metadata changes (title, artist, etc.) do not trigger regeneration.
// Only a change in the actual audio content (new acoustic ID) will.
//
// Returns the number of songs processed.
func (g *Generator) ProcessBatch(ctx context.Context, batchSize int) (int, error) {
	mfs, err := g.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.NotEq{"media_file.acoustic_id": ""},
			squirrel.Eq{"media_file.missing": false},
		},
		Max: batchSize,
	})
	if err != nil {
		return 0, fmt.Errorf("querying media files: %w", err)
	}

	if len(mfs) == 0 {
		return 0, nil
	}

	vizDir := visualizationDir()
	processed := 0
	for _, mf := range mfs {
		if ctx.Err() != nil {
			break
		}
		if mf.AcousticID == "" {
			continue
		}

		dirName := hashAcousticID(mf.AcousticID)
		songDir := filepath.Join(vizDir, dirName)

		if allVisualizationsExist(songDir) {
			continue
		}

		filePath := mf.AbsolutePath()
		log.Info(ctx, "Generating visualizations", "id", mf.ID, "title", mf.Title, "hash", dirName)
		if err := generateVisualizations(ctx, filePath, songDir); err != nil {
			log.Warn(ctx, "Failed to generate visualizations", "id", mf.ID, "title", mf.Title, err)
			continue
		}

		processed++
	}

	if processed > 0 {
		log.Info(ctx, "Visualization batch complete", "processed", processed)
	}
	return processed, nil
}

// ProcessAll processes all songs that need visualizations.
func (g *Generator) ProcessAll(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n, err := g.ProcessBatch(ctx, DefaultBatchSize)
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// CleanupStale removes visualization directories whose hash no longer
// corresponds to any media file's acoustic ID. This handles the case
// where a file's audio content changed and got a new acoustic ID.
func (g *Generator) CleanupStale(ctx context.Context) error {
	vizDir := visualizationDir()
	entries, err := os.ReadDir(vizDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Build a set of all current hashes
	activeHashes := make(map[string]bool)
	offset := 0
	for {
		mfs, err := g.ds.MediaFile(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.NotEq{"media_file.acoustic_id": ""},
			Max:     500,
			Offset:  offset,
		})
		if err != nil {
			return fmt.Errorf("querying acoustic IDs for cleanup: %w", err)
		}
		if len(mfs) == 0 {
			break
		}
		for _, mf := range mfs {
			activeHashes[hashAcousticID(mf.AcousticID)] = true
		}
		offset += len(mfs)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirHash := entry.Name()
		if !activeHashes[dirHash] {
			dir := filepath.Join(vizDir, dirHash)
			log.Info(ctx, "Removing stale visualizations", "hash", dirHash)
			if err := os.RemoveAll(dir); err != nil {
				log.Warn(ctx, "Failed to remove stale visualization dir", "dir", dir, err)
			}
		}
	}
	return nil
}

// GetVisualizationPath returns the path to a specific visualization SVG.
func GetVisualizationPath(acousticID, mode string) string {
	return filepath.Join(visualizationDir(), hashAcousticID(acousticID), mode+".svg")
}

// hashAcousticID returns a short, filesystem-safe hash of the acoustic fingerprint.
// Acoustic fingerprints can be 1000+ chars, exceeding filesystem name limits.
func hashAcousticID(acousticID string) string {
	h := sha256.Sum256([]byte(acousticID))
	return hex.EncodeToString(h[:])
}

func visualizationDir() string {
	return filepath.Join(conf.Server.DataFolder, "visualizations")
}

func allVisualizationsExist(dir string) bool {
	for _, mode := range Modes {
		path := filepath.Join(dir, mode+".svg")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func generateVisualizations(ctx context.Context, audioPath, outputDir string) error {
	pcm, err := decodeToPCM(ctx, audioPath)
	if err != nil {
		return fmt.Errorf("decoding audio: %w", err)
	}

	analysis := songviz.Analyze(pcm, songviz.DefaultConfig())

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating visualization dir: %w", err)
	}

	for _, mode := range Modes {
		svg := songviz.Render(analysis, svgSize, nil, mode)
		outPath := filepath.Join(outputDir, mode+".svg")
		if err := os.WriteFile(outPath, []byte(svg), 0644); err != nil {
			return fmt.Errorf("writing %s visualization: %w", mode, err)
		}
	}

	return nil
}

// decodeToPCM uses ffmpeg to decode an audio file to raw mono f32le PCM at 22050 Hz.
func decodeToPCM(ctx context.Context, audioPath string) (songviz.PcmData, error) {
	ffmpegPath := conf.Server.FFmpegPath
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	sampleRate := 22050
	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-i", audioPath,
		"-ac", "1",
		"-ar", fmt.Sprintf("%d", sampleRate),
		"-f", "f32le",
		"-",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return songviz.PcmData{}, fmt.Errorf("ffmpeg decode failed: %w (stderr: %s)", err, stderr.String())
	}

	raw := stdout.Bytes()
	if len(raw) < 4 {
		return songviz.PcmData{}, fmt.Errorf("ffmpeg produced no audio data for %s", audioPath)
	}

	samples := make([]float32, len(raw)/4)
	for i := range samples {
		bits := binary.LittleEndian.Uint32(raw[i*4 : (i+1)*4])
		samples[i] = math.Float32frombits(bits)
	}

	duration := float32(len(samples)) / float32(sampleRate)

	return songviz.PcmData{
		Samples:         samples,
		SampleRate:      sampleRate,
		DurationSeconds: duration,
	}, nil
}
