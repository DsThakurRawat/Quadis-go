package ota

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
	"quadis-backend-go/pkg/dateutil"
)

type POSCredentials struct {
	Username  string `json:"Username"`
	Password  string `json:"Password"`
	IDContext string `json:"ID_Context,omitempty"`
}

type OTAService struct {
	repo       repository.Repository
	cfg        *config.Config
	httpClient *http.Client
}

func NewOTAService(repo repository.Repository, cfg *config.Config) *OTAService {
	return &OTAService{
		repo:       repo,
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *OTAService) Authenticate(username, password, idContext string) bool {
	if s.cfg.ResAvenueUsername == "" || s.cfg.ResAvenuePassword == "" {
		return false // Service unavailable if credentials unset
	}

	userMatch := subtle.ConstantTimeCompare([]byte(username), []byte(s.cfg.ResAvenueUsername)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(password), []byte(s.cfg.ResAvenuePassword)) == 1

	if s.cfg.ResAvenueIDContext != "" {
		idMatch := subtle.ConstantTimeCompare([]byte(idContext), []byte(s.cfg.ResAvenueIDContext)) == 1
		return userMatch && passMatch && idMatch
	}

	return userMatch && passMatch
}

// GetPropertyDetails returns rooms and rate plans for a given hotel code
func (s *OTAService) GetPropertyDetails(ctx context.Context, hotelCode int) (map[string]interface{}, error) {
	propID := fmt.Sprintf("prop-%d", hotelCode)
	prop, err := s.repo.GetPropertyByID(ctx, propID)
	if err != nil {
		return nil, err
	}
	if prop == nil {
		return nil, fmt.Errorf("hotel code %d not found", hotelCode)
	}

	rooms, err := s.repo.GetRoomTypesByPropertyID(ctx, prop.ID)
	if err != nil {
		return nil, err
	}

	var roomDetails []map[string]interface{}
	for _, r := range rooms {
		roomCode := RoomOTACode(prop.ID, r.Slug)
		var ratePlans []map[string]interface{}
		for _, plan := range []domain.MealPlan{domain.MealPlanRoomOnly, domain.MealPlanWithBreakfast, domain.MealPlanAllMealsIncluded} {
			ratePlans = append(ratePlans, map[string]interface{}{
				"RatePlanCode": RatePlanOTACode(prop.ID, r.Slug, plan),
				"RatePlanName": string(plan),
			})
		}

		roomDetails = append(roomDetails, map[string]interface{}{
			"InvTypeCode": roomCode,
			"RoomName":    r.Name,
			"TotalUnits":  r.TotalUnits,
			"RatePlans":   ratePlans,
		})
	}

	return map[string]interface{}{
		"HotelCode": hotelCode,
		"HotelName": prop.Name,
		"Rooms":     roomDetails,
	}, nil
}

// UpdateInventory applies InvCountNotif updates
func (s *OTAService) UpdateInventory(ctx context.Context, hotelCode int, roomCode int, startDate, endDate string, invCount *int, stopSell, closeArrival, closeDeparture bool, cutOff int) (int, error) {
	propID, roomSlug, err := ParseRoomCode(roomCode)
	if err != nil {
		return 0, err
	}

	rooms, err := s.repo.GetRoomTypesByPropertyID(ctx, propID)
	if err != nil {
		return 0, err
	}
	var targetRoom *domain.RoomTypeRecord
	for _, r := range rooms {
		if r.Slug == roomSlug {
			targetRoom = &r
			break
		}
	}
	if targetRoom == nil {
		return 0, fmt.Errorf("room code %d not found for property %s", roomCode, propID)
	}

	nights, err := dateutil.NightsBetween(startDate, endDate)
	if err != nil {
		return 0, err
	}

	var records []domain.InventoryDayRecord
	for _, n := range nights {
		records = append(records, domain.InventoryDayRecord{
			RoomTypeID:       targetRoom.ID,
			StayDate:         n,
			InvCount:         invCount,
			StopSell:         stopSell,
			CloseOnArrival:   closeArrival,
			CloseOnDeparture: closeDeparture,
			CutOff:           cutOff,
		})
	}

	return s.repo.UpsertInventoryDays(ctx, records)
}

// UpdateRates applies RateAmountNotif updates
func (s *OTAService) UpdateRates(ctx context.Context, ratePlanCode int, startDate, endDate string, single, double, triple, quad, extraAdult, extraChild *float64, minStay, maxStay *int, stopSell bool) (int, error) {
	propID, roomSlug, plan, err := ParseRatePlanCode(ratePlanCode)
	if err != nil {
		return 0, err
	}

	rooms, err := s.repo.GetRoomTypesByPropertyID(ctx, propID)
	if err != nil {
		return 0, err
	}
	var targetRoom *domain.RoomTypeRecord
	for _, r := range rooms {
		if r.Slug == roomSlug {
			targetRoom = &r
			break
		}
	}
	if targetRoom == nil {
		return 0, fmt.Errorf("rate plan code %d room not found for property %s", ratePlanCode, propID)
	}

	nights, err := dateutil.NightsBetween(startDate, endDate)
	if err != nil {
		return 0, err
	}

	var records []domain.RateDayRecord
	for _, n := range nights {
		records = append(records, domain.RateDayRecord{
			RoomTypeID: targetRoom.ID,
			MealPlan:   plan,
			StayDate:   n,
			Single:     single,
			Double:     double,
			Triple:     triple,
			Quad:       quad,
			ExtraAdult: extraAdult,
			ExtraChild: extraChild,
			MinStay:    minStay,
			MaxStay:    maxStay,
			StopSell:   stopSell,
		})
	}

	return s.repo.UpsertRateDays(ctx, records)
}

// PullBookings serves confirmed or cancelled bookings
func (s *OTAService) PullBookings(ctx context.Context, hotelCode int, fromDate, toDate string) ([]domain.BookingRecord, error) {
	propID := ""
	if hotelCode > 0 {
		propID = fmt.Sprintf("prop-%d", hotelCode)
	}
	return s.repo.GetBookingsForChannelPull(ctx, propID, fromDate, toDate, 500)
}

// PushBooking sends a reservation to ResAvenue's webhook URL
func (s *OTAService) PushBooking(ctx context.Context, booking *domain.BookingRecord) error {
	if s.cfg.ResAvenuePushURL == "" {
		return nil // Push not configured, channel manager will pull
	}

	payload := map[string]interface{}{
		"Target":      s.cfg.ResAvenueTarget,
		"TimeStamp":   time.Now().UTC().Format(time.RFC3339),
		"BookingCode": booking.BookingCode,
		"Status":      booking.CMResStatus,
		"CheckIn":     booking.CheckIn,
		"CheckOut":    booking.CheckOut,
		"GuestName":   booking.GuestName,
		"GuestPhone":  booking.GuestPhone,
		"TotalAmount": booking.TotalAmount,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.ResAvenuePushURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.ResAvenuePushAuth != "" {
		req.Header.Set("Authorization", s.cfg.ResAvenuePushAuth)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		errStr := err.Error()
		_ = s.repo.MarkChannelSync(ctx, booking.ID, domain.CMSyncStatusFailed, nil, &errStr)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_ = s.repo.MarkChannelSync(ctx, booking.ID, domain.CMSyncStatusSent, nil, nil)
		return nil
	}

	errStr := fmt.Sprintf("status code: %d", resp.StatusCode)
	_ = s.repo.MarkChannelSync(ctx, booking.ID, domain.CMSyncStatusFailed, nil, &errStr)
	return fmt.Errorf("push failed with HTTP %d", resp.StatusCode)
}

// FetchInventory fetches per-night sellable inventory and restrictions for rooms
func (s *OTAService) FetchInventory(ctx context.Context, hotelCode int, roomCodes []int, startDate, endDate string) (map[string]interface{}, error) {
	propID := fmt.Sprintf("prop-%d", hotelCode)
	prop, err := s.repo.GetPropertyByID(ctx, propID)
	if err != nil {
		return nil, err
	}
	if prop == nil {
		return nil, fmt.Errorf("hotel code %d not found", hotelCode)
	}

	allRooms, err := s.repo.GetRoomTypesByPropertyID(ctx, prop.ID)
	if err != nil {
		return nil, err
	}

	roomByCode := make(map[int]domain.RoomTypeRecord)
	for _, r := range allRooms {
		c := RoomOTACode(prop.ID, r.Slug)
		roomByCode[c] = r
	}

	var targetRooms []domain.RoomTypeRecord
	var roomIDs []string
	for _, code := range roomCodes {
		r, ok := roomByCode[code]
		if !ok {
			return nil, fmt.Errorf("unknown InvCode %d for hotel %d", code, hotelCode)
		}
		targetRooms = append(targetRooms, r)
		roomIDs = append(roomIDs, r.ID)
	}

	nights, err := dateutil.NightsBetween(startDate, endDate)
	if err != nil {
		return nil, err
	}

	invDays, err := s.repo.GetInventoryDays(ctx, roomIDs, nights)
	if err != nil {
		return nil, err
	}
	invMap := make(map[string]domain.InventoryDayRecord)
	for _, d := range invDays {
		invMap[fmt.Sprintf("%s|%s", d.RoomTypeID, d.StayDate)] = d
	}

	heldMap, err := s.repo.GetHeldUnitsByNight(ctx, roomIDs, nights)
	if err != nil {
		return nil, err
	}

	var inventories []map[string]interface{}
	for _, r := range targetRooms {
		code := RoomOTACode(prop.ID, r.Slug)
		var nightList []map[string]interface{}
		for _, n := range nights {
			rule, hasRule := invMap[fmt.Sprintf("%s|%s", r.ID, n)]
			cap := r.TotalUnits
			if hasRule && rule.InvCount != nil {
				cap = *rule.InvCount
			}
			held := heldMap[fmt.Sprintf("%s|%s", r.ID, n)]
			free := cap - held
			if free < 0 {
				free = 0
			}

			stopSell := !r.IsAvailable
			closeArr := false
			closeDep := false
			cutOff := 0
			if hasRule {
				if rule.StopSell {
					stopSell = true
				}
				closeArr = rule.CloseOnArrival
				closeDep = rule.CloseOnDeparture
				cutOff = rule.CutOff
			}

			nightList = append(nightList, map[string]interface{}{
				"Date":             n,
				"InvCount":         free,
				"StopSell":         stopSell,
				"CloseOnArrival":   closeArr,
				"CloseOnDeparture": closeDep,
				"CutOff":           cutOff,
			})
		}
		inventories = append(inventories, map[string]interface{}{
			"InvCode":   code,
			"Inventory": nightList,
		})
	}

	return map[string]interface{}{
		"HotelName":   prop.Name,
		"HotelCode":   fmt.Sprintf("%d", hotelCode),
		"Inventories": inventories,
	}, nil
}

// FetchRates fetches per-night rates for rate plans
func (s *OTAService) FetchRates(ctx context.Context, hotelCode int, rateCodes []int, startDate, endDate string) (map[string]interface{}, error) {
	propID := fmt.Sprintf("prop-%d", hotelCode)
	prop, err := s.repo.GetPropertyByID(ctx, propID)
	if err != nil {
		return nil, err
	}
	if prop == nil {
		return nil, fmt.Errorf("hotel code %d not found", hotelCode)
	}

	allRooms, err := s.repo.GetRoomTypesByPropertyID(ctx, prop.ID)
	if err != nil {
		return nil, err
	}
	roomBySlug := make(map[string]domain.RoomTypeRecord)
	for _, r := range allRooms {
		roomBySlug[r.Slug] = r
	}

	nights, err := dateutil.NightsBetween(startDate, endDate)
	if err != nil {
		return nil, err
	}

	type targetPlan struct {
		code int
		room domain.RoomTypeRecord
		plan domain.MealPlan
	}
	var targets []targetPlan
	var roomIDs []string
	var plans []domain.MealPlan

	for _, code := range rateCodes {
		pID, slug, plan, err := ParseRatePlanCode(code)
		if err != nil || pID != propID {
			return nil, fmt.Errorf("unknown RateCode %d for hotel %d", code, hotelCode)
		}
		room, ok := roomBySlug[slug]
		if !ok {
			return nil, fmt.Errorf("unknown room for rate code %d", code)
		}
		targets = append(targets, targetPlan{code: code, room: room, plan: plan})
		roomIDs = append(roomIDs, room.ID)
		plans = append(plans, plan)
	}

	rateDays, err := s.repo.GetRateDays(ctx, roomIDs, plans, nights)
	if err != nil {
		return nil, err
	}
	rateMap := make(map[string]domain.RateDayRecord)
	for _, rd := range rateDays {
		rateMap[fmt.Sprintf("%s|%s|%s", rd.RoomTypeID, string(rd.MealPlan), rd.StayDate)] = rd
	}

	invDays, err := s.repo.GetInventoryDays(ctx, roomIDs, nights)
	if err != nil {
		return nil, err
	}
	invMap := make(map[string]domain.InventoryDayRecord)
	for _, id := range invDays {
		invMap[fmt.Sprintf("%s|%s", id.RoomTypeID, id.StayDate)] = id
	}

	var ratesOut []map[string]interface{}
	for _, target := range targets {
		var rateList []map[string]interface{}
		basePrice := prop.BasePrice
		roomOffset := target.room.PriceOffset
		mealOffset := 0.0
		switch target.plan {
		case domain.MealPlanWithBreakfast:
			mealOffset = (basePrice + roomOffset) * 0.25
		case domain.MealPlanAllMealsIncluded:
			mealOffset = (basePrice + roomOffset) * 0.50
		}
		nightly := basePrice + roomOffset + mealOffset

		for _, n := range nights {
			nt, _ := time.Parse(dateutil.DateFormat, n)
			isWeekend := dateutil.IsWeekendNight(nt)
			calcRate := nightly
			if isWeekend {
				calcRate = nightly * (1.0 + prop.WeekendSurchargePercent/100.0)
			}

			rd, hasRd := rateMap[fmt.Sprintf("%s|%s|%s", target.room.ID, string(target.plan), n)]
			inv, hasInv := invMap[fmt.Sprintf("%s|%s", target.room.ID, n)]

			single := calcRate
			double := calcRate
			triple := calcRate * 1.30
			quad := calcRate * 1.60
			childPercent := 20.0
			if prop.ChildPercent != nil {
				childPercent = *prop.ChildPercent
			}
			extraAdult := calcRate * (prop.ExtraAdultPercent / 100.0)
			extraChild := calcRate * (childPercent / 100.0)
			minStay := 1
			maxStay := 30
			stopSell := !target.room.IsAvailable

			if hasRd {
				if rd.Single != nil {
					single = *rd.Single
				}
				if rd.Double != nil {
					double = *rd.Double
				}
				if rd.Triple != nil {
					triple = *rd.Triple
				}
				if rd.Quad != nil {
					quad = *rd.Quad
				}
				if rd.ExtraAdult != nil {
					extraAdult = *rd.ExtraAdult
				}
				if rd.ExtraChild != nil {
					extraChild = *rd.ExtraChild
				}
				if rd.MinStay != nil {
					minStay = *rd.MinStay
				}
				if rd.MaxStay != nil {
					maxStay = *rd.MaxStay
				}
				if rd.StopSell {
					stopSell = true
				}
			}
			if hasInv && inv.StopSell {
				stopSell = true
			}

			rateList = append(rateList, map[string]interface{}{
				"Date":       n,
				"Single":     math.Round(single),
				"Double":     math.Round(double),
				"Triple":     math.Round(triple),
				"Quad":       math.Round(quad),
				"ExtraPax":   math.Round(extraAdult),
				"ExtraChild": math.Round(extraChild),
				"MinStay":    minStay,
				"MaxStay":    maxStay,
				"StopSell":   stopSell,
			})
		}

		ratesOut = append(ratesOut, map[string]interface{}{
			"RateCode": target.code,
			"Rate":     rateList,
		})
	}

	return map[string]interface{}{
		"HotelName": prop.Name,
		"HotelCode": fmt.Sprintf("%d", hotelCode),
		"Rates":     ratesOut,
	}, nil
}

// StartChannelSyncWorker periodically retries pushing pending channel bookings
func (s *OTAService) StartChannelSyncWorker(ctx context.Context, interval time.Duration, maxAttempts int) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pending, err := s.repo.GetBookingsPendingChannelSync(ctx, maxAttempts, 20)
				if err != nil {
					continue
				}
				for _, b := range pending {
					_ = s.PushBooking(ctx, &b)
				}
			}
		}
	}()
}
