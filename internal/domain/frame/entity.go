package frame

import (
	"github.com/google/uuid"
)

const (
	SelectionReasonFixedInterval = "fixed_interval"
	SelectionReasonSceneChange   = "scene_change"
)

type Frame struct {
	ID              uuid.UUID
	VideoID         uuid.UUID
	SegmentIndex    int
	Index           int
	TimestampMs     int64
	StorageKey      string
	SelectionReason string
	PHashDistance   int
	Retained        bool
	PruneReason     string
}

func NewFrame(
	videoID uuid.UUID,
	segmentIndex, index int,
	timestampMs int64,
	storageKey, selectionReason string,
	phashDistance int,
) *Frame {
	return &Frame{
		ID:              uuid.New(),
		VideoID:         videoID,
		SegmentIndex:    segmentIndex,
		Index:           index,
		TimestampMs:     timestampMs,
		StorageKey:      storageKey,
		SelectionReason: selectionReason,
		PHashDistance:   phashDistance,
		Retained:        true,
	}
}

func (f *Frame) ApplyRetention(retained bool, pruneReason string) {
	f.Retained = retained
	f.PruneReason = pruneReason
}
