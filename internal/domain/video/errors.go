package video

import "errors"

var (
	ErrInvalidFilename    = errors.New("invalid video filename")
	ErrUnsupportedType    = errors.New("unsupported video type")
	ErrInvalidTransition  = errors.New("invalid video status transition")
	ErrVideoNotFound      = errors.New("video not found")
	ErrRemoteURLForbidden = errors.New("remote video url is not allowed")
	ErrVideoTooLarge      = errors.New("video exceeds maximum size")
)
