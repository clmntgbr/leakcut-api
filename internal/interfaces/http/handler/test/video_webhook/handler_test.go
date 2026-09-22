package videowebhooktest

import (
	"context"
	"errors"
	"net/http"
	"testing"

	cmdvideo "go-api/internal/application/command/video"
	domainvideo "go-api/internal/domain/video"
	"go-api/internal/interfaces/http/dto"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/testutil"
)

type mockRequestIngestHandler struct {
	called bool
	cmd    cmdvideo.RequestIngestCommand
	result *cmdvideo.RequestIngestResult
	err    error
}

func (m *mockRequestIngestHandler) Handle(_ context.Context, cmd cmdvideo.RequestIngestCommand) (*cmdvideo.RequestIngestResult, error) {
	m.called = true
	m.cmd = cmd
	return m.result, m.err
}

func newHandler(ingest *mockRequestIngestHandler) *handler.VideoWebhookHandler {
	if ingest == nil {
		ingest = &mockRequestIngestHandler{}
	}
	return handler.NewVideoWebhookHandler(ingest)
}

func validIngest() dto.VideoIngestRequest {
	return dto.VideoIngestRequest{
		VideoURL:    "https://cdn.example.com/demo.mp4",
		Filename:    "demo.mp4",
		ContentType: "video/mp4",
		SizeBytes:   4096,
	}
}

func postWebhook(t *testing.T, h *handler.VideoWebhookHandler, payload any) *http.Response {
	t.Helper()

	app := testutil.NewTestApp()
	app.Post("/webhooks/videos", testutil.WithLocal("payload", payload), h.Ingest)

	req, err := testutil.JSONRequest(http.MethodPost, "/webhooks/videos", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	return resp
}

func TestVideoWebhookHandler_Ingest_Success(t *testing.T) {
	ingest := &mockRequestIngestHandler{
		result: &cmdvideo.RequestIngestResult{VideoID: testutil.TestVideoID.String()},
	}
	h := newHandler(ingest)

	resp := postWebhook(t, h, validIngest())
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusAccepted)
	}
	if ingest.cmd.RemoteURL != "https://cdn.example.com/demo.mp4" {
		t.Fatalf("remote url: got %q", ingest.cmd.RemoteURL)
	}

	var out presenter.VideoIngestResponse
	testutil.DecodeJSON(t, resp, &out)
	if out.VideoID != testutil.TestVideoID.String() {
		t.Fatalf("video id: got %s", out.VideoID)
	}
}

func TestVideoWebhookHandler_Ingest_FilenameFromURL(t *testing.T) {
	ingest := &mockRequestIngestHandler{
		result: &cmdvideo.RequestIngestResult{VideoID: testutil.TestVideoID.String()},
	}
	h := newHandler(ingest)

	resp := postWebhook(t, h, dto.VideoIngestRequest{
		VideoURL:    "https://cdn.example.com/path/clip.webm",
		ContentType: "video/webm",
	})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusAccepted)
	}
	if ingest.cmd.Filename != "clip.webm" {
		t.Fatalf("filename: got %q", ingest.cmd.Filename)
	}
}

func TestVideoWebhookHandler_Ingest_InvalidPayload(t *testing.T) {
	h := newHandler(nil)
	resp := postWebhook(t, h, "not-a-payload")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestVideoWebhookHandler_Ingest_ValidationError(t *testing.T) {
	h := newHandler(nil)
	resp := postWebhook(t, h, dto.VideoIngestRequest{})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestVideoWebhookHandler_Ingest_HandlerError_UnsupportedType(t *testing.T) {
	ingest := &mockRequestIngestHandler{err: domainvideo.ErrUnsupportedType}
	h := newHandler(ingest)

	resp := postWebhook(t, h, validIngest())
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestVideoWebhookHandler_Ingest_HandlerError_InvalidFilename(t *testing.T) {
	ingest := &mockRequestIngestHandler{err: domainvideo.ErrInvalidFilename}
	h := newHandler(ingest)

	resp := postWebhook(t, h, validIngest())
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestVideoWebhookHandler_Ingest_HandlerError_Internal(t *testing.T) {
	ingest := &mockRequestIngestHandler{err: errors.New("database unavailable")}
	h := newHandler(ingest)

	resp := postWebhook(t, h, validIngest())
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusInternalServerError)
	}

	body := testutil.DecodeJSONMap(t, resp)
	if body["message"] != "Failed to ingest video" {
		t.Fatalf("message: got %v", body["message"])
	}
}
