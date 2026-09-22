package dto

type VideoIngestRequest struct {
	VideoURL    string `json:"videoUrl" validate:"required,url,max=2048"`
	Filename    string `json:"filename" validate:"omitempty,max=255"`
	ContentType string `json:"contentType" validate:"omitempty,max=127"`
	SizeBytes   int64  `json:"sizeBytes" validate:"gte=0"`
}
