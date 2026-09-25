package write

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	domainocr "go-api/internal/domain/ocr"

	"github.com/google/uuid"
)

type ocrLinesJSON string

func (c ocrLinesJSON) Value() (driver.Value, error) {
	if c == "" {
		return "[]", nil
	}
	return string(c), nil
}

func (c *ocrLinesJSON) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		*c = ocrLinesJSON(v)
	case string:
		*c = ocrLinesJSON(v)
	case nil:
		*c = "[]"
	default:
		return fmt.Errorf("unsupported ocr lines type %T", value)
	}
	return nil
}

type OCRModel struct {
	ID          uuid.UUID    `gorm:"column:id;primaryKey"`
	FrameID     uuid.UUID    `gorm:"column:frame_id"`
	Text        string       `gorm:"column:text"`
	Confidence  float64      `gorm:"column:confidence"`
	Status      string       `gorm:"column:status"`
	ErrorReason string       `gorm:"column:error_reason"`
	Lines       ocrLinesJSON `gorm:"column:lines;type:jsonb"`
}

func (OCRModel) TableName() string {
	return "ocrs"
}

func ocrModelFromDomain(r *domainocr.Result) *OCRModel {
	lines := r.Lines
	if lines == nil {
		lines = []domainocr.Line{}
	}
	raw, err := json.Marshal(lines)
	if err != nil {
		raw = []byte("[]")
	}
	return &OCRModel{
		ID:          r.ID,
		FrameID:     r.FrameID,
		Text:        r.Text,
		Confidence:  r.Confidence,
		Status:      r.Status,
		ErrorReason: r.ErrorReason,
		Lines:       ocrLinesJSON(raw),
	}
}

func ocrDomainFromModel(m *OCRModel) *domainocr.Result {
	lines := []domainocr.Line{}
	if m.Lines != "" {
		_ = json.Unmarshal([]byte(m.Lines), &lines)
	}
	return &domainocr.Result{
		ID:          m.ID,
		FrameID:     m.FrameID,
		Text:        m.Text,
		Confidence:  m.Confidence,
		Status:      m.Status,
		ErrorReason: m.ErrorReason,
		Lines:       lines,
	}
}
