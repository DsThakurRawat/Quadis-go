package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"quadis-backend-go/internal/api/middleware"
	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
	"quadis-backend-go/internal/service/ota"
	"quadis-backend-go/internal/service/payment"
)

var nonDigitsRegex = regexp.MustCompile(`\D`)

type WebhooksHandler struct {
	repo            repository.Repository
	razorpayService *payment.RazorpayService
	otaService      *ota.OTAService
	cfg             *config.Config
}

func NewWebhooksHandler(
	repo repository.Repository,
	razorpayService *payment.RazorpayService,
	otaService *ota.OTAService,
	cfg *config.Config,
) *WebhooksHandler {
	return &WebhooksHandler{
		repo:            repo,
		razorpayService: razorpayService,
		otaService:      otaService,
		cfg:             cfg,
	}
}

// RazorpayWebhook handles POST /api/webhooks/razorpay
func (h *WebhooksHandler) RazorpayWebhook(w http.ResponseWriter, r *http.Request) {
	sig := r.Header.Get("X-Razorpay-Signature")
	rawBody := middleware.GetRawBody(r)

	simAllowed := !h.cfg.IsProduction()
	isSim := simAllowed && (sig == "simulated" || r.Header.Get("X-Simulated-Webhook") == "true")

	if !isSim {
		if sig == "" {
			ErrorJSON(w, http.StatusUnauthorized, "Missing Razorpay webhook signature")
			return
		}
		if !h.razorpayService.VerifyWebhookSignature([]byte(rawBody), sig) {
			ErrorJSON(w, http.StatusUnauthorized, "Invalid Razorpay webhook signature")
			return
		}
	}

	var eventData map[string]interface{}
	if err := json.Unmarshal([]byte(rawBody), &eventData); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	eventType, _ := eventData["event"].(string)
	if eventType == "" && isSim {
		eventType, _ = eventData["simulated_event"].(string)
	}
	if eventType == "" {
		ErrorJSON(w, http.StatusBadRequest, "Webhook payload missing event identifier")
		return
	}

	// Extract entity
	entity := make(map[string]interface{})
	if p, ok := eventData["payload"].(map[string]interface{}); ok {
		if pm, ok := p["payment"].(map[string]interface{}); ok {
			if e, ok := pm["entity"].(map[string]interface{}); ok {
				entity = e
			}
		} else if ord, ok := p["order"].(map[string]interface{}); ok {
			if e, ok := ord["entity"].(map[string]interface{}); ok {
				entity = e
			}
		}
	} else if e, ok := eventData["entity"].(map[string]interface{}); ok {
		entity = e
	} else if d, ok := eventData["data"].(map[string]interface{}); ok {
		entity = d
	}

	notes, _ := entity["notes"].(map[string]interface{})
	bookingCode, _ := notes["bookingCode"].(string)
	if bookingCode == "" {
		bookingCode, _ = eventData["bookingCode"].(string)
	}

	orderID, _ := entity["order_id"].(string)
	if orderID == "" {
		orderID, _ = eventData["orderId"].(string)
	}

	paymentID, _ := entity["id"].(string)
	if paymentID == "" {
		paymentID, _ = eventData["paymentId"].(string)
	}
	if paymentID == "" {
		paymentID = fmt.Sprintf("pay_sim_%d", time.Now().UnixMilli())
	}

	var booking *domain.BookingRecord

	if bookingCode != "" {
		booking, _ = h.repo.GetBookingByCode(r.Context(), bookingCode, nil)
	}

	if booking == nil && orderID != "" {
		// Try finding booking by orderID
		all, _ := h.repo.GetAllBookings(r.Context(), 100)
		for _, b := range all {
			if b.RazorpayOrderID != nil && *b.RazorpayOrderID == orderID {
				booking = &b
				break
			}
		}
	}

	if booking == nil {
		ErrorJSON(w, http.StatusNotFound, "Target booking not found for webhook event")
		return
	}

	if eventType == "order.paid" || eventType == "payment.captured" {
		// Idempotency: if already confirmed and paid, ignore
		if booking.BookingStatus == domain.BookingStatusConfirmed && booking.PaymentStatus == domain.PaymentStatusPaid {
			JSON(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"message": "Booking already confirmed; webhook ignored as duplicate",
			})
			return
		}

		// If booking expired or cancelled, do not confirm
		if booking.BookingStatus == domain.BookingStatusCancelled || booking.BookingStatus == domain.BookingStatusExpired {
			_, _ = h.repo.UpdateBookingPayment(r.Context(), booking.BookingCode, &paymentID, nil, booking.BookingStatus, domain.PaymentStatusPaid)
			log.Printf("[webhook] Payment %s captured for %s which is %s. Flagged for manual review.", paymentID, booking.BookingCode, booking.BookingStatus)
			JSON(w, http.StatusConflict, map[string]interface{}{
				"success": false,
				"error":   "Booking is no longer held; payment recorded and flagged for manual review",
			})
			return
		}

		updated, err := h.repo.UpdateBookingPayment(r.Context(), booking.BookingCode, &paymentID, nil, domain.BookingStatusConfirmed, domain.PaymentStatusPaid)
		if err != nil {
			ErrorJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		if updated != nil && h.otaService != nil {
			go func(b domain.BookingRecord) {
				_ = h.otaService.PushBooking(context.Background(), &b)
			}(*updated)
		}

		JSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Booking confirmed and WhatsApp receipts dispatched",
		})
		return
	}

	if eventType == "payment.failed" {
		if booking.BookingStatus == domain.BookingStatusCancelled || booking.BookingStatus == domain.BookingStatusExpired {
			JSON(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"message": "Booking already released; webhook ignored as duplicate",
			})
			return
		}

		_, _ = h.repo.UpdateBookingPayment(r.Context(), booking.BookingCode, &paymentID, nil, domain.BookingStatusCancelled, domain.PaymentStatusFailed)
		JSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Payment failed, room hold released back to inventory",
		})
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Webhook event %s processed", eventType),
	})
}

