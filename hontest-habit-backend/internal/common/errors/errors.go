package errors

import (
	goerrors "errors"
	"net/http"
)

type ErrorCode string

const (
	CodeBadRequest   ErrorCode = "bad_request"
	CodeUnauthorized ErrorCode = "unauthorized"
	CodeForbidden    ErrorCode = "forbidden"
	CodeNotFound     ErrorCode = "not_found"
	CodeConflict     ErrorCode = "conflict"
	CodeInternal     ErrorCode = "internal"
)

var statusByCode = map[ErrorCode]int{
	CodeBadRequest:   http.StatusBadRequest,
	CodeUnauthorized: http.StatusUnauthorized,
	CodeForbidden:    http.StatusForbidden,
	CodeNotFound:     http.StatusNotFound,
	CodeConflict:     http.StatusConflict,
	CodeInternal:     http.StatusInternalServerError,
}

// HError is a categorized, HTTP-status-aware application error.
type HError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// New builds an HError of an arbitrary category. Prefer the category
// constructors below at call sites; New exists for generic/dynamic cases.
func New(code ErrorCode, message string, cause error) *HError {
	return &HError{Code: code, Message: message, Err: cause}
}

func BadRequest(message string, cause error) *HError {
	return New(CodeBadRequest, message, cause)
}

func Unauthorized(message string, cause error) *HError {
	return New(CodeUnauthorized, message, cause)
}

func Forbidden(message string, cause error) *HError {
	return New(CodeForbidden, message, cause)
}

func NotFound(message string, cause error) *HError {
	return New(CodeNotFound, message, cause)
}

func Conflict(message string, cause error) *HError {
	return New(CodeConflict, message, cause)
}

func Internal(message string, cause error) *HError {
	return New(CodeInternal, message, cause)
}

func (e *HError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap enables errors.Is/errors.As to traverse into the wrapped cause.
func (e *HError) Unwrap() error {
	return e.Err
}

// StatusCode returns the HTTP status for this error's category, defaulting
// to 500 for an unrecognized/zero-value code.
func (e *HError) StatusCode() int {
	if s, ok := statusByCode[e.Code]; ok {
		return s
	}
	return http.StatusInternalServerError
}

// StatusCode finds an *HError anywhere in err's chain and returns its HTTP
// status, or 500 if none is found.
func StatusCode(err error) int {
	if he, ok := goerrors.AsType[*HError](err); ok {
		return he.StatusCode()
	}
	return http.StatusInternalServerError
}
