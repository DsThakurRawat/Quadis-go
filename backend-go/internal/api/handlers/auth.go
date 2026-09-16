package handlers

import (
	"net/http"
	"strings"

	"quadis-backend-go/internal/api/middleware"
	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
	"quadis-backend-go/internal/service/auth"
)

type AuthHandler struct {
	repo           repository.Repository
	sessionService *auth.SessionService
	cfg            *config.Config
}

func NewAuthHandler(
	repo repository.Repository,
	sessionService *auth.SessionService,
	cfg *config.Config,
) *AuthHandler {
	return &AuthHandler{
		repo:           repo,
		sessionService: sessionService,
		cfg:            cfg,
	}
}

type registerPayload struct {
	FullName string  `json:"fullName"`
	Email    string  `json:"email"`
	Phone    *string `json:"phone,omitempty"`
	Password string  `json:"password"`
}

type loginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func toPublicUser(u *domain.UserRecord) map[string]interface{} {
	return map[string]interface{}{
		"id":        u.ID,
		"full_name": u.FullName,
		"email":     u.Email,
		"phone":     u.Phone,
	}
}

// Register handles POST /api/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var payload registerPayload
	if err := ParseJSON(r, &payload); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	fieldErrors := make(map[string][]string)
	if len(strings.TrimSpace(payload.FullName)) < 2 {
		fieldErrors["fullName"] = []string{"Please enter your full name"}
	}
	if !strings.Contains(payload.Email, "@") || len(payload.Email) < 5 {
		fieldErrors["email"] = []string{"Enter a valid email address"}
	}
	if len(payload.Password) < 8 {
		fieldErrors["password"] = []string{"Password must be at least 8 characters"}
	}

	if len(fieldErrors) > 0 {
		ValidationErrorJSON(w, "Invalid registration details", fieldErrors)
		return
	}

	existing, err := h.repo.GetUserByEmail(r.Context(), payload.Email)
	if err == nil && existing != nil {
		ErrorJSON(w, http.StatusConflict, "An account with this email already exists")
		return
	}

	hash, err := auth.HashPassword(payload.Password)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "Could not hash password")
		return
	}

	user := domain.UserRecord{
		FullName:     payload.FullName,
		Email:        strings.ToLower(strings.TrimSpace(payload.Email)),
		Phone:        payload.Phone,
		PasswordHash: hash,
	}

	if err := h.repo.CreateUser(r.Context(), &user); err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "Could not create your account")
		return
	}

	token, err := h.sessionService.SignSession(user.ID, user.Email, nil, 0)
	if err != nil || token == "" {
		ErrorJSON(w, http.StatusServiceUnavailable, "Sign-in is not configured on this server")
		return
	}

	JSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"token":   token,
		"data":    toPublicUser(&user),
	})
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var payload loginPayload
	if err := ParseJSON(r, &payload); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "Enter your email and password")
		return
	}

	if strings.TrimSpace(payload.Email) == "" || strings.TrimSpace(payload.Password) == "" {
		ErrorJSON(w, http.StatusBadRequest, "Enter your email and password")
		return
	}

	user, err := h.repo.GetUserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(payload.Email)))
	ok := false
	if err == nil && user != nil {
		ok = auth.VerifyPassword(payload.Password, user.PasswordHash)
	}

	if !ok || user == nil {
		ErrorJSON(w, http.StatusUnauthorized, "Email or password is incorrect")
		return
	}

	token, err := h.sessionService.SignSession(user.ID, user.Email, nil, 0)
	if err != nil || token == "" {
		ErrorJSON(w, http.StatusServiceUnavailable, "Sign-in is not configured on this server")
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"token":   token,
		"data":    toPublicUser(user),
	})
}

// Me handles GET /api/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		ErrorJSON(w, http.StatusUnauthorized, "Not signed in")
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	session := h.sessionService.VerifySession(token)
	if session == nil {
		ErrorJSON(w, http.StatusUnauthorized, "Session expired")
		return
	}

	user, err := h.repo.GetUserByID(r.Context(), session.Sub)
	if err != nil || user == nil {
		ErrorJSON(w, http.StatusUnauthorized, "Account no longer exists")
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    toPublicUser(user),
	})
}

// Bookings handles GET /api/auth/bookings (requires user session)
func (h *AuthHandler) Bookings(w http.ResponseWriter, r *http.Request) {
	session := middleware.GetSession(r)
	if session == nil {
		ErrorJSON(w, http.StatusUnauthorized, "Not signed in")
		return
	}

	bookings, err := h.repo.GetBookingsForUser(r.Context(), session.Sub, session.Email)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "Could not load your bookings")
		return
	}

	enriched := make([]map[string]interface{}, 0, len(bookings))
	for _, b := range bookings {
		prop, _ := h.repo.GetPropertyByID(r.Context(), b.PropertyID)
		room, _ := h.repo.GetRoomTypeByID(r.Context(), b.RoomTypeID)

		propName := "Quadis Hotel"
		propAddr := ""
		propSlug := ""
		if prop != nil {
			propName = prop.Name
			propAddr = prop.Address
			propSlug = prop.Slug
		}

		roomName := ""
		if room != nil {
			roomName = room.Name
		}

		bMap := map[string]interface{}{
			"id":               b.ID,
			"booking_code":     b.BookingCode,
			"property_id":      b.PropertyID,
			"room_type_id":     b.RoomTypeID,
			"check_in":         b.CheckIn,
			"check_out":        b.CheckOut,
			"rooms_count":      b.RoomsCount,
			"adults_count":     b.AdultsCount,
			"guest_name":       b.GuestName,
			"guest_phone":      b.GuestPhone,
			"guest_email":      b.GuestEmail,
			"total_amount":     b.TotalAmount,
			"booking_status":   b.BookingStatus,
			"payment_status":   b.PaymentStatus,
			"created_at":       b.CreatedAt,
			"property_name":    propName,
			"property_address": propAddr,
			"property_slug":    propSlug,
			"room_type_name":   roomName,
		}
		enriched = append(enriched, bMap)
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"count":   len(enriched),
		"data":    enriched,
	})
}
