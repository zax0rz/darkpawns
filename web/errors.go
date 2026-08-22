package web

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the stable error envelope returned by the HTTP API.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail gives machine clients a stable code and a recovery hint.
type ErrorDetail struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Resolution string `json:"resolution"`
}

// WriteJSONError writes a structured, machine-readable API error.
func WriteJSONError(w http.ResponseWriter, status int, code, message, resolution string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: ErrorDetail{
		Code:       code,
		Message:    message,
		Resolution: resolution,
	}})
}
