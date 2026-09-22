package write

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	domainfinding "go-api/internal/domain/finding"

	"github.com/google/uuid"
)

type categoriesJSON string

func (c categoriesJSON) Value() (driver.Value, error) {
	if c == "" {
		return "[]", nil
	}
	return string(c), nil
}

func (c *categoriesJSON) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		*c = categoriesJSON(v)
	case string:
		*c = categoriesJSON(v)
	case nil:
		*c = "[]"
	default:
		return fmt.Errorf("unsupported categories type %T", value)
	}
	return nil
}

type FindingModel struct {
	ID           uuid.UUID      `gorm:"column:id;primaryKey"`
	FrameID      uuid.UUID      `gorm:"column:frame_id"`
	Confidential bool           `gorm:"column:confidential"`
	Probability  float64        `gorm:"column:probability"`
	Categories   categoriesJSON `gorm:"column:categories;type:jsonb"`
	Status       string         `gorm:"column:status"`
	ErrorReason  string         `gorm:"column:error_reason"`
}

func (FindingModel) TableName() string {
	return "frame_findings"
}

func findingModelFromDomain(f *domainfinding.Finding) *FindingModel {
	categories := f.Categories
	if categories == nil {
		categories = []domainfinding.Category{}
	}
	raw, err := json.Marshal(categories)
	if err != nil {
		raw = []byte("[]")
	}
	return &FindingModel{
		ID:           f.ID,
		FrameID:      f.FrameID,
		Confidential: f.Confidential,
		Probability:  f.Probability,
		Categories:   categoriesJSON(raw),
		Status:       f.Status,
		ErrorReason:  f.ErrorReason,
	}
}

func findingDomainFromModel(m *FindingModel) *domainfinding.Finding {
	categories := []domainfinding.Category{}
	if m.Categories != "" {
		_ = json.Unmarshal([]byte(m.Categories), &categories)
	}
	return &domainfinding.Finding{
		ID:           m.ID,
		FrameID:      m.FrameID,
		Confidential: m.Confidential,
		Probability:  m.Probability,
		Categories:   categories,
		Status:       m.Status,
		ErrorReason:  m.ErrorReason,
	}
}
