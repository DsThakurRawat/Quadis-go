package domain

type MealPlan string

const (
	MealPlanRoomOnly         MealPlan = "Room Only"
	MealPlanWithBreakfast    MealPlan = "With Breakfast"
	MealPlanAllMealsIncluded MealPlan = "All Meals Included"
)

type RoomTypeRecord struct {
	ID              string  `json:"id" db:"id"`
	PropertyID      string  `json:"property_id" db:"property_id"`
	Slug            string  `json:"slug" db:"slug"`
	Name            string  `json:"name" db:"name"`
	Description     string  `json:"description" db:"description"`
	SizeSqft        string  `json:"size_sqft" db:"size_sqft"`
	BedType         string  `json:"bed_type" db:"bed_type"`
	MaxGuests       int     `json:"max_guests" db:"max_guests"`
	PriceOffset     float64 `json:"price_offset" db:"price_offset"`
	BreakfastOffset float64 `json:"breakfast_offset" db:"breakfast_offset"`
	AllMealsOffset  float64 `json:"all_meals_offset" db:"all_meals_offset"`
	TotalUnits      int     `json:"total_units" db:"total_units"`
	AvailableUnits  int     `json:"available_units" db:"available_units"`
	IsAvailable     bool    `json:"is_available" db:"is_available"`
}
