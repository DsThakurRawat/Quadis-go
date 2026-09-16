package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
)

type PropertiesHandler struct {
	repo repository.Repository
}

func NewPropertiesHandler(repo repository.Repository) *PropertiesHandler {
	return &PropertiesHandler{repo: repo}
}

type propertyWithImages struct {
	domain.PropertyRecord
	Images []domain.PropertyImageRecord `json:"images"`
}

type propertyDetailResponse struct {
	domain.PropertyRecord
	Rooms  []domain.RoomTypeRecord      `json:"rooms"`
	Images []domain.PropertyImageRecord `json:"images"`
}

// ListProperties handles GET /api/properties
func (h *PropertiesHandler) ListProperties(w http.ResponseWriter, r *http.Request) {
	props, err := h.repo.GetProperties(r.Context())
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]propertyWithImages, 0, len(props))
	for _, p := range props {
		imgs, _ := h.repo.GetPropertyImages(r.Context(), p.ID)
		if imgs == nil {
			imgs = []domain.PropertyImageRecord{}
		}
		result = append(result, propertyWithImages{
			PropertyRecord: p,
			Images:         imgs,
		})
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"count":   len(result),
		"data":    result,
	})
}

// GetPropertyBySlug handles GET /api/properties/:slug
func (h *PropertiesHandler) GetPropertyBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		ErrorJSON(w, http.StatusBadRequest, "Property slug is required")
		return
	}

	prop, rooms, err := h.repo.GetPropertyBySlug(r.Context(), slug)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if prop == nil {
		ErrorJSON(w, http.StatusNotFound, "Property not found")
		return
	}

	imgs, _ := h.repo.GetPropertyImages(r.Context(), prop.ID)
	if imgs == nil {
		imgs = []domain.PropertyImageRecord{}
	}
	if rooms == nil {
		rooms = []domain.RoomTypeRecord{}
	}

	resp := propertyDetailResponse{
		PropertyRecord: *prop,
		Rooms:          rooms,
		Images:         imgs,
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    resp,
	})
}
