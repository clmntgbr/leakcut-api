package video

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"go-api/internal/domain/port"
)

type SegmentSplitter struct{}

func NewSegmentSplitter() *SegmentSplitter {
	return &SegmentSplitter{}
}

func (s *SegmentSplitter) ProbeDurationMs(ctx context.Context, videoPath string) (int64, error) {
	cmd := exec.CommandContext(
		ctx,
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("ffprobe duration failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	raw := strings.TrimSpace(stdout.String())
	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("parse duration %q: %w", raw, err)
	}
	if seconds < 0 {
		seconds = 0
	}
	return int64(seconds * 1000), nil
}

func (s *SegmentSplitter) Split(
	ctx context.Context,
	videoPath string,
	segmentDurationSeconds float64,
	outputDir string,
) ([]port.SegmentFile, error) {
	if segmentDurationSeconds <= 0 {
		return nil, fmt.Errorf("segment duration must be positive")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}

	pattern := filepath.Join(outputDir, "segment_%02d.mp4")
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", videoPath,
		"-c", "copy",
		"-f", "segment",
		"-segment_time", strconv.FormatFloat(segmentDurationSeconds, 'f', -1, 64),
		"-reset_timestamps", "1",
		pattern,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg segment failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}

	files := make([]port.SegmentFile, 0, len(entries))
	offsetMs := int64(0)
	stepMs := int64(segmentDurationSeconds * 1000)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "segment_") || !strings.HasSuffix(entry.Name(), ".mp4") {
			continue
		}
		index, err := parseSegmentIndex(entry.Name())
		if err != nil {
			continue
		}
		path := filepath.Join(outputDir, entry.Name())
		durationMs, err := s.ProbeDurationMs(ctx, path)
		if err != nil {
			durationMs = stepMs
		}
		files = append(files, port.SegmentFile{
			Index:      index,
			Path:       path,
			OffsetMs:   int64(index) * stepMs,
			DurationMs: durationMs,
		})
		_ = offsetMs
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("ffmpeg produced no segment files")
	}
	return files, nil
}

func parseSegmentIndex(name string) (int, error) {
	// segment_00.mp4
	base := strings.TrimSuffix(strings.TrimPrefix(name, "segment_"), ".mp4")
	return strconv.Atoi(base)
}
