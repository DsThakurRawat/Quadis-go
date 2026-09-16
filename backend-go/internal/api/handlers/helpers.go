package handlers

import (
	"encoding/json"
	"net/http"

	"quadis-backend-go/internal/api/middleware"
)

// JSON writes a JSON response with status code.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// ErrorJSON writes an error response matching Express { success: false, error: msg }
func ErrorJSON(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// ValidationErrorJSON writes a 400 validation error response matching Express Zod errors
func ValidationErrorJSON(w http.ResponseWriter, message string, details map[string][]string) {
	JSON(w, http.StatusBadRequest, map[string]interface{}{
		"success": false,
		"error":   message,
		"details": details,
	})
}

// ParseJSON parses the request body into target struct, reading from preserved raw body if available.
func ParseJSON(r *http.Request, target interface{}) error {
	raw := middleware.GetRawBody(r)
	if raw != "" {
		return json.Unmarshal([]byte(raw), target)
	}
	return json.NewDecoder(r.Body).Decode(target)
}
