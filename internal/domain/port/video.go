package port

import "context"

type FrameSelectionParams struct {
	AnalysisFPS        float64
	DiffThreshold      float64
	MaxIntervalSeconds int
	MaxWidthPx         int
}

type ExtractedFrame struct {
	Index           int
	TimestampMs     int64
	Data            []byte
	SelectionReason string
	DiffScore       float64
}

type FrameSink func(frame ExtractedFrame) error

type FrameExtractor interface {
	ExtractFrames(ctx context.Context, videoPath string, params FrameSelectionParams, emit FrameSink) error
	ExtractThumbnail(ctx context.Context, videoPath string) ([]byte, error)
}