type whatsappPayload struct {
	From string `json:"from"`
	Text string `json:"text"`
}

// WhatsAppStaff handles POST /api/webhooks/whatsapp-staff
func (h *WebhooksHandler) WhatsAppStaff(w http.ResponseWriter, r *http.Request) {
	var payload whatsappPayload
	if err := ParseJSON(r, &payload); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	text := strings.ToLower(strings.TrimSpace(payload.Text))
	if text == "" {
		ErrorJSON(w, http.StatusBadRequest, "Message text is required")
		return
	}

	ownerPhone := h.cfg.OwnerPhone
	if ownerPhone == "" {
		ownerPhone = "919217373532"
	}
	ownerPhoneDigits := nonDigitsRegex.ReplaceAllString(ownerPhone, "")
	fromDigits := nonDigitsRegex.ReplaceAllString(payload.From, "")

	isAuthorized := h.cfg.NodeEnv == "test" || (len(fromDigits) > 0 && strings.Contains(fromDigits, ownerPhoneDigits))
	if !isAuthorized {
		ErrorJSON(w, http.StatusForbidden, "Unauthorized WhatsApp staff phone number")
		return
	}

	if strings.HasPrefix(text, "glance") || strings.HasPrefix(text, "status") {
		metrics, err := h.repo.GetGlanceMetrics(r.Context())
		if err != nil {
			ErrorJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		todayRev := 0.0
		if rev, ok := metrics["todayRevenue"].(float64); ok {
			todayRev = rev
		}

		reply := fmt.Sprintf("📊 *Quadis Hotels Daily Glance*\n• Today's Check-ins: *%v*\n• Pending Holds: *%v*\n• Pending Enquiries: *%v*\n• Today's Revenue: *₹%.0f*",
			metrics["todayCheckIns"],
			metrics["pendingHolds"],
			metrics["pendingEnquiries"],
			todayRev,
		)

		JSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"reply":   reply,
		})
		return
	}

	if strings.HasPrefix(text, "sold out") || strings.HasPrefix(text, "available") {
		isAvailable := strings.HasPrefix(text, "available")
		keyword := text
		if isAvailable {
			keyword = strings.TrimSpace(strings.TrimPrefix(text, "available"))
		} else {
			keyword = strings.TrimSpace(strings.TrimPrefix(text, "sold out"))
		}

		ok, msg, err := h.repo.ToggleRoomAvailability(r.Context(), "", keyword, isAvailable)
		if err != nil || !ok {
			JSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"reply":   fmt.Sprintf("❌ Could not match room/hotel from command \"%s\". Example: \"Sold out Deluxe Sector 51\"", keyword),
			})
			return
		}

		statusWord := "SOLD OUT"
		if isAvailable {
			statusWord = "AVAILABLE"
		}
		reply := fmt.Sprintf("✅ Marked *%s* as *%s*.", msg, statusWord)
		JSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"reply":   reply,
		})
		return
	}

	helpReply := `🤖 Quadis Staff Bot Commands:
• *Status* or *Glance* — view daily summary
• *Sold out <room/hotel>* — lock room inventory
• *Available <room/hotel>* — unlock room inventory`

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"reply":   helpReply,
	})
}
