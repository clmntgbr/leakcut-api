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
	Index           int
	TimestampMs     int64
	StorageKey      string
	SelectionReason string
	PHashDistance   int
}

func NewFrame(
	videoID uuid.UUID,
	index int,
	timestampMs int64,
	storageKey, selectionReason string,
	phashDistance int,
) *Frame {
	return &Frame{
		ID:              uuid.New(),
		VideoID:         videoID,
		Index:           index,
		TimestampMs:     timestampMs,
		StorageKey:      storageKey,
		SelectionReason: selectionReason,
		PHashDistance:   phashDistance,
	}
}
