package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
	"quadis-backend-go/internal/service/payment"
)

const maxEnquiryLinkAmount = 500000.0

type PaymentsHandler struct {
	repo            repository.Repository
	razorpayService *payment.RazorpayService
}

func NewPaymentsHandler(repo repository.Repository, razorpayService *payment.RazorpayService) *PaymentsHandler {
	return &PaymentsHandler{
		repo:            repo,
		razorpayService: razorpayService,
	}
}

type bookingCodePayload struct {
	BookingCode string `json:"bookingCode"`
}

// CreateOrder handles POST /api/payments/create-order
func (h *PaymentsHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var payload bookingCodePayload
	if err := ParseJSON(r, &payload); err != nil || strings.TrimSpace(payload.BookingCode) == "" {
		ValidationErrorJSON(w, "Invalid request payload", map[string][]string{
			"bookingCode": {"Booking code is required"},
		})
		return
	}

	booking, err := h.repo.GetBookingByCode(r.Context(), payload.BookingCode, nil)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if booking == nil {
		ErrorJSON(w, http.StatusNotFound, "Booking not found")
		return
	}

	if booking.BookingStatus != domain.BookingStatusPendingPayment {
		ErrorJSON(w, http.StatusBadRequest, fmt.Sprintf("Cannot create payment order for booking with status: %s", booking.BookingStatus))
		return
	}

	orderRes, err := h.razorpayService.CreateOrder(r.Context(), booking)
	if err != nil || (orderRes != nil && !orderRes.Success) {
		errMsg := "Failed to generate Razorpay order"
		if orderRes != nil && orderRes.Error != "" {
			errMsg = orderRes.Error
		} else if err != nil {
			errMsg = err.Error()
		}
		ErrorJSON(w, http.StatusInternalServerError, errMsg)
		return
	}

	if orderRes.OrderID != "" {
		_, _ = h.repo.UpdateBookingPayment(r.Context(), booking.BookingCode, nil, &orderRes.OrderID, booking.BookingStatus, booking.PaymentStatus)
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Razorpay checkout order initialized",
		"data": map[string]interface{}{
			"orderId":     orderRes.OrderID,
			"keyId":       orderRes.KeyID,
			"amount":      orderRes.Amount,
			"currency":    orderRes.Currency,
			"isSimulated": orderRes.IsSimulated,
		},
	})
}

// PaymentLink handles POST /api/payments/payment-link
func (h *PaymentsHandler) PaymentLink(w http.ResponseWriter, r *http.Request) {
	var payload bookingCodePayload
	if err := ParseJSON(r, &payload); err != nil || strings.TrimSpace(payload.BookingCode) == "" {
		ErrorJSON(w, http.StatusBadRequest, "Booking code is required")
		return
	}

	booking, err := h.repo.GetBookingByCode(r.Context(), payload.BookingCode, nil)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if booking == nil {
		ErrorJSON(w, http.StatusNotFound, "Booking not found")
		return
	}

	linkRes, err := h.razorpayService.CreatePaymentLink(r.Context(), booking)
	if err != nil || (linkRes != nil && !linkRes.Success) {
		errMsg := "Failed to create payment link"
		if linkRes != nil && linkRes.Error != "" {
			errMsg = linkRes.Error
		} else if err != nil {
			errMsg = err.Error()
		}
		ErrorJSON(w, http.StatusInternalServerError, errMsg)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Payment link generated successfully",
		"data": map[string]interface{}{
			"paymentLinkId": linkRes.PaymentLinkID,
			"shortUrl":      linkRes.ShortURL,
			"isSimulated":   linkRes.IsSimulated,
		},
	})
}

type enquiryPaymentLinkPayload struct {
	EnquiryID   string  `json:"enquiryId"`
	Amount      float64 `json:"amount"`
	Description *string `json:"description,omitempty"`
}

// EnquiryPaymentLink handles POST /api/payments/enquiry-payment-link (requireAdmin)
func (h *PaymentsHandler) EnquiryPaymentLink(w http.ResponseWriter, r *http.Request) {
	var payload enquiryPaymentLinkPayload
	if err := ParseJSON(r, &payload); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	fieldErrors := make(map[string][]string)
	if strings.TrimSpace(payload.EnquiryID) == "" {
		fieldErrors["enquiryId"] = []string{"Enquiry ID is required"}
	}
	if payload.Amount <= 0 {
		fieldErrors["amount"] = []string{"Amount must be positive"}
	} else if payload.Amount > maxEnquiryLinkAmount {
		fieldErrors["amount"] = []string{fmt.Sprintf("Amount must not exceed ₹%.0f", maxEnquiryLinkAmount)}
	}

	if len(fieldErrors) > 0 {
		ValidationErrorJSON(w, "Invalid enquiry payment link parameters", fieldErrors)
		return
	}

	enquiry, err := h.repo.GetEnquiryByID(r.Context(), payload.EnquiryID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if enquiry == nil {
		ErrorJSON(w, http.StatusNotFound, "Enquiry not found")
		return
	}

	linkRes, err := h.razorpayService.CreateEnquiryPaymentLink(r.Context(), payment.CreateEnquiryLinkInput{
		EnquiryID:   enquiry.ID,
		Amount:      payload.Amount,
		GuestName:   enquiry.GuestName,
		GuestPhone:  enquiry.GuestPhone,
		GuestEmail:  enquiry.GuestEmail,
		Description: payload.Description,
	})
	if err != nil || (linkRes != nil && !linkRes.Success) {
		errMsg := "Failed to create enquiry payment link"
		if linkRes != nil && linkRes.Error != "" {
			errMsg = linkRes.Error
		} else if err != nil {
			errMsg = err.Error()
		}
		ErrorJSON(w, http.StatusInternalServerError, errMsg)
		return
	}

	if linkRes.PaymentLinkID != "" {
		_, _ = h.repo.UpdateEnquiryStatus(r.Context(), enquiry.ID, domain.EnquiryStatusLinkSent, &linkRes.PaymentLinkID)
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Enquiry payment link generated and status updated to LINK_SENT",
		"data": map[string]interface{}{
			"paymentLinkId": linkRes.PaymentLinkID,
			"shortUrl":      linkRes.ShortURL,
			"isSimulated":   linkRes.IsSimulated,
		},
	})
}
