package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"quadis-backend-go/internal/api"
	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/repository/memory"
	"quadis-backend-go/internal/service/ai"
	"quadis-backend-go/internal/service/auth"
	"quadis-backend-go/internal/service/booking"
	"quadis-backend-go/internal/service/ota"
	"quadis-backend-go/internal/service/payment"
)

func setupTestRouter() (http.Handler, *config.Config) {
	cfg := &config.Config{
		NodeEnv:               "test",
		Port:                  3001,
		AdminPIN:              "998877",
		AdminPassword:         "test-admin-secret",
		SessionSecret:         "test-session-secret-key-at-least-32-chars-long",
		RazorpayKeyID:         "rzp_test_simulated",
		RazorpayKeySecret:     "rzp_test_secret",
		RazorpayWebhookSecret: "secret_simulated_webhook",
		OwnerPhone:            "919217373532",
		ResAvenueUsername:      "resavenue_user",
		ResAvenuePassword:      "resavenue_pass",
		ResAvenueTarget:        "Test",
	}

	repo := memory.NewMemoryStore()
	sessionService := auth.NewSessionService(cfg)
	bookingService := booking.NewBookingService(repo)
	razorpayService := payment.NewRazorpayService(cfg)
	otaService := ota.NewOTAService(repo, cfg)
	aiService := ai.NewAIService(repo, cfg)

	router := api.NewRouter(api.RouterDeps{
		Config:          cfg,
		Repo:            repo,
		SessionService:  sessionService,
		BookingService:  bookingService,
		RazorpayService: razorpayService,
		OTAService:      otaService,
		AIService:       aiService,
	})

	return router, cfg
}

func TestHealthEndpoints(t *testing.T) {
	router, _ := setupTestRouter()

	// 1. GET /healthz
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /healthz, got %d", rr.Code)
	}
	if rr.Body.String() != "ok\n" {
		t.Fatalf("expected 'ok\\n', got %q", rr.Body.String())
	}

	// 2. GET /api/health
	req, _ = http.NewRequest(http.MethodGet, "/api/health", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/health, got %d", rr.Code)
	}
	var res map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode /api/health JSON: %v", err)
	}
	if res["status"] != "healthy" {
		t.Fatalf("expected status healthy, got %v", res["status"])
	}
}

