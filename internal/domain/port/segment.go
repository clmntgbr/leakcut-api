package port

import "context"

// SegmentFile is one output file from a stream-copy split.
type SegmentFile struct {
	Index      int
	Path       string
	OffsetMs   int64
	DurationMs int64
}

type SegmentSplitter interface {
	ProbeDurationMs(ctx context.Context, videoPath string) (int64, error)
	Split(ctx context.Context, videoPath string, segmentDurationSeconds float64, outputDir string) ([]SegmentFile, error)
}
