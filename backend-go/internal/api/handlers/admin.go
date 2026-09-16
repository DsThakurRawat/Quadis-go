package handlers

import (
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
	"quadis-backend-go/internal/service/auth"
	"quadis-backend-go/internal/service/payment"
)

const adminSessionTTL = 60 * 60 * 12 // 12 hours

type AdminHandler struct {
	repo            repository.Repository
	sessionService  *auth.SessionService
	razorpayService *payment.RazorpayService
	cfg             *config.Config
}

func NewAdminHandler(
	repo repository.Repository,
	sessionService *auth.SessionService,
	razorpayService *payment.RazorpayService,
	cfg *config.Config,
) *AdminHandler {
	return &AdminHandler{
		repo:            repo,
		sessionService:  sessionService,
		razorpayService: razorpayService,
		cfg:             cfg,
	}
}

var sixDigitsRegex = regexp.MustCompile(`^\d{6}$`)

func validatePIN(pin string) string {
	if !sixDigitsRegex.MatchString(pin) {
		return "PIN must be exactly 6 digits"
	}
	// Same digit six times
	allSame := true
	for i := 1; i < len(pin); i++ {
		if pin[i] != pin[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return "PIN cannot be the same digit six times"
	}
	// Straight run forward or backward (e.g. 123456 or 987654)
	isRunAsc := true
	isRunDesc := true
	for i := 1; i < len(pin); i++ {
		if int(pin[i])-int(pin[i-1]) != 1 {
			isRunAsc = false
		}
		if int(pin[i-1])-int(pin[i]) != 1 {
			isRunDesc = false
		}
	}
	if isRunAsc || isRunDesc {
		return "PIN cannot be six digits in a row"
	}
	return ""
}

type adminAuthPayload struct {
	PIN string `json:"pin"`
}

// Auth handles POST /api/admin/auth
func (h *AdminHandler) Auth(w http.ResponseWriter, r *http.Request) {
	bootstrapPin := h.cfg.AdminPIN
	adminPassword := h.cfg.AdminPassword

	if bootstrapPin == "" || adminPassword == "" {
		log.Println("ADMIN_PIN / ADMIN_PASSWORD are not set — refusing admin sign-in.")
		ErrorJSON(w, http.StatusServiceUnavailable, "Admin access is not configured")
		return
	}

	var payload adminAuthPayload
	if err := ParseJSON(r, &payload); err != nil || payload.PIN == "" {
		ErrorJSON(w, http.StatusUnauthorized, "Invalid Admin PIN")
		return
	}

	storedHash, err := h.repo.GetAdminPINHash(r.Context())
	usingBootstrap := err != nil || storedHash == ""

	ok := false
	if usingBootstrap {
		ok = payload.PIN == bootstrapPin
	} else {
		ok = auth.VerifyPassword(payload.PIN, storedHash)
	}

	if !ok {
		log.Printf("Admin sign-in rejected — mode=%v submitted_length=%d (expected 6)",
			map[bool]string{true: "bootstrap", false: "stored"}[usingBootstrap],
			len(payload.PIN),
		)
		ErrorJSON(w, http.StatusUnauthorized, "Invalid Admin PIN")
		return
	}

	roleAdmin := "admin"
	token, err := h.sessionService.SignSession("admin", "", &roleAdmin, adminSessionTTL)
	if err != nil || token == "" {
		ErrorJSON(w, http.StatusServiceUnavailable, "Admin access is not configured")
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success":          true,
		"token":            token,
		"expiresInSeconds": adminSessionTTL,
		"mustChangePin":    usingBootstrap,
		"message":          "Authenticated successfully as Hotel Management",
	})
}

type changePinPayload struct {
	CurrentPIN string `json:"currentPin"`
	NewPIN     string `json:"newPin"`
}

