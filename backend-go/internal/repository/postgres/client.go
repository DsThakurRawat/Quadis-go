package postgres

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
	"quadis-backend-go/pkg/dateutil"
)

//go:embed schema.sql
var SchemaSQL string

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, connString string) (*PostgresStore, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store := &PostgresStore{pool: pool}
	return store, nil
}

func (s *PostgresStore) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *PostgresStore) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, SchemaSQL)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

func (s *PostgresStore) Ping(ctx context.Context) (string, bool, error) {
	err := s.pool.Ping(ctx)
	if err != nil {
		return "postgres", false, err
	}
	return "postgres", true, nil
}

// ── Properties ─────────────────────────────────────────────────────────────

func (s *PostgresStore) GetProperties(ctx context.Context) ([]domain.PropertyRecord, error) {
	query := `SELECT id, slug, name, city, address, map_link, phone, whatsapp, email,
                     base_price, rating, is_active, weekend_surcharge_percent,
                     extra_adult_percent, child_free_under_age, child_percent, adult_from_age,
                     lat, lng, place_id, tier, tier_label
              FROM properties
              WHERE is_active = true
              ORDER BY id ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var props []domain.PropertyRecord
	for rows.Next() {
		var p domain.PropertyRecord
		if err := rows.Scan(
			&p.ID, &p.Slug, &p.Name, &p.City, &p.Address, &p.MapLink, &p.Phone, &p.WhatsApp, &p.Email,
			&p.BasePrice, &p.Rating, &p.IsActive, &p.WeekendSurchargePercent,
			&p.ExtraAdultPercent, &p.ChildFreeUnderAge, &p.ChildPercent, &p.AdultFromAge,
			&p.Lat, &p.Lng, &p.PlaceID, &p.Tier, &p.TierLabel,
		); err != nil {
			return nil, err
		}
		props = append(props, p)
	}
	return props, rows.Err()
}

func (s *PostgresStore) GetPropertyByID(ctx context.Context, id string) (*domain.PropertyRecord, error) {
	query := `SELECT id, slug, name, city, address, map_link, phone, whatsapp, email,
                     base_price, rating, is_active, weekend_surcharge_percent,
                     extra_adult_percent, child_free_under_age, child_percent, adult_from_age,
                     lat, lng, place_id, tier, tier_label
              FROM properties WHERE id = $1`

	var p domain.PropertyRecord
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Slug, &p.Name, &p.City, &p.Address, &p.MapLink, &p.Phone, &p.WhatsApp, &p.Email,
		&p.BasePrice, &p.Rating, &p.IsActive, &p.WeekendSurchargePercent,
		&p.ExtraAdultPercent, &p.ChildFreeUnderAge, &p.ChildPercent, &p.AdultFromAge,
		&p.Lat, &p.Lng, &p.PlaceID, &p.Tier, &p.TierLabel,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (s *PostgresStore) GetPropertyBySlug(ctx context.Context, slug string) (*domain.PropertyRecord, []domain.RoomTypeRecord, error) {
	query := `SELECT id, slug, name, city, address, map_link, phone, whatsapp, email,
                     base_price, rating, is_active, weekend_surcharge_percent,
                     extra_adult_percent, child_free_under_age, child_percent, adult_from_age,
                     lat, lng, place_id, tier, tier_label
              FROM properties WHERE slug = $1 AND is_active = true`

	var p domain.PropertyRecord
	err := s.pool.QueryRow(ctx, query, slug).Scan(
		&p.ID, &p.Slug, &p.Name, &p.City, &p.Address, &p.MapLink, &p.Phone, &p.WhatsApp, &p.Email,
		&p.BasePrice, &p.Rating, &p.IsActive, &p.WeekendSurchargePercent,
		&p.ExtraAdultPercent, &p.ChildFreeUnderAge, &p.ChildPercent, &p.AdultFromAge,
		&p.Lat, &p.Lng, &p.PlaceID, &p.Tier, &p.TierLabel,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	rooms, err := s.GetRoomTypesByPropertyID(ctx, p.ID)
	if err != nil {
		return nil, nil, err
	}

	return &p, rooms, nil
}

func (s *PostgresStore) GetPropertyByIDOrSlug(ctx context.Context, idOrSlug string) (*domain.PropertyRecord, error) {
	query := `SELECT id, slug, name, city, address, map_link, phone, whatsapp, email,
                     base_price, rating, is_active, weekend_surcharge_percent,
                     extra_adult_percent, child_free_under_age, child_percent, adult_from_age,
                     lat, lng, place_id, tier, tier_label
              FROM properties WHERE id = $1 OR slug = $1 LIMIT 1`

	var p domain.PropertyRecord
	err := s.pool.QueryRow(ctx, query, idOrSlug).Scan(
		&p.ID, &p.Slug, &p.Name, &p.City, &p.Address, &p.MapLink, &p.Phone, &p.WhatsApp, &p.Email,
		&p.BasePrice, &p.Rating, &p.IsActive, &p.WeekendSurchargePercent,
		&p.ExtraAdultPercent, &p.ChildFreeUnderAge, &p.ChildPercent, &p.AdultFromAge,
		&p.Lat, &p.Lng, &p.PlaceID, &p.Tier, &p.TierLabel,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (s *PostgresStore) GetPropertiesWithRooms(ctx context.Context) ([]domain.PropertyRecord, error) {
	props, err := s.GetProperties(ctx)
	if err != nil {
		return nil, err
	}

	for i := range props {
		rooms, err := s.GetRoomTypesByPropertyID(ctx, props[i].ID)
		if err != nil {
			return nil, err
		}
		props[i].Rooms = rooms
	}
	return props, nil
}

func (s *PostgresStore) UpdateProperty(ctx context.Context, prop domain.PropertyRecord) error {
	query := `UPDATE properties SET
                base_price = $1,
                weekend_surcharge_percent = $2,
                rating = $3,
                phone = $4,
                whatsapp = $5,
                email = $6,
                is_active = $7
              WHERE id = $8`

	_, err := s.pool.Exec(ctx, query,
		prop.BasePrice, prop.WeekendSurchargePercent, prop.Rating,
		prop.Phone, prop.WhatsApp, prop.Email, prop.IsActive, prop.ID,
	)
	return err
}

func (s *PostgresStore) UpdateWeekendSurcharge(ctx context.Context, propertySlugOrName string, percent float64) (bool, string, error) {
	query := `UPDATE properties
              SET weekend_surcharge_percent = $1
              WHERE LOWER(slug) = LOWER($2) OR LOWER(name) ILIKE '%' || LOWER($2) || '%'
              RETURNING name`

	var name string
	err := s.pool.QueryRow(ctx, query, percent, strings.TrimSpace(propertySlugOrName)).Scan(&name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, "", nil
		}
		return false, "", err
	}
	return true, name, nil
}

// ── Property Images ────────────────────────────────────────────────────────

func (s *PostgresStore) GetPropertyImages(ctx context.Context, propertyID string) ([]domain.PropertyImageRecord, error) {
	query := `SELECT id, property_id, url, thumb_url, storage_key, alt_text, sort_order, created_at
              FROM property_images
              WHERE property_id = $1
              ORDER BY sort_order ASC`

	rows, err := s.pool.Query(ctx, query, propertyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []domain.PropertyImageRecord
	for rows.Next() {
		var img domain.PropertyImageRecord
		if err := rows.Scan(
			&img.ID, &img.PropertyID, &img.URL, &img.ThumbURL, &img.StorageKey,
			&img.AltText, &img.SortOrder, &img.CreatedAt,
		); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

func (s *PostgresStore) AddPropertyImage(ctx context.Context, img domain.PropertyImageRecord) error {
	if img.ID == "" {
		img.ID = uuid.New().String()
	}
	query := `INSERT INTO property_images (id, property_id, url, thumb_url, storage_key, alt_text, sort_order)
              VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := s.pool.Exec(ctx, query, img.ID, img.PropertyID, img.URL, img.ThumbURL, img.StorageKey, img.AltText, img.SortOrder)
	return err
}

