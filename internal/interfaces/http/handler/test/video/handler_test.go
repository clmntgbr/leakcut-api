package videotest

import (
	"net/http"
	"testing"

	domainvideo "go-api/internal/domain/video"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/testutil"

	"github.com/google/uuid"
)

func TestVideoHandler_RequestUploadURL_Success(t *testing.T) {
	upload := &mockRequestUploadURLHandler{result: sampleUploadResult()}
	h := newVideoHandler(upload, nil)

	app := testutil.NewTestApp()
	app.Post("/videos/upload-url", testutil.WithUserWithoutProject(testutil.TestUserID), h.RequestUploadURL)

	req, err := testutil.JSONRequest(http.MethodPost, "/videos/upload-url", validUploadBody())
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusCreated)
	}
	if !upload.called {
		t.Fatal("expected upload handler to be called")
	}
	if upload.cmd.UserID != testutil.TestUserID {
		t.Fatalf("user id: got %s", upload.cmd.UserID)
	}
	if upload.cmd.Filename != "demo.mp4" {
		t.Fatalf("filename: got %q", upload.cmd.Filename)
	}

	var out presenter.RequestUploadURLResponse
	testutil.DecodeJSON(t, resp, &out)
	if out.VideoID != testutil.TestVideoID.String() {
		t.Fatalf("video id: got %s", out.VideoID)
	}
	if out.UploadURL == "" {
		t.Fatal("expected upload url")
	}
}

func TestVideoHandler_RequestUploadURL_Unauthorized(t *testing.T) {
	upload := &mockRequestUploadURLHandler{}
	h := newVideoHandler(upload, nil)

	app := testutil.NewTestApp()
	app.Post("/videos/upload-url", h.RequestUploadURL)

	req, err := testutil.JSONRequest(http.MethodPost, "/videos/upload-url", validUploadBody())
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if upload.called {
		t.Fatal("upload handler must not be called without auth")
	}
}

func TestVideoHandler_RequestUploadURL_InvalidInput(t *testing.T) {
	h := newVideoHandler(nil, nil)

	app := testutil.NewTestApp()
	app.Post("/videos/upload-url", testutil.WithUserWithoutProject(testutil.TestUserID), h.RequestUploadURL)

	req, err := testutil.JSONRequest(http.MethodPost, "/videos/upload-url", map[string]any{"filename": ""})
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestVideoHandler_RequestUploadURL_HandlerError_UnsupportedType(t *testing.T) {
	upload := &mockRequestUploadURLHandler{err: domainvideo.ErrUnsupportedType}
	h := newVideoHandler(upload, nil)

	app := testutil.NewTestApp()
	app.Post("/videos/upload-url", testutil.WithUserWithoutProject(testutil.TestUserID), h.RequestUploadURL)

	req, err := testutil.JSONRequest(http.MethodPost, "/videos/upload-url", validUploadBody())
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusBadRequest)
	}

	body := testutil.DecodeJSONMap(t, resp)
	if body["message"] != "Unsupported video type" {
		t.Fatalf("message: got %v", body["message"])
	}
}

func TestVideoHandler_RequestUploadURL_HandlerError_InvalidFilename(t *testing.T) {
	upload := &mockRequestUploadURLHandler{err: domainvideo.ErrInvalidFilename}
	h := newVideoHandler(upload, nil)

	app := testutil.NewTestApp()
	app.Post("/videos/upload-url", testutil.WithUserWithoutProject(testutil.TestUserID), h.RequestUploadURL)

	req, err := testutil.JSONRequest(http.MethodPost, "/videos/upload-url", validUploadBody())
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestVideoHandler_RequestUploadURL_HandlerError_Internal(t *testing.T) {
	upload := &mockRequestUploadURLHandler{err: errUnexpected}
	h := newVideoHandler(upload, nil)

	app := testutil.NewTestApp()
	app.Post("/videos/upload-url", testutil.WithUserWithoutProject(testutil.TestUserID), h.RequestUploadURL)

	req, err := testutil.JSONRequest(http.MethodPost, "/videos/upload-url", validUploadBody())
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusInternalServerError)
	}

	body := testutil.DecodeJSONMap(t, resp)
	if body["message"] != "Failed to generate upload url" {
		t.Fatalf("message: got %v", body["message"])
	}
}

