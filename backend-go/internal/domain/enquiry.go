package domain

import "time"

type EnquiryType string

const (
	EnquiryTypeRoomHold     EnquiryType = "ROOM_HOLD"
	EnquiryTypeBanquet      EnquiryType = "BANQUET"
	EnquiryTypeCorporateRFP EnquiryType = "CORPORATE_RFP"
	EnquiryTypeGeneral      EnquiryType = "GENERAL"
)

type EnquiryStatus string

const (
	EnquiryStatusNew       EnquiryStatus = "NEW"
	EnquiryStatusContacted EnquiryStatus = "CONTACTED"
	EnquiryStatusLinkSent  EnquiryStatus = "LINK_SENT"
	EnquiryStatusConverted EnquiryStatus = "CONVERTED"
	EnquiryStatusClosed    EnquiryStatus = "CLOSED"
)

type EnquiryRecord struct {
	ID                    string        `json:"id" db:"id"`
	EnquiryType           EnquiryType   `json:"enquiry_type" db:"enquiry_type"`
	PropertyID            *string       `json:"property_id,omitempty" db:"property_id"`
	GuestName             string        `json:"guest_name" db:"guest_name"`
	GuestPhone            string        `json:"guest_phone" db:"guest_phone"`
	GuestEmail            *string       `json:"guest_email,omitempty" db:"guest_email"`
	EventDate             *string       `json:"event_date,omitempty" db:"event_date"`
	GuestCount            *int          `json:"guest_count,omitempty" db:"guest_count"`
	Message               *string       `json:"message,omitempty" db:"message"`
	Status                EnquiryStatus `json:"status" db:"status"`
	RazorpayPaymentLinkID *string       `json:"razorpay_payment_link_id,omitempty" db:"razorpay_payment_link_id"`
	CreatedAt             time.Time     `json:"created_at" db:"created_at"`
}