// ChangePIN handles POST /api/admin/change-pin (requireAdmin)
func (h *AdminHandler) ChangePIN(w http.ResponseWriter, r *http.Request) {
	var payload changePinPayload
	if err := ParseJSON(r, &payload); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if prob := validatePIN(payload.NewPIN); prob != "" {
		ErrorJSON(w, http.StatusBadRequest, prob)
		return
	}

	storedHash, err := h.repo.GetAdminPINHash(r.Context())
	bootstrapPin := h.cfg.AdminPIN

	currentOk := false
	if err != nil || storedHash == "" {
		currentOk = bootstrapPin != "" && payload.CurrentPIN == bootstrapPin
	} else {
		currentOk = auth.VerifyPassword(payload.CurrentPIN, storedHash)
	}

	if !currentOk {
		ErrorJSON(w, http.StatusUnauthorized, "Current PIN is incorrect")
		return
	}

	if payload.NewPIN == payload.CurrentPIN {
		ErrorJSON(w, http.StatusBadRequest, "New PIN must be different from the current one")
		return
	}

	newHash, err := auth.HashPassword(payload.NewPIN)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "Failed to hash new PIN")
		return
	}

	if err := h.repo.SetAdminPINHash(r.Context(), newHash); err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "Failed to save new PIN")
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Admin PIN updated",
	})
}

// Dashboard handles GET /api/admin/dashboard (requireAdmin)
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.repo.GetGlanceMetrics(r.Context())
	if err != nil {
		metrics = map[string]interface{}{}
	}

	props, err := h.repo.GetPropertiesWithRooms(r.Context())
	if err != nil {
		props = []domain.PropertyRecord{}
	}

	bookings, err := h.repo.GetAllBookings(r.Context(), 15)
	if err != nil {
		bookings = []domain.BookingRecord{}
	}

	enquiries, err := h.repo.GetEnquiries(r.Context(), nil)
	if err != nil {
		enquiries = []domain.EnquiryRecord{}
	}
	recentEnquiries := enquiries
	if len(recentEnquiries) > 15 {
		recentEnquiries = recentEnquiries[:15]
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"metrics":         metrics,
			"properties":      props,
			"recentBookings":  bookings,
			"recentEnquiries": recentEnquiries,
		},
	})
}

type roomAvailPayload struct {
	RoomTypeID   string  `json:"roomTypeId"`
	PropertySlug *string `json:"propertySlug,omitempty"`
	IsAvailable  bool    `json:"isAvailable"`
}

// RoomAvailability handles PATCH /api/admin/room-availability (requireAdmin)
func (h *AdminHandler) RoomAvailability(w http.ResponseWriter, r *http.Request) {
	var payload roomAvailPayload
	if err := ParseJSON(r, &payload); err != nil || payload.RoomTypeID == "" {
		ErrorJSON(w, http.StatusBadRequest, "roomTypeId is required")
		return
	}

	propSlug := ""
	if payload.PropertySlug != nil {
		propSlug = *payload.PropertySlug
	}

	ok, msg, err := h.repo.ToggleRoomAvailability(r.Context(), propSlug, payload.RoomTypeID, payload.IsAvailable)
	if err != nil || !ok {
		ErrorJSON(w, http.StatusNotFound, "Room category not found")
		return
	}

	statusWord := "SOLD OUT"
	if payload.IsAvailable {
		statusWord = "AVAILABLE"
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Room category status updated to %s", statusWord),
		"data": map[string]interface{}{
			"room":         msg,
			"is_available": payload.IsAvailable,
		},
	})
}

