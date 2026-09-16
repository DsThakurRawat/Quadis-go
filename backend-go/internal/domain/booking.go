package domain

import "time"

type PaymentMode string

const (
	PaymentModeInstantFullPayment PaymentMode = "INSTANT_FULL_PAYMENT"
	PaymentModeTokenDeposit        PaymentMode = "TOKEN_DEPOSIT"
	PaymentModeEnquiryPaymentLink  PaymentMode = "ENQUIRY_PAYMENT_LINK"
)

type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "PENDING"
	PaymentStatusPaid     PaymentStatus = "PAID"
	PaymentStatusFailed   PaymentStatus = "FAILED"
	PaymentStatusRefunded PaymentStatus = "REFUNDED"
)

type BookingStatus string

const (
	BookingStatusPendingPayment BookingStatus = "PENDING_PAYMENT"
	BookingStatusConfirmed      BookingStatus = "CONFIRMED"
	BookingStatusCancelled      BookingStatus = "CANCELLED"
	BookingStatusExpired        BookingStatus = "EXPIRED"
)

type CMSyncStatus string

const (
	CMSyncStatusNone    CMSyncStatus = "NONE"
	CMSyncStatusPending CMSyncStatus = "PENDING"
	CMSyncStatusSent    CMSyncStatus = "SENT"
	CMSyncStatusFailed  CMSyncStatus = "FAILED"
)

type BookingRecord struct {
	ID                    string        `json:"id" db:"id"`
	BookingCode           string        `json:"booking_code" db:"booking_code"`
	UserID                *string       `json:"user_id,omitempty" db:"user_id"`
	PropertyID            string        `json:"property_id" db:"property_id"`
	RoomTypeID            string        `json:"room_type_id" db:"room_type_id"`
	GuestName             string        `json:"guest_name" db:"guest_name"`
	GuestPhone            string        `json:"guest_phone" db:"guest_phone"`
	GuestEmail            *string       `json:"guest_email,omitempty" db:"guest_email"`
	CompanyName           *string       `json:"company_name,omitempty" db:"company_name"`
	GSTIN                 *string       `json:"gstin,omitempty" db:"gstin"`
	CheckIn               string        `json:"check_in" db:"check_in"`
	CheckOut              string        `json:"check_out" db:"check_out"`
	RoomsCount            int           `json:"rooms_count" db:"rooms_count"`
	GuestsCount           int           `json:"guests_count" db:"guests_count"`
	AdultsCount           int           `json:"adults_count" db:"adults_count"`
	ChildrenCount         int           `json:"children_count" db:"children_count"`
	ChildAges             []int         `json:"child_ages" db:"child_ages"`
	ExtraAdults           int           `json:"extra_adults" db:"extra_adults"`
	ExtraAdultPercent     float64       `json:"extra_adult_percent" db:"extra_adult_percent"`
	ExtraAdultCharge      float64       `json:"extra_adult_charge" db:"extra_adult_charge"`
	TotalAmount           float64       `json:"total_amount" db:"total_amount"`
	PaymentMode           PaymentMode   `json:"payment_mode" db:"payment_mode"`
	PaymentStatus         PaymentStatus `json:"payment_status" db:"payment_status"`
	RazorpayOrderID       *string       `json:"razorpay_order_id,omitempty" db:"razorpay_order_id"`
	RazorpayPaymentID     *string       `json:"razorpay_payment_id,omitempty" db:"razorpay_payment_id"`
	RazorpayPaymentLinkID *string       `json:"razorpay_payment_link_id,omitempty" db:"razorpay_payment_link_id"`
	BookingStatus         BookingStatus `json:"booking_status" db:"booking_status"`
	MealPlan              *MealPlan     `json:"meal_plan,omitempty" db:"meal_plan"`
	CreatedAt             time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at" db:"updated_at"`

	// ResAvenue Channel Manager integration
	CMSyncStatus CMSyncStatus `json:"cm_sync_status,omitempty" db:"cm_sync_status"`
	CMResStatus  *string      `json:"cm_res_status,omitempty" db:"cm_res_status"`
	CMSyncedAt   *time.Time   `json:"cm_synced_at,omitempty" db:"cm_synced_at"`
	CMAttempts   int          `json:"cm_attempts" db:"cm_attempts"`
	CMLastError  *string      `json:"cm_last_error,omitempty" db:"cm_last_error"`
}
