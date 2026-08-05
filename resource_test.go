package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sjlit/rest/v3/query"
	"github.com/sjlit/rest/v3/schema"
)

func TestBuildQuerySkipsMalformedSort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/list?sort=name,", nil)
	r := &Resource{}
	qb := r.buildQuery(req, nil)
	spec := qb.Spec()
	if len(spec.OrderBy) != 1 {
		t.Fatalf("want 1 order clause (empty segment skipped), got %d: %+v", len(spec.OrderBy), spec.OrderBy)
	}
	if spec.OrderBy[0] != (query.Order{Field: "name", Direction: "ASC"}) {
		t.Errorf("unexpected order: %+v", spec.OrderBy[0])
	}
}

func TestBuildQuerySortPrefixDirections(t *testing.T) {
	// 裸 + 在 URL query 中会被解码为空格, 升序前缀需使用 %2B
	req := httptest.NewRequest(http.MethodGet, "http://example.com/list?sort=-age,%2Bcreated_at", nil)
	r := &Resource{}
	qb := r.buildQuery(req, nil)
	spec := qb.Spec()
	want := []query.Order{
		{Field: "age", Direction: "DESC"},
		{Field: "created_at", Direction: "ASC"},
	}
	if len(spec.OrderBy) != len(want) {
		t.Fatalf("want %d orders, got %d: %+v", len(want), len(spec.OrderBy), spec.OrderBy)
	}
	for i := range want {
		if spec.OrderBy[i] != want[i] {
			t.Errorf("order[%d]: want %+v, got %+v", i, want[i], spec.OrderBy[i])
		}
	}
}

func TestBuildQueryTimeRangeRequiresSeparator(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/list?created_at=12", nil)
	r := &Resource{}
	schemas := []schema.Schema{{Column: "created_at", Format: schema.FormatDatetime, Native: 1}}
	qb := r.buildQuery(req, schemas)
	spec := qb.Spec()
	if len(spec.Where) != 1 {
		t.Fatalf("want 1 where clause, got %d", len(spec.Where))
	}
	cond, ok := spec.Where[0].(query.Condition)
	if !ok {
		t.Fatalf("want plain Condition (no separator must not become a range), got %T", spec.Where[0])
	}
	if cond.Op != query.OpEq {
		t.Errorf("op: want %s, got %s", query.OpEq, cond.Op)
	}
}

func TestBuildQueryTimeRangeWithSeparator(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/list?created_at=2026-01-01,2026-02-01", nil)
	r := &Resource{}
	schemas := []schema.Schema{{Column: "created_at", Format: schema.FormatDatetime, Native: 1}}
	qb := r.buildQuery(req, schemas)
	spec := qb.Spec()
	if len(spec.Where) != 1 {
		t.Fatalf("want 1 where clause, got %d", len(spec.Where))
	}
	if _, ok := spec.Where[0].(query.AndGroup); !ok {
		t.Fatalf("want AndGroup range with separator, got %T", spec.Where[0])
	}
}
