package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sjlit/rest/v3/schema"
)

// TestRespondErrorCodeMapping 验证 Respond 会按 error 类型映射状态码，
// 且错误响应是 JSON 而不是纯文本。
func TestRespondErrorCodeMapping(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"permission", ErrPermissionDenied, http.StatusForbidden},
		{"not_found", ErrRecordNotFound, http.StatusNotFound},
		{"payload", ErrPayloadInvalid, http.StatusBadRequest},
		{"create", ErrCreateFailed, http.StatusInternalServerError},
		{"update", ErrUpdateFailed, http.StatusInternalServerError},
		{"delete", ErrDeleteFailed, http.StatusInternalServerError},
		{"unavailable", ErrUnavailable, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Resource{}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			r.Respond(rec, req, tc.err)
			if rec.Code != tc.wantCode {
				t.Fatalf("status: want %d, got %d", tc.wantCode, rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type: want application/json, got %q", ct)
			}
			// 必须是合法 JSON（不再返回纯文本）
			var got map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("error response must be JSON, got %q (err=%v)", rec.Body.String(), err)
			}
		})
	}
}

// TestRespondUnknownErrorReturnsInternal 未知错误类型 → 500 + JSON。
func TestRespondUnknownErrorReturnsInternal(t *testing.T) {
	r := &Resource{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.Respond(rec, req, &testStringError{s: "boom"})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unknown error: want 500, got %d", rec.Code)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("error response must be JSON, got %q", rec.Body.String())
	}
}

type testStringError struct{ s string }

func (e *testStringError) Error() string { return e.s }

// TestCreateHandlerPropagatesError 验证 Create handler 把底层 error 透传
// 给 Respond（不再 swallow 成 ErrUnavailable）。
// 用非法 JSON 触发 ErrPayloadInvalid（应映射到 400）。
func TestCreateHandlerPropagatesError(t *testing.T) {
	db := setupDynamicTestDB(t)
	m, err := NewModel(&dynUser{}, WithDB(db), WithModuleName("dyn"))
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	tr := &testRouter{}
	r := NewResource(m, ResourceConfig{Router: tr, Prefix: "/api/v1"})
	r.Register()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dyn/dyn_user", strings.NewReader(`{not-json`))
	tr.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for ErrPayloadInvalid, got %d (%s)", rec.Code, rec.Body.String())
	}
}

// TestUpdateHandlerPropagatesError Update 非法 JSON → 400。
func TestUpdateHandlerPropagatesError(t *testing.T) {
	db := setupDynamicTestDB(t)
	m, err := NewModel(&dynUser{}, WithDB(db), WithModuleName("dyn"))
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	tr := &testRouter{}
	r := NewResource(m, ResourceConfig{Router: tr, Prefix: "/api/v1"})
	r.Register()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dyn/dyn_user/1", strings.NewReader(`{not-json`))
	tr.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for ErrPayloadInvalid, got %d (%s)", rec.Code, rec.Body.String())
	}
}

// TestDeleteHandlerPropagatesError Delete 找不到记录时返回 404。
func TestDeleteHandlerPropagatesError(t *testing.T) {
	db := setupDynamicTestDB(t)
	m, err := NewModel(&dynUser{}, WithDB(db), WithModuleName("dyn"))
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	tr := &testRouter{}
	r := NewResource(m, ResourceConfig{Router: tr, Prefix: "/api/v1"})
	r.Register()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/dyn/dyn_user/9999", nil)
	tr.ServeHTTP(rec, req)
	if rec.Code == http.StatusServiceUnavailable {
		t.Fatalf("Delete should not always return 503, got %d (%s)", rec.Code, rec.Body.String())
	}
}

// TestDetailHandlerPropagatesError Detail 找不到记录时返回 404。
func TestDetailHandlerPropagatesError(t *testing.T) {
	db := setupDynamicTestDB(t)
	m, err := NewModel(&dynUser{}, WithDB(db), WithModuleName("dyn"))
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	tr := &testRouter{}
	r := NewResource(m, ResourceConfig{Router: tr, Prefix: "/api/v1"})
	r.Register()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dyn/dyn_user/detail/9999", nil)
	tr.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("Detail missing record: want 404, got %d (%s)", rec.Code, rec.Body.String())
	}
}

// TestFindPrimaryKeyHandlesTrailingSlash findPrimaryKey 不能因多余前缀段而错位。
func TestFindPrimaryKeyHandlesTrailingSlash(t *testing.T) {
	db := setupDynamicTestDB(t)
	m, err := NewModel(&dynUser{}, WithDB(db), WithModuleName("dyn"))
	if err != nil {
		t.Fatalf("NewModel: %v", err)
	}
	r := &Resource{prefix: "/api/v1", model: m}
	got := r.findPrimaryKey(httptest.NewRequest(http.MethodGet,
		"/api/v1/dyn/dyn_user/detail/42", nil), schema.ScenarioDetail)
	if got != "42" {
		t.Errorf("want 42, got %q", got)
	}
	// path 短于 fullUri 时应安全返回空串而不是 panic
	got = r.findPrimaryKey(httptest.NewRequest(http.MethodGet, "/", nil), schema.ScenarioDetail)
	if got != "" {
		t.Errorf("want empty for too-short path, got %q", got)
	}
}
