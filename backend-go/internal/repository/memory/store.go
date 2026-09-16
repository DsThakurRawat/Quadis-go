package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
	"quadis-backend-go/pkg/dateutil"
)

type MemoryStore struct {
	mu             sync.RWMutex
	properties     map[string]domain.PropertyRecord
	rooms          map[string]domain.RoomTypeRecord
	propertyImages map[string][]domain.PropertyImageRecord
	bookings       map[string]domain.BookingRecord // keyed by booking_code
	users          map[string]domain.UserRecord    // keyed by ID
	userEmailIndex map[string]string               // email -> ID
	inventoryDays  map[string]domain.InventoryDayRecord // "roomTypeId:stayDate"
	rateDays       map[string]domain.RateDayRecord      // "roomTypeId:mealPlan:stayDate"
	enquiries      map[string]domain.EnquiryRecord      // keyed by ID
	chatLogs       map[string]domain.ChatLogRecord      // keyed by ID
	siteContent    map[string]string
	adminPinHash   string
}

func NewMemoryStore() *MemoryStore {
	ms := &MemoryStore{
		properties:     make(map[string]domain.PropertyRecord),
		rooms:          make(map[string]domain.RoomTypeRecord),
		propertyImages: make(map[string][]domain.PropertyImageRecord),
		bookings:       make(map[string]domain.BookingRecord),
		users:          make(map[string]domain.UserRecord),
		userEmailIndex: make(map[string]string),
		inventoryDays:  make(map[string]domain.InventoryDayRecord),
		rateDays:       make(map[string]domain.RateDayRecord),
		enquiries:      make(map[string]domain.EnquiryRecord),
		chatLogs:       make(map[string]domain.ChatLogRecord),
		siteContent:    make(map[string]string),
	}

	// Seed properties
	for _, p := range repository.SeedProperties {
		ms.properties[p.ID] = p
	}

	// Seed rooms
	for _, r := range repository.BuildSeedRoomTypes() {
		ms.rooms[r.ID] = r
	}

	return ms
}

func (m *MemoryStore) Ping(ctx context.Context) (string, bool, error) {
	return "in-memory", true, nil
}

// ── Properties ─────────────────────────────────────────────────────────────

func (m *MemoryStore) GetProperties(ctx context.Context) ([]domain.PropertyRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []domain.PropertyRecord
	for _, p := range m.properties {
		result = append(result, p)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (m *MemoryStore) GetPropertyByID(ctx context.Context, id string) (*domain.PropertyRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, exists := m.properties[id]
	if !exists {
		return nil, nil
	}
	return &p, nil
}

func (m *MemoryStore) GetPropertyBySlug(ctx context.Context, slug string) (*domain.PropertyRecord, []domain.RoomTypeRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var foundProp *domain.PropertyRecord
	for _, p := range m.properties {
		if p.Slug == slug {
			cp := p
			foundProp = &cp
			break
		}
	}

	if foundProp == nil {
		return nil, nil, nil
	}

	var rooms []domain.RoomTypeRecord
	for _, r := range m.rooms {
		if r.PropertyID == foundProp.ID {
			rooms = append(rooms, r)
		}
	}

	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].PriceOffset < rooms[j].PriceOffset
	})

	return foundProp, rooms, nil
}

