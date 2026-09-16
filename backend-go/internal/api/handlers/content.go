package handlers

import (
	"net/http"

	"quadis-backend-go/internal/repository"
)

type ContentHandler struct {
	repo repository.Repository
}

func NewContentHandler(repo repository.Repository) *ContentHandler {
	return &ContentHandler{repo: repo}
}

// GetContent handles GET /api/content
func (h *ContentHandler) GetContent(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.GetSiteContent(r.Context())
	if err != nil {
		// Never fail the page over editable copy — an error here just returns empty map
		data = map[string]string{}
	}
	if data == nil {
		data = map[string]string{}
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}
