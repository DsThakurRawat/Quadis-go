package invoice

import (
	"fmt"
	"math"
	"time"

	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/service/pricing"
	"quadis-backend-go/pkg/dateutil"
)

const (
	LegalEntity       = "Quadis Services Private Limited"
	GSTIN             = "07AAACQ4872H1ZA"
	RegisteredAddress = "G/F, 9/2672, Plot No 22-H, Gali Number 17, Kailash Nagar,\nEast Delhi, Delhi — 110031"
	SACCode           = "996311"
)

type InvoiceDetails struct {
	InvoiceNumber     string  `json:"invoice_number"`
	Date              string  `json:"date"`
	LegalEntity       string  `json:"legal_entity"`
	GSTIN             string  `json:"gstin"`
	RegisteredAddress string  `json:"registered_address"`
	SACCode           string  `json:"sac_code"`

	// Guest
	GuestName   string  `json:"guest_name"`
	GuestPhone  string  `json:"guest_phone"`
	GuestEmail  *string `json:"guest_email,omitempty"`
	CompanyName *string `json:"company_name,omitempty"`
	GuestGSTIN  *string `json:"guest_gstin,omitempty"`

	// Property & Room
	PropertyName    string `json:"property_name"`
	PropertyAddress string `json:"property_address"`
	RoomName        string `json:"room_name"`
	CheckIn         string `json:"check_in"`
	CheckOut        string `json:"check_out"`
	Nights          int    `json:"nights"`
	RoomsCount      int    `json:"rooms_count"`

	// Financials
	TotalAmount    float64 `json:"total_amount"`
	BaseAmount     float64 `json:"base_amount"`
	GSTRatePercent float64 `json:"gst_rate_percent"`
	CGSTPercent    float64 `json:"cgst_percent"`
	CGSTAmount     float64 `json:"cgst_amount"`
	SGSTPercent    float64 `json:"sgst_percent"`
	SGSTAmount     float64 `json:"sgst_amount"`
}

func ComputeInvoiceDetails(
	booking *domain.BookingRecord,
	property *domain.PropertyRecord,
	room *domain.RoomTypeRecord,
) *InvoiceDetails {
	propName := "Quadis Hotel"
	propAddr := "Sector 51, Noida, Uttar Pradesh"
	if property != nil {
		propName = property.Name
		propAddr = property.Address
	}

	roomName := "Executive Room"
	if room != nil {
		roomName = room.Name
	}

	nightsList, err := dateutil.NightsBetween(booking.CheckIn, booking.CheckOut)
	nights := len(nightsList)
	if err != nil || nights <= 0 {
		nights = 1
	}

	totalAmount := booking.TotalAmount
	rooms := booking.RoomsCount
	if rooms <= 0 {
		rooms = 1
	}

	ratePerRoomNight := totalAmount / float64(nights*rooms)
	gstRate := pricing.GSTRatePercentFor(ratePerRoomNight)

	baseAmount := math.Round((totalAmount/(1.0+gstRate/100.0))*100) / 100
	taxAmount := math.Round((totalAmount-baseAmount)*100) / 100

	halfRate := gstRate / 2.0
	halfTax := math.Round((taxAmount/2.0)*100) / 100

	return &InvoiceDetails{
		InvoiceNumber:     fmt.Sprintf("INV-%s", booking.BookingCode),
		Date:              time.Now().UTC().Format("02 Jan 2006"),
		LegalEntity:       LegalEntity,
		GSTIN:             GSTIN,
		RegisteredAddress: RegisteredAddress,
		SACCode:           SACCode,
		GuestName:         booking.GuestName,
		GuestPhone:        booking.GuestPhone,
		GuestEmail:        booking.GuestEmail,
		CompanyName:       booking.CompanyName,
		GuestGSTIN:        booking.GSTIN,
		PropertyName:      propName,
		PropertyAddress:   propAddr,
		RoomName:          roomName,
		CheckIn:           booking.CheckIn,
		CheckOut:          booking.CheckOut,
		Nights:            nights,
		RoomsCount:        rooms,
		TotalAmount:       totalAmount,
		BaseAmount:        baseAmount,
		GSTRatePercent:    gstRate,
		CGSTPercent:       halfRate,
		CGSTAmount:        halfTax,
		SGSTPercent:       halfRate,
		SGSTAmount:        halfTax,
	}
}