func (m *MemoryStore) GetPropertyByIDOrSlug(ctx context.Context, idOrSlug string) (*domain.PropertyRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if p, exists := m.properties[idOrSlug]; exists {
		return &p, nil
	}

	for _, p := range m.properties {
		if p.Slug == idOrSlug {
			cp := p
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *MemoryStore) GetPropertiesWithRooms(ctx context.Context) ([]domain.PropertyRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []domain.PropertyRecord
	for _, p := range m.properties {
		cp := p
		var rooms []domain.RoomTypeRecord
		for _, r := range m.rooms {
			if r.PropertyID == p.ID {
				rooms = append(rooms, r)
			}
		}
		sort.Slice(rooms, func(i, j int) bool {
			return rooms[i].PriceOffset < rooms[j].PriceOffset
		})
		cp.Rooms = rooms
		result = append(result, cp)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (m *MemoryStore) UpdateProperty(ctx context.Context, prop domain.PropertyRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.properties[prop.ID]; !exists {
		return fmt.Errorf("property not found: %s", prop.ID)
	}
	m.properties[prop.ID] = prop
	return nil
}

func (m *MemoryStore) UpdateWeekendSurcharge(ctx context.Context, propertySlugOrName string, percent float64) (bool, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	target := strings.ToLower(strings.TrimSpace(propertySlugOrName))
	for id, p := range m.properties {
		if strings.ToLower(p.Slug) == target || strings.Contains(strings.ToLower(p.Name), target) {
			p.WeekendSurchargePercent = percent
			m.properties[id] = p
			return true, p.Name, nil
		}
	}
	return false, "", nil
}

// ── Property Images ────────────────────────────────────────────────────────

func (m *MemoryStore) GetPropertyImages(ctx context.Context, propertyID string) ([]domain.PropertyImageRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	imgs := m.propertyImages[propertyID]
	var out []domain.PropertyImageRecord
	out = append(out, imgs...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].SortOrder < out[j].SortOrder
	})
	return out, nil
}

func (m *MemoryStore) AddPropertyImage(ctx context.Context, img domain.PropertyImageRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if img.ID == "" {
		img.ID = uuid.New().String()
	}
	m.propertyImages[img.PropertyID] = append(m.propertyImages[img.PropertyID], img)
	return nil
}

func (m *MemoryStore) DeletePropertyImage(ctx context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for propID, list := range m.propertyImages {
		for i, img := range list {
			if img.ID == id {
				m.propertyImages[propID] = append(list[:i], list[i+1:]...)
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *MemoryStore) ReorderPropertyImages(ctx context.Context, propertyID string, orderedIDs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	list := m.propertyImages[propertyID]
	imgMap := make(map[string]domain.PropertyImageRecord)
	for _, img := range list {
		imgMap[img.ID] = img
	}

	var updated []domain.PropertyImageRecord
	for order, id := range orderedIDs {
		if img, ok := imgMap[id]; ok {
			img.SortOrder = order
			updated = append(updated, img)
			delete(imgMap, id)
		}
	}
	for _, leftover := range imgMap {
		updated = append(updated, leftover)
	}
	m.propertyImages[propertyID] = updated
	return nil
}

// ── Rooms ──────────────────────────────────────────────────────────────────

func (m *MemoryStore) GetRoomTypesByPropertyID(ctx context.Context, propertyID string) ([]domain.RoomTypeRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var rooms []domain.RoomTypeRecord
	for _, r := range m.rooms {
		if r.PropertyID == propertyID {
			rooms = append(rooms, r)
		}
	}
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].PriceOffset < rooms[j].PriceOffset
	})
	return rooms, nil
}

func (m *MemoryStore) GetRoomTypeByID(ctx context.Context, id string) (*domain.RoomTypeRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	r, exists := m.rooms[id]
	if !exists {
		return nil, nil
	}
	return &r, nil
}

func (m *MemoryStore) UpdateRoomType(ctx context.Context, room domain.RoomTypeRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.rooms[room.ID]; !exists {
		return fmt.Errorf("room type not found: %s", room.ID)
	}
	m.rooms[room.ID] = room
	return nil
}

func (m *MemoryStore) ToggleRoomAvailability(ctx context.Context, propertySlugOrName, roomSlugOrName string, isAvailable bool) (bool, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pTarget := strings.ToLower(strings.TrimSpace(propertySlugOrName))
	rTarget := strings.ToLower(strings.TrimSpace(roomSlugOrName))

	var targetPropID string
	var propName string
	for id, p := range m.properties {
		if strings.ToLower(p.Slug) == pTarget || strings.Contains(strings.ToLower(p.Name), pTarget) {
			targetPropID = id
			propName = p.Name
			break
		}
	}
	if targetPropID == "" {
		return false, "", nil
	}

	for id, r := range m.rooms {
		if r.PropertyID == targetPropID {
			if strings.ToLower(r.Slug) == rTarget || strings.Contains(strings.ToLower(r.Name), rTarget) {
				r.IsAvailable = isAvailable
				m.rooms[id] = r
				desc := fmt.Sprintf("%s (%s)", r.Name, propName)
				return true, desc, nil
			}
		}
	}
	return false, "", nil
}

// ── Inventory & Availability ───────────────────────────────────────────────

func (m *MemoryStore) GetAvailableUnits(ctx context.Context, roomTypeID string, checkIn, checkOut string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	room, exists := m.rooms[roomTypeID]
	if !exists {
		return 0, fmt.Errorf("room type not found: %s", roomTypeID)
	}
	if !room.IsAvailable {
		return 0, nil
	}

	nights, err := dateutil.NightsBetween(checkIn, checkOut)
	if err != nil {
		return 0, err
	}

	minAvailable := room.TotalUnits

	for _, night := range nights {
		invKey := fmt.Sprintf("%s:%s", roomTypeID, night)
		invCeiling := room.TotalUnits
		if invRecord, hasOverride := m.inventoryDays[invKey]; hasOverride {
			if invRecord.StopSell {
				return 0, nil
			}
			if invRecord.InvCount != nil {
				invCeiling = *invRecord.InvCount
			}
		}

		// Count booked or active held units for this room on this night
		bookedUnits := 0
		now := time.Now()
		for _, b := range m.bookings {
			if b.RoomTypeID != roomTypeID {
				continue
			}
			// Status check: CONFIRMED always holds units.
			// PENDING_PAYMENT holds units only if within 15 minutes of created_at.
			isActiveHold := b.BookingStatus == domain.BookingStatusPendingPayment && now.Sub(b.CreatedAt) < 15*time.Minute
			isConfirmed := b.BookingStatus == domain.BookingStatusConfirmed
			if !isActiveHold && !isConfirmed {
				continue
			}

			// Check if booking date range overlaps with this night
			// booking spans check_in to check_out
			bNights, err := dateutil.NightsBetween(b.CheckIn, b.CheckOut)
			if err != nil {
				continue
			}
			for _, bn := range bNights {
				if bn == night {
					bookedUnits += b.RoomsCount
					break
				}
			}
		}

		avail := invCeiling - bookedUnits
		if avail < 0 {
			avail = 0
		}
		if avail < minAvailable {
			minAvailable = avail
		}
	}

	return minAvailable, nil
}

func (m *MemoryStore) GetHeldUnitsByNight(ctx context.Context, roomTypeIDs []string, dates []string) (map[string]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]int)
	roomSet := make(map[string]bool)
	for _, id := range roomTypeIDs {
		roomSet[id] = true
	}
	dateSet := make(map[string]bool)
	for _, d := range dates {
		dateSet[d] = true
	}

	now := time.Now()
	for _, b := range m.bookings {
		if !roomSet[b.RoomTypeID] {
			continue
		}
		isActiveHold := b.BookingStatus == domain.BookingStatusPendingPayment && now.Sub(b.CreatedAt) < 15*time.Minute
		isConfirmed := b.BookingStatus == domain.BookingStatusConfirmed
		if !isActiveHold && !isConfirmed {
			continue
		}

		bNights, err := dateutil.NightsBetween(b.CheckIn, b.CheckOut)
		if err != nil {
			continue
		}
		for _, bn := range bNights {
			if dateSet[bn] {
				key := fmt.Sprintf("%s:%s", b.RoomTypeID, bn)
				result[key] += b.RoomsCount
			}
		}
	}

	return result, nil
}

