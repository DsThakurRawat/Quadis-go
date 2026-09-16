package handlers

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
	"quadis-backend-go/internal/service/auth"
	"quadis-backend-go/internal/service/booking"
	"quadis-backend-go/internal/service/invoice"
)

var isoDateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type BookingsHandler struct {
	repo           repository.Repository
	bookingService *booking.BookingService
	sessionService *auth.SessionService
	cfg            *config.Config
}

func NewBookingsHandler(
	repo repository.Repository,
	bookingService *booking.BookingService,
	sessionService *auth.SessionService,
	cfg *config.Config,
) *BookingsHandler {
	return &BookingsHandler{
		repo:           repo,
		bookingService: bookingService,
		sessionService: sessionService,
		cfg:            cfg,
	}
}

type initiateBookingPayload struct {
	PropertySlug string   `json:"propertySlug"`
	RoomTypeSlug string   `json:"roomTypeSlug"`
	CheckIn      string   `json:"checkIn"`
	CheckOut     string   `json:"checkOut"`
	RoomsCount   int      `json:"roomsCount"`
	GuestsCount  int      `json:"guestsCount"`
	AdultsCount  *int     `json:"adultsCount,omitempty"`
	ChildAges    []int    `json:"childAges,omitempty"`
	GuestName    string   `json:"guestName"`
	GuestPhone   string   `json:"guestPhone"`
	GuestEmail   *string  `json:"guestEmail,omitempty"`
	CompanyName  *string  `json:"companyName,omitempty"`
	GSTIN        *string  `json:"gstin,omitempty"`
	MealPlan     *string  `json:"mealPlan,omitempty"`
}

// InitiateBooking handles POST /api/bookings/initiate
func (h *BookingsHandler) InitiateBooking(w http.ResponseWriter, r *http.Request) {
	var payload initiateBookingPayload
	if err := ParseJSON(r, &payload); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	fieldErrors := make(map[string][]string)

	if strings.TrimSpace(payload.PropertySlug) == "" {
		fieldErrors["propertySlug"] = []string{"Property slug is required"}
	}
	if strings.TrimSpace(payload.RoomTypeSlug) == "" {
		fieldErrors["roomTypeSlug"] = []string{"Room category slug is required"}
	}
	if !isoDateRegex.MatchString(payload.CheckIn) {
		fieldErrors["checkIn"] = []string{"Check-in must be YYYY-MM-DD format"}
	}
	if !isoDateRegex.MatchString(payload.CheckOut) {
		fieldErrors["checkOut"] = []string{"Check-out must be YYYY-MM-DD format"}
	}

	today := time.Now().Format("2006-01-02")
	if payload.CheckIn != "" && payload.CheckIn < today {
		fieldErrors["checkIn"] = []string{"Check-in date cannot be in the past"}
	}
	if payload.CheckIn != "" && payload.CheckOut != "" && payload.CheckOut <= payload.CheckIn {
		fieldErrors["checkOut"] = []string{"Check-out date must be strictly after check-in date"}
	}

	if payload.RoomsCount < 1 || payload.RoomsCount > 20 {
		fieldErrors["roomsCount"] = []string{"Rooms count must be between 1 and 20"}
	}
	if payload.GuestsCount < 1 || payload.GuestsCount > 80 {
		fieldErrors["guestsCount"] = []string{"Guests count must be between 1 and 80"}
	}

	if payload.AdultsCount != nil || len(payload.ChildAges) > 0 {
		adults := 0
		if payload.AdultsCount != nil {
			adults = *payload.AdultsCount
		}
		if adults+len(payload.ChildAges) != payload.GuestsCount {
			fieldErrors["guestsCount"] = []string{"Adults plus children must equal the total number of guests"}
		}
	}

	if len(strings.TrimSpace(payload.GuestName)) < 2 {
		fieldErrors["guestName"] = []string{"Guest name must be at least 2 characters"}
	}
	if len(strings.TrimSpace(payload.GuestPhone)) < 10 {
		fieldErrors["guestPhone"] = []string{"Valid 10-digit mobile number required"}
	}

	if len(fieldErrors) > 0 {
		ValidationErrorJSON(w, "Invalid request payload", fieldErrors)
		return
	}

	// Resolve property and room type
	prop, rooms, err := h.repo.GetPropertyBySlug(r.Context(), payload.PropertySlug)
	if err != nil || prop == nil {
		// Try by ID
		prop, err = h.repo.GetPropertyByIDOrSlug(r.Context(), payload.PropertySlug)
		if err != nil || prop == nil {
			ErrorJSON(w, http.StatusBadRequest, fmt.Sprintf("Property not found: %s", payload.PropertySlug))
			return
		}
		rooms, _ = h.repo.GetRoomTypesByPropertyID(r.Context(), prop.ID)
	}

	var targetRoom *domain.RoomTypeRecord
	for _, room := range rooms {
		if room.Slug == payload.RoomTypeSlug || room.ID == payload.RoomTypeSlug {
			targetRoom = &room
			break
		}
	}
	if targetRoom == nil {
		ErrorJSON(w, http.StatusBadRequest, fmt.Sprintf("Room category %s not found for property %s", payload.RoomTypeSlug, prop.Name))
		return
	}

	// Check optional auth session
	var userID *string
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if session := h.sessionService.VerifySession(token); session != nil {
			userID = &session.Sub
		}
	}

	adultsCount := payload.GuestsCount
	if payload.AdultsCount != nil && *payload.AdultsCount > 0 {
		adultsCount = *payload.AdultsCount
	}

	var mealPlan *domain.MealPlan
	if payload.MealPlan != nil && *payload.MealPlan != "" {
		mp := domain.MealPlan(*payload.MealPlan)
		mealPlan = &mp
	}

	record, _, err := h.bookingService.CreateHold(r.Context(), booking.CreateHoldRequest{
		UserID:      userID,
		PropertyID:  prop.ID,
		RoomTypeID:  targetRoom.ID,
		GuestName:   payload.GuestName,
		GuestPhone:  payload.GuestPhone,
		GuestEmail:  payload.GuestEmail,
		CompanyName: payload.CompanyName,
		GSTIN:       payload.GSTIN,
		CheckIn:     payload.CheckIn,
		CheckOut:    payload.CheckOut,
		RoomsCount:  payload.RoomsCount,
		AdultsCount: adultsCount,
		ChildAges:   payload.ChildAges,
		MealPlan:    mealPlan,
	})

	if err != nil {
		ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	JSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Booking soft hold created successfully for 15 minutes. Complete payment to confirm.",
		"data":    record,
	})
}

