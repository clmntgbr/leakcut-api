package write

import (
	domainocr "go-api/internal/domain/ocrresult"

	"github.com/google/uuid"
)

type OCRResultModel struct {
	ID          uuid.UUID `gorm:"column:id;primaryKey"`
	FrameID     uuid.UUID `gorm:"column:frame_id"`
	Text        string    `gorm:"column:text"`
	Confidence  float64   `gorm:"column:confidence"`
	Status      string    `gorm:"column:status"`
	ErrorReason string    `gorm:"column:error_reason"`
}

func (OCRResultModel) TableName() string {
	return "ocr_results"
}

func ocrResultModelFromDomain(r *domainocr.Result) *OCRResultModel {
	return &OCRResultModel{
		ID:          r.ID,
		FrameID:     r.FrameID,
		Text:        r.Text,
		Confidence:  r.Confidence,
		Status:      r.Status,
		ErrorReason: r.ErrorReason,
	}
}

func ocrResultDomainFromModel(m *OCRResultModel) *domainocr.Result {
	return &domainocr.Result{
		ID:          m.ID,
		FrameID:     m.FrameID,
		Text:        m.Text,
		Confidence:  m.Confidence,
		Status:      m.Status,
		ErrorReason: m.ErrorReason,
	}
}
