package pricing

import (
	"math"
	"time"

	"quadis-backend-go/pkg/dateutil"
)

const (
	GSTLuxuryThresholdPerRoomNight = 7500.0
	GSTPercentStandard             = 5.0
	GSTPercentLuxury               = 18.0
)

type NightOverride struct {
	Rate             float64
	ExtraAdultCharge *float64
	ExtraChildCharge *float64
}

type StayPricingInput struct {
	BasePrice               float64
	RoomOffset              float64
	MealOffset              float64
	WeekendSurchargePercent float64
	CheckIn                 string
	CheckOut                string
	RoomsCount              int
	ExtraAdults             int
	ExtraChildren           int
	ExtraAdultPercent       *float64
	ChildPercent            *float64
	NightOverrides          map[string]NightOverride
}

type StayPricingBreakdown struct {
	Nights                   int     `json:"nights"`
	RoomTotal                float64 `json:"room_total"`
	ExtraAdultTotal          float64 `json:"extra_adult_total"`
	ExtraAdults              int     `json:"extra_adults"`
	ExtraChildTotal          float64 `json:"extra_child_total"`
	ExtraChildren            int     `json:"extra_children"`
	ExtraAdultPercent        float64 `json:"extra_adult_percent"`
	ChildPercent             float64 `json:"child_percent"`
	ExtraAdultChargePerNight float64 `json:"extra_adult_charge_per_night"`
	Total                    float64 `json:"total"`
}

func round(val float64) float64 {
	return math.Round(val*100) / 100
}

func ComputeStayBreakdown(input StayPricingInput) StayPricingBreakdown {
	nightly := input.BasePrice + input.RoomOffset + input.MealOffset
	surcharge := input.WeekendSurchargePercent

	extraAdults := input.ExtraAdults
	if extraAdults < 0 {
		extraAdults = 0
	}

	extraChildren := input.ExtraChildren
	if extraChildren < 0 {
		extraChildren = 0
	}

	rooms := input.RoomsCount
	if rooms <= 0 {
		rooms = 1
	}

	extraAdultPercent := DefaultExtraAdultPercent
	if input.ExtraAdultPercent != nil && *input.ExtraAdultPercent >= 0 {
		extraAdultPercent = *input.ExtraAdultPercent
	}

	childPercent := DefaultChildPercent
	if input.ChildPercent != nil && *input.ChildPercent >= 0 {
		childPercent = *input.ChildPercent
	}

	nightsList, err := dateutil.NightsBetween(input.CheckIn, input.CheckOut)
	if err != nil || len(nightsList) == 0 {
		// Minimum 1 night fallback
		upliftAdult := math.Round(nightly * (extraAdultPercent / 100.0))
		upliftChild := math.Round(nightly * (childPercent / 100.0))
		roomCharge := round(nightly * float64(rooms))
		extraCharge := round(upliftAdult * float64(extraAdults))
		childCharge := round(upliftChild * float64(extraChildren))

		return StayPricingBreakdown{
			Nights:                   1,
			RoomTotal:                roomCharge,
			ExtraAdultTotal:          extraCharge,
			ExtraAdults:              extraAdults,
			ExtraChildTotal:          childCharge,
			ExtraChildren:            extraChildren,
			ExtraAdultPercent:        extraAdultPercent,
			ChildPercent:             childPercent,
			ExtraAdultChargePerNight: upliftAdult,
			Total:                    round(roomCharge + extraCharge + childCharge),
		}
	}

	upliftPerAdult := func(rate float64) float64 {
		return math.Round(rate * (extraAdultPercent / 100.0))
	}
	upliftPerChild := func(rate float64) float64 {
		return math.Round(rate * (childPercent / 100.0))
	}

	charge := func(custom *float64, fallback float64) float64 {
		if custom != nil && *custom >= 0 {
			return *custom
		}
		return fallback
	}

	var roomTotal float64
	var extraAdultTotal float64
	var extraChildTotal float64
	nights := len(nightsList)

	for _, nightStr := range nightsList {
		nightTime, _ := time.Parse(dateutil.DateFormat, nightStr)
		override, hasOverride := input.NightOverrides[nightStr]

		var rate float64
		if hasOverride && override.Rate > 0 {
			rate = override.Rate
		} else if dateutil.IsWeekendNight(nightTime) {
			rate = nightly * (1.0 + surcharge/100.0)
		} else {
			rate = nightly
		}

		roomTotal += rate
		extraAdultTotal += charge(override.ExtraAdultCharge, upliftPerAdult(rate)) * float64(extraAdults)
		extraChildTotal += charge(override.ExtraChildCharge, upliftPerChild(rate)) * float64(extraChildren)
	}

	roomCharge := round(roomTotal * float64(rooms))
	extraCharge := round(extraAdultTotal)
	childCharge := round(extraChildTotal)

	var extraAdultChargePerNight float64
	if extraAdults > 0 && nights > 0 {
		extraAdultChargePerNight = round(extraCharge / float64(extraAdults*nights))
	}

	return StayPricingBreakdown{
		Nights:                   nights,
		RoomTotal:                roomCharge,
		ExtraAdultTotal:          extraCharge,
		ExtraAdults:              extraAdults,
		ExtraChildTotal:          childCharge,
		ExtraChildren:            extraChildren,
		ExtraAdultPercent:        extraAdultPercent,
		ChildPercent:             childPercent,
		ExtraAdultChargePerNight: extraAdultChargePerNight,
		Total:                    round(roomCharge + extraCharge + childCharge),
	}
}

func ComputeStayTotal(input StayPricingInput) float64 {
	return ComputeStayBreakdown(input).Total
}

// GSTRatePercentFor calculates the applicable GST slab percentage (5% or 18%)
// based on the value of supply net of GST.
func GSTRatePercentFor(inclusiveRatePerRoomNight float64) float64 {
	valueOfSupply := inclusiveRatePerRoomNight / (1.0 + GSTPercentStandard/100.0)
	if valueOfSupply > GSTLuxuryThresholdPerRoomNight {
		return GSTPercentLuxury
	}
	return GSTPercentStandard
}
