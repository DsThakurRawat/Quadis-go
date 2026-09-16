package pricing

import (
	"fmt"
	"math"
	"testing"

	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
)

func TestGST_ThresholdAndRates(t *testing.T) {
	// Standard rate checks
	if rate := GSTRatePercentFor(2000); rate != 5 {
		t.Errorf("expected 5%% GST for 2000, got %f", rate)
	}
	if rate := GSTRatePercentFor(7499.99); rate != 5 {
		t.Errorf("expected 5%% GST for 7499.99, got %f", rate)
	}

	// 7875 inclusive = exactly 7500 net of 5% tax
	if rate := GSTRatePercentFor(7875); rate != 5 {
		t.Errorf("expected 5%% GST for 7875, got %f", rate)
	}
	if rate := GSTRatePercentFor(7875.01); rate != 18 {
		t.Errorf("expected 18%% GST for 7875.01, got %f", rate)
	}
	if rate := GSTRatePercentFor(20000); rate != 18 {
		t.Errorf("expected 18%% GST for 20000, got %f", rate)
	}

	// Royal Suite at 7500 inclusive must remain at 5%
	if rate := GSTRatePercentFor(7500); rate != 5 {
		t.Errorf("expected 5%% GST for 7500, got %f", rate)
	}
}

func TestGST_AmaltasSuperiorIsOnlyLuxury(t *testing.T) {
	plans := []domain.MealPlan{
		domain.MealPlanRoomOnly,
		domain.MealPlanWithBreakfast,
		domain.MealPlanAllMealsIncluded,
	}

	var luxuryList []string

	rooms := repository.BuildSeedRoomTypes()
	for _, room := range rooms {
		var prop *domain.PropertyRecord
		for _, p := range repository.SeedProperties {
			if p.ID == room.PropertyID {
				prop = &p
				break
			}
		}
		if prop == nil {
			continue
		}

		base := BaseRoomRateFor(prop.BasePrice, room.PriceOffset)
		for _, plan := range plans {
			inclusive := base + MealUpliftFor(plan, base)
			if GSTRatePercentFor(inclusive) == GSTPercentLuxury {
				luxuryList = append(luxuryList, fmt.Sprintf("%s/%s/%s", prop.Slug, room.Slug, plan))
			}
		}
	}

	expected := []string{"hotel-amaltas-international/superior-room/All Meals Included"}
	if len(luxuryList) != len(expected) {
		t.Fatalf("expected exactly %v, got %v", expected, luxuryList)
	}
	for i := range expected {
		if luxuryList[i] != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], luxuryList[i])
		}
	}

	// Amaltas Superior on Room Only (6000) and Breakfast (7500) must be 5%
	if rate := GSTRatePercentFor(6000); rate != 5 {
		t.Errorf("expected 5%% for 6000, got %f", rate)
	}
	if rate := GSTRatePercentFor(7500); rate != 5 {
		t.Errorf("expected 5%% for 7500, got %f", rate)
	}
	if rate := GSTRatePercentFor(9000); rate != 18 {
		t.Errorf("expected 18%% for 9000, got %f", rate)
	}
}

func TestMealPlans_Uplifts(t *testing.T) {
	if MealPlanUpliftPercent(domain.MealPlanRoomOnly) != 0 {
		t.Errorf("expected 0 for Room Only")
	}
	if MealPlanUpliftPercent(domain.MealPlanWithBreakfast) != 25 {
		t.Errorf("expected 25 for With Breakfast")
	}
	if MealPlanUpliftPercent(domain.MealPlanAllMealsIncluded) != 50 {
		t.Errorf("expected 50 for All Meals Included")
	}

	// Downtown EOK: 3000 base + 1000 super deluxe = 4000
	base := BaseRoomRateFor(3000, 1000)
	if base != 4000 {
		t.Errorf("expected base 4000, got %f", base)
	}
	if uplift := MealUpliftFor(domain.MealPlanWithBreakfast, base); uplift != 1000 {
		t.Errorf("expected 1000 for CP, got %f", uplift)
	}
	if uplift := MealUpliftFor(domain.MealPlanAllMealsIncluded, base); uplift != 2000 {
		t.Errorf("expected 2000 for MAP, got %f", uplift)
	}
}

