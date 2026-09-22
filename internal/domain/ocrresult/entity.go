package ocrresult

import (
	"github.com/google/uuid"
)

const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusEmpty   = "empty"
)

type Result struct {
	ID          uuid.UUID
	FrameID     uuid.UUID
	Text        string
	Confidence  float64
	Status      string
	ErrorReason string
}

func NewResult(frameID uuid.UUID, text string, confidence float64, status, errorReason string) *Result {
	return &Result{
		ID:          uuid.New(),
		FrameID:     frameID,
		Text:        text,
		Confidence:  confidence,
		Status:      status,
		ErrorReason: errorReason,
	}
}

func (r *Result) IsFinal() bool {
	return r.Status == StatusSuccess || r.Status == StatusEmpty
}
