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

	"github.com/corona10/goimagehash"
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
	if params.PHashDistanceThreshold <= 0 {
		params.PHashDistanceThreshold = domainjob.DefaultPHashDistanceThreshold
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
	var lastKeptHash *goimagehash.ImageHash
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
		hash, err := perceptionHashPNG(data)
		if err != nil {
			readErr = err
			break
		}

		reason := ""
		distance := 0
		if lastKeptHash == nil {
			reason = domainframe.SelectionReasonFixedInterval
		} else {
			distance, err = hash.Distance(lastKeptHash)
			if err != nil {
				readErr = err
				break
			}
			if distance >= params.PHashDistanceThreshold {
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
			PHashDistance:   distance,
		}); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return err
		}
		kept++
		lastKeptHash = hash
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

func perceptionHashPNG(raw []byte) (*goimagehash.ImageHash, error) {
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode frame: %w", err)
	}
	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return nil, fmt.Errorf("phash: %w", err)
	}
	return hash, nil
}
