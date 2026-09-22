package video

import "time"

const (
	EventTypeVideoCreated               = "video.created.v1"
	EventTypeVideoIngestRequested       = "video.ingest_requested.v1"
	EventTypeVideoUploaded              = "video.uploaded.v1"
	EventTypeVideoExtracting            = "video.extracting.v1"
	EventTypeVideoFramesExtracted       = "video.frames_extracted.v1"
	EventTypeVideoFrameExtractionFailed = "video.frame_extraction_failed.v1"
	EventTypeVideoOCRProcessing         = "video.ocr_processing.v1"
	EventTypeVideoOCRBatchCompleted     = "video.ocr_batch_completed.v1"
	EventTypeVideoFramesOCRCompleted    = "video.frames_ocr_completed.v1"
	EventTypeVideoOCRFailed             = "video.ocr_failed.v1"
	EventTypeVideoClassifying           = "video.classifying.v1"
	EventTypeVideoFramesClassified      = "video.frames_classified.v1"
	EventTypeVideoClassifyFailed        = "video.classify_failed.v1"
	EventTypeVideoUploadExpired         = "video.upload_expired.v1"
)

type VideoCreated struct {
	ID         string    `json:"eventId"`
	VideoID    string    `json:"videoId"`
	UserID     string    `json:"userId,omitempty"`
	Filename   string    `json:"filename"`
	StorageKey string    `json:"storageKey"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e VideoCreated) EventID() string       { return e.ID }
func (e VideoCreated) EventType() string     { return EventTypeVideoCreated }
func (e VideoCreated) AggregateID() string   { return e.VideoID }
func (e VideoCreated) OccurredAt() time.Time { return e.Timestamp }

type VideoIngestRequested struct {
	ID         string    `json:"eventId"`
	VideoID    string    `json:"videoId"`
	RemoteURL  string    `json:"remoteUrl"`
	StorageKey string    `json:"storageKey"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e VideoIngestRequested) EventID() string       { return e.ID }
func (e VideoIngestRequested) EventType() string     { return EventTypeVideoIngestRequested }
func (e VideoIngestRequested) AggregateID() string   { return e.VideoID }
func (e VideoIngestRequested) OccurredAt() time.Time { return e.Timestamp }

type VideoUploaded struct {
	ID          string    `json:"eventId"`
	VideoID     string    `json:"videoId"`
	UserID      string    `json:"userId,omitempty"`
	StorageKey  string    `json:"storageKey"`
	ContentType string    `json:"contentType"`
	SizeBytes   int64     `json:"sizeBytes"`
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e VideoUploaded) EventID() string       { return e.ID }
func (e VideoUploaded) EventType() string     { return EventTypeVideoUploaded }
func (e VideoUploaded) AggregateID() string   { return e.VideoID }
func (e VideoUploaded) OccurredAt() time.Time { return e.Timestamp }

type VideoExtracting struct {
	ID        string    `json:"eventId"`
	VideoID   string    `json:"videoId"`
	UserID    string    `json:"userId,omitempty"`
	JobID     string    `json:"jobId"`
	JobType   string    `json:"jobType,omitempty"`
	Status    string    `json:"status"`
	JobStatus string    `json:"jobStatus"`
	Timestamp time.Time `json:"timestamp"`
}

func (e VideoExtracting) EventID() string       { return e.ID }
func (e VideoExtracting) EventType() string     { return EventTypeVideoExtracting }
func (e VideoExtracting) AggregateID() string   { return e.VideoID }
func (e VideoExtracting) OccurredAt() time.Time { return e.Timestamp }

type ExtractedFramePayload struct {
	ID              string  `json:"id,omitempty"`
	Index           int     `json:"index"`
	TimestampMs     int64   `json:"timestampMs"`
	StorageKey      string  `json:"storageKey"`
	SelectionReason string  `json:"selectionReason"`
	DiffScore       float64 `json:"diffScore"`
}

type VideoFramesExtracted struct {
	ID         string                  `json:"eventId"`
	VideoID    string                  `json:"videoId"`
	UserID     string                  `json:"userId,omitempty"`
	JobID      string                  `json:"jobId"`
	JobType    string                  `json:"jobType,omitempty"`
	Status     string                  `json:"status"`
	JobStatus  string                  `json:"jobStatus"`
	FrameCount int                     `json:"frameCount"`
	Frames     []ExtractedFramePayload `json:"frames"`
	Timestamp  time.Time               `json:"timestamp"`
}

