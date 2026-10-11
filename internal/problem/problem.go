package problem

import "errors"

// Error is safe to expose to a client. Never wrap a database or provider message here.
type Error struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string                    { return e.Code }
func New(status int, code, message string) *Error { return &Error{status, code, message} }
func Invalid(message string) *Error               { return New(400, "INVALID_REQUEST", message) }
func Unavailable() *Error {
	return New(503, "DEPENDENCY_UNAVAILABLE", "Required dependency is unavailable")
}
func NotFound() *Error { return New(404, "NOT_FOUND", "Resource not found") }
func Public(err error) *Error {
	var p *Error
	if errors.As(err, &p) {
		return p
	}
	return Unavailable()
}
