package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/domain"
)

type RazorpayOrderResponse struct {
	Success     bool   `json:"success"`
	IsSimulated bool   `json:"isSimulated,omitempty"`
	OrderID     string `json:"orderId,omitempty"`
	KeyID       string `json:"keyId,omitempty"`
	Amount      int64  `json:"amount,omitempty"`
	Currency    string `json:"currency,omitempty"`
	Error       string `json:"error,omitempty"`
}

type RazorpayPaymentLinkResponse struct {
	Success       bool   `json:"success"`
	IsSimulated   bool   `json:"isSimulated,omitempty"`
	PaymentLinkID string `json:"paymentLinkId,omitempty"`
	ShortURL      string `json:"shortUrl,omitempty"`
	Error         string `json:"error,omitempty"`
}

type CreateEnquiryLinkInput struct {
	EnquiryID   string
	Amount      float64
	GuestName   string
	GuestPhone  string
	GuestEmail  *string
	Description *string
}

type RazorpayService struct {
	cfg        *config.Config
	httpClient *http.Client
	isLiveMode bool
}

func NewRazorpayService(cfg *config.Config) *RazorpayService {
	liveKeyRequired := cfg.IsProduction()
	keyLooksLive := strings.HasPrefix(cfg.RazorpayKeyID, "rzp_live_")

	isLiveMode := false
	if cfg.RazorpayKeyID != "" && cfg.RazorpayKeyID != "mock" && cfg.RazorpayKeyID != "rzp_test_simulated" &&
		cfg.RazorpayKeySecret != "" && cfg.RazorpayKeySecret != "mock" &&
		(!liveKeyRequired || keyLooksLive) {
		isLiveMode = true
	}

	return &RazorpayService{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		isLiveMode: isLiveMode,
	}
}

func (s *RazorpayService) CreateOrder(ctx context.Context, booking *domain.BookingRecord) (*RazorpayOrderResponse, error) {
	amountInPaisa := int64(math.Round(booking.TotalAmount * 100))

	if s.isLiveMode {
		payload := map[string]interface{}{
			"amount":   amountInPaisa,
			"currency": "INR",
			"receipt":  booking.BookingCode,
			"notes": map[string]string{
				"bookingCode": booking.BookingCode,
				"guestPhone":  booking.GuestPhone,
				"propertyId":  booking.PropertyID,
			},
		}

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.razorpay.com/v1/orders", bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(s.cfg.RazorpayKeyID, s.cfg.RazorpayKeySecret)
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return &RazorpayOrderResponse{
				Success: false,
				Error:   fmt.Sprintf("Razorpay API network error: %v", err),
			}, nil
		}
		defer resp.Body.Close()

		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return &RazorpayOrderResponse{
				Success: false,
				Error:   fmt.Sprintf("Razorpay error status %d: %s", resp.StatusCode, string(respBytes)),
			}, nil
		}

		var apiResp struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(respBytes, &apiResp); err != nil {
			return nil, err
		}

		return &RazorpayOrderResponse{
			Success:     true,
			IsSimulated: false,
			OrderID:     apiResp.ID,
			KeyID:       s.cfg.RazorpayKeyID,
			Amount:      amountInPaisa,
			Currency:    "INR",
		}, nil
	}

	// Simulation fallback
	simulatedID := fmt.Sprintf("order_sim_%d_%d", time.Now().UnixMilli(), time.Now().Nanosecond()%1000)
	return &RazorpayOrderResponse{
		Success:     true,
		IsSimulated: true,
		OrderID:     simulatedID,
		KeyID:       "rzp_test_simulated_key",
		Amount:      amountInPaisa,
		Currency:    "INR",
	}, nil
}

