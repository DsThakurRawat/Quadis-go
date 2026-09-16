package memory

import (
	"context"
	"testing"

	"quadis-backend-go/internal/domain"
)

func TestMemoryStore_PropertiesAndRoomsSeed(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	props, err := store.GetProperties(ctx)
	if err != nil {
		t.Fatalf("failed to get properties: %v", err)
	}

	// Must have 10 properties including Amaltas International
	if len(props) != 10 {
		t.Errorf("expected 10 properties, got %d", len(props))
	}

	// Verify Hotel Amaltas International
	p, rooms, err := store.GetPropertyBySlug(ctx, "hotel-amaltas-international")
	if err != nil {
		t.Fatalf("failed to get Amaltas: %v", err)
	}
	if p == nil {
		t.Fatal("expected Hotel Amaltas International to exist in seed data")
	}
	if p.ID != "prop-11" {
		t.Errorf("expected prop-11, got %s", p.ID)
	}
	if p.BasePrice != 4000 {
		t.Errorf("expected base price 4000, got %f", p.BasePrice)
	}

	// Verify Amaltas room types (Deluxe 6 keys, Superior 2 keys = 8 keys)
	if len(rooms) != 2 {
		t.Fatalf("expected 2 room types for Amaltas, got %d", len(rooms))
	}
	totalAmaltasKeys := 0
	hasSuperior := false
	for _, r := range rooms {
		totalAmaltasKeys += r.TotalUnits
		if r.Slug == "superior-room" {
			hasSuperior = true
			if r.PriceOffset != 2000 {
				t.Errorf("expected superior price offset 2000, got %f", r.PriceOffset)
			}
		}
	}
	if !hasSuperior {
		t.Error("expected superior-room in Amaltas rooms")
	}
	if totalAmaltasKeys != 8 {
		t.Errorf("expected 8 keys for Amaltas, got %d", totalAmaltasKeys)
	}

	// Verify total keys across all properties = 205 (197 old + 8 Amaltas)
	allPropsWithRooms, err := store.GetPropertiesWithRooms(ctx)
	if err != nil {
		t.Fatalf("failed to get all properties with rooms: %v", err)
	}
	totalGroupKeys := 0
	for _, prop := range allPropsWithRooms {
		for _, room := range prop.Rooms {
			totalGroupKeys += room.TotalUnits
		}
	}
	if totalGroupKeys != 205 {
		t.Errorf("expected 205 group keys across 10 properties, got %d", totalGroupKeys)
	}
}

func TestMemoryStore_BookingHoldAndAvailability(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	// Pick a room: Quadis 51 deluxe room has 25 keys
	roomID := "room-prop-2-deluxe-room"
	checkIn := "2026-10-01"
	checkOut := "2026-10-03" // 2 nights

	avail, err := store.GetAvailableUnits(ctx, roomID, checkIn, checkOut)
	if err != nil {
		t.Fatalf("failed to get available units: %v", err)
	}
	if avail != 25 {
		t.Errorf("expected 25 units initially, got %d", avail)
	}

	// Create a hold for 3 rooms
	booking := &domain.BookingRecord{
		PropertyID:  "prop-2",
		RoomTypeID:  roomID,
		GuestName:   "Aarav Sharma",
		GuestPhone:  "+91 9811223344",
		CheckIn:     checkIn,
		CheckOut:    checkOut,
		RoomsCount:  3,
		GuestsCount: 6,
		TotalAmount: 9000,
		PaymentMode: domain.PaymentModeInstantFullPayment,
	}

	err = store.InitiateBookingHold(ctx, booking)
	if err != nil {
		t.Fatalf("failed to initiate booking hold: %v", err)
	}
	if booking.BookingCode == "" {
		t.Error("expected booking code to be generated")
	}

	// Availability must now drop to 25 - 3 = 22
	availAfter, err := store.GetAvailableUnits(ctx, roomID, checkIn, checkOut)
	if err != nil {
		t.Fatalf("failed to get available units after hold: %v", err)
	}
	if availAfter != 22 {
		t.Errorf("expected 22 available units after 3-room hold, got %d", availAfter)
	}

	// Confirm the booking
	payID := "pay_test_123"
	orderID := "order_test_123"
	updated, err := store.UpdateBookingPayment(
		ctx,
		booking.BookingCode,
		&payID,
		&orderID,
		domain.BookingStatusConfirmed,
		domain.PaymentStatusPaid,
	)
	if err != nil {
		t.Fatalf("failed to update booking payment: %v", err)
	}
	if updated.BookingStatus != domain.BookingStatusConfirmed {
		t.Errorf("expected CONFIRMED status, got %s", updated.BookingStatus)
	}
	if updated.CMSyncStatus != domain.CMSyncStatusPending {
		t.Errorf("expected PENDING channel sync status on confirmation, got %s", updated.CMSyncStatus)
	}
}

func TestMemoryStore_ResAvenueOverrides(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	roomID := "room-prop-2-deluxe-room"
	date := "2026-10-05"

	// Upsert inventory override: set units to 2 and stop_sell = false
	units := 2
	n, err := store.UpsertInventoryDays(ctx, []domain.InventoryDayRecord{
		{
			RoomTypeID: roomID,
			StayDate:   date,
			InvCount:   &units,
			StopSell:   false,
		},
	})
	if err != nil || n != 1 {
		t.Fatalf("UpsertInventoryDays failed: %v", err)
	}

	// Check available units on that night
	avail, err := store.GetAvailableUnits(ctx, roomID, "2026-10-05", "2026-10-06")
	if err != nil {
		t.Fatalf("failed to check available units: %v", err)
	}
	if avail != 2 {
		t.Errorf("expected 2 available units from ResAvenue override, got %d", avail)
	}

	// Now set stop_sell = true
	_, err = store.UpsertInventoryDays(ctx, []domain.InventoryDayRecord{
		{
			RoomTypeID: roomID,
			StayDate:   date,
			InvCount:   &units,
			StopSell:   true,
		},
	})
	if err != nil {
		t.Fatalf("UpsertInventoryDays stop_sell failed: %v", err)
	}

	availStopped, err := store.GetAvailableUnits(ctx, roomID, "2026-10-05", "2026-10-06")
	if err != nil {
		t.Fatalf("failed to check available units: %v", err)
	}
	if availStopped != 0 {
		t.Errorf("expected 0 available units with stop_sell active, got %d", availStopped)
	}
}