func (e VideoFramesExtracted) EventID() string       { return e.ID }
func (e VideoFramesExtracted) EventType() string     { return EventTypeVideoFramesExtracted }
func (e VideoFramesExtracted) AggregateID() string   { return e.VideoID }
func (e VideoFramesExtracted) OccurredAt() time.Time { return e.Timestamp }

type VideoFrameExtractionFailed struct {
	ID        string    `json:"eventId"`
	VideoID   string    `json:"videoId"`
	UserID    string    `json:"userId,omitempty"`
	JobID     string    `json:"jobId"`
	JobType   string    `json:"jobType,omitempty"`
	Status    string    `json:"status"`
	JobStatus string    `json:"jobStatus"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

func (e VideoFrameExtractionFailed) EventID() string       { return e.ID }
func (e VideoFrameExtractionFailed) EventType() string     { return EventTypeVideoFrameExtractionFailed }
func (e VideoFrameExtractionFailed) AggregateID() string   { return e.VideoID }
func (e VideoFrameExtractionFailed) OccurredAt() time.Time { return e.Timestamp }

type VideoOCRProcessing struct {
	ID                 string    `json:"eventId"`
	VideoID            string    `json:"videoId"`
	UserID             string    `json:"userId,omitempty"`
	JobID              string    `json:"jobId"`
	JobType            string    `json:"jobType,omitempty"`
	Status             string    `json:"status"`
	JobStatus          string    `json:"jobStatus"`
	ExpectedFrameCount int       `json:"expectedFrameCount"`
	OCRCompletedCount  int       `json:"ocrCompletedCount"`
	Timestamp          time.Time `json:"timestamp"`
}

func (e VideoOCRProcessing) EventID() string       { return e.ID }
func (e VideoOCRProcessing) EventType() string     { return EventTypeVideoOCRProcessing }
func (e VideoOCRProcessing) AggregateID() string   { return e.VideoID }
func (e VideoOCRProcessing) OccurredAt() time.Time { return e.Timestamp }

type VideoOCRBatchCompleted struct {
	ID        string                  `json:"eventId"`
	VideoID   string                  `json:"videoId"`
	Results   []OCRFrameResultPayload `json:"results"`
	Timestamp time.Time               `json:"timestamp"`
}

func (e VideoOCRBatchCompleted) EventID() string       { return e.ID }
func (e VideoOCRBatchCompleted) EventType() string     { return EventTypeVideoOCRBatchCompleted }
func (e VideoOCRBatchCompleted) AggregateID() string   { return e.VideoID }
func (e VideoOCRBatchCompleted) OccurredAt() time.Time { return e.Timestamp }

type OCRFrameResultPayload struct {
	FrameID     string  `json:"frameId"`
	FrameIndex  int     `json:"frameIndex"`
	TimestampMs int64   `json:"timestampMs"`
	Text        string  `json:"text"`
	Confidence  float64 `json:"confidence"`
	Status      string  `json:"status"`
}

type VideoFramesOCRCompleted struct {
	ID                 string                  `json:"eventId"`
	VideoID            string                  `json:"videoId"`
	UserID             string                  `json:"userId,omitempty"`
	JobID              string                  `json:"jobId"`
	JobType            string                  `json:"jobType,omitempty"`
	Status             string                  `json:"status"`
	JobStatus          string                  `json:"jobStatus"`
	ExpectedFrameCount int                     `json:"expectedFrameCount"`
	OCRCompletedCount  int                     `json:"ocrCompletedCount"`
	Results            []OCRFrameResultPayload `json:"results"`
	Timestamp          time.Time               `json:"timestamp"`
}

func (e VideoFramesOCRCompleted) EventID() string       { return e.ID }
func (e VideoFramesOCRCompleted) EventType() string     { return EventTypeVideoFramesOCRCompleted }
func (e VideoFramesOCRCompleted) AggregateID() string   { return e.VideoID }
func (e VideoFramesOCRCompleted) OccurredAt() time.Time { return e.Timestamp }

type VideoOCRFailed struct {
	ID                 string    `json:"eventId"`
	VideoID            string    `json:"videoId"`
	UserID             string    `json:"userId,omitempty"`
	JobID              string    `json:"jobId"`
	JobType            string    `json:"jobType,omitempty"`
	Status             string    `json:"status"`
	JobStatus          string    `json:"jobStatus"`
	Reason             string    `json:"reason"`
	ExpectedFrameCount int       `json:"expectedFrameCount"`
	OCRCompletedCount  int       `json:"ocrCompletedCount"`
	Timestamp          time.Time `json:"timestamp"`
}

func (e VideoOCRFailed) EventID() string       { return e.ID }
func (e VideoOCRFailed) EventType() string     { return EventTypeVideoOCRFailed }
func (e VideoOCRFailed) AggregateID() string   { return e.VideoID }
func (e VideoOCRFailed) OccurredAt() time.Time { return e.Timestamp }

type VideoClassifying struct {
	ID                 string    `json:"eventId"`
	VideoID            string    `json:"videoId"`
	UserID             string    `json:"userId,omitempty"`
	JobID              string    `json:"jobId"`
	JobType            string    `json:"jobType,omitempty"`
	Status             string    `json:"status"`
	JobStatus          string    `json:"jobStatus"`
	ExpectedFrameCount int       `json:"expectedFrameCount"`
	Timestamp          time.Time `json:"timestamp"`
}

func (e VideoClassifying) EventID() string       { return e.ID }
func (e VideoClassifying) EventType() string     { return EventTypeVideoClassifying }
func (e VideoClassifying) AggregateID() string   { return e.VideoID }
func (e VideoClassifying) OccurredAt() time.Time { return e.Timestamp }

type FindingCategoryPayload struct {
	Name        string  `json:"name"`
	Probability float64 `json:"probability"`
}

type FrameFindingPayload struct {
	FrameID      string                   `json:"frameId"`
	Confidential bool                     `json:"confidential"`
	Probability  float64                  `json:"probability"`
	Categories   []FindingCategoryPayload `json:"categories"`
	Status       string                   `json:"status"`
}

type VideoFramesClassified struct {
	ID                 string                `json:"eventId"`
	VideoID            string                `json:"videoId"`
	UserID             string                `json:"userId,omitempty"`
	JobID              string                `json:"jobId"`
	JobType            string                `json:"jobType,omitempty"`
	Status             string                `json:"status"`
	JobStatus          string                `json:"jobStatus"`
	ExpectedFrameCount int                   `json:"expectedFrameCount"`
	Findings           []FrameFindingPayload `json:"findings"`
	Timestamp          time.Time             `json:"timestamp"`
}

func (e VideoFramesClassified) EventID() string       { return e.ID }
func (e VideoFramesClassified) EventType() string     { return EventTypeVideoFramesClassified }
func (e VideoFramesClassified) AggregateID() string   { return e.VideoID }
func (e VideoFramesClassified) OccurredAt() time.Time { return e.Timestamp }

type VideoClassifyFailed struct {
	ID                 string    `json:"eventId"`
	VideoID            string    `json:"videoId"`
	UserID             string    `json:"userId,omitempty"`
	JobID              string    `json:"jobId"`
	JobType            string    `json:"jobType,omitempty"`
	Status             string    `json:"status"`
	JobStatus          string    `json:"jobStatus"`
	Reason             string    `json:"reason"`
	ExpectedFrameCount int       `json:"expectedFrameCount"`
	Timestamp          time.Time `json:"timestamp"`
}

func (e VideoClassifyFailed) EventID() string       { return e.ID }
func (e VideoClassifyFailed) EventType() string     { return EventTypeVideoClassifyFailed }
func (e VideoClassifyFailed) AggregateID() string   { return e.VideoID }
func (e VideoClassifyFailed) OccurredAt() time.Time { return e.Timestamp }

type VideoUploadExpired struct {
	ID        string    `json:"eventId"`
	VideoID   string    `json:"videoId"`
	Timestamp time.Time `json:"timestamp"`
}

func (e VideoUploadExpired) EventID() string       { return e.ID }
func (e VideoUploadExpired) EventType() string     { return EventTypeVideoUploadExpired }
func (e VideoUploadExpired) AggregateID() string   { return e.VideoID }
func (e VideoUploadExpired) OccurredAt() time.Time { return e.Timestamp }
