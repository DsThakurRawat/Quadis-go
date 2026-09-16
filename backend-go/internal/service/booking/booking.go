package booking

import (
	"context"
	"errors"
	"fmt"
	"time"

	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
	"quadis-backend-go/internal/service/pricing"
	"quadis-backend-go/pkg/dateutil"
)

var (
	ErrRoomUnavailable = errors.New("no units available for selected dates")
	ErrInvalidDates    = errors.New("invalid stay dates")
	ErrInvalidGuests   = errors.New("invalid guest count")
)

type CreateHoldRequest struct {
	UserID      *string
	PropertyID  string
	RoomTypeID  string
	GuestName   string
	GuestPhone  string
	GuestEmail  *string
	CompanyName *string
	GSTIN       *string
	CheckIn     string
	CheckOut    string
	RoomsCount  int
	AdultsCount int
	ChildAges   []int
	MealPlan    *domain.MealPlan
	PaymentMode domain.PaymentMode
}

type BookingService struct {
	repo repository.Repository
}

func NewBookingService(repo repository.Repository) *BookingService {
	return &BookingService{repo: repo}
}

func (s *BookingService) CreateHold(ctx context.Context, req CreateHoldRequest) (*domain.BookingRecord, *pricing.StayPricingBreakdown, error) {
	if req.RoomsCount <= 0 {
		req.RoomsCount = 1
	}
	if req.AdultsCount <= 0 {
		req.AdultsCount = 1
	}

	nights, err := dateutil.NightsBetween(req.CheckIn, req.CheckOut)
	if err != nil || len(nights) == 0 {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidDates, err)
	}

	// Verify property & room existence
	prop, err := s.repo.GetPropertyByIDOrSlug(ctx, req.PropertyID)
	if err != nil {
		return nil, nil, err
	}
	if prop == nil {
		return nil, nil, fmt.Errorf("property not found: %s", req.PropertyID)
	}

	room, err := s.repo.GetRoomTypeByID(ctx, req.RoomTypeID)
	if err != nil {
		return nil, nil, err
	}
	if room == nil {
		return nil, nil, fmt.Errorf("room type not found: %s", req.RoomTypeID)
	}

	// Check availability
	avail, err := s.repo.GetAvailableUnits(ctx, room.ID, req.CheckIn, req.CheckOut)
	if err != nil {
		return nil, nil, err
	}
	if avail < req.RoomsCount {
		return nil, nil, fmt.Errorf("%w: requested %d, available %d", ErrRoomUnavailable, req.RoomsCount, avail)
	}

	// Calculate occupancy
	chargeable := pricing.ChargeableGuestsFor(pricing.OccupancyInput{
		Adults:            req.AdultsCount,
		ChildAges:         req.ChildAges,
		RoomsCount:        req.RoomsCount,
		ChildFreeUnderAge: &prop.ChildFreeUnderAge,
		AdultFromAge:      prop.AdultFromAge,
	})

	// Calculate meal supplement
	var mealOffset float64
	if req.MealPlan != nil {
		mealOffset = pricing.MealOffsetFor(*req.MealPlan, prop.BasePrice, room.PriceOffset, room.BreakfastOffset, room.AllMealsOffset)
	}

	// Query per-night rate overrides from channel manager if any
	rateDays, _ := s.repo.GetRateDays(ctx, []string{room.ID}, []domain.MealPlan{domain.MealPlanRoomOnly, domain.MealPlanWithBreakfast, domain.MealPlanAllMealsIncluded}, nights)
	overridesMap := make(map[string]pricing.NightOverride)
	for _, rd := range rateDays {
		if req.MealPlan != nil && rd.MealPlan != *req.MealPlan {
			continue
		}
		var rateVal float64
		if rd.Double != nil && *rd.Double > 0 {
			rateVal = *rd.Double
		}
		overridesMap[rd.StayDate] = pricing.NightOverride{
			Rate:             rateVal,
			ExtraAdultCharge: rd.ExtraAdult,
			ExtraChildCharge: rd.ExtraChild,
		}
	}

	pricingInput := pricing.StayPricingInput{
		BasePrice:               prop.BasePrice,
		RoomOffset:              room.PriceOffset,
		MealOffset:              mealOffset,
		WeekendSurchargePercent: prop.WeekendSurchargePercent,
		CheckIn:                 req.CheckIn,
		CheckOut:                req.CheckOut,
		RoomsCount:              req.RoomsCount,
		ExtraAdults:             chargeable.ExtraAdults,
		ExtraChildren:           chargeable.ExtraChildren,
		ExtraAdultPercent:       &prop.ExtraAdultPercent,
		ChildPercent:            prop.ChildPercent,
		NightOverrides:          overridesMap,
	}

	breakdown := pricing.ComputeStayBreakdown(pricingInput)

	paymentMode := req.PaymentMode
	if paymentMode == "" {
		paymentMode = domain.PaymentModeInstantFullPayment
	}

	booking := &domain.BookingRecord{
		UserID:            req.UserID,
		PropertyID:        prop.ID,
		RoomTypeID:        room.ID,
		GuestName:         req.GuestName,
		GuestPhone:        req.GuestPhone,
		GuestEmail:        req.GuestEmail,
		CompanyName:       req.CompanyName,
		GSTIN:             req.GSTIN,
		CheckIn:           req.CheckIn,
		CheckOut:          req.CheckOut,
		RoomsCount:        req.RoomsCount,
		GuestsCount:       req.AdultsCount + len(req.ChildAges),
		AdultsCount:       req.AdultsCount,
		ChildrenCount:     len(req.ChildAges),
		ChildAges:         req.ChildAges,
		ExtraAdults:       breakdown.ExtraAdults,
		ExtraAdultPercent: breakdown.ExtraAdultPercent,
		ExtraAdultCharge:  breakdown.ExtraAdultChargePerNight,
		TotalAmount:       breakdown.Total,
		PaymentMode:       paymentMode,
		MealPlan:          req.MealPlan,
	}

	err = s.repo.InitiateBookingHold(ctx, booking)
	if err != nil {
		return nil, nil, err
	}

	return booking, &breakdown, nil
}

func (s *BookingService) GetBooking(ctx context.Context, code string, guestPhone *string) (*domain.BookingRecord, error) {
	return s.repo.GetBookingByCode(ctx, code, guestPhone)
}

// StartCleanupWorker periodically expires holds older than thresholdMinutes in the background
func (s *BookingService) StartCleanupWorker(ctx context.Context, interval time.Duration, thresholdMinutes int) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = s.repo.CleanupExpiredHolds(ctx, thresholdMinutes)
			}
		}
	}()
}
