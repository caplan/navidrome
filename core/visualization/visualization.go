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
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/caplan/music-visualizer/go/songviz"
	"github.com/caplan/navidrome/conf"
	"github.com/caplan/navidrome/log"
	"github.com/caplan/navidrome/model"
)

// Modes is the set of visualization types generated for each song.
var Modes = []string{"radio", "blocky", "ribbons", "heatmap"}

const (
	// DefaultBatchSize is the number of songs to process per scheduled run.
	DefaultBatchSize = 10
	// DefaultSchedule is how often the background processor runs.
	DefaultSchedule = "@every 5m"
	// svgSize is the pixel dimension passed to songviz.Render.
	svgSize = 800
)

// SpecVersion is the current songviz spec version, derived from the module version.
// When the songviz library is upgraded, this changes and triggers regeneration.
var SpecVersion = detectSpecVersion()

func detectSpecVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dep := range bi.Deps {
		if dep.Path == "github.com/caplan/music-visualizer/go" {
			return dep.Version
		}
	}
	return "unknown"
}

// Generator creates visualization SVGs for songs.
type Generator struct {
	ds model.DataStore
}

func NewGenerator(ds model.DataStore) *Generator {
	return &Generator{ds: ds}
}

// ProcessBatch generates visualizations with two priority levels:
//  1. Songs with NO visualizations at any version (missing)
//  2. Songs with visualizations at an older version (upgrades)
//
// Returns the number of songs processed.
func (g *Generator) ProcessBatch(ctx context.Context, batchSize int) (int, error) {
	// Priority 1: generate missing visualizations
	processed, err := g.processMissing(ctx, batchSize)
	if err != nil {
		return processed, err
	}

	// Priority 2: upgrade older versions (only if we have budget left)
	if processed < batchSize {
		upgraded, err := g.processUpgrades(ctx, batchSize-processed)
		processed += upgraded
		if err != nil {
			return processed, err
		}
	}

	if processed > 0 {
		log.Info(ctx, "Visualization batch complete", "processed", processed, "specVersion", SpecVersion)
	}
	return processed, nil
}

// processMissing generates visualizations for songs that have no visualization at any version.
func (g *Generator) processMissing(ctx context.Context, limit int) (int, error) {
	vizDir := visualizationDir()
	processed := 0
	offset := 0
	const pageSize = 100

	for processed < limit {
		if ctx.Err() != nil {
			break
		}

		mfs, err := g.ds.MediaFile(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.And{
				squirrel.NotEq{"media_file.acoustic_id": ""},
				squirrel.Eq{"media_file.missing": false},
			},
			Max:    pageSize,
			Offset: offset,
		})
		if err != nil {
			return processed, fmt.Errorf("querying media files: %w", err)
		}
		if len(mfs) == 0 {
			break
		}
		offset += len(mfs)

		for _, mf := range mfs {
			if ctx.Err() != nil || processed >= limit {
				break
			}
			if mf.AcousticID == "" {
				continue
			}

			hashDir := filepath.Join(vizDir, hashAcousticID(mf.AcousticID))

			// Skip if any version exists at all
			if hasAnyVersion(hashDir) {
				continue
			}

			if err := g.generateForSong(ctx, mf, hashDir); err != nil {
				continue
			}
			processed++
		}
	}
	return processed, nil
}

// processUpgrades regenerates visualizations for songs that have an older spec version.
func (g *Generator) processUpgrades(ctx context.Context, limit int) (int, error) {
	vizDir := visualizationDir()
	processed := 0
	offset := 0
	const pageSize = 100

	for processed < limit {
		if ctx.Err() != nil {
			break
		}

		mfs, err := g.ds.MediaFile(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.And{
				squirrel.NotEq{"media_file.acoustic_id": ""},
				squirrel.Eq{"media_file.missing": false},
			},
			Max:    pageSize,
			Offset: offset,
		})
		if err != nil {
			return processed, fmt.Errorf("querying media files: %w", err)
		}
		if len(mfs) == 0 {
			break
		}
		offset += len(mfs)

		for _, mf := range mfs {
			if ctx.Err() != nil || processed >= limit {
				break
			}
			if mf.AcousticID == "" {
				continue
			}

			hashDir := filepath.Join(vizDir, hashAcousticID(mf.AcousticID))

			// Skip if current version already exists
			if allVisualizationsExist(filepath.Join(hashDir, SpecVersion)) {
				continue
			}

			// Skip if no version exists (handled by processMissing)
			if !hasAnyVersion(hashDir) {
				continue
			}

			if err := g.generateForSong(ctx, mf, hashDir); err != nil {
				continue
			}
			processed++
		}
	}
	return processed, nil
}

func (g *Generator) generateForSong(ctx context.Context, mf model.MediaFile, hashDir string) error {
	versionDir := filepath.Join(hashDir, SpecVersion)

	filePath := mf.AbsolutePath()
	log.Info(ctx, "Generating visualizations", "id", mf.ID, "title", mf.Title,
		"hash", filepath.Base(hashDir), "version", SpecVersion)

	if err := generateVisualizations(ctx, filePath, versionDir); err != nil {
		log.Warn(ctx, "Failed to generate visualizations", "id", mf.ID, "title", mf.Title, err)
		return err
	}

	// Delete older versions now that the new one is ready
	deleteOlderVersions(ctx, hashDir, SpecVersion)
	return nil
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
// corresponds to any media file's acoustic ID.
func (g *Generator) CleanupStale(ctx context.Context) error {
	vizDir := visualizationDir()
	entries, err := os.ReadDir(vizDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

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

// GetVisualizationPath returns the path to a specific visualization SVG,
// using the latest available version on disk.
func GetVisualizationPath(acousticID, mode string) string {
	hashDir := filepath.Join(visualizationDir(), hashAcousticID(acousticID))
	version := latestVersionOnDisk(hashDir)
	if version == "" {
		// Return the current spec version path (will 404 if not generated yet)
		return filepath.Join(hashDir, SpecVersion, mode+".svg")
	}
	return filepath.Join(hashDir, version, mode+".svg")
}

// latestVersionOnDisk returns the highest semver directory name inside hashDir,
// or "" if none exist.
func latestVersionOnDisk(hashDir string) string {
	entries, err := os.ReadDir(hashDir)
	if err != nil {
		return ""
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "v") {
			versions = append(versions, e.Name())
		}
	}
	if len(versions) == 0 {
		return ""
	}
	sort.Strings(versions)
	return versions[len(versions)-1]
}

// hasAnyVersion returns true if the hash directory has at least one version subdirectory.
func hasAnyVersion(hashDir string) bool {
	return latestVersionOnDisk(hashDir) != ""
}

// deleteOlderVersions removes all version directories except the specified one.
func deleteOlderVersions(ctx context.Context, hashDir, keepVersion string) {
	entries, err := os.ReadDir(hashDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() && e.Name() != keepVersion {
			old := filepath.Join(hashDir, e.Name())
			log.Debug(ctx, "Removing old visualization version", "dir", old, "kept", keepVersion)
			_ = os.RemoveAll(old)
		}
	}
}

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
