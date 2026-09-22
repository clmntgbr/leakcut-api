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
	ScanJobID       uuid.UUID
	Index           int
	TimestampMs     int64
	StorageKey      string
	SelectionReason string
	DiffScore       float64
}

func NewFrame(
	scanJobID uuid.UUID,
	index int,
	timestampMs int64,
	storageKey, selectionReason string,
	diffScore float64,
) *Frame {
	return &Frame{
		ID:              uuid.New(),
		ScanJobID:       scanJobID,
		Index:           index,
		TimestampMs:     timestampMs,
		StorageKey:      storageKey,
		SelectionReason: selectionReason,
		DiffScore:       diffScore,
	}
}
