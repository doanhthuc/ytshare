package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

// APIError is the wire format for error responses.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type HTTPError struct {
	Status  int
	Code    string
	Message string
	Cause   error
}

func (e *HTTPError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *HTTPError) Unwrap() error { return e.Cause }

func NewError(status int, code, message string) *HTTPError {
	return &HTTPError{Status: status, Code: code, Message: message}
}

func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteError(w http.ResponseWriter, err error) {
	var he *HTTPError
	if errors.As(err, &he) {
		JSON(w, he.Status, APIError{Code: he.Code, Message: he.Message})
		return
	}
	JSON(w, http.StatusInternalServerError, APIError{
		Code:    "internal_error",
		Message: "internal server error",
	})
}

// DecodeJSON parses the body into dst, returning a 400 HTTPError on malformed payloads.
func DecodeJSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return NewError(http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
	}
	return nil
}