// UpdateProperty handles PATCH /api/admin/properties/:idOrSlug (requireAdmin)
func (h *AdminHandler) UpdateProperty(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "idOrSlug")
	if idOrSlug == "" {
		ErrorJSON(w, http.StatusBadRequest, "Property ID or slug is required")
		return
	}

	existing, err := h.repo.GetPropertyByIDOrSlug(r.Context(), idOrSlug)
	if err != nil || existing == nil {
		ErrorJSON(w, http.StatusNotFound, "Property not found")
		return
	}

	var patch domain.PropertyRecord
	if err := ParseJSON(r, &patch); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "Invalid property update")
		return
	}

	// Merge fields onto existing
	if patch.Name != "" {
		existing.Name = patch.Name
	}
	if patch.City != "" {
		existing.City = patch.City
	}
	if patch.Address != "" {
		existing.Address = patch.Address
	}
	if patch.MapLink != nil {
		existing.MapLink = patch.MapLink
	}
	if patch.Phone != "" {
		existing.Phone = patch.Phone
	}
	if patch.WhatsApp != "" {
		existing.WhatsApp = patch.WhatsApp
	}
	if patch.Email != "" {
		existing.Email = patch.Email
	}
	if patch.BasePrice > 0 {
		existing.BasePrice = patch.BasePrice
	}
	if patch.Rating > 0 {
		existing.Rating = patch.Rating
	}
	if patch.WeekendSurchargePercent >= 0 {
		existing.WeekendSurchargePercent = patch.WeekendSurchargePercent
	}
	if patch.ExtraAdultPercent >= 0 {
		existing.ExtraAdultPercent = patch.ExtraAdultPercent
	}
	if patch.ChildPercent != nil {
		existing.ChildPercent = patch.ChildPercent
	}
	if patch.AdultFromAge != nil {
		existing.AdultFromAge = patch.AdultFromAge
	}
	if patch.ChildFreeUnderAge >= 0 {
		existing.ChildFreeUnderAge = patch.ChildFreeUnderAge
	}

	if err := h.repo.UpdateProperty(r.Context(), *existing); err != nil {
		ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("%s updated", existing.Name),
		"data":    existing,
	})
}

// UpdateRoomType handles PATCH /api/admin/room-types/:id (requireAdmin)
func (h *AdminHandler) UpdateRoomType(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		ErrorJSON(w, http.StatusBadRequest, "Room type ID is required")
		return
	}

	existing, err := h.repo.GetRoomTypeByID(r.Context(), id)
	if err != nil || existing == nil {
		ErrorJSON(w, http.StatusNotFound, "Room category not found")
		return
	}

	var patch domain.RoomTypeRecord
	if err := ParseJSON(r, &patch); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "Invalid room update")
		return
	}

	if patch.Name != "" {
		existing.Name = patch.Name
	}
	if patch.Description != "" {
		existing.Description = patch.Description
	}
	if patch.SizeSqft != "" {
		existing.SizeSqft = patch.SizeSqft
	}
	if patch.BedType != "" {
		existing.BedType = patch.BedType
	}
	if patch.MaxGuests > 0 {
		existing.MaxGuests = patch.MaxGuests
	}
	if patch.PriceOffset >= 0 {
		existing.PriceOffset = patch.PriceOffset
	}
	if patch.TotalUnits >= 0 {
		existing.TotalUnits = patch.TotalUnits
	}

	if err := h.repo.UpdateRoomType(r.Context(), *existing); err != nil {
		ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("%s updated", existing.Name),
		"data":    existing,
	})
}

type contentPayload struct {
	Entries map[string]string `json:"entries"`
}

// UpdateContent handles PUT /api/admin/content (requireAdmin)
func (h *AdminHandler) UpdateContent(w http.ResponseWriter, r *http.Request) {
	var payload contentPayload
	if err := ParseJSON(r, &payload); err != nil || payload.Entries == nil {
		ErrorJSON(w, http.StatusBadRequest, "Invalid content payload")
		return
	}

	saved, err := h.repo.SetSiteContent(r.Context(), payload.Entries)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "Failed to save content")
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Website content updated",
		"data":    saved,
	})
}

type surchargePayload struct {
	SurchargePercent float64 `json:"surchargePercent"`
	PropertyID       *string `json:"propertyId,omitempty"`
}

// Surcharge handles PATCH /api/admin/surcharge (requireAdmin)
func (h *AdminHandler) Surcharge(w http.ResponseWriter, r *http.Request) {
	var payload surchargePayload
	if err := ParseJSON(r, &payload); err != nil || payload.SurchargePercent < 0 {
		ErrorJSON(w, http.StatusBadRequest, "Valid surchargePercent (>= 0) is required")
		return
	}

	propID := "all"
	if payload.PropertyID != nil && *payload.PropertyID != "" {
		propID = *payload.PropertyID
	}

	_, _, err := h.repo.UpdateWeekendSurcharge(r.Context(), propID, payload.SurchargePercent)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "Failed to update weekend surcharge")
		return
	}

	updatedProps, _ := h.repo.GetPropertiesWithRooms(r.Context())
	scope := "all properties"
	if propID != "all" {
		scope = "selected property"
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Weekend surcharge updated to %.0f%% across %s", payload.SurchargePercent, scope),
		"data":    updatedProps,
	})
}

