package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
)

type EnquiriesHandler struct {
	repo repository.Repository
}

func NewEnquiriesHandler(repo repository.Repository) *EnquiriesHandler {
	return &EnquiriesHandler{repo: repo}
}

type createEnquiryPayload struct {
	EnquiryType  string  `json:"enquiryType"`
	PropertySlug *string `json:"propertySlug,omitempty"`
	PropertyID   *string `json:"propertyId,omitempty"`
	GuestName    string  `json:"guestName"`
	GuestPhone   string  `json:"guestPhone"`
	GuestEmail   *string `json:"guestEmail,omitempty"`
	EventDate    *string `json:"eventDate,omitempty"`
	GuestCount   *int    `json:"guestCount,omitempty"`
	Message      *string `json:"message,omitempty"`
}

// CreateEnquiry handles POST /api/enquiries
func (h *EnquiriesHandler) CreateEnquiry(w http.ResponseWriter, r *http.Request) {
	var payload createEnquiryPayload
	if err := ParseJSON(r, &payload); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	fieldErrors := make(map[string][]string)
	validTypes := map[string]bool{
		"ROOM_HOLD":     true,
		"BANQUET":       true,
		"CORPORATE_RFP": true,
		"GENERAL":       true,
	}
	if !validTypes[payload.EnquiryType] {
		fieldErrors["enquiryType"] = []string{"Invalid enquiry type (ROOM_HOLD, BANQUET, CORPORATE_RFP, GENERAL)"}
	}
	if strings.TrimSpace(payload.GuestName) == "" {
		fieldErrors["guestName"] = []string{"Guest name is required"}
	}
	if len(strings.TrimSpace(payload.GuestPhone)) < 10 {
		fieldErrors["guestPhone"] = []string{"Valid 10-digit phone number is required"}
	}

	if len(fieldErrors) > 0 {
		ValidationErrorJSON(w, "Invalid enquiry parameters", fieldErrors)
		return
	}

	var propID *string
	if payload.PropertySlug != nil && *payload.PropertySlug != "" {
		p, _, err := h.repo.GetPropertyBySlug(r.Context(), *payload.PropertySlug)
		if err == nil && p != nil {
			propID = &p.ID
		}
	} else if payload.PropertyID != nil && *payload.PropertyID != "" {
		p, err := h.repo.GetPropertyByID(r.Context(), *payload.PropertyID)
		if err == nil && p != nil {
			propID = &p.ID
		}
	}

	enq := domain.EnquiryRecord{
		EnquiryType: domain.EnquiryType(payload.EnquiryType),
		PropertyID:  propID,
		GuestName:   payload.GuestName,
		GuestPhone:  payload.GuestPhone,
		GuestEmail:  payload.GuestEmail,
		EventDate:    payload.EventDate,
		GuestCount:   payload.GuestCount,
		Message:      payload.Message,
		Status:       domain.EnquiryStatusNew,
	}

	if err := h.repo.CreateEnquiry(r.Context(), &enq); err != nil {
		ErrorJSON(w, http.StatusInternalServerError, fmt.Sprintf("Failed to submit enquiry: %v", err))
		return
	}

	JSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Enquiry submitted successfully and WhatsApp alert dispatched to hotel management",
		"data":    enq,
	})
}

// GetEnquiries handles GET /api/enquiries (requireAdmin)
func (h *EnquiriesHandler) GetEnquiries(w http.ResponseWriter, r *http.Request) {
	var statusFilter *domain.EnquiryStatus
	qStatus := r.URL.Query().Get("status")
	if qStatus != "" {
		st := domain.EnquiryStatus(qStatus)
		statusFilter = &st
	}

	enquiries, err := h.repo.GetEnquiries(r.Context(), statusFilter)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if enquiries == nil {
		enquiries = []domain.EnquiryRecord{}
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"count":   len(enquiries),
		"data":    enquiries,
	})
}

// GetEnquiryByID handles GET /api/enquiries/:id (requireAdmin)
func (h *EnquiriesHandler) GetEnquiryByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		ErrorJSON(w, http.StatusBadRequest, "Enquiry ID is required")
		return
	}

	enq, err := h.repo.GetEnquiryByID(r.Context(), id)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if enq == nil {
		ErrorJSON(w, http.StatusNotFound, "Enquiry not found")
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    enq,
	})
}

type updateEnquiryStatusPayload struct {
	Status                string  `json:"status"`
	RazorpayPaymentLinkID *string `json:"razorpayPaymentLinkId,omitempty"`
}

// UpdateEnquiryStatus handles PATCH /api/enquiries/:id/status (requireAdmin)
func (h *EnquiriesHandler) UpdateEnquiryStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		ErrorJSON(w, http.StatusBadRequest, "Enquiry ID is required")
		return
	}

	var payload updateEnquiryStatusPayload
	if err := ParseJSON(r, &payload); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	validStatuses := map[string]bool{
		"NEW":       true,
		"CONTACTED": true,
		"LINK_SENT": true,
		"CONVERTED": true,
		"CLOSED":    true,
	}
	if !validStatuses[payload.Status] {
		ErrorJSON(w, http.StatusBadRequest, "Invalid status update parameters")
		return
	}

	updated, err := h.repo.UpdateEnquiryStatus(r.Context(), id, domain.EnquiryStatus(payload.Status), payload.RazorpayPaymentLinkID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if updated == nil {
		ErrorJSON(w, http.StatusNotFound, "Enquiry not found")
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Enquiry status updated",
		"data":    updated,
	})
}