func (s *RazorpayService) CreatePaymentLink(ctx context.Context, booking *domain.BookingRecord) (*RazorpayPaymentLinkResponse, error) {
	amountInPaisa := int64(math.Round(booking.TotalAmount * 100))

	if s.isLiveMode {
		email := "guest@quadishotels.com"
		if booking.GuestEmail != nil && *booking.GuestEmail != "" {
			email = *booking.GuestEmail
		}
		payload := map[string]interface{}{
			"amount":         amountInPaisa,
			"currency":       "INR",
			"accept_partial": false,
			"description":    fmt.Sprintf("Quadis Hotels Booking %s", booking.BookingCode),
			"customer": map[string]interface{}{
				"name":    booking.GuestName,
				"contact": booking.GuestPhone,
				"email":   email,
			},
			"notify": map[string]interface{}{
				"sms":   true,
				"email": booking.GuestEmail != nil && *booking.GuestEmail != "",
			},
			"reminder_enable": true,
			"notes": map[string]string{
				"bookingCode": booking.BookingCode,
			},
		}

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.razorpay.com/v1/payment_links", bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(s.cfg.RazorpayKeyID, s.cfg.RazorpayKeySecret)
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return &RazorpayPaymentLinkResponse{
				Success: false,
				Error:   fmt.Sprintf("Razorpay API error: %v", err),
			}, nil
		}
		defer resp.Body.Close()

		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return &RazorpayPaymentLinkResponse{
				Success: false,
				Error:   fmt.Sprintf("Razorpay error status %d: %s", resp.StatusCode, string(respBytes)),
			}, nil
		}

		var apiResp struct {
			ID       string `json:"id"`
			ShortURL string `json:"short_url"`
		}
		if err := json.Unmarshal(respBytes, &apiResp); err != nil {
			return nil, err
		}

		return &RazorpayPaymentLinkResponse{
			Success:       true,
			IsSimulated:   false,
			PaymentLinkID: apiResp.ID,
			ShortURL:      apiResp.ShortURL,
		}, nil
	}

	// Simulation fallback
	simLinkID := fmt.Sprintf("plink_sim_%d", time.Now().UnixMilli())
	simShortURL := fmt.Sprintf("https://checkout.quadishotels.com/pay/%s", booking.BookingCode)
	return &RazorpayPaymentLinkResponse{
		Success:       true,
		IsSimulated:   true,
		PaymentLinkID: simLinkID,
		ShortURL:      simShortURL,
	}, nil
}

func (s *RazorpayService) CreateEnquiryPaymentLink(ctx context.Context, input CreateEnquiryLinkInput) (*RazorpayPaymentLinkResponse, error) {
	amountInPaisa := int64(math.Round(input.Amount * 100))

	if s.isLiveMode {
		email := "guest@quadishotels.com"
		if input.GuestEmail != nil && *input.GuestEmail != "" {
			email = *input.GuestEmail
		}
		desc := fmt.Sprintf("Quadis Hotels Enquiry Deposit (%s)", input.EnquiryID)
		if input.Description != nil && *input.Description != "" {
			desc = *input.Description
		}

		payload := map[string]interface{}{
			"amount":         amountInPaisa,
			"currency":       "INR",
			"accept_partial": false,
			"description":    desc,
			"customer": map[string]interface{}{
				"name":    input.GuestName,
				"contact": input.GuestPhone,
				"email":   email,
			},
			"notify": map[string]interface{}{
				"sms":   true,
				"email": input.GuestEmail != nil && *input.GuestEmail != "",
			},
			"reminder_enable": true,
			"notes": map[string]string{
				"enquiryId": input.EnquiryID,
			},
		}

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.razorpay.com/v1/payment_links", bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(s.cfg.RazorpayKeyID, s.cfg.RazorpayKeySecret)
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return &RazorpayPaymentLinkResponse{
				Success: false,
				Error:   fmt.Sprintf("Razorpay API error: %v", err),
			}, nil
		}
		defer resp.Body.Close()

		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return &RazorpayPaymentLinkResponse{
				Success: false,
				Error:   fmt.Sprintf("Razorpay error status %d: %s", resp.StatusCode, string(respBytes)),
			}, nil
		}

		var apiResp struct {
			ID       string `json:"id"`
			ShortURL string `json:"short_url"`
		}
		if err := json.Unmarshal(respBytes, &apiResp); err != nil {
			return nil, err
		}

		return &RazorpayPaymentLinkResponse{
			Success:       true,
			IsSimulated:   false,
			PaymentLinkID: apiResp.ID,
			ShortURL:      apiResp.ShortURL,
		}, nil
	}

	// Simulation fallback
	simLinkID := fmt.Sprintf("plink_enq_sim_%d", time.Now().UnixMilli())
	simShortURL := fmt.Sprintf("https://checkout.quadishotels.com/pay-enquiry/%s", input.EnquiryID)
	return &RazorpayPaymentLinkResponse{
		Success:       true,
		IsSimulated:   true,
		PaymentLinkID: simLinkID,
		ShortURL:      simShortURL,
	}, nil
}

func (s *RazorpayService) VerifyWebhookSignature(rawBody []byte, signature string) bool {
	secret := s.cfg.RazorpayWebhookSecret
	if secret == "" {
		if s.cfg.IsProduction() {
			return false // Fail closed in production
		}
		secret = "secret_simulated_webhook"
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return subtle.ConstantTimeCompare([]byte(expectedSignature), []byte(signature)) == 1
}
