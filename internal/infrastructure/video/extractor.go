package video

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	domainframe "go-api/internal/domain/frame"
	"go-api/internal/domain/port"
)

type FrameExtractor struct{}

func NewFrameExtractor() *FrameExtractor {
	return &FrameExtractor{}
}

func (e *FrameExtractor) ExtractThumbnail(ctx context.Context, videoPath string) ([]byte, error) {
	tmp, err := os.CreateTemp("", "video-thumb-*.jpg")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp thumbnail: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	var lastErr error
	for _, seek := range []string{"1", "0"} {
		cmd := exec.CommandContext(
			ctx,
			"ffmpeg",
			"-y",
			"-ss", seek,
			"-i", videoPath,
			"-frames:v", "1",
			"-update", "1",
			"-vf", "scale='min(640,iw)':-2",
			"-q:v", "3",
			tmpPath,
		)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			lastErr = fmt.Errorf("ffmpeg thumbnail failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
			continue
		}

		data, err := os.ReadFile(tmpPath)
		if err != nil {
			lastErr = err
			continue
		}
		if len(data) == 0 {
			lastErr = errors.New("ffmpeg produced an empty thumbnail")
			continue
		}
		return data, nil
	}

	if lastErr == nil {
		lastErr = errors.New("ffmpeg produced no thumbnail")
	}
	return nil, lastErr
}

func (e *FrameExtractor) ExtractFrames(ctx context.Context, videoPath string, params port.FrameSelectionParams) ([]port.ExtractedFrame, error) {
	if params.AnalysisFPS <= 0 {
		params.AnalysisFPS = 3
	}
	if params.DiffThreshold <= 0 {
		params.DiffThreshold = 0.08
	}
	if params.MaxIntervalSeconds <= 0 {
		params.MaxIntervalSeconds = 10
	}

	tmpDir, err := os.MkdirTemp("", "frame-extract-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	pattern := filepath.Join(tmpDir, "candidate-%06d.png")
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", videoPath,
		"-vf", fmt.Sprintf("fps=%s", strconv.FormatFloat(params.AnalysisFPS, 'f', -1, 64)),
		pattern,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	candidates, err := filepath.Glob(filepath.Join(tmpDir, "candidate-*.png"))
	if err != nil {
		return nil, err
	}
	sort.Strings(candidates)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("ffmpeg produced no frames")
	}

	maxIntervalMs := int64(params.MaxIntervalSeconds) * 1000
	selected := make([]port.ExtractedFrame, 0)
	var lastGray []uint8
	var lastWidth, lastHeight int
	var lastKeptAt int64 = -1

	for i, path := range candidates {
		timestampMs := int64(math.Round(float64(i) * 1000 / params.AnalysisFPS))
		gray, width, height, data, err := loadGrayPNG(path)
		if err != nil {
			return nil, err
		}

		reason := ""
		score := 0.0
		if lastKeptAt < 0 {
			reason = domainframe.SelectionReasonFixedInterval
		} else {
			score = meanAbsDiff(lastGray, gray, lastWidth, lastHeight, width, height)
			if score >= params.DiffThreshold {
				reason = domainframe.SelectionReasonSceneChange
			} else if timestampMs-lastKeptAt >= maxIntervalMs {
				reason = domainframe.SelectionReasonFixedInterval
			}
		}

		if reason == "" {
			continue
		}

		selected = append(selected, port.ExtractedFrame{
			Index:           len(selected),
			TimestampMs:     timestampMs,
			Data:            data,
			SelectionReason: reason,
			DiffScore:       score,
		})
		lastGray = gray
		lastWidth, lastHeight = width, height
		lastKeptAt = timestampMs
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no frames retained")
	}
	return selected, nil
}

func loadGrayPNG(path string) ([]uint8, int, int, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, 0, nil, err
	}

	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, nil, fmt.Errorf("decode frame: %w", err)
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	gray := make([]uint8, width*height)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			gray[(y-bounds.Min.Y)*width+(x-bounds.Min.X)] = uint8(((r*299 + g*587 + b*114) / 1000) >> 8)
		}
	}

	return gray, width, height, raw, nil
}

func meanAbsDiff(a []uint8, b []uint8, aw, ah, bw, bh int) float64 {
	if len(a) == 0 || len(b) == 0 || aw == 0 || ah == 0 || bw == 0 || bh == 0 {
		return 1
	}

	width := aw
	height := ah
	if bw < width {
		width = bw
	}
	if bh < height {
		height = bh
	}

	var total float64
	pixels := width * height
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			diff := int(a[y*aw+x]) - int(b[y*bw+x])
			if diff < 0 {
				diff = -diff
			}
			total += float64(diff)
		}
	}
	return total / float64(pixels) / 255
}
