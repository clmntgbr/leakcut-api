package video

import "time"

const (
	EventTypeVideoCreated               = "video.created.v1"
	EventTypeVideoIngestRequested       = "video.ingest_requested.v1"
	EventTypeVideoUploaded              = "video.uploaded.v1"
	EventTypeVideoFramesExtracted       = "video.frames_extracted.v1"
	EventTypeVideoFrameExtractionFailed = "video.frame_extraction_failed.v1"
	EventTypeVideoUploadExpired         = "video.upload_expired.v1"
)

type VideoCreated struct {
	ID         string    `json:"eventId"`
	VideoID    string    `json:"videoId"`
	Filename   string    `json:"filename"`
	StorageKey string    `json:"storageKey"`
	Source     string    `json:"source"`
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
	StorageKey  string    `json:"storageKey"`
	ContentType string    `json:"contentType"`
	SizeBytes   int64     `json:"sizeBytes"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e VideoUploaded) EventID() string       { return e.ID }
func (e VideoUploaded) EventType() string     { return EventTypeVideoUploaded }
func (e VideoUploaded) AggregateID() string   { return e.VideoID }
func (e VideoUploaded) OccurredAt() time.Time { return e.Timestamp }

type ExtractedFramePayload struct {
	Index           int     `json:"index"`
	TimestampMs     int64   `json:"timestampMs"`
	StorageKey      string  `json:"storageKey"`
	SelectionReason string  `json:"selectionReason"`
	DiffScore       float64 `json:"diffScore"`
}

type VideoFramesExtracted struct {
	ID         string                  `json:"eventId"`
	VideoID    string                  `json:"videoId"`
	ScanJobID  string                  `json:"scanJobId"`
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
	ScanJobID string    `json:"scanJobId"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

func (e VideoFrameExtractionFailed) EventID() string       { return e.ID }
func (e VideoFrameExtractionFailed) EventType() string     { return EventTypeVideoFrameExtractionFailed }
func (e VideoFrameExtractionFailed) AggregateID() string   { return e.VideoID }
func (e VideoFrameExtractionFailed) OccurredAt() time.Time { return e.Timestamp }

type VideoUploadExpired struct {
	ID        string    `json:"eventId"`
	VideoID   string    `json:"videoId"`
	Timestamp time.Time `json:"timestamp"`
}

func (e VideoUploadExpired) EventID() string       { return e.ID }
func (e VideoUploadExpired) EventType() string     { return EventTypeVideoUploadExpired }
func (e VideoUploadExpired) AggregateID() string   { return e.VideoID }
func (e VideoUploadExpired) OccurredAt() time.Time { return e.Timestamp }
