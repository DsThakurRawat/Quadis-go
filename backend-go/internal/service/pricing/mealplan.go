package pricing

import (
	"math"
	"quadis-backend-go/internal/domain"
)

func MealPlanUpliftPercent(plan domain.MealPlan) float64 {
	switch plan {
	case domain.MealPlanWithBreakfast:
		return 25.0
	case domain.MealPlanAllMealsIncluded:
		return 50.0
	default:
		return 0.0
	}
}

func BaseRoomRateFor(basePrice, roomOffset float64) float64 {
	return basePrice + roomOffset
}

func MealUpliftFor(plan domain.MealPlan, baseRoomRate float64) float64 {
	percent := MealPlanUpliftPercent(plan)
	if baseRoomRate <= 0 || percent <= 0 {
		return 0
	}
	return math.Round(baseRoomRate * (percent / 100.0))
}

func MealOffsetFor(
	plan domain.MealPlan,
	basePrice, roomOffset float64,
	fallbackBreakfast, fallbackAllMeals float64,
) float64 {
	baseRate := BaseRoomRateFor(basePrice, roomOffset)
	if baseRate > 0 {
		return MealUpliftFor(plan, baseRate)
	}

	if plan == domain.MealPlanWithBreakfast {
		return fallbackBreakfast
	}
	if plan == domain.MealPlanAllMealsIncluded {
		return fallbackAllMeals
	}
	return 0
}
