package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResourceDynamicHTTPRoundTrip(t *testing.T) {
	db := setupDynamicTestDB(t)
	model, err := NewModel(&dynUser{}, WithDB(db), WithModuleName("dyn"), WithOpenAPI(true))
	if err != nil {
		t.Fatalf("NewModel failed: %v", err)
	}
	tr := &testRouter{}
	res := NewResource(model, ResourceConfig{Router: tr, Prefix: "/api/v1"})
	res.Register()

	// Create via HTTP
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dyn/dyn_user", strings.NewReader(`{"name":"alice"}`))
	rec := httptest.NewRecorder()
	tr.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Create: want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var createResult CreateResult
	if err := json.Unmarshal(rec.Body.Bytes(), &createResult); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if createResult.ID == nil {
		t.Error("Create response: want id, got nil")
	}

	// Search via HTTP
	req = httptest.NewRequest(http.MethodGet, "/api/v1/dyn/dyn_users?page=0&page_size=10", nil)
	rec = httptest.NewRecorder()
	tr.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Search: want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var page PageResult
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("unmarshal search response: %v", err)
	}
	if page.TotalCount != 1 {
		t.Errorf("Search TotalCount: want 1, got %d", page.TotalCount)
	}

	// OpenAPI endpoint via dynamic model
	req = httptest.NewRequest(http.MethodGet, "/api/v1/dyn/dyn_user/openapi.json", nil)
	rec = httptest.NewRecorder()
	tr.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("OpenAPI: want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestResourceBatchRegistration(t *testing.T) {
	db := setupDynamicTestDB(t)
	tr := &testRouter{}
	models := []any{&dynUser{}, &dynOrder{}}
	for _, model := range models {
		m, err := NewModel(model, WithDB(db), WithModuleName("dyn"))
		if err != nil {
			t.Fatalf("NewModel(%T): %v", model, err)
		}
		NewResource(m, ResourceConfig{Router: tr, Prefix: "/api/v1"}).Register()
	}
	for _, key := range []string{
		"POST /api/v1/dyn/dyn_user",
		"GET /api/v1/dyn/dyn_users",
		"GET /api/v1/dyn/dyn_order/detail/:id",
		"GET /api/v1/dyn/dyn_orders",
	} {
		if _, ok := tr.handlers[key]; !ok {
			t.Errorf("route %q not registered", key)
		}
	}
}

func TestNewResourceWithOptionsDynamic(t *testing.T) {
	db := setupDynamicTestDB(t)
	tr := &testRouter{}
	res, err := NewResourceWithOptions(&dynUser{}, ResourceConfig{Router: tr, Prefix: "/api/v1"}, WithDB(db), WithModuleName("dyn"))
	if err != nil {
		t.Fatalf("NewResourceWithOptions: %v", err)
	}
	if res.ModelValue() == nil {
		t.Fatal("ModelValue is nil")
	}
	if _, ok := tr.handlers["POST /api/v1/dyn/dyn_user"]; !ok {
		t.Error("route not auto-registered by NewResourceWithOptions")
	}
}