func TestVideoHandler_GetByID_Success(t *testing.T) {
	get := &mockGetVideoByIDHandler{view: sampleVideoView()}
	h := newVideoHandler(nil, get)

	app := testutil.NewTestApp()
	app.Get("/videos/:id", testutil.WithUserWithoutProject(testutil.TestUserID), h.GetByID)

	req, err := testutil.JSONRequest(http.MethodGet, "/videos/"+testutil.TestVideoID.String(), nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	if get.query.ID != testutil.TestVideoID {
		t.Fatalf("query id: got %s", get.query.ID)
	}
	if get.query.UserID != testutil.TestUserID {
		t.Fatalf("query user id: got %s", get.query.UserID)
	}

	var out presenter.VideoDetailResponse
	testutil.DecodeJSON(t, resp, &out)
	if out.ID != testutil.TestVideoID.String() {
		t.Fatalf("id: got %s", out.ID)
	}
	if out.Status != domainvideo.StatusPendingUpload {
		t.Fatalf("status: got %s", out.Status)
	}
	if out.ThumbnailKey == nil || *out.ThumbnailKey != domainvideo.NewThumbnailStorageKey(testutil.TestVideoID) {
		t.Fatalf("thumbnail key: got %v", out.ThumbnailKey)
	}
	if out.ThumbnailURL == nil || *out.ThumbnailURL == "" {
		t.Fatal("expected thumbnail url")
	}
	if out.JobID == nil || *out.JobID != testutil.TestJobID.String() {
		t.Fatalf("job id: got %v", out.JobID)
	}
	if out.JobStatus == nil || *out.JobStatus != "pending" {
		t.Fatalf("job status: got %v", out.JobStatus)
	}
}

func TestVideoHandler_GetByID_Success_NoThumbnail(t *testing.T) {
	view := sampleVideoView()
	view.ThumbnailKey = ""
	view.ThumbnailURL = ""
	get := &mockGetVideoByIDHandler{view: view}
	h := newVideoHandler(nil, get)

	app := testutil.NewTestApp()
	app.Get("/videos/:id", testutil.WithUserWithoutProject(testutil.TestUserID), h.GetByID)

	req, err := testutil.JSONRequest(http.MethodGet, "/videos/"+testutil.TestVideoID.String(), nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}

	var out presenter.VideoDetailResponse
	testutil.DecodeJSON(t, resp, &out)
	if out.ThumbnailKey != nil {
		t.Fatalf("thumbnail key: got %v", out.ThumbnailKey)
	}
	if out.ThumbnailURL != nil {
		t.Fatalf("thumbnail url: got %v", out.ThumbnailURL)
	}
}

func TestVideoHandler_GetByID_Unauthorized(t *testing.T) {
	get := &mockGetVideoByIDHandler{}
	h := newVideoHandler(nil, get)

	app := testutil.NewTestApp()
	app.Get("/videos/:id", h.GetByID)

	req, err := testutil.JSONRequest(http.MethodGet, "/videos/"+testutil.TestVideoID.String(), nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if get.called {
		t.Fatal("get handler must not be called without auth")
	}
}

func TestVideoHandler_GetByID_InvalidInput(t *testing.T) {
	h := newVideoHandler(nil, nil)

	app := testutil.NewTestApp()
	app.Get("/videos/:id", testutil.WithUserWithoutProject(testutil.TestUserID), h.GetByID)

	req, err := testutil.JSONRequest(http.MethodGet, "/videos/not-a-uuid", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestVideoHandler_GetByID_HandlerError_NotFound(t *testing.T) {
	get := &mockGetVideoByIDHandler{err: domainvideo.ErrVideoNotFound}
	h := newVideoHandler(nil, get)

	app := testutil.NewTestApp()
	app.Get("/videos/:id", testutil.WithUserWithoutProject(testutil.TestUserID), h.GetByID)

	req, err := testutil.JSONRequest(http.MethodGet, "/videos/"+uuid.New().String(), nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestVideoHandler_GetByID_HandlerError_Internal(t *testing.T) {
	get := &mockGetVideoByIDHandler{err: errUnexpected}
	h := newVideoHandler(nil, get)

	app := testutil.NewTestApp()
	app.Get("/videos/:id", testutil.WithUserWithoutProject(testutil.TestUserID), h.GetByID)

	req, err := testutil.JSONRequest(http.MethodGet, "/videos/"+testutil.TestVideoID.String(), nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusInternalServerError)
	}

	body := testutil.DecodeJSONMap(t, resp)
	if body["message"] != "Failed to get video" {
		t.Fatalf("message: got %v", body["message"])
	}
}
