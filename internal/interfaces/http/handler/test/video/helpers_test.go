package videotest

import (
	"context"
	"errors"
	"time"

	cmdvideo "go-api/internal/application/command/video"
	queryvideo "go-api/internal/application/query/video"
	domainvideo "go-api/internal/domain/video"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"
)

type mockRequestUploadURLHandler struct {
	called bool
	cmd    cmdvideo.RequestUploadURLCommand
	result *cmdvideo.RequestUploadURLResult
	err    error
}

func (m *mockRequestUploadURLHandler) Handle(_ context.Context, cmd cmdvideo.RequestUploadURLCommand) (*cmdvideo.RequestUploadURLResult, error) {
	m.called = true
	m.cmd = cmd
	return m.result, m.err
}

type mockGetVideoByIDHandler struct {
	called bool
	query  queryvideo.GetVideoByIDQuery
	view   *domainvideo.VideoView
	err    error
}

func (m *mockGetVideoByIDHandler) Handle(_ context.Context, q queryvideo.GetVideoByIDQuery) (*domainvideo.VideoView, error) {
	m.called = true
	m.query = q
	return m.view, m.err
}

type mockListVideosHandler struct {
	called bool
	query  queryvideo.ListVideosQuery
	views  []domainvideo.VideoListView
	total  int64
	err    error
}

func (m *mockListVideosHandler) Handle(_ context.Context, q queryvideo.ListVideosQuery) ([]domainvideo.VideoListView, int64, error) {
	m.called = true
	m.query = q
	return m.views, m.total, m.err
}

func newVideoHandler(upload *mockRequestUploadURLHandler, get *mockGetVideoByIDHandler, list ...*mockListVideosHandler) *handler.VideoHandler {
	if upload == nil {
		upload = &mockRequestUploadURLHandler{}
	}
	if get == nil {
		get = &mockGetVideoByIDHandler{}
	}
	listHandler := &mockListVideosHandler{}
	if len(list) > 0 && list[0] != nil {
		listHandler = list[0]
	}
	return handler.NewVideoHandler(upload, get, listHandler)
}

func validUploadBody() map[string]any {
	return map[string]any{
		"filename":    "demo.mp4",
		"contentType": "video/mp4",
		"sizeBytes":   1024,
	}
}

func sampleUploadResult() *cmdvideo.RequestUploadURLResult {
	return &cmdvideo.RequestUploadURLResult{
		VideoID:   testutil.TestVideoID.String(),
		UploadURL: "http://localhost:9000/media/videos/" + testutil.TestVideoID.String() + "/original.mp4",
		ExpiresAt: time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC),
	}
}

func sampleVideoView() *domainvideo.VideoView {
	jobID := testutil.TestJobID
	return &domainvideo.VideoView{
		ID:               testutil.TestVideoID,
		OriginalFilename: "demo.mp4",
		StorageKey:       domainvideo.NewStorageKey(testutil.TestVideoID),
		ThumbnailKey:     domainvideo.NewThumbnailStorageKey(testutil.TestVideoID),
		ThumbnailURL:     "http://localhost:9000/media/videos/" + testutil.TestVideoID.String() + "/thumbnail.jpg",
		VideoURL:         "http://localhost:9000/media/videos/" + testutil.TestVideoID.String() + "/original.mp4",
		SizeBytes:        1024,
		ContentType:      "video/mp4",
		Status:           domainvideo.StatusPendingUpload,
		CreatedAt:        time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC),
		JobID:            &jobID,
		JobStatus:        "pending",
		FrameCount:       0,
		Jobs: []domainvideo.JobView{{
			ID:     jobID,
			Type:   "extract_frames",
			Status: "pending",
		}},
		Frames: []domainvideo.VideoFrameDetailView{},
	}
}

func sampleVideoListView() domainvideo.VideoListView {
	return domainvideo.VideoListView{
		ID:               testutil.TestVideoID,
		OriginalFilename: "demo.mp4",
		ThumbnailKey:     domainvideo.NewThumbnailStorageKey(testutil.TestVideoID),
		ThumbnailURL:     "http://localhost:9000/media/videos/" + testutil.TestVideoID.String() + "/thumbnail.jpg",
		Status:           domainvideo.StatusFramesReady,
		CreatedAt:        time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC),
	}
}

var errUnexpected = errors.New("unexpected handler call")