func (s *PostgresStore) DeletePropertyImage(ctx context.Context, id string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM property_images WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *PostgresStore) ReorderPropertyImages(ctx context.Context, propertyID string, orderedIDs []string) error {
	for order, id := range orderedIDs {
		_, err := s.pool.Exec(ctx, `UPDATE property_images SET sort_order = $1 WHERE id = $2 AND property_id = $3`, order, id, propertyID)
		if err != nil {
			return err
		}
	}
	return nil
}

// ── Rooms ──────────────────────────────────────────────────────────────────

func (s *PostgresStore) GetRoomTypesByPropertyID(ctx context.Context, propertyID string) ([]domain.RoomTypeRecord, error) {
	query := `SELECT id, property_id, slug, name, description, size_sqft, bed_type, max_guests,
                     price_offset, breakfast_offset, all_meals_offset, total_units, available_units, is_available
              FROM room_types
              WHERE property_id = $1
              ORDER BY price_offset ASC`

	rows, err := s.pool.Query(ctx, query, propertyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []domain.RoomTypeRecord
	for rows.Next() {
		var r domain.RoomTypeRecord
		if err := rows.Scan(
			&r.ID, &r.PropertyID, &r.Slug, &r.Name, &r.Description, &r.SizeSqft, &r.BedType, &r.MaxGuests,
			&r.PriceOffset, &r.BreakfastOffset, &r.AllMealsOffset, &r.TotalUnits, &r.AvailableUnits, &r.IsAvailable,
		); err != nil {
			return nil, err
		}
		rooms = append(rooms, r)
	}
	return rooms, rows.Err()
}

func (s *PostgresStore) GetRoomTypeByID(ctx context.Context, id string) (*domain.RoomTypeRecord, error) {
	query := `SELECT id, property_id, slug, name, description, size_sqft, bed_type, max_guests,
                     price_offset, breakfast_offset, all_meals_offset, total_units, available_units, is_available
              FROM room_types WHERE id = $1`

	var r domain.RoomTypeRecord
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&r.ID, &r.PropertyID, &r.Slug, &r.Name, &r.Description, &r.SizeSqft, &r.BedType, &r.MaxGuests,
		&r.PriceOffset, &r.BreakfastOffset, &r.AllMealsOffset, &r.TotalUnits, &r.AvailableUnits, &r.IsAvailable,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (s *PostgresStore) UpdateRoomType(ctx context.Context, room domain.RoomTypeRecord) error {
	query := `UPDATE room_types SET
                name = $1,
                description = $2,
                price_offset = $3,
                total_units = $4,
                is_available = $5
              WHERE id = $6`
	_, err := s.pool.Exec(ctx, query, room.Name, room.Description, room.PriceOffset, room.TotalUnits, room.IsAvailable, room.ID)
	return err
}

func (s *PostgresStore) ToggleRoomAvailability(ctx context.Context, propertySlugOrName, roomSlugOrName string, isAvailable bool) (bool, string, error) {
	query := `UPDATE room_types rt
              SET is_available = $1
              FROM properties p
              WHERE rt.property_id = p.id
                AND (LOWER(p.slug) = LOWER($2) OR LOWER(p.name) ILIKE '%' || LOWER($2) || '%')
                AND (LOWER(rt.slug) = LOWER($3) OR LOWER(rt.name) ILIKE '%' || LOWER($3) || '%')
              RETURNING rt.name, p.name`

	var roomName, propName string
	err := s.pool.QueryRow(ctx, query, isAvailable, strings.TrimSpace(propertySlugOrName), strings.TrimSpace(roomSlugOrName)).Scan(&roomName, &propName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, "", nil
		}
		return false, "", err
	}
	return true, fmt.Sprintf("%s (%s)", roomName, propName), nil
}

// ── Inventory & Availability ───────────────────────────────────────────────

func (s *PostgresStore) GetAvailableUnits(ctx context.Context, roomTypeID string, checkIn, checkOut string) (int, error) {
	room, err := s.GetRoomTypeByID(ctx, roomTypeID)
	if err != nil {
		return 0, err
	}
	if room == nil {
		return 0, fmt.Errorf("room type not found: %s", roomTypeID)
	}
	if !room.IsAvailable {
		return 0, nil
	}

	nights, err := dateutil.NightsBetween(checkIn, checkOut)
	if err != nil {
		return 0, err
	}

	// Fetch per-night inventory overrides for this room
	invDays, err := s.GetInventoryDays(ctx, []string{roomTypeID}, nights)
	if err != nil {
		return 0, err
	}
	overrideMap := make(map[string]domain.InventoryDayRecord)
	for _, inv := range invDays {
		overrideMap[inv.StayDate] = inv
	}

	minAvailable := room.TotalUnits

	for _, night := range nights {
		invCeiling := room.TotalUnits
		if rec, exists := overrideMap[night]; exists {
			if rec.StopSell {
				return 0, nil
			}
			if rec.InvCount != nil {
				invCeiling = *rec.InvCount
			}
		}

		// Count booked units from active holds (within 15m) or confirmed bookings
		query := `SELECT COALESCE(SUM(rooms_count), 0)
                  FROM bookings
                  WHERE room_type_id = $1
                    AND check_in <= $2 AND check_out > $2
                    AND (
                      booking_status = 'CONFIRMED'
                      OR (booking_status = 'PENDING_PAYMENT' AND created_at > NOW() - INTERVAL '15 minutes')
                    )`

		var bookedUnits int
		err := s.pool.QueryRow(ctx, query, roomTypeID, night).Scan(&bookedUnits)
		if err != nil {
			return 0, err
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

func (s *PostgresStore) GetHeldUnitsByNight(ctx context.Context, roomTypeIDs []string, dates []string) (map[string]int, error) {
	if len(roomTypeIDs) == 0 || len(dates) == 0 {
		return map[string]int{}, nil
	}

	query := `SELECT room_type_id, check_in, check_out, rooms_count
              FROM bookings
              WHERE room_type_id = ANY($1)
                AND (
                  booking_status = 'CONFIRMED'
                  OR (booking_status = 'PENDING_PAYMENT' AND created_at > NOW() - INTERVAL '15 minutes')
                )`

	rows, err := s.pool.Query(ctx, query, roomTypeIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dateSet := make(map[string]bool)
	for _, d := range dates {
		dateSet[d] = true
	}

	result := make(map[string]int)
	for rows.Next() {
		var rID, cin, cout string
		var count int
		if err := rows.Scan(&rID, &cin, &cout, &count); err != nil {
			return nil, err
		}
		nights, err := dateutil.NightsBetween(cin, cout)
		if err != nil {
			continue
		}
		for _, n := range nights {
			if dateSet[n] {
				k := fmt.Sprintf("%s:%s", rID, n)
				result[k] += count
			}
		}
	}
	return result, rows.Err()
}

// ── Bookings ───────────────────────────────────────────────────────────────

func (s *PostgresStore) InitiateBookingHold(ctx context.Context, booking *domain.BookingRecord) error {
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

	query := `INSERT INTO bookings (
                id, booking_code, user_id, property_id, room_type_id,
                guest_name, guest_phone, guest_email, company_name, gstin,
                check_in, check_out, rooms_count, guests_count, adults_count, children_count,
                child_ages, extra_adults, extra_adult_percent, extra_adult_charge,
                total_amount, payment_mode, payment_status, booking_status, meal_plan,
                created_at, updated_at
              ) VALUES (
                $1, $2, $3, $4, $5,
                $6, $7, $8, $9, $10,
                $11, $12, $13, $14, $15, $16,
                $17, $18, $19, $20,
                $21, $22, $23, $24, $25,
                NOW(), NOW()
              ) RETURNING created_at, updated_at`

	return s.pool.QueryRow(ctx, query,
		booking.ID, booking.BookingCode, booking.UserID, booking.PropertyID, booking.RoomTypeID,
		booking.GuestName, booking.GuestPhone, booking.GuestEmail, booking.CompanyName, booking.GSTIN,
		booking.CheckIn, booking.CheckOut, booking.RoomsCount, booking.GuestsCount, booking.AdultsCount, booking.ChildrenCount,
		booking.ChildAges, booking.ExtraAdults, booking.ExtraAdultPercent, booking.ExtraAdultCharge,
		booking.TotalAmount, booking.PaymentMode, booking.PaymentStatus, booking.BookingStatus, booking.MealPlan,
	).Scan(&booking.CreatedAt, &booking.UpdatedAt)
}

func (s *PostgresStore) GetBookingByCode(ctx context.Context, bookingCode string, guestPhone *string) (*domain.BookingRecord, error) {
	query := `SELECT id, booking_code, user_id, property_id, room_type_id,
                     guest_name, guest_phone, guest_email, company_name, gstin,
                     check_in, check_out, rooms_count, guests_count, adults_count, children_count,
                     child_ages, extra_adults, extra_adult_percent, extra_adult_charge,
                     total_amount, payment_mode, payment_status, razorpay_order_id,
                     razorpay_payment_id, razorpay_payment_link_id, booking_status,
                     meal_plan, created_at, updated_at, cm_sync_status, cm_res_status,
                     cm_synced_at, cm_attempts, cm_last_error
              FROM bookings WHERE booking_code = $1`

	var b domain.BookingRecord
	err := s.pool.QueryRow(ctx, query, bookingCode).Scan(
		&b.ID, &b.BookingCode, &b.UserID, &b.PropertyID, &b.RoomTypeID,
		&b.GuestName, &b.GuestPhone, &b.GuestEmail, &b.CompanyName, &b.GSTIN,
		&b.CheckIn, &b.CheckOut, &b.RoomsCount, &b.GuestsCount, &b.AdultsCount, &b.ChildrenCount,
		&b.ChildAges, &b.ExtraAdults, &b.ExtraAdultPercent, &b.ExtraAdultCharge,
		&b.TotalAmount, &b.PaymentMode, &b.PaymentStatus, &b.RazorpayOrderID,
		&b.RazorpayPaymentID, &b.RazorpayPaymentLinkID, &b.BookingStatus,
		&b.MealPlan, &b.CreatedAt, &b.UpdatedAt, &b.CMSyncStatus, &b.CMResStatus,
		&b.CMSyncedAt, &b.CMAttempts, &b.CMLastError,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
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

func (s *PostgresStore) GetBookingsForUser(ctx context.Context, userID, email string) ([]domain.BookingRecord, error) {
	query := `SELECT id, booking_code, user_id, property_id, room_type_id,
                     guest_name, guest_phone, guest_email, company_name, gstin,
                     check_in, check_out, rooms_count, guests_count, adults_count, children_count,
                     child_ages, extra_adults, extra_adult_percent, extra_adult_charge,
                     total_amount, payment_mode, payment_status, razorpay_order_id,
                     razorpay_payment_id, razorpay_payment_link_id, booking_status,
                     meal_plan, created_at, updated_at, cm_sync_status, cm_res_status,
                     cm_synced_at, cm_attempts, cm_last_error
              FROM bookings
              WHERE user_id = $1 OR LOWER(guest_email) = LOWER($2)
              ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, query, userID, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.BookingRecord
	for rows.Next() {
		var b domain.BookingRecord
		if err := rows.Scan(
			&b.ID, &b.BookingCode, &b.UserID, &b.PropertyID, &b.RoomTypeID,
			&b.GuestName, &b.GuestPhone, &b.GuestEmail, &b.CompanyName, &b.GSTIN,
			&b.CheckIn, &b.CheckOut, &b.RoomsCount, &b.GuestsCount, &b.AdultsCount, &b.ChildrenCount,
			&b.ChildAges, &b.ExtraAdults, &b.ExtraAdultPercent, &b.ExtraAdultCharge,
			&b.TotalAmount, &b.PaymentMode, &b.PaymentStatus, &b.RazorpayOrderID,
			&b.RazorpayPaymentID, &b.RazorpayPaymentLinkID, &b.BookingStatus,
			&b.MealPlan, &b.CreatedAt, &b.UpdatedAt, &b.CMSyncStatus, &b.CMResStatus,
			&b.CMSyncedAt, &b.CMAttempts, &b.CMLastError,
		); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (s *PostgresStore) UpdateBookingPayment(
	ctx context.Context,
	bookingCode string,
	paymentID, orderID *string,
	status domain.BookingStatus,
	payStatus domain.PaymentStatus,
) (*domain.BookingRecord, error) {
	cmSync := domain.CMSyncStatusNone
	var cmRes *string
	if status == domain.BookingStatusConfirmed {
		cmSync = domain.CMSyncStatusPending
		confirm := "Confirm"
		cmRes = &confirm
	} else if status == domain.BookingStatusCancelled {
		cmSync = domain.CMSyncStatusPending
		cancel := "Cancel"
		cmRes = &cancel
	}

	query := `UPDATE bookings SET
                razorpay_payment_id = COALESCE($1, razorpay_payment_id),
                razorpay_order_id = COALESCE($2, razorpay_order_id),
                booking_status = $3,
                payment_status = $4,
                cm_sync_status = CASE WHEN $5 != 'NONE' THEN $5 ELSE cm_sync_status END,
                cm_res_status = CASE WHEN $6 IS NOT NULL THEN $6 ELSE cm_res_status END,
                updated_at = NOW()
              WHERE booking_code = $7
              RETURNING id, booking_code, user_id, property_id, room_type_id,
                        guest_name, guest_phone, guest_email, company_name, gstin,
                        check_in, check_out, rooms_count, guests_count, adults_count, children_count,
                        child_ages, extra_adults, extra_adult_percent, extra_adult_charge,
                        total_amount, payment_mode, payment_status, razorpay_order_id,
                        razorpay_payment_id, razorpay_payment_link_id, booking_status,
                        meal_plan, created_at, updated_at, cm_sync_status, cm_res_status,
                        cm_synced_at, cm_attempts, cm_last_error`

	var b domain.BookingRecord
	err := s.pool.QueryRow(ctx, query, paymentID, orderID, status, payStatus, cmSync, cmRes, bookingCode).Scan(
		&b.ID, &b.BookingCode, &b.UserID, &b.PropertyID, &b.RoomTypeID,
		&b.GuestName, &b.GuestPhone, &b.GuestEmail, &b.CompanyName, &b.GSTIN,
		&b.CheckIn, &b.CheckOut, &b.RoomsCount, &b.GuestsCount, &b.AdultsCount, &b.ChildrenCount,
		&b.ChildAges, &b.ExtraAdults, &b.ExtraAdultPercent, &b.ExtraAdultCharge,
		&b.TotalAmount, &b.PaymentMode, &b.PaymentStatus, &b.RazorpayOrderID,
		&b.RazorpayPaymentID, &b.RazorpayPaymentLinkID, &b.BookingStatus,
		&b.MealPlan, &b.CreatedAt, &b.UpdatedAt, &b.CMSyncStatus, &b.CMResStatus,
		&b.CMSyncedAt, &b.CMAttempts, &b.CMLastError,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *PostgresStore) CleanupExpiredHolds(ctx context.Context, expireThresholdMinutes int) (int, error) {
	query := `UPDATE bookings
              SET booking_status = 'EXPIRED',
                  payment_status = 'FAILED',
                  updated_at = NOW()
              WHERE booking_status = 'PENDING_PAYMENT'
                AND created_at < NOW() - ($1 || ' minutes')::INTERVAL`

	tag, err := s.pool.Exec(ctx, query, expireThresholdMinutes)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (s *PostgresStore) GetAllBookings(ctx context.Context, limit int) ([]domain.BookingRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, booking_code, user_id, property_id, room_type_id,
                     guest_name, guest_phone, guest_email, company_name, gstin,
                     check_in, check_out, rooms_count, guests_count, adults_count, children_count,
                     child_ages, extra_adults, extra_adult_percent, extra_adult_charge,
                     total_amount, payment_mode, payment_status, razorpay_order_id,
                     razorpay_payment_id, razorpay_payment_link_id, booking_status,
                     meal_plan, created_at, updated_at, cm_sync_status, cm_res_status,
                     cm_synced_at, cm_attempts, cm_last_error
              FROM bookings
              ORDER BY created_at DESC
              LIMIT $1`

	rows, err := s.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.BookingRecord
	for rows.Next() {
		var b domain.BookingRecord
		if err := rows.Scan(
			&b.ID, &b.BookingCode, &b.UserID, &b.PropertyID, &b.RoomTypeID,
			&b.GuestName, &b.GuestPhone, &b.GuestEmail, &b.CompanyName, &b.GSTIN,
			&b.CheckIn, &b.CheckOut, &b.RoomsCount, &b.GuestsCount, &b.AdultsCount, &b.ChildrenCount,
			&b.ChildAges, &b.ExtraAdults, &b.ExtraAdultPercent, &b.ExtraAdultCharge,
			&b.TotalAmount, &b.PaymentMode, &b.PaymentStatus, &b.RazorpayOrderID,
			&b.RazorpayPaymentID, &b.RazorpayPaymentLinkID, &b.BookingStatus,
			&b.MealPlan, &b.CreatedAt, &b.UpdatedAt, &b.CMSyncStatus, &b.CMResStatus,
			&b.CMSyncedAt, &b.CMAttempts, &b.CMLastError,
		); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (s *PostgresStore) GetGlanceMetrics(ctx context.Context) (map[string]interface{}, error) {
	today := time.Now().UTC().Format("2006-01-02")
	query := `SELECT
                COALESCE(COUNT(*) FILTER (WHERE booking_status = 'CONFIRMED'), 0) AS total_bookings,
                COALESCE(SUM(total_amount) FILTER (WHERE booking_status = 'CONFIRMED'), 0) AS total_revenue,
                COALESCE(COUNT(*) FILTER (WHERE booking_status = 'PENDING_PAYMENT' AND created_at > NOW() - INTERVAL '15 minutes'), 0) AS active_holds,
                COALESCE(COUNT(*) FILTER (WHERE booking_status = 'CONFIRMED' AND check_in = $1), 0) AS today_checkins
              FROM bookings`

	var totalBookings, activeHolds, todayCheckins int
	var totalRevenue float64
	err := s.pool.QueryRow(ctx, query, today).Scan(&totalBookings, &totalRevenue, &activeHolds, &todayCheckins)
	if err != nil {
		return nil, err
	}

	var activeHotels int
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM properties WHERE is_active = true`).Scan(&activeHotels)

	return map[string]interface{}{
		"totalBookings": totalBookings,
		"totalRevenue":  totalRevenue,
		"activeHolds":   activeHolds,
		"todayCheckins": todayCheckins,
		"activeHotels":  activeHotels,
	}, nil
}

// ── Channel Manager (ResAvenue) ────────────────────────────────────────────

func (s *PostgresStore) GetInventoryDays(ctx context.Context, roomTypeIDs []string, dates []string) ([]domain.InventoryDayRecord, error) {
	if len(roomTypeIDs) == 0 || len(dates) == 0 {
		return nil, nil
	}
	query := `SELECT room_type_id, stay_date, inv_count, stop_sell, close_on_arrival, close_on_departure, cut_off, updated_at
              FROM room_inventory_days
              WHERE room_type_id = ANY($1) AND stay_date = ANY($2)`

	rows, err := s.pool.Query(ctx, query, roomTypeIDs, dates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.InventoryDayRecord
	for rows.Next() {
		var rec domain.InventoryDayRecord
		var stayTime time.Time
		if err := rows.Scan(
			&rec.RoomTypeID, &stayTime, &rec.InvCount, &rec.StopSell,
			&rec.CloseOnArrival, &rec.CloseOnDeparture, &rec.CutOff, &rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rec.StayDate = stayTime.Format("2006-01-02")
		result = append(result, rec)
	}
	return result, rows.Err()
}

func (s *PostgresStore) UpsertInventoryDays(ctx context.Context, rows []domain.InventoryDayRecord) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	count := 0
	for _, r := range rows {
		query := `INSERT INTO room_inventory_days (
                    room_type_id, stay_date, inv_count, stop_sell, close_on_arrival, close_on_departure, cut_off, updated_at
                  ) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
                  ON CONFLICT (room_type_id, stay_date) DO UPDATE SET
                    inv_count = COALESCE(EXCLUDED.inv_count, room_inventory_days.inv_count),
                    stop_sell = EXCLUDED.stop_sell,
                    close_on_arrival = EXCLUDED.close_on_arrival,
                    close_on_departure = EXCLUDED.close_on_departure,
                    cut_off = EXCLUDED.cut_off,
                    updated_at = NOW()`
		_, err := s.pool.Exec(ctx, query, r.RoomTypeID, r.StayDate, r.InvCount, r.StopSell, r.CloseOnArrival, r.CloseOnDeparture, r.CutOff)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *PostgresStore) GetRateDays(ctx context.Context, roomTypeIDs []string, plans []domain.MealPlan, dates []string) ([]domain.RateDayRecord, error) {
	if len(roomTypeIDs) == 0 || len(dates) == 0 {
		return nil, nil
	}
	query := `SELECT room_type_id, meal_plan, stay_date, single, double, triple, quad,
                     extra_adult, extra_child, min_stay, max_stay, stop_sell, updated_at
              FROM room_rate_days
              WHERE room_type_id = ANY($1) AND stay_date = ANY($2)`

	rows, err := s.pool.Query(ctx, query, roomTypeIDs, dates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.RateDayRecord
	for rows.Next() {
		var rec domain.RateDayRecord
		var stayTime time.Time
		if err := rows.Scan(
			&rec.RoomTypeID, &rec.MealPlan, &stayTime, &rec.Single, &rec.Double, &rec.Triple, &rec.Quad,
			&rec.ExtraAdult, &rec.ExtraChild, &rec.MinStay, &rec.MaxStay, &rec.StopSell, &rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rec.StayDate = stayTime.Format("2006-01-02")
		result = append(result, rec)
	}
	return result, rows.Err()
}

func (s *PostgresStore) UpsertRateDays(ctx context.Context, rows []domain.RateDayRecord) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	count := 0
	for _, r := range rows {
		query := `INSERT INTO room_rate_days (
                    room_type_id, meal_plan, stay_date, single, double, triple, quad,
                    extra_adult, extra_child, min_stay, max_stay, stop_sell, updated_at
                  ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
                  ON CONFLICT (room_type_id, meal_plan, stay_date) DO UPDATE SET
                    single = COALESCE(EXCLUDED.single, room_rate_days.single),
                    double = COALESCE(EXCLUDED.double, room_rate_days.double),
                    triple = COALESCE(EXCLUDED.triple, room_rate_days.triple),
                    quad = COALESCE(EXCLUDED.quad, room_rate_days.quad),
                    extra_adult = COALESCE(EXCLUDED.extra_adult, room_rate_days.extra_adult),
                    extra_child = COALESCE(EXCLUDED.extra_child, room_rate_days.extra_child),
                    min_stay = COALESCE(EXCLUDED.min_stay, room_rate_days.min_stay),
                    max_stay = COALESCE(EXCLUDED.max_stay, room_rate_days.max_stay),
                    stop_sell = EXCLUDED.stop_sell,
                    updated_at = NOW()`
		_, err := s.pool.Exec(ctx, query,
			r.RoomTypeID, r.MealPlan, r.StayDate, r.Single, r.Double, r.Triple, r.Quad,
			r.ExtraAdult, r.ExtraChild, r.MinStay, r.MaxStay, r.StopSell,
		)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *PostgresStore) GetBookingsForChannelPull(ctx context.Context, propertyID, fromDate, toDate string, limit int) ([]domain.BookingRecord, error) {
	if limit <= 0 {
		limit = 500
	}
	query := `SELECT id, booking_code, user_id, property_id, room_type_id,
                     guest_name, guest_phone, guest_email, company_name, gstin,
                     check_in, check_out, rooms_count, guests_count, adults_count, children_count,
                     child_ages, extra_adults, extra_adult_percent, extra_adult_charge,
                     total_amount, payment_mode, payment_status, razorpay_order_id,
                     razorpay_payment_id, razorpay_payment_link_id, booking_status,
                     meal_plan, created_at, updated_at, cm_sync_status, cm_res_status,
                     cm_synced_at, cm_attempts, cm_last_error
              FROM bookings
              WHERE ($1 = '' OR property_id = $1)
                AND ($2 = '' OR updated_at::date >= $2::date)
                AND ($3 = '' OR updated_at::date <= $3::date)
              ORDER BY updated_at DESC
              LIMIT $4`

	rows, err := s.pool.Query(ctx, query, propertyID, fromDate, toDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.BookingRecord
	for rows.Next() {
		var b domain.BookingRecord
		if err := rows.Scan(
			&b.ID, &b.BookingCode, &b.UserID, &b.PropertyID, &b.RoomTypeID,
			&b.GuestName, &b.GuestPhone, &b.GuestEmail, &b.CompanyName, &b.GSTIN,
			&b.CheckIn, &b.CheckOut, &b.RoomsCount, &b.GuestsCount, &b.AdultsCount, &b.ChildrenCount,
			&b.ChildAges, &b.ExtraAdults, &b.ExtraAdultPercent, &b.ExtraAdultCharge,
			&b.TotalAmount, &b.PaymentMode, &b.PaymentStatus, &b.RazorpayOrderID,
			&b.RazorpayPaymentID, &b.RazorpayPaymentLinkID, &b.BookingStatus,
			&b.MealPlan, &b.CreatedAt, &b.UpdatedAt, &b.CMSyncStatus, &b.CMResStatus,
			&b.CMSyncedAt, &b.CMAttempts, &b.CMLastError,
		); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (s *PostgresStore) GetBookingsPendingChannelSync(ctx context.Context, maxAttempts, limit int) ([]domain.BookingRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, booking_code, user_id, property_id, room_type_id,
                     guest_name, guest_phone, guest_email, company_name, gstin,
                     check_in, check_out, rooms_count, guests_count, adults_count, children_count,
                     child_ages, extra_adults, extra_adult_percent, extra_adult_charge,
                     total_amount, payment_mode, payment_status, razorpay_order_id,
                     razorpay_payment_id, razorpay_payment_link_id, booking_status,
                     meal_plan, created_at, updated_at, cm_sync_status, cm_res_status,
                     cm_synced_at, cm_attempts, cm_last_error
              FROM bookings
              WHERE cm_sync_status IN ('PENDING', 'FAILED')
                AND cm_attempts < $1
              ORDER BY updated_at ASC
              LIMIT $2`

	rows, err := s.pool.Query(ctx, query, maxAttempts, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.BookingRecord
	for rows.Next() {
		var b domain.BookingRecord
		if err := rows.Scan(
			&b.ID, &b.BookingCode, &b.UserID, &b.PropertyID, &b.RoomTypeID,
			&b.GuestName, &b.GuestPhone, &b.GuestEmail, &b.CompanyName, &b.GSTIN,
			&b.CheckIn, &b.CheckOut, &b.RoomsCount, &b.GuestsCount, &b.AdultsCount, &b.ChildrenCount,
			&b.ChildAges, &b.ExtraAdults, &b.ExtraAdultPercent, &b.ExtraAdultCharge,
			&b.TotalAmount, &b.PaymentMode, &b.PaymentStatus, &b.RazorpayOrderID,
			&b.RazorpayPaymentID, &b.RazorpayPaymentLinkID, &b.BookingStatus,
			&b.MealPlan, &b.CreatedAt, &b.UpdatedAt, &b.CMSyncStatus, &b.CMResStatus,
			&b.CMSyncedAt, &b.CMAttempts, &b.CMLastError,
		); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (s *PostgresStore) MarkChannelSync(ctx context.Context, bookingID string, status domain.CMSyncStatus, resStatus *string, errStr *string) error {
	query := `UPDATE bookings SET
                cm_sync_status = $1,
                cm_res_status = COALESCE($2, cm_res_status),
                cm_attempts = cm_attempts + 1,
                cm_synced_at = NOW(),
                cm_last_error = $3
              WHERE id = $4`

	_, err := s.pool.Exec(ctx, query, status, resStatus, errStr, bookingID)
	return err
}

// ── Users ──────────────────────────────────────────────────────────────────

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*domain.UserRecord, error) {
	query := `SELECT id, full_name, email, phone, password_hash, created_at
              FROM users WHERE LOWER(email) = LOWER($1)`

	var u domain.UserRecord
	err := s.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.FullName, &u.Email, &u.Phone, &u.PasswordHash, &u.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (*domain.UserRecord, error) {
	query := `SELECT id, full_name, email, phone, password_hash, created_at
              FROM users WHERE id = $1`

	var u domain.UserRecord
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.FullName, &u.Email, &u.Phone, &u.PasswordHash, &u.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (s *PostgresStore) CreateUser(ctx context.Context, user *domain.UserRecord) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	query := `INSERT INTO users (id, full_name, email, phone, password_hash, created_at)
              VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING created_at`
	return s.pool.QueryRow(ctx, query, user.ID, user.FullName, user.Email, user.Phone, user.PasswordHash).Scan(&user.CreatedAt)
}

// ── Admin PIN & Site Content ───────────────────────────────────────────────

func (s *PostgresStore) GetAdminPINHash(ctx context.Context) (string, error) {
	var hash string
	err := s.pool.QueryRow(ctx, `SELECT pin_hash FROM admin_pin WHERE id = 1`).Scan(&hash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return hash, nil
}

func (s *PostgresStore) SetAdminPINHash(ctx context.Context, hash string) error {
	query := `INSERT INTO admin_pin (id, pin_hash, updated_at)
              VALUES (1, $1, NOW())
              ON CONFLICT (id) DO UPDATE SET pin_hash = EXCLUDED.pin_hash, updated_at = NOW()`
	_, err := s.pool.Exec(ctx, query, hash)
	return err
}

func (s *PostgresStore) GetSiteContent(ctx context.Context) (map[string]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT key, value FROM site_content`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}

func (s *PostgresStore) SetSiteContent(ctx context.Context, entries map[string]string) (map[string]string, error) {
	for k, v := range entries {
		query := `INSERT INTO site_content (key, value, updated_at)
                  VALUES ($1, $2, NOW())
                  ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`
		if _, err := s.pool.Exec(ctx, query, k, v); err != nil {
			return nil, err
		}
	}
	return s.GetSiteContent(ctx)
}

// ── Enquiries ──────────────────────────────────────────────────────────────

func (s *PostgresStore) CreateEnquiry(ctx context.Context, enq *domain.EnquiryRecord) error {
	if enq.ID == "" {
		enq.ID = uuid.New().String()
	}
	enq.Status = domain.EnquiryStatusNew
	query := `INSERT INTO enquiries (id, enquiry_type, property_id, guest_name, guest_phone, guest_email, event_date, guest_count, message, status, created_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW()) RETURNING created_at`
	return s.pool.QueryRow(ctx, query, enq.ID, enq.EnquiryType, enq.PropertyID, enq.GuestName, enq.GuestPhone, enq.GuestEmail, enq.EventDate, enq.GuestCount, enq.Message, enq.Status).Scan(&enq.CreatedAt)
}

func (s *PostgresStore) GetEnquiries(ctx context.Context, status *domain.EnquiryStatus) ([]domain.EnquiryRecord, error) {
	query := `SELECT id, enquiry_type, property_id, guest_name, guest_phone, guest_email, event_date, guest_count, message, status, razorpay_payment_link_id, created_at
              FROM enquiries
              WHERE ($1::text IS NULL OR status = $1)
              ORDER BY created_at DESC`

	var statusStr *string
	if status != nil {
		str := string(*status)
		statusStr = &str
	}

	rows, err := s.pool.Query(ctx, query, statusStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.EnquiryRecord
	for rows.Next() {
		var e domain.EnquiryRecord
		if err := rows.Scan(&e.ID, &e.EnquiryType, &e.PropertyID, &e.GuestName, &e.GuestPhone, &e.GuestEmail, &e.EventDate, &e.GuestCount, &e.Message, &e.Status, &e.RazorpayPaymentLinkID, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

func (s *PostgresStore) GetEnquiryByID(ctx context.Context, id string) (*domain.EnquiryRecord, error) {
	query := `SELECT id, enquiry_type, property_id, guest_name, guest_phone, guest_email, event_date, guest_count, message, status, razorpay_payment_link_id, created_at
              FROM enquiries WHERE id = $1`

	var e domain.EnquiryRecord
	err := s.pool.QueryRow(ctx, query, id).Scan(&e.ID, &e.EnquiryType, &e.PropertyID, &e.GuestName, &e.GuestPhone, &e.GuestEmail, &e.EventDate, &e.GuestCount, &e.Message, &e.Status, &e.RazorpayPaymentLinkID, &e.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (s *PostgresStore) UpdateEnquiryStatus(ctx context.Context, id string, status domain.EnquiryStatus, paymentLinkID *string) (*domain.EnquiryRecord, error) {
	query := `UPDATE enquiries SET
                status = $1,
                razorpay_payment_link_id = COALESCE($2, razorpay_payment_link_id)
              WHERE id = $3
              RETURNING id, enquiry_type, property_id, guest_name, guest_phone, guest_email, event_date, guest_count, message, status, razorpay_payment_link_id, created_at`

	var e domain.EnquiryRecord
	err := s.pool.QueryRow(ctx, query, status, paymentLinkID, id).Scan(&e.ID, &e.EnquiryType, &e.PropertyID, &e.GuestName, &e.GuestPhone, &e.GuestEmail, &e.EventDate, &e.GuestCount, &e.Message, &e.Status, &e.RazorpayPaymentLinkID, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ── Chat Logs ──────────────────────────────────────────────────────────────

func (s *PostgresStore) CreateChatLog(ctx context.Context, chat *domain.ChatLogRecord) error {
	if chat.ID == "" {
		chat.ID = uuid.New().String()
	}
	query := `INSERT INTO chat_logs (id, session_id, user_message, bot_response, tools_invoked, handoff_triggered, created_at)
              VALUES ($1, $2, $3, $4, $5, $6, NOW()) RETURNING created_at`
	return s.pool.QueryRow(ctx, query, chat.ID, chat.SessionID, chat.UserMessage, chat.BotResponse, chat.ToolsInvoked, chat.HandoffTriggered).Scan(&chat.CreatedAt)
}