// ── Bookings ───────────────────────────────────────────────────────────────

func (m *MemoryStore) InitiateBookingHold(ctx context.Context, booking *domain.BookingRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if booking.ID == "" {
		booking.ID = uuid.New().String()
	}
	if booking.BookingCode == "" {
		code, err := repository.GenerateBookingCode()
		if err != nil {
			return err
		}
		booking.BookingCode = code
	}

	now := time.Now()
	booking.CreatedAt = now
	booking.UpdatedAt = now
	booking.BookingStatus = domain.BookingStatusPendingPayment
	booking.PaymentStatus = domain.PaymentStatusPending

	m.bookings[booking.BookingCode] = *booking
	return nil
}

func (m *MemoryStore) GetBookingByCode(ctx context.Context, bookingCode string, guestPhone *string) (*domain.BookingRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	b, exists := m.bookings[bookingCode]
	if !exists {
		return nil, nil
	}
	if guestPhone != nil && *guestPhone != "" {
		cleanExpected := strings.ReplaceAll(*guestPhone, " ", "")
		cleanActual := strings.ReplaceAll(b.GuestPhone, " ", "")
		if !strings.HasSuffix(cleanActual, cleanExpected) && !strings.HasSuffix(cleanExpected, cleanActual) {
			return nil, nil
		}
	}
	return &b, nil
}

