package classification

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

type Classification struct {
	ID           uuid.UUID
	FrameID      uuid.UUID
	Confidential bool
	Probability  float64
	Categories   []Category
	Status       string
	ErrorReason  string
}

func New(
	frameID uuid.UUID,
	confidential bool,
	probability float64,
	categories []Category,
	status, errorReason string,
) *Classification {
	if categories == nil {
		categories = []Category{}
	}
	return &Classification{
		ID:           uuid.New(),
		FrameID:      frameID,
		Confidential: confidential,
		Probability:  probability,
		Categories:   categories,
		Status:       status,
		ErrorReason:  errorReason,
	}
}

func (c *Classification) IsFinal() bool {
	return c.Status == StatusSuccess || c.Status == StatusSkipped
}
