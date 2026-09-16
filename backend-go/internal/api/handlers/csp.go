package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// CSPReport handles POST /api/csp-report
func CSPReport(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil && body != nil {
		report, _ := body["csp-report"].(map[string]interface{})
		if report == nil {
			report = body
		}

		at := func(k string) string {
			if v, ok := report[k].(string); ok {
				if len(v) > 200 {
					return v[:200]
				}
				return v
			}
			return ""
		}

		blocked := at("blocked-uri")
		if blocked == "" {
			blocked = at("blockedURL")
		}
		directive := at("violated-directive")
		if directive == "" {
			directive = at("effectiveDirective")
		}
		docURI := at("document-uri")
		if docURI == "" {
			docURI = at("documentURL")
		}

		log.Printf("[CSP] blocked=%s directive=%s on=%s", blocked, directive, docURI)
	}

	w.WriteHeader(http.StatusNoContent)
}
