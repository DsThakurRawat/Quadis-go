package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
)

type contextKey string

const RawBodyKey contextKey = "rawBody"

// PreserveRawBody reads the request body up to maxBytes, saves the raw string/bytes in context,
// and resets r.Body with a new reader so downstream handlers can parse it.
func PreserveRawBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Read up to maxBytes + 1 to detect oversize
			bodyBytes, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBytes))
			if err != nil {
				http.Error(w, `{"success":false,"error":"Request payload too large"}`, http.StatusRequestEntityTooLarge)
				return
			}
			_ = r.Body.Close()

			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			ctx := context.WithValue(r.Context(), RawBodyKey, string(bodyBytes))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRawBody retrieves the preserved raw body from context.
func GetRawBody(r *http.Request) string {
	if val, ok := r.Context().Value(RawBodyKey).(string); ok {
		return val
	}
	return ""
}