type adminPaymentLinkPayload struct {
	Phone        string  `json:"phone"`
	Amount       float64 `json:"amount"`
	Description  *string `json:"description,omitempty"`
	GuestName    *string `json:"guestName,omitempty"`
	PropertyName *string `json:"propertyName,omitempty"`
}

// PaymentLink handles POST /api/admin/payment-link (requireAdmin)
func (h *AdminHandler) PaymentLink(w http.ResponseWriter, r *http.Request) {
	var payload adminPaymentLinkPayload
	if err := ParseJSON(r, &payload); err != nil || payload.Phone == "" || payload.Amount <= 0 {
		ErrorJSON(w, http.StatusBadRequest, "Guest phone and positive amount are required")
		return
	}

	guestName := "Walk-in / Custom Guest"
	if payload.GuestName != nil && *payload.GuestName != "" {
		guestName = *payload.GuestName
	}

	propName := "Quadis Hotel"
	if payload.PropertyName != nil && *payload.PropertyName != "" {
		propName = *payload.PropertyName
	}

	desc := fmt.Sprintf("Instant admin payment link (%s)", propName)
	if payload.Description != nil && *payload.Description != "" {
		desc = *payload.Description
	}

	enq := domain.EnquiryRecord{
		EnquiryType: domain.EnquiryTypeRoomHold,
		GuestName:   guestName,
		GuestPhone:  payload.Phone,
		Message:      &desc,
		Status:       domain.EnquiryStatusNew,
	}

	if err := h.repo.CreateEnquiry(r.Context(), &enq); err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "Failed to record enquiry for payment link")
		return
	}

	linkRes, err := h.razorpayService.CreateEnquiryPaymentLink(r.Context(), payment.CreateEnquiryLinkInput{
		EnquiryID:   enq.ID,
		Amount:      payload.Amount,
		GuestName:   guestName,
		GuestPhone:  payload.Phone,
		Description: &desc,
	})
	if err != nil || (linkRes != nil && !linkRes.Success) {
		errMsg := "Failed to generate admin payment link"
		if linkRes != nil && linkRes.Error != "" {
			errMsg = linkRes.Error
		}
		ErrorJSON(w, http.StatusInternalServerError, errMsg)
		return
	}

	if linkRes.PaymentLinkID != "" {
		_, _ = h.repo.UpdateEnquiryStatus(r.Context(), enq.ID, domain.EnquiryStatusLinkSent, &linkRes.PaymentLinkID)
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Instant payment link generated and recorded",
		"data": map[string]interface{}{
			"enquiryId":     enq.ID,
			"paymentLinkId": linkRes.PaymentLinkID,
			"shortUrl":      linkRes.ShortURL,
			"isSimulated":   linkRes.IsSimulated,
		},
	})
}

type reorderImagesPayload struct {
	Order []string `json:"order"`
}

// ReorderImages handles PATCH /api/admin/properties/:idOrSlug/images/order (requireAdmin)
func (h *AdminHandler) ReorderImages(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "idOrSlug")
	var payload reorderImagesPayload
	if err := ParseJSON(r, &payload); err != nil || len(payload.Order) == 0 {
		ErrorJSON(w, http.StatusBadRequest, "Expected { order: [imageId, ...] }")
		return
	}

	prop, err := h.repo.GetPropertyByIDOrSlug(r.Context(), idOrSlug)
	if err != nil || prop == nil {
		ErrorJSON(w, http.StatusNotFound, "Property not found")
		return
	}

	if err := h.repo.ReorderPropertyImages(r.Context(), prop.ID, payload.Order); err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "Reorder failed")
		return
	}

	images, _ := h.repo.GetPropertyImages(r.Context(), prop.ID)
	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    images,
	})
}

// DeleteImage handles DELETE /api/admin/images/:id (requireAdmin)
func (h *AdminHandler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		ErrorJSON(w, http.StatusBadRequest, "Image ID is required")
		return
	}

	ok, err := h.repo.DeletePropertyImage(r.Context(), id)
	if err != nil || !ok {
		ErrorJSON(w, http.StatusNotFound, "Photo not found")
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Photo removed",
	})
}
