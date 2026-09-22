package video

import (
	"fmt"
	"mime"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	OriginalObjectName  = "original.mp4"
	ThumbnailObjectName = "thumbnail.jpg"
)

var videoExtensions = map[string]struct{}{
	"mp4":  {},
	"mov":  {},
	"avi":  {},
	"mkv":  {},
	"m4v":  {},
	"mpeg": {},
	"mpg":  {},
	"wmv":  {},
	"asf":  {},
	"flv":  {},
	"webm": {},
	"ogg":  {},
	"ogv":  {},
}

func NewStorageKey(videoID uuid.UUID) string {
	return "videos/" + videoID.String() + "/" + OriginalObjectName
}

func NewFrameStorageKey(videoID uuid.UUID, index int) string {
	return fmt.Sprintf("frames/%s/frame_%06d.png", videoID.String(), index)
}

func NewThumbnailStorageKey(videoID uuid.UUID) string {
	return "videos/" + videoID.String() + "/" + ThumbnailObjectName
}

func IsFrameObjectKey(key string) bool {
	return strings.HasPrefix(strings.TrimPrefix(key, "/"), "frames/")
}

func IsThumbnailObjectKey(key string) bool {
	key = strings.TrimPrefix(key, "/")
	return strings.HasPrefix(key, "videos/") && strings.HasSuffix(key, "/"+ThumbnailObjectName)
}

func DecodeObjectKey(key string) (string, error) {
	decoded, err := url.QueryUnescape(key)
	if err != nil {
		return "", fmt.Errorf("invalid object key: %w", err)
	}
	return decoded, nil
}

func VideoIDFromStorageKey(encodedKey string) (uuid.UUID, error) {
	key, err := DecodeObjectKey(encodedKey)
	if err != nil {
		return uuid.Nil, err
	}

	parts := strings.Split(strings.TrimPrefix(key, "/"), "/")
	if len(parts) != 3 || parts[0] != "videos" || parts[2] != OriginalObjectName {
		return uuid.Nil, fmt.Errorf("invalid video storage key: %q", key)
	}

	return uuid.Parse(parts[1])
}

func FileExtension(filename string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
}

func IsVideoFilename(filename string) bool {
	_, ok := videoExtensions[FileExtension(filename)]
	return ok
}

func IsVideoContentType(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(contentType), "video/")
}

func ContentTypeFromFilename(filename, fallback string) string {
	if fallback != "" {
		return fallback
	}
	if contentType := mime.TypeByExtension(filepath.Ext(filename)); contentType != "" {
		return contentType
	}
	return "video/mp4"
}

func SanitizeFilename(filename string) string {
	return filepath.Base(filename)
}
