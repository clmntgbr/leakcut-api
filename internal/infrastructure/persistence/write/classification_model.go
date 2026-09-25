package write

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	domainclassification "go-api/internal/domain/classification"

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

type ClassificationModel struct {
	ID           uuid.UUID      `gorm:"column:id;primaryKey"`
	FrameID      uuid.UUID      `gorm:"column:frame_id"`
	Confidential bool           `gorm:"column:confidential"`
	Probability  float64        `gorm:"column:probability"`
	Categories   categoriesJSON `gorm:"column:categories;type:jsonb"`
	Status       string         `gorm:"column:status"`
	ErrorReason  string         `gorm:"column:error_reason"`
}

func (ClassificationModel) TableName() string {
	return "classifications"
}

func classificationModelFromDomain(c *domainclassification.Classification) *ClassificationModel {
	categories := c.Categories
	if categories == nil {
		categories = []domainclassification.Category{}
	}
	raw, err := json.Marshal(categories)
	if err != nil {
		raw = []byte("[]")
	}
	return &ClassificationModel{
		ID:           c.ID,
		FrameID:      c.FrameID,
		Confidential: c.Confidential,
		Probability:  c.Probability,
		Categories:   categoriesJSON(raw),
		Status:       c.Status,
		ErrorReason:  c.ErrorReason,
	}
}

func classificationDomainFromModel(m *ClassificationModel) *domainclassification.Classification {
	categories := []domainclassification.Category{}
	if m.Categories != "" {
		_ = json.Unmarshal([]byte(m.Categories), &categories)
	}
	return &domainclassification.Classification{
		ID:           m.ID,
		FrameID:      m.FrameID,
		Confidential: m.Confidential,
		Probability:  m.Probability,
		Categories:   categories,
		Status:       m.Status,
		ErrorReason:  m.ErrorReason,
	}
}
