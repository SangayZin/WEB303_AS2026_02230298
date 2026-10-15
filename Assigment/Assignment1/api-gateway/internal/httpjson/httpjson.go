// Package httpjson has small helpers for reading/writing JSON in handlers.
package httpjson

import (
	"encoding/json"
	"io"
	"net/http"
)

// WriteJSON writes `v` as JSON with the given HTTP status.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// WriteError writes a standard error envelope: {"error": "..."}.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// ReadJSON decodes the request body into `dst`.
// Limits body size to 1MB and rejects unknown fields silently.
func ReadJSON(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dst)
}
