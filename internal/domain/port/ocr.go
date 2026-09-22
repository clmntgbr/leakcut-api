package port

import "context"

type OCRImage struct {
	FrameID string
	Data    []byte
}

type OCRItemResult struct {
	FrameID    string
	Text       string
	Confidence float64
	Status     string
}

type OCREngine interface {
	Recognize(ctx context.Context, images []OCRImage, lang string) ([]OCRItemResult, error)
}
