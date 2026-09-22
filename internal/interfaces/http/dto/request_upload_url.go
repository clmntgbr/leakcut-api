package dto

type RequestUploadURLRequest struct {
	Filename    string `json:"filename" validate:"required,min=1,max=255"`
	ContentType string `json:"contentType" validate:"required,max=127"`
	SizeBytes   int64  `json:"sizeBytes" validate:"gte=0"`
}
