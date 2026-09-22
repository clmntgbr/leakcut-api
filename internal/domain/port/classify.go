package port

import "context"

type ClassifyFrame struct {
	FrameID string
	Text    string
}

type ClassifyCategory struct {
	Name        string
	Probability float64
}

type ClassifyItemResult struct {
	FrameID      string
	Confidential bool
	Probability  float64
	Categories   []ClassifyCategory
	Status       string
}

type Classifier interface {
	Classify(ctx context.Context, frames []ClassifyFrame, threshold float64) ([]ClassifyItemResult, error)
}