func (m *MemoryStore) GetBookingsForUser(ctx context.Context, userID, email string) ([]domain.BookingRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []domain.BookingRecord
	for _, b := range m.bookings {
		matchesUser := b.UserID != nil && *b.UserID == userID
		matchesEmail := b.GuestEmail != nil && strings.EqualFold(*b.GuestEmail, email)
		if matchesUser || matchesEmail {
			result = append(result, b)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (m *MemoryStore) UpdateBookingPayment(
	ctx context.Context,
	bookingCode string,
	paymentID, orderID *string,
	status domain.BookingStatus,
	payStatus domain.PaymentStatus,
) (*domain.BookingRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, exists := m.bookings[bookingCode]
	if !exists {
		return nil, fmt.Errorf("booking not found: %s", bookingCode)
	}

	if paymentID != nil {
		b.RazorpayPaymentID = paymentID
	}
	if orderID != nil {
		b.RazorpayOrderID = orderID
	}
	b.BookingStatus = status
	b.PaymentStatus = payStatus
	b.UpdatedAt = time.Now()

	// If confirmed, queue for channel manager sync
	if status == domain.BookingStatusConfirmed {
		b.CMSyncStatus = domain.CMSyncStatusPending
		confirmStr := "Confirm"
		b.CMResStatus = &confirmStr
	} else if status == domain.BookingStatusCancelled {
		b.CMSyncStatus = domain.CMSyncStatusPending
		cancelStr := "Cancel"
		b.CMResStatus = &cancelStr
	}

	m.bookings[bookingCode] = b
	return &b, nil
}

func (m *MemoryStore) CleanupExpiredHolds(ctx context.Context, expireThresholdMinutes int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	threshold := time.Duration(expireThresholdMinutes) * time.Minute
	now := time.Now()
	expiredCount := 0

	for code, b := range m.bookings {
		if b.BookingStatus == domain.BookingStatusPendingPayment && now.Sub(b.CreatedAt) > threshold {
			b.BookingStatus = domain.BookingStatusExpired
			b.PaymentStatus = domain.PaymentStatusFailed
			b.UpdatedAt = now
			m.bookings[code] = b
			expiredCount++
		}
	}

	return expiredCount, nil
}

func (m *MemoryStore) GetAllBookings(ctx context.Context, limit int) ([]domain.BookingRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []domain.BookingRecord
	for _, b := range m.bookings {
		list = append(list, b)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (m *MemoryStore) GetGlanceMetrics(ctx context.Context) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	today := now.UTC().Format("2006-01-02")

	var confirmedCount int
	var totalRevenue float64
	var activeHolds int
	var todayCheckins int

	for _, b := range m.bookings {
		if b.BookingStatus == domain.BookingStatusConfirmed {
			confirmedCount++
			totalRevenue += b.TotalAmount
			if b.CheckIn == today {
				todayCheckins++
			}
		} else if b.BookingStatus == domain.BookingStatusPendingPayment && now.Sub(b.CreatedAt) < 15*time.Minute {
			activeHolds++
		}
	}

	metrics := map[string]interface{}{
		"totalBookings": confirmedCount,
		"totalRevenue":  totalRevenue,
		"activeHolds":   activeHolds,
		"todayCheckins": todayCheckins,
		"activeHotels":  len(m.properties),
	}
	return metrics, nil
}

// ── Channel Manager (ResAvenue) ────────────────────────────────────────────

func (m *MemoryStore) GetInventoryDays(ctx context.Context, roomTypeIDs []string, dates []string) ([]domain.InventoryDayRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []domain.InventoryDayRecord
	for _, rID := range roomTypeIDs {
		for _, d := range dates {
			k := fmt.Sprintf("%s:%s", rID, d)
			if rec, ok := m.inventoryDays[k]; ok {
				result = append(result, rec)
			}
		}
	}
	return result, nil
}

func (m *MemoryStore) UpsertInventoryDays(ctx context.Context, rows []domain.InventoryDayRecord) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for _, r := range rows {
		r.UpdatedAt = now
		k := fmt.Sprintf("%s:%s", r.RoomTypeID, r.StayDate)
		m.inventoryDays[k] = r
	}
	return len(rows), nil
}

func (m *MemoryStore) GetRateDays(ctx context.Context, roomTypeIDs []string, plans []domain.MealPlan, dates []string) ([]domain.RateDayRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []domain.RateDayRecord
	for _, rID := range roomTypeIDs {
		for _, plan := range plans {
			for _, d := range dates {
				k := fmt.Sprintf("%s:%s:%s", rID, plan, d)
				if rec, ok := m.rateDays[k]; ok {
					result = append(result, rec)
				}
			}
		}
	}
	return result, nil
}

func (m *MemoryStore) UpsertRateDays(ctx context.Context, rows []domain.RateDayRecord) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for _, r := range rows {
		r.UpdatedAt = now
		k := fmt.Sprintf("%s:%s:%s", r.RoomTypeID, r.MealPlan, r.StayDate)
		m.rateDays[k] = r
	}
	return len(rows), nil
}

func (m *MemoryStore) GetBookingsForChannelPull(ctx context.Context, propertyID, fromDate, toDate string, limit int) ([]domain.BookingRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []domain.BookingRecord
	for _, b := range m.bookings {
		if propertyID != "" && b.PropertyID != propertyID {
			continue
		}
		// check updated_at or created_at in range
		bDate := b.UpdatedAt.UTC().Format("2006-01-02")
		if (fromDate == "" || bDate >= fromDate) && (toDate == "" || bDate <= toDate) {
			list = append(list, b)
		}
	}

	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (m *MemoryStore) GetBookingsPendingChannelSync(ctx context.Context, maxAttempts, limit int) ([]domain.BookingRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []domain.BookingRecord
	for _, b := range m.bookings {
		if (b.CMSyncStatus == domain.CMSyncStatusPending || b.CMSyncStatus == domain.CMSyncStatusFailed) && b.CMAttempts < maxAttempts {
			list = append(list, b)
		}
	}

	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

func (m *MemoryStore) MarkChannelSync(ctx context.Context, bookingID string, status domain.CMSyncStatus, resStatus *string, errStr *string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for code, b := range m.bookings {
		if b.ID == bookingID {
			b.CMSyncStatus = status
			b.CMAttempts++
			now := time.Now()
			b.CMSyncedAt = &now
			if resStatus != nil {
				b.CMResStatus = resStatus
			}
			b.CMLastError = errStr
			m.bookings[code] = b
			return nil
		}
	}
	return fmt.Errorf("booking ID not found: %s", bookingID)
}

// ── Users ──────────────────────────────────────────────────────────────────

func (m *MemoryStore) GetUserByEmail(ctx context.Context, email string) (*domain.UserRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, exists := m.userEmailIndex[strings.ToLower(email)]
	if !exists {
		return nil, nil
	}
	u := m.users[id]
	return &u, nil
}

func (m *MemoryStore) GetUserByID(ctx context.Context, id string) (*domain.UserRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	u, exists := m.users[id]
	if !exists {
		return nil, nil
	}
	return &u, nil
}

func (m *MemoryStore) CreateUser(ctx context.Context, user *domain.UserRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lowerEmail := strings.ToLower(user.Email)
	if _, exists := m.userEmailIndex[lowerEmail]; exists {
		return fmt.Errorf("user with email %s already exists", user.Email)
	}

	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	user.CreatedAt = time.Now()

	m.users[user.ID] = *user
	m.userEmailIndex[lowerEmail] = user.ID
	return nil
}

// ── Admin PIN & Site Content ───────────────────────────────────────────────

func (m *MemoryStore) GetAdminPINHash(ctx context.Context) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.adminPinHash, nil
}

func (m *MemoryStore) SetAdminPINHash(ctx context.Context, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.adminPinHash = hash
	return nil
}

func (m *MemoryStore) GetSiteContent(ctx context.Context) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	copy := make(map[string]string)
	for k, v := range m.siteContent {
		copy[k] = v
	}
	return copy, nil
}

func (m *MemoryStore) SetSiteContent(ctx context.Context, entries map[string]string) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for k, v := range entries {
		m.siteContent[k] = v
	}

	copy := make(map[string]string)
	for k, v := range m.siteContent {
		copy[k] = v
	}
	return copy, nil
}

// ── Enquiries ──────────────────────────────────────────────────────────────

func (m *MemoryStore) CreateEnquiry(ctx context.Context, enq *domain.EnquiryRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if enq.ID == "" {
		enq.ID = uuid.New().String()
	}
	enq.Status = domain.EnquiryStatusNew
	enq.CreatedAt = time.Now()
	m.enquiries[enq.ID] = *enq
	return nil
}

func (m *MemoryStore) GetEnquiries(ctx context.Context, status *domain.EnquiryStatus) ([]domain.EnquiryRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []domain.EnquiryRecord
	for _, e := range m.enquiries {
		if status == nil || e.Status == *status {
			list = append(list, e)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	return list, nil
}

func (m *MemoryStore) GetEnquiryByID(ctx context.Context, id string) (*domain.EnquiryRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	e, exists := m.enquiries[id]
	if !exists {
		return nil, nil
	}
	return &e, nil
}

func (m *MemoryStore) UpdateEnquiryStatus(ctx context.Context, id string, status domain.EnquiryStatus, paymentLinkID *string) (*domain.EnquiryRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, exists := m.enquiries[id]
	if !exists {
		return nil, fmt.Errorf("enquiry not found: %s", id)
	}
	e.Status = status
	if paymentLinkID != nil {
		e.RazorpayPaymentLinkID = paymentLinkID
	}
	m.enquiries[id] = e
	return &e, nil
}

// ── Chat Logs ──────────────────────────────────────────────────────────────

func (m *MemoryStore) CreateChatLog(ctx context.Context, chat *domain.ChatLogRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if chat.ID == "" {
		chat.ID = uuid.New().String()
	}
	chat.CreatedAt = time.Now()
	m.chatLogs[chat.ID] = *chat
	return nil
}
