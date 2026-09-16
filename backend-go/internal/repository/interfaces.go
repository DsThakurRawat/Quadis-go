package repository

import (
	"context"
	"quadis-backend-go/internal/domain"
)

type Repository interface {
	Ping(ctx context.Context) (storage string, reachable bool, err error)

	// Properties
	GetProperties(ctx context.Context) ([]domain.PropertyRecord, error)
	GetPropertyByID(ctx context.Context, id string) (*domain.PropertyRecord, error)
	GetPropertyBySlug(ctx context.Context, slug string) (*domain.PropertyRecord, []domain.RoomTypeRecord, error)
	GetPropertyByIDOrSlug(ctx context.Context, idOrSlug string) (*domain.PropertyRecord, error)
	GetPropertiesWithRooms(ctx context.Context) ([]domain.PropertyRecord, error)
	UpdateProperty(ctx context.Context, prop domain.PropertyRecord) error
	UpdateWeekendSurcharge(ctx context.Context, propertySlugOrName string, percent float64) (bool, string, error)

	// Property Images
	GetPropertyImages(ctx context.Context, propertyID string) ([]domain.PropertyImageRecord, error)
	AddPropertyImage(ctx context.Context, img domain.PropertyImageRecord) error
	DeletePropertyImage(ctx context.Context, id string) (bool, error)
	ReorderPropertyImages(ctx context.Context, propertyID string, orderedIDs []string) error

	// Rooms
	GetRoomTypesByPropertyID(ctx context.Context, propertyID string) ([]domain.RoomTypeRecord, error)
	GetRoomTypeByID(ctx context.Context, id string) (*domain.RoomTypeRecord, error)
	UpdateRoomType(ctx context.Context, room domain.RoomTypeRecord) error
	ToggleRoomAvailability(ctx context.Context, propertySlugOrName, roomSlugOrName string, isAvailable bool) (bool, string, error)

	// Inventory & Availability
	GetAvailableUnits(ctx context.Context, roomTypeID string, checkIn, checkOut string) (int, error)
	GetHeldUnitsByNight(ctx context.Context, roomTypeIDs []string, dates []string) (map[string]int, error)

	// Bookings
	InitiateBookingHold(ctx context.Context, booking *domain.BookingRecord) error
	GetBookingByCode(ctx context.Context, bookingCode string, guestPhone *string) (*domain.BookingRecord, error)
	GetBookingsForUser(ctx context.Context, userID, email string) ([]domain.BookingRecord, error)
	UpdateBookingPayment(ctx context.Context, bookingCode string, paymentID, orderID *string, status domain.BookingStatus, payStatus domain.PaymentStatus) (*domain.BookingRecord, error)
	CleanupExpiredHolds(ctx context.Context, expireThresholdMinutes int) (int, error)
	GetAllBookings(ctx context.Context, limit int) ([]domain.BookingRecord, error)
	GetGlanceMetrics(ctx context.Context) (map[string]interface{}, error)

	// Channel Manager (ResAvenue)
	GetInventoryDays(ctx context.Context, roomTypeIDs []string, dates []string) ([]domain.InventoryDayRecord, error)
	UpsertInventoryDays(ctx context.Context, rows []domain.InventoryDayRecord) (int, error)
	GetRateDays(ctx context.Context, roomTypeIDs []string, plans []domain.MealPlan, dates []string) ([]domain.RateDayRecord, error)
	UpsertRateDays(ctx context.Context, rows []domain.RateDayRecord) (int, error)
	GetBookingsForChannelPull(ctx context.Context, propertyID, fromDate, toDate string, limit int) ([]domain.BookingRecord, error)
	GetBookingsPendingChannelSync(ctx context.Context, maxAttempts, limit int) ([]domain.BookingRecord, error)
	MarkChannelSync(ctx context.Context, bookingID string, status domain.CMSyncStatus, resStatus *string, errStr *string) error

	// Users
	GetUserByEmail(ctx context.Context, email string) (*domain.UserRecord, error)
	GetUserByID(ctx context.Context, id string) (*domain.UserRecord, error)
	CreateUser(ctx context.Context, user *domain.UserRecord) error

	// Admin PIN & Content
	GetAdminPINHash(ctx context.Context) (string, error)
	SetAdminPINHash(ctx context.Context, hash string) error
	GetSiteContent(ctx context.Context) (map[string]string, error)
	SetSiteContent(ctx context.Context, entries map[string]string) (map[string]string, error)

	// Enquiries
	CreateEnquiry(ctx context.Context, enq *domain.EnquiryRecord) error
	GetEnquiries(ctx context.Context, status *domain.EnquiryStatus) ([]domain.EnquiryRecord, error)
	GetEnquiryByID(ctx context.Context, id string) (*domain.EnquiryRecord, error)
	UpdateEnquiryStatus(ctx context.Context, id string, status domain.EnquiryStatus, paymentLinkID *string) (*domain.EnquiryRecord, error)

	// Chat Logs
	CreateChatLog(ctx context.Context, chat *domain.ChatLogRecord) error
}
