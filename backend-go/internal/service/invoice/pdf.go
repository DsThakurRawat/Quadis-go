package invoice

import (
	"bytes"
	"fmt"
	"math"
	"time"

	"github.com/go-pdf/fpdf"
	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/service/pricing"
	"quadis-backend-go/pkg/dateutil"
)

// GenerateGstInvoicePdf generates an official SAC 996311 GST Tax Invoice PDF.
func GenerateGstInvoicePdf(
	booking *domain.BookingRecord,
	property *domain.PropertyRecord,
	room *domain.RoomTypeRecord,
) ([]byte, error) {
	propName := "Quadis Hotel"
	roomName := "Executive Room"
	if property != nil {
		propName = property.Name
	}
	if room != nil {
		roomName = room.Name
	}

	nightsList, err := dateutil.NightsBetween(booking.CheckIn, booking.CheckOut)
	nights := len(nightsList)
	if err != nil || nights <= 0 {
		nights = 1
	}

	rooms := booking.RoomsCount
	if rooms <= 0 {
		rooms = 1
	}

	totalAmount := booking.TotalAmount
	ratePerRoomNight := totalAmount / float64(nights*rooms)

	gstRate := pricing.GSTRatePercentFor(ratePerRoomNight)
	halfRate := gstRate / 2.0

	taxableBase := math.Round((totalAmount/(1.0+gstRate/100.0))*100) / 100
	totalTax := math.Round((totalAmount-taxableBase)*100) / 100
	cgst := math.Round((totalTax/2.0)*100) / 100
	sgst := math.Round((totalTax-cgst)*100) / 100

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	// Header - Brand
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetTextColor(17, 24, 39)
	pdf.Cell(0, 10, "QUADIS HOTELS & RESORTS")
	pdf.Ln(8)

	// Company details
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(75, 85, 99)
	pdf.Cell(0, 5, LegalEntity)
	pdf.Ln(5)
	pdf.Cell(0, 5, "G/F, 9/2672, Plot No 22-H, Gali Number 17, Kailash Nagar, East Delhi, Delhi - 110031")
	pdf.Ln(5)
	pdf.Cell(0, 5, fmt.Sprintf("GSTIN: %s | SAC Code: %s", GSTIN, SACCode))
	pdf.Ln(8)

	// Divider
	pdf.SetDrawColor(229, 231, 235)
	pdf.SetLineWidth(0.5)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(6)

	// Title
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(17, 24, 39)
	pdf.Cell(0, 8, "TAX INVOICE / RECEIPT")
	pdf.Ln(8)

	// Invoice metadata (Left Column) & Guest details (Right Column)
	startY := pdf.GetY()
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(55, 65, 81)

	// Left: Invoice details
	pdf.Text(20, startY+4, fmt.Sprintf("Invoice No: INV-%s", booking.BookingCode))
	pdf.Text(20, startY+10, fmt.Sprintf("Invoice Date: %s", time.Now().Format("02/01/2006")))
	pdf.Text(20, startY+16, fmt.Sprintf("Booking Status: %s", string(booking.BookingStatus)))
	pdf.Text(20, startY+22, fmt.Sprintf("Payment Status: %s", string(booking.PaymentStatus)))
	payID := "ONLINE_INSTANT"
	if booking.RazorpayPaymentID != nil && *booking.RazorpayPaymentID != "" {
		payID = *booking.RazorpayPaymentID
	}
	pdf.Text(20, startY+28, fmt.Sprintf("Razorpay Payment ID: %s", payID))

	// Right: Guest details
	pdf.SetFont("Helvetica", "B", 9)
	pdf.Text(115, startY+4, "BILLED TO (GUEST):")
	pdf.SetFont("Helvetica", "", 9)
	pdf.Text(115, startY+10, fmt.Sprintf("Name: %s", booking.GuestName))
	pdf.Text(115, startY+16, fmt.Sprintf("Phone: +91 %s", booking.GuestPhone))
	email := "N/A"
	if booking.GuestEmail != nil && *booking.GuestEmail != "" {
		email = *booking.GuestEmail
	}
	pdf.Text(115, startY+22, fmt.Sprintf("Email: %s", email))

	offsetY := 28.0
	if booking.CompanyName != nil && *booking.CompanyName != "" {
		pdf.Text(115, startY+offsetY, fmt.Sprintf("Company: %s", *booking.CompanyName))
		offsetY += 6
	}
	if booking.GSTIN != nil && *booking.GSTIN != "" {
		pdf.Text(115, startY+offsetY, fmt.Sprintf("Guest GSTIN: %s", *booking.GSTIN))
		offsetY += 6
	}

	pdf.SetY(startY + offsetY + 6)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(4)

	// Table Header
	pdf.SetFillColor(243, 244, 246)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(85, 8, "ACCOMMODATION DESCRIPTION", "0", 0, "L", true, 0, "")
	pdf.CellFormat(25, 8, "NIGHTS", "0", 0, "C", true, 0, "")
	pdf.CellFormat(25, 8, "ROOMS", "0", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "AMOUNT (INR)", "0", 1, "R", true, 0, "")

	// Table Row
	pdf.SetFont("Helvetica", "", 9)
	pdf.Cell(85, 6, fmt.Sprintf("%s - %s", propName, roomName))
	pdf.CellFormat(25, 6, fmt.Sprintf("%d", nights), "0", 0, "C", false, 0, "")
	pdf.CellFormat(25, 6, fmt.Sprintf("%d", rooms), "0", 0, "C", false, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("%.2f", taxableBase), "0", 1, "R", false, 0, "")

	// Dates Subtext
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(107, 114, 128)
	pdf.Cell(0, 5, fmt.Sprintf("Check-In: %s | Check-Out: %s", booking.CheckIn, booking.CheckOut))
	pdf.Ln(8)

	pdf.SetDrawColor(229, 231, 235)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(4)

	// Breakdown Table (Right Aligned)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(55, 65, 81)
	row := func(label string, val float64) {
		pdf.SetX(115)
		pdf.Cell(40, 6, label)
		pdf.CellFormat(35, 6, fmt.Sprintf("%.2f", val), "0", 1, "R", false, 0, "")
	}

	row("Taxable Base Amount:", taxableBase)
	row(fmt.Sprintf("CGST (%.1f%%):", halfRate), cgst)
	row(fmt.Sprintf("SGST (%.1f%%):", halfRate), sgst)

	pdf.SetDrawColor(17, 24, 39)
	pdf.SetLineWidth(0.5)
	pdf.Line(115, pdf.GetY()+2, 190, pdf.GetY()+2)
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(17, 24, 39)
	pdf.SetX(115)
	pdf.Cell(40, 8, "TOTAL PAID (INR):")
	pdf.CellFormat(35, 8, fmt.Sprintf("%.2f", totalAmount), "0", 1, "R", false, 0, "")
	pdf.Ln(12)

	// Footer
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(107, 114, 128)
	pdf.Cell(0, 4, "Note: This is an electronically generated SAC 996311 GST Tax Invoice and receipt.")
	pdf.Ln(4)
	pdf.Cell(0, 4, "Thank you for staying at Quadis Hotels. We wish you a wonderful journey!")
	pdf.Ln(8)

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(0, 5, fmt.Sprintf("For %s", LegalEntity), "0", 1, "R", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(0, 4, "Authorized Signatory", "0", 1, "R", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
