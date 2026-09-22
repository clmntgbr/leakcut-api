package storagewebhooktest

import (
	"context"
	"errors"
	"net/http"
	"testing"

	cmdvideo "go-api/internal/application/command/video"
	domainvideo "go-api/internal/domain/video"
	"go-api/internal/interfaces/http/dto"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"
)

const testBucket = "media"

type mockConfirmUploadHandler struct {
	called bool
	cmd    cmdvideo.ConfirmUploadCommand
	err    error
}

func (m *mockConfirmUploadHandler) Handle(_ context.Context, cmd cmdvideo.ConfirmUploadCommand) error {
	m.called = true
	m.cmd = cmd
	return m.err
}

func newHandler(confirm *mockConfirmUploadHandler) *handler.StorageWebhookHandler {
	if confirm == nil {
		confirm = &mockConfirmUploadHandler{}
	}
	return handler.NewStorageWebhookHandler(testBucket, confirm)
}

func objectCreated(key, bucket string) dto.ObjectCreatedEvent {
	return dto.ObjectCreatedEvent{
		Records: []dto.ObjectCreatedRecord{{
			EventName: "s3:ObjectCreated:Put",
			S3: dto.S3Entity{
				Bucket: dto.S3Bucket{Name: bucket},
				Object: dto.S3Object{
					Key:         key,
					Size:        2048,
					ContentType: "video/mp4",
				},
			},
		}},
	}
}

func postWebhook(t *testing.T, h *handler.StorageWebhookHandler, payload any) *http.Response {
	t.Helper()

	app := testutil.NewTestApp()
	app.Post("/webhooks/minio/object-created", testutil.WithLocal("payload", payload), h.ObjectCreated)

	req, err := testutil.JSONRequest(http.MethodPost, "/webhooks/minio/object-created", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	return resp
}

func TestStorageWebhookHandler_ObjectCreated_Success(t *testing.T) {
	confirm := &mockConfirmUploadHandler{}
	h := newHandler(confirm)
	key := domainvideo.NewStorageKey(testutil.TestVideoID)

	resp := postWebhook(t, h, objectCreated(key, testBucket))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	if !confirm.called {
		t.Fatal("expected confirm handler to be called")
	}
	if confirm.cmd.StorageKey != key {
		t.Fatalf("storage key: got %q", confirm.cmd.StorageKey)
	}
	if confirm.cmd.SizeBytes != 2048 {
		t.Fatalf("size: got %d", confirm.cmd.SizeBytes)
	}
}

func TestStorageWebhookHandler_ObjectCreated_InvalidPayload(t *testing.T) {
	h := newHandler(nil)
	resp := postWebhook(t, h, "not-an-event")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestStorageWebhookHandler_ObjectCreated_ValidationError(t *testing.T) {
	h := newHandler(nil)
	resp := postWebhook(t, h, dto.ObjectCreatedEvent{})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestStorageWebhookHandler_ObjectCreated_SkipNonCreatedEvent(t *testing.T) {
	confirm := &mockConfirmUploadHandler{}
	h := newHandler(confirm)
	event := objectCreated(domainvideo.NewStorageKey(testutil.TestVideoID), testBucket)
	event.Records[0].EventName = "s3:ObjectRemoved:Delete"

	resp := postWebhook(t, h, event)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	if confirm.called {
		t.Fatal("confirm handler must not run for non-created events")
	}
}

func TestStorageWebhookHandler_ObjectCreated_HandlerError_TooLarge(t *testing.T) {
	confirm := &mockConfirmUploadHandler{err: domainvideo.ErrVideoTooLarge}
	h := newHandler(confirm)

	resp := postWebhook(t, h, objectCreated(domainvideo.NewStorageKey(testutil.TestVideoID), testBucket))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestStorageWebhookHandler_ObjectCreated_WrongBucket(t *testing.T) {
	confirm := &mockConfirmUploadHandler{}
	h := newHandler(confirm)
	key := domainvideo.NewStorageKey(testutil.TestVideoID)

	resp := postWebhook(t, h, objectCreated(key, "other"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	if confirm.called {
		t.Fatal("confirm handler must not run for another bucket")
	}
}

func TestStorageWebhookHandler_ObjectCreated_SkipFrameKey(t *testing.T) {
	confirm := &mockConfirmUploadHandler{}
	h := newHandler(confirm)

	resp := postWebhook(t, h, objectCreated(domainvideo.NewFrameStorageKey(testutil.TestVideoID, 0), testBucket))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	if confirm.called {
		t.Fatal("confirm handler must not run for frame keys")
	}
}

func TestStorageWebhookHandler_ObjectCreated_SkipThumbnailKey(t *testing.T) {
	confirm := &mockConfirmUploadHandler{}
	h := newHandler(confirm)

	resp := postWebhook(t, h, objectCreated(domainvideo.NewThumbnailStorageKey(testutil.TestVideoID), testBucket))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	if confirm.called {
		t.Fatal("confirm handler must not run for thumbnail keys")
	}
}

func TestStorageWebhookHandler_ObjectCreated_InvalidKey(t *testing.T) {
	confirm := &mockConfirmUploadHandler{}
	h := newHandler(confirm)

	resp := postWebhook(t, h, objectCreated("not-a-video-key", testBucket))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	if confirm.called {
		t.Fatal("confirm handler must not run for invalid keys")
	}
}

func TestStorageWebhookHandler_ObjectCreated_BrokenEncoding(t *testing.T) {
	confirm := &mockConfirmUploadHandler{}
	h := newHandler(confirm)

	resp := postWebhook(t, h, objectCreated("%zz", testBucket))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	if confirm.called {
		t.Fatal("confirm handler must not run for invalid encoding")
	}
}

func TestStorageWebhookHandler_ObjectCreated_HandlerError_NotFound(t *testing.T) {
	confirm := &mockConfirmUploadHandler{err: domainvideo.ErrVideoNotFound}
	h := newHandler(confirm)

	resp := postWebhook(t, h, objectCreated(domainvideo.NewStorageKey(testutil.TestVideoID), testBucket))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestStorageWebhookHandler_ObjectCreated_HandlerError_InvalidTransition(t *testing.T) {
	confirm := &mockConfirmUploadHandler{err: domainvideo.ErrInvalidTransition}
	h := newHandler(confirm)

	resp := postWebhook(t, h, objectCreated(domainvideo.NewStorageKey(testutil.TestVideoID), testBucket))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestStorageWebhookHandler_ObjectCreated_HandlerError_Internal(t *testing.T) {
	confirm := &mockConfirmUploadHandler{err: errors.New("database unavailable")}
	h := newHandler(confirm)

	resp := postWebhook(t, h, objectCreated(domainvideo.NewStorageKey(testutil.TestVideoID), testBucket))
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusInternalServerError)
	}

	body := testutil.DecodeJSONMap(t, resp)
	if body["message"] != "Failed to confirm upload" {
		t.Fatalf("message: got %v", body["message"])
	}
}
