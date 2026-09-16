package ota

import (
	"fmt"
	"regexp"
	"strconv"

	"quadis-backend-go/internal/domain"
)

var propIDRegex = regexp.MustCompile(`^prop-(\d+)$`)

var roomCategoryIndex = map[string]int{
	"deluxe-room":   1,
	"super-deluxe":  2,
	"superior-room": 3,
	"royal-suite":   4,
}

var categoryByNumber = map[int]string{
	1: "deluxe-room",
	2: "super-deluxe",
	3: "superior-room",
	4: "royal-suite",
}

var ratePlanIndex = map[domain.MealPlan]int{
	domain.MealPlanRoomOnly:         1,
	domain.MealPlanWithBreakfast:    2,
	domain.MealPlanAllMealsIncluded: 3,
}

var planByNumber = map[int]domain.MealPlan{
	1: domain.MealPlanRoomOnly,
	2: domain.MealPlanWithBreakfast,
	3: domain.MealPlanAllMealsIncluded,
}

func PropertyOTACode(propertyID string) int {
	matches := propIDRegex.FindStringSubmatch(propertyID)
	if len(matches) == 2 {
		if n, err := strconv.Atoi(matches[1]); err == nil {
			return n
		}
	}
	return 999
}

func RoomOTACode(propertyID, roomSlug string) int {
	hotel := PropertyOTACode(propertyID)
	idx, exists := roomCategoryIndex[roomSlug]
	if !exists {
		idx = 9
	}
	return hotel*100 + idx
}

func RatePlanOTACode(propertyID, roomSlug string, plan domain.MealPlan) int {
	roomCode := RoomOTACode(propertyID, roomSlug)
	planIdx, exists := ratePlanIndex[plan]
	if !exists {
		planIdx = 1
	}
	return roomCode*10 + planIdx
}

func ParseRatePlanCode(code int) (propertyID string, roomSlug string, plan domain.MealPlan, err error) {
	planNum := code % 10
	roomCode := code / 10
	categoryNum := roomCode % 100
	hotelNum := roomCode / 100

	p, ok := planByNumber[planNum]
	if !ok {
		return "", "", "", fmt.Errorf("unknown rate plan number: %d", planNum)
	}

	c, ok := categoryByNumber[categoryNum]
	if !ok {
		return "", "", "", fmt.Errorf("unknown room category number: %d", categoryNum)
	}

	propID := fmt.Sprintf("prop-%d", hotelNum)
	return propID, c, p, nil
}

func ParseRoomCode(code int) (propertyID string, roomSlug string, err error) {
	categoryNum := code % 100
	hotelNum := code / 100

	c, ok := categoryByNumber[categoryNum]
	if !ok {
		return "", "", fmt.Errorf("unknown room category number: %d", categoryNum)
	}

	propID := fmt.Sprintf("prop-%d", hotelNum)
	return propID, c, nil
}
