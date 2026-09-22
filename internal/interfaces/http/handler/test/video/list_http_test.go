package videotest

import (
	"net/http"
	"testing"

	"go-api/internal/domain/paginate"
	domainvideo "go-api/internal/domain/video"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/testutil"
)

func TestVideoHandler_List_Success(t *testing.T) {
	list := &mockListVideosHandler{
		views: []domainvideo.VideoListView{sampleVideoListView()},
		total: 1,
	}
	h := newVideoHandler(nil, nil, list)

	app := testutil.NewTestApp()
	app.Get("/videos", testutil.WithUserWithoutProject(testutil.TestUserID), h.List)

	req, err := testutil.JSONRequest(http.MethodGet, "/videos", nil)
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
	if list.query.UserID != testutil.TestUserID {
		t.Fatalf("user id: got %s", list.query.UserID)
	}
	if list.query.Query.SortBy != "created_at" {
		t.Fatalf("sortBy: got %q", list.query.Query.SortBy)
	}
	if list.query.Query.OrderBy != paginate.OrderByDesc {
		t.Fatalf("orderBy: got %q", list.query.Query.OrderBy)
	}

	var out paginate.PaginateResponse
	testutil.DecodeJSON(t, resp, &out)
	if out.Total != 1 || out.Page != 1 {
		t.Fatalf("pagination: total=%d page=%d", out.Total, out.Page)
	}

	members, ok := out.Members.([]any)
	if !ok {
		t.Fatalf("members type: %T", out.Members)
	}
	if len(members) != 1 {
		t.Fatalf("members: got %d", len(members))
	}
	item, ok := members[0].(map[string]any)
	if !ok {
		t.Fatalf("member type: %T", members[0])
	}
	if item["id"] != testutil.TestVideoID.String() {
		t.Fatalf("id: got %v", item["id"])
	}
	if item["status"] != domainvideo.StatusFramesReady {
		t.Fatalf("status: got %v", item["status"])
	}
	if _, exists := item["storageKey"]; exists {
		t.Fatal("list item must not include storageKey")
	}
	if _, exists := item["frameCount"]; exists {
		t.Fatal("list item must not include frameCount")
	}
}

func TestVideoHandler_List_Success_QueryParams(t *testing.T) {
	list := &mockListVideosHandler{
		views: []domainvideo.VideoListView{sampleVideoListView()},
		total: 21,
	}
	h := newVideoHandler(nil, nil, list)

	app := testutil.NewTestApp()
	app.Get("/videos", testutil.WithUserWithoutProject(testutil.TestUserID), h.List)

	req, err := testutil.JSONRequest(http.MethodGet, "/videos?page=2&limit=10&sortBy=status&orderBy=asc&search=demo", nil)
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
	if list.query.Query.Page != 2 {
		t.Fatalf("page: got %d", list.query.Query.Page)
	}
	if list.query.Query.Limit != 10 {
		t.Fatalf("limit: got %d", list.query.Query.Limit)
	}
	if list.query.Query.SortBy != "status" {
		t.Fatalf("sortBy: got %q", list.query.Query.SortBy)
	}
	if list.query.Query.OrderBy != paginate.OrderByAsc {
		t.Fatalf("orderBy: got %q", list.query.Query.OrderBy)
	}
	if list.query.Query.Search != "demo" {
		t.Fatalf("search: got %q", list.query.Query.Search)
	}
}

func TestVideoHandler_List_Success_InvalidOrderByDefaultsDesc(t *testing.T) {
	list := &mockListVideosHandler{}
	h := newVideoHandler(nil, nil, list)

	app := testutil.NewTestApp()
	app.Get("/videos", testutil.WithUserWithoutProject(testutil.TestUserID), h.List)

	req, err := testutil.JSONRequest(http.MethodGet, "/videos?orderBy=bogus&limit=9999", nil)
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
	if list.query.Query.OrderBy != paginate.OrderByDesc {
		t.Fatalf("orderBy: got %q", list.query.Query.OrderBy)
	}
	if list.query.Query.Limit != paginate.MaxLimit {
		t.Fatalf("limit: got %d", list.query.Query.Limit)
	}
}

func TestVideoHandler_List_Success_Empty(t *testing.T) {
	list := &mockListVideosHandler{views: nil, total: 0}
	h := newVideoHandler(nil, nil, list)

	app := testutil.NewTestApp()
	app.Get("/videos", testutil.WithUserWithoutProject(testutil.TestUserID), h.List)

	req, err := testutil.JSONRequest(http.MethodGet, "/videos", nil)
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

	var out struct {
		Members []presenter.VideoListItemResponse `json:"members"`
		Total   int                               `json:"total"`
	}
	testutil.DecodeJSON(t, resp, &out)
	if out.Members == nil {
		t.Fatal("members must be [] not null")
	}
	if len(out.Members) != 0 {
		t.Fatalf("members: got %d", len(out.Members))
	}
}

func TestVideoHandler_List_Unauthorized(t *testing.T) {
	list := &mockListVideosHandler{}
	h := newVideoHandler(nil, nil, list)

	app := testutil.NewTestApp()
	app.Get("/videos", h.List)

	req, err := testutil.JSONRequest(http.MethodGet, "/videos", nil)
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
	if list.called {
		t.Fatal("list handler must not be called without auth")
	}
}

func TestVideoHandler_List_InvalidInput(t *testing.T) {
	h := newVideoHandler(nil, nil)

	app := testutil.NewTestApp()
	app.Get("/videos", testutil.WithUserWithoutProject(testutil.TestUserID), h.List)

	req, err := testutil.JSONRequest(http.MethodGet, "/videos?page=not-a-number", nil)
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
	if body["message"] != "Invalid query parameters" {
		t.Fatalf("message: got %v", body["message"])
	}
}

func TestVideoHandler_List_HandlerError_Internal(t *testing.T) {
	list := &mockListVideosHandler{err: errUnexpected}
	h := newVideoHandler(nil, nil, list)

	app := testutil.NewTestApp()
	app.Get("/videos", testutil.WithUserWithoutProject(testutil.TestUserID), h.List)

	req, err := testutil.JSONRequest(http.MethodGet, "/videos", nil)
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
	if body["message"] != "Failed to list videos" {
		t.Fatalf("message: got %v", body["message"])
	}
}
