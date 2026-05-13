package rest

var (
	ErrPermissionDenied = NewError(4003, "permission denied")
	ErrRecordNotFound   = NewError(4004, "record not found")
	ErrPayloadInvalid   = NewError(1001, "invalid payload")
	ErrCreateFailed     = NewError(6001, "create failed")
	ErrUpdateFailed     = NewError(6002, "update failed")
	ErrDeleteFailed     = NewError(6003, "delete failed")
	ErrUnavailable      = NewError(1003, "internal server error")
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail"`
}

func (e Error) Error() string {
	return e.Message
}

func NewError(code int, message string) Error {
	return Error{
		Code:    code,
		Message: message,
	}
}
