package video

const (
	StatusPendingUpload    = "pending_upload"
	StatusUploaded         = "uploaded"
	StatusExtractionQueued = "extraction_queued"
	StatusExtracting       = "extracting"
	StatusFramesReady      = "frames_ready"
	StatusExtractionFailed = "extraction_failed"
	StatusOCRProcessing    = "ocr_processing"
	StatusOCRReady         = "ocr_ready"
	StatusOCRFailed        = "ocr_failed"
	StatusClassifying      = "classifying"
	StatusClassified       = "classified"
	StatusClassifyFailed   = "classify_failed"
	StatusUploadExpired    = "upload_expired"
)

func ExtractionAlreadyDone(status string) bool {
	switch status {
	case StatusFramesReady, StatusOCRProcessing, StatusOCRReady, StatusOCRFailed,
		StatusClassifying, StatusClassified, StatusClassifyFailed:
		return true
	default:
		return false
	}
}
