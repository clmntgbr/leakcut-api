package dto

type ObjectCreatedEvent struct {
	Records []ObjectCreatedRecord `json:"Records" validate:"required,min=1,dive"`
}

type ObjectCreatedRecord struct {
	EventName string   `json:"eventName" validate:"required"`
	S3        S3Entity `json:"s3" validate:"required"`
}

type S3Entity struct {
	Bucket S3Bucket `json:"bucket" validate:"required"`
	Object S3Object `json:"object" validate:"required"`
}

type S3Bucket struct {
	Name string `json:"name" validate:"required"`
}

type S3Object struct {
	Key         string `json:"key" validate:"required,min=1,max=1024"`
	Size        int64  `json:"size" validate:"gte=0"`
	ContentType string `json:"contentType" validate:"omitempty,max=127"`
}