func TestPropertiesEndpoints(t *testing.T) {
	router, _ := setupTestRouter()

	// GET /api/properties
	req, _ := http.NewRequest(http.MethodGet, "/api/properties", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var res struct {
		Success bool          `json:"success"`
		Count   int           `json:"count"`
		Data    []interface{} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !res.Success || res.Count != 10 {
		t.Fatalf("expected 10 seeded properties, got %d", res.Count)
	}

	// GET /api/properties/hotel-amaltas-international
	req, _ = http.NewRequest(http.MethodGet, "/api/properties/hotel-amaltas-international", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for Hotel Amaltas, got %d: %s", rr.Code, rr.Body.String())
	}
	var detail struct {
		Success bool `json:"success"`
		Data    struct {
			Name  string        `json:"name"`
			City  string        `json:"city"`
			Rooms []interface{} `json:"rooms"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if detail.Data.Name != "Hotel Amaltas International" {
		t.Fatalf("expected Hotel Amaltas International, got %s", detail.Data.Name)
	}
	if detail.Data.City != "New Delhi" {
		t.Fatalf("expected New Delhi, got %s", detail.Data.City)
	}
}

func TestBookingAndPaymentLifecycle(t *testing.T) {
	router, _ := setupTestRouter()

	// 1. Initiate Hold
	checkIn := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	checkOut := time.Now().AddDate(0, 0, 9).Format("2006-01-02")
	payload := map[string]interface{}{
		"propertySlug": "hotel-amaltas-international",
		"roomTypeSlug": "superior-room",
		"checkIn":      checkIn,
		"checkOut":     checkOut,
		"roomsCount":   1,
		"guestsCount":  2,
		"adultsCount":  2,
		"guestName":    "Test Guest",
		"guestPhone":   "9876543210",
		"guestEmail":   "test@example.com",
		"mealPlan":     "All Meals Included",
	}
	bodyBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/bookings/initiate", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
	}

	var holdRes struct {
		Success bool `json:"success"`
		Data    struct {
			BookingCode string  `json:"booking_code"`
			TotalAmount float64 `json:"total_amount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &holdRes); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	bookingCode := holdRes.Data.BookingCode
	if bookingCode == "" {
		t.Fatalf("empty booking code returned")
	}

	// 2. Lookup Booking Status
	req, _ = http.NewRequest(http.MethodGet, "/api/bookings/"+bookingCode, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for lookup, got %d", rr.Code)
	}

	// 3. Create Razorpay Order
	orderPayload := map[string]string{"bookingCode": bookingCode}
	orderBytes, _ := json.Marshal(orderPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/payments/create-order", bytes.NewReader(orderBytes))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for create-order, got %d: %s", rr.Code, rr.Body.String())
	}

	// 4. Simulate Razorpay Webhook Confirmation
	webhookPayload := map[string]interface{}{
		"event":       "order.paid",
		"bookingCode": bookingCode,
		"entity": map[string]interface{}{
			"id": "pay_test_12345",
		},
	}
	whBytes, _ := json.Marshal(webhookPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/webhooks/razorpay", bytes.NewReader(whBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Razorpay-Signature", "simulated")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for webhook, got %d: %s", rr.Code, rr.Body.String())
	}

	// 5. Download Invoice PDF
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/bookings/%s/invoice?phone=9876543210", bookingCode), nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for invoice, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("expected application/pdf, got %s", rr.Header().Get("Content-Type"))
	}
	if len(rr.Body.Bytes()) < 500 {
		t.Fatalf("invoice PDF suspiciously small (%d bytes)", len(rr.Body.Bytes()))
	}
}

func TestAuthAndAdminFlows(t *testing.T) {
	router, cfg := setupTestRouter()

	// 1. Guest Registration
	regPayload := map[string]string{
		"fullName": "Test User",
		"email":    "user@example.com",
		"password": "Password123!",
		"phone":    "9123456789",
	}
	rb, _ := json.Marshal(regPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(rb))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for register, got %d: %s", rr.Code, rr.Body.String())
	}

	var regRes struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &regRes)
	guestToken := regRes.Token

	// 2. Guest /me
	req, _ = http.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+guestToken)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /me, got %d", rr.Code)
	}

	// 3. Admin Sign-In with Bootstrap PIN
	adminAuthPayload := map[string]string{"pin": cfg.AdminPIN}
	ab, _ := json.Marshal(adminAuthPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/admin/auth", bytes.NewReader(ab))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin auth, got %d: %s", rr.Code, rr.Body.String())
	}
	var adminRes struct {
		Token         string `json:"token"`
		MustChangePin bool   `json:"mustChangePin"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &adminRes)
	adminToken := adminRes.Token

	// 4. Admin Dashboard
	req, _ = http.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin dashboard, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestOTAIntegrationFlow(t *testing.T) {
	router, _ := setupTestRouter()

	// 1. GET /api/ota/codes with Basic Auth
	req, _ := http.NewRequest(http.MethodGet, "/api/ota/codes", nil)
	req.SetBasicAuth("resavenue_user", "resavenue_pass")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for ota codes, got %d: %s", rr.Code, rr.Body.String())
	}

	// 2. Dispatch OTA_HotelDetailsRQ
	detailsPayload := map[string]interface{}{
		"OTA_HotelDetailsRQ": map[string]interface{}{
			"POS": map[string]interface{}{
				"Username": "resavenue_user",
				"Password": "resavenue_pass",
			},
			"HotelCode": 11, // Hotel Amaltas International
		},
	}
	dbBytes, _ := json.Marshal(detailsPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/ota/resavenue", bytes.NewReader(dbBytes))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for OTA dispatch, got %d: %s", rr.Code, rr.Body.String())
	}
	var detailsRes struct {
		OTA_HotelDetailsRS struct {
			Status string `json:"Status"`
			Hotel  struct {
				HotelCode int `json:"HotelCode"`
			} `json:"Hotel"`
		} `json:"OTA_HotelDetailsRS"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &detailsRes); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if detailsRes.OTA_HotelDetailsRS.Status != "Success" {
		t.Fatalf("expected Success status, got %s, body: %s", detailsRes.OTA_HotelDetailsRS.Status, rr.Body.String())
	}
	if detailsRes.OTA_HotelDetailsRS.Hotel.HotelCode != 11 {
		t.Fatalf("expected HotelCode 11, got %d", detailsRes.OTA_HotelDetailsRS.Hotel.HotelCode)
	}
}
