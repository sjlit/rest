package rest

import "errors"

var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrRecordNotFound   = errors.New("record not found")
	ErrPayloadInvalid   = errors.New("invalid payload")
	ErrCreateFailed     = errors.New("create failed")
	ErrUpdateFailed     = errors.New("update failed")
	ErrDeleteFailed     = errors.New("delete failed")
	ErrInternal         = errors.New("internal server error")
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
