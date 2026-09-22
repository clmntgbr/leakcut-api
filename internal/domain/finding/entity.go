package finding

import "github.com/google/uuid"

const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"
)

type Category struct {
	Name        string  `json:"name"`
	Probability float64 `json:"probability"`
}

type Finding struct {
	ID           uuid.UUID
	FrameID      uuid.UUID
	Confidential bool
	Probability  float64
	Categories   []Category
	Status       string
	ErrorReason  string
}

func NewFinding(frameID uuid.UUID, confidential bool, probability float64, categories []Category, status, errorReason string) *Finding {
	if categories == nil {
		categories = []Category{}
	}
	return &Finding{
		ID:           uuid.New(),
		FrameID:      frameID,
		Confidential: confidential,
		Probability:  probability,
		Categories:   categories,
		Status:       status,
		ErrorReason:  errorReason,
	}
}

func (f *Finding) IsFinal() bool {
	return f.Status == StatusSuccess || f.Status == StatusSkipped
}