// GetBookingByCode handles GET /api/bookings/:code
func (h *BookingsHandler) GetBookingByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		ErrorJSON(w, http.StatusBadRequest, "Booking code is required")
		return
	}

	var phone *string
	qPhone := strings.TrimSpace(r.URL.Query().Get("phone"))
	if qPhone != "" {
		phone = &qPhone
	}

	bookingRecord, err := h.repo.GetBookingByCode(r.Context(), code, phone)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if bookingRecord == nil {
		ErrorJSON(w, http.StatusNotFound, "Booking not found or phone number mismatch")
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    bookingRecord,
	})
}

// GetInvoice handles GET /api/bookings/:code/invoice
func (h *BookingsHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		ErrorJSON(w, http.StatusBadRequest, "Booking code is required")
		return
	}

	bookingRecord, err := h.repo.GetBookingByCode(r.Context(), code, nil)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if bookingRecord == nil {
		ErrorJSON(w, http.StatusNotFound, "Booking not found")
		return
	}

	// Verify ownership: matching phone, user session, or admin token
	authHeader := r.Header.Get("Authorization")
	bearer := ""
	if strings.HasPrefix(authHeader, "Bearer ") {
		bearer = strings.TrimPrefix(authHeader, "Bearer ")
	}

	var session *auth.SessionPayload
	if bearer != "" {
		session = h.sessionService.VerifySession(bearer)
	}

	nonDigits := regexp.MustCompile(`\D`)
	queryPhone := nonDigits.ReplaceAllString(r.URL.Query().Get("phone"), "")
	bookingPhone := nonDigits.ReplaceAllString(bookingRecord.GuestPhone, "")

	ownsByPhone := queryPhone != "" && len(queryPhone) >= 10 && queryPhone == bookingPhone
	ownsBySession := session != nil && bookingRecord.UserID != nil && session.Sub == *bookingRecord.UserID

	adminPass := h.cfg.AdminPassword
	isAdmin := (adminPass != "" && bearer != "" && subtle.ConstantTimeCompare([]byte(bearer), []byte(adminPass)) == 1) ||
		(session != nil && session.Role != nil && *session.Role == "admin")

	if !ownsByPhone && !ownsBySession && !isAdmin {
		ErrorJSON(w, http.StatusForbidden, "Provide the phone number on the booking, or sign in, to download this invoice")
		return
	}

	prop, _ := h.repo.GetPropertyByID(r.Context(), bookingRecord.PropertyID)
	room, _ := h.repo.GetRoomTypeByID(r.Context(), bookingRecord.RoomTypeID)

	pdfBytes, err := invoice.GenerateGstInvoicePdf(bookingRecord, prop, room)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate invoice PDF: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="Quadis-Invoice-%s.pdf"`, bookingRecord.BookingCode))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}
