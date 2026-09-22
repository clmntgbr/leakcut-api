package video

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	domainframe "go-api/internal/domain/frame"
	domainjob "go-api/internal/domain/job"
	"go-api/internal/domain/port"
)

var pngSignature = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

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

func (e *FrameExtractor) ExtractFrames(
	ctx context.Context,
	videoPath string,
	params port.FrameSelectionParams,
	emit port.FrameSink,
) error {
	if emit == nil {
		return errors.New("frame sink is required")
	}
	if params.AnalysisFPS <= 0 {
		params.AnalysisFPS = domainjob.DefaultAnalysisFPS
	}
	if params.DiffThreshold <= 0 {
		params.DiffThreshold = domainjob.DefaultDiffThreshold
	}
	if params.MaxIntervalSeconds <= 0 {
		params.MaxIntervalSeconds = domainjob.DefaultMaxIntervalSeconds
	}
	if params.MaxWidthPx <= 0 {
		params.MaxWidthPx = domainjob.DefaultFrameMaxWidthPx
	}

	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", videoPath,
		"-vf", fmt.Sprintf(
			"fps=%s,scale='min(%d,iw)':-2",
			strconv.FormatFloat(params.AnalysisFPS, 'f', -1, 64),
			params.MaxWidthPx,
		),
		"-f", "image2pipe",
		"-vcodec", "png",
		"pipe:1",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w", err)
	}

	maxIntervalMs := int64(params.MaxIntervalSeconds) * 1000
	kept := 0
	candidate := 0
	var lastGray []uint8
	var lastWidth, lastHeight int
	var lastKeptAt int64 = -1
	readErr := error(nil)

	for {
		data, err := readPNG(stdout)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			readErr = err
			break
		}

		timestampMs := int64(math.Round(float64(candidate) * 1000 / params.AnalysisFPS))
		candidate++
		gray, width, height, err := loadGrayPNG(data)
		if err != nil {
			readErr = err
			break
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

		if err := emit(port.ExtractedFrame{
			Index:           kept,
			TimestampMs:     timestampMs,
			Data:            data,
			SelectionReason: reason,
			DiffScore:       score,
		}); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return err
		}
		kept++
		lastGray = gray
		lastWidth, lastHeight = width, height
		lastKeptAt = timestampMs
	}

	waitErr := cmd.Wait()
	if readErr != nil {
		return readErr
	}
	if waitErr != nil && kept == 0 {
		return fmt.Errorf("ffmpeg failed: %w (%s)", waitErr, strings.TrimSpace(stderr.String()))
	}
	if kept == 0 {
		return fmt.Errorf("no frames retained")
	}
	return nil
}

func readPNG(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	sig := make([]byte, 8)
	if _, err := io.ReadFull(r, sig); err != nil {
		return nil, err
	}
	if !bytes.Equal(sig, pngSignature) {
		return nil, fmt.Errorf("invalid png signature")
	}
	buf.Write(sig)

	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(r, header); err != nil {
			return nil, err
		}
		buf.Write(header)
		length := binary.BigEndian.Uint32(header[:4])
		typ := string(header[4:8])
		chunk := make([]byte, int(length)+4)
		if _, err := io.ReadFull(r, chunk); err != nil {
			return nil, err
		}
		buf.Write(chunk)
		if typ == "IEND" {
			return buf.Bytes(), nil
		}
	}
}

func loadGrayPNG(raw []byte) ([]uint8, int, int, error) {
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decode frame: %w", err)
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
	return gray, width, height, nil
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
