package ocrresult

import (
	"github.com/google/uuid"
)

const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusEmpty   = "empty"
)

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Line struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Box        []Point `json:"box"`
}

type Result struct {
	ID          uuid.UUID
	FrameID     uuid.UUID
	Text        string
	Confidence  float64
	Status      string
	ErrorReason string
	Lines       []Line
}

func NewResult(frameID uuid.UUID, text string, confidence float64, status, errorReason string, lines []Line) *Result {
	if lines == nil {
		lines = []Line{}
	}
	return &Result{
		ID:          uuid.New(),
		FrameID:     frameID,
		Text:        text,
		Confidence:  confidence,
		Status:      status,
		ErrorReason: errorReason,
		Lines:       lines,
	}
}

func (r *Result) IsFinal() bool {
	return r.Status == StatusSuccess || r.Status == StatusEmpty
}