func TestStayPricing_Composition(t *testing.T) {
	base := 4000.0
	stay := StayPricingInput{
		BasePrice:               3000,
		RoomOffset:              1000,
		WeekendSurchargePercent: 20,
		CheckIn:                 "2026-09-07", // Mon
		CheckOut:                "2026-09-08", // Tue
		RoomsCount:              1,
	}

	// 1. All Meals Included on weekday: 4000 * 1.5 = 6000
	stay1 := stay
	stay1.MealOffset = MealUpliftFor(domain.MealPlanAllMealsIncluded, base)
	b1 := ComputeStayBreakdown(stay1)
	if b1.Total != 6000 {
		t.Errorf("expected total 6000, got %f", b1.Total)
	}

	// 2. Weekend surcharge (Friday) runs on meal-inclusive rate: 6000 * 1.2 = 7200
	stay2 := stay1
	stay2.CheckIn = "2026-09-11"  // Fri
	stay2.CheckOut = "2026-09-12" // Sat
	b2 := ComputeStayBreakdown(stay2)
	if b2.Total != 7200 {
		t.Errorf("expected total 7200 for Friday stay, got %f", b2.Total)
	}

	// 3. Extra adult charged on meal-inclusive rate: 6000 + (30% of 6000 = 1800) = 7800
	stay3 := stay1
	stay3.ExtraAdults = 1
	pAdult := 30.0
	stay3.ExtraAdultPercent = &pAdult
	b3 := ComputeStayBreakdown(stay3)
	if b3.RoomTotal != 6000 {
		t.Errorf("expected room total 6000, got %f", b3.RoomTotal)
	}
	if b3.ExtraAdultTotal != 1800 {
		t.Errorf("expected extra adult total 1800, got %f", b3.ExtraAdultTotal)
	}
	if b3.Total != 7800 {
		t.Errorf("expected total 7800, got %f", b3.Total)
	}

	// 4. Multiple rooms: 3 rooms with breakfast (5000 * 3 = 15000), 1 extra adult = 1500
	stay4 := stay
	stay4.RoomsCount = 3
	stay4.MealOffset = MealUpliftFor(domain.MealPlanWithBreakfast, base) // 1000 -> nightly = 5000
	stay4.ExtraAdults = 1
	stay4.ExtraAdultPercent = &pAdult
	b4 := ComputeStayBreakdown(stay4)
	if b4.RoomTotal != 15000 {
		t.Errorf("expected room total 15000, got %f", b4.RoomTotal)
	}
	if b4.ExtraAdultTotal != 1500 {
		t.Errorf("expected extra adult total 1500, got %f", b4.ExtraAdultTotal)
	}
	if b4.Total != 16500 {
		t.Errorf("expected total 16500, got %f", b4.Total)
	}
}

func TestStayPricing_NightOverrides(t *testing.T) {
	stay := StayPricingInput{
		BasePrice:               2500,
		RoomOffset:              0,
		MealOffset:              0,
		WeekendSurchargePercent: 0,
		CheckIn:                 "2026-10-01",
		CheckOut:                "2026-10-03", // 2 nights
		RoomsCount:              1,
		ExtraAdults:             1,
		NightOverrides: map[string]NightOverride{
			"2026-10-01": {
				Rate: 3500, // pushed rate
			},
		},
	}

	b := ComputeStayBreakdown(stay)
	// Night 1: 3500 + 30% extra adult (1050) = 4550
	// Night 2: 2500 + 30% extra adult (750) = 3250
	// Total room: 6000, Total extra adult: 1800 -> Total: 7800
	if b.RoomTotal != 6000 {
		t.Errorf("expected room total 6000, got %f", b.RoomTotal)
	}
	if b.ExtraAdultTotal != 1800 {
		t.Errorf("expected extra adult total 1800, got %f", b.ExtraAdultTotal)
	}
	if b.Total != 7800 {
		t.Errorf("expected total 7800, got %f", b.Total)
	}
}

func TestOccupancy_ChargeableGuests(t *testing.T) {
	tests := []struct {
		name          string
		adults        int
		childAges     []int
		roomsCount    int
		wantExtraAd   int
		wantExtraCh   int
	}{
		{"2 adults 1 room", 2, nil, 1, 0, 0},
		{"3 adults 1 room", 3, nil, 1, 1, 0},
		{"2 adults + 1 toddler (age 5)", 2, []int{5}, 1, 0, 0},
		{"2 adults + 1 concession child (age 10)", 2, []int{10}, 1, 0, 1},
		{"2 adults + 1 teen (age 14)", 2, []int{14}, 1, 1, 0},
		{"3 adults in 2 rooms", 3, nil, 2, 0, 0},
		{"4 adults + 2 children (age 9, 11) in 2 rooms", 4, []int{9, 11}, 2, 0, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ChargeableGuestsFor(OccupancyInput{
				Adults:     tt.adults,
				ChildAges:  tt.childAges,
				RoomsCount: tt.roomsCount,
			})
			if res.ExtraAdults != tt.wantExtraAd {
				t.Errorf("extra adults: got %d, want %d", res.ExtraAdults, tt.wantExtraAd)
			}
			if res.ExtraChildren != tt.wantExtraCh {
				t.Errorf("extra children: got %d, want %d", res.ExtraChildren, tt.wantExtraCh)
			}
		})
	}
}

func TestGST_ValueSplit(t *testing.T) {
	total := 6000.0
	rate := GSTRatePercentFor(total)
	base := math.Round((total/(1.0+rate/100.0))*100) / 100
	tax := math.Round((total-base)*100) / 100

	if rate != 5 {
		t.Errorf("expected 5%%, got %f", rate)
	}
	if math.Abs(base-5714.29) > 0.01 {
		t.Errorf("expected base ~5714.29, got %f", base)
	}
	if math.Abs(tax-285.71) > 0.01 {
		t.Errorf("expected tax ~285.71, got %f", tax)
	}
	if math.Round((base+tax)*100)/100 != total {
		t.Errorf("base + tax does not equal total")
	}
}
