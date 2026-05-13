package rest

import (
	"net/http"
)

type (
	Naming struct {
		Pluralize  string
		Singular   string
		ModuleName string
		TableName  string
	}

	DiffAttr struct {
		Column   string `json:"column"`
		Label    string `json:"label"`
		Previous any    `json:"previous"`
		Current  any    `json:"current"`
	}
)

type (
	Router interface {
		Handle(method string, path string, handler http.HandlerFunc)
	}

	Responder interface {
		Respond(w http.ResponseWriter, r *http.Request, data any)
	}

	CreateResult struct {
		ID string `json:"id"`
	}

	UpdateResult struct {
		ID string `json:"id"`
	}

	DeletedResult struct {
		ID string `json:"id"`
	}

	PageResult[T any] struct {
		Page       int   `json:"page"`
		PageSize   int   `json:"page_size"`
		TotalCount int64 `json:"total_count"`
		TotalPages int   `json:"total_pages"`
		Data       []*T  `json:"data"`
	}

	CursorResult[T any] struct {
		Data       []*T   `json:"data"`
		NextCursor string `json:"next_cursor,omitempty"`
		HasMore    bool   `json:"has_more"`
	}
)
