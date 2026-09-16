package domain

import "time"

// InventoryDayRecord stores per-night inventory and restrictions pushed by ResAvenue
type InventoryDayRecord struct {
	RoomTypeID       string    `json:"room_type_id" db:"room_type_id"`
	StayDate         string    `json:"stay_date" db:"stay_date"`
	InvCount         *int      `json:"inv_count" db:"inv_count"`
	StopSell         bool      `json:"stop_sell" db:"stop_sell"`
	CloseOnArrival   bool      `json:"close_on_arrival" db:"close_on_arrival"`
	CloseOnDeparture bool      `json:"close_on_departure" db:"close_on_departure"`
	CutOff           int       `json:"cut_off" db:"cut_off"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// RateDayRecord stores per-night pricing and length-of-stay restrictions pushed by ResAvenue
type RateDayRecord struct {
	RoomTypeID string    `json:"room_type_id" db:"room_type_id"`
	MealPlan   MealPlan  `json:"meal_plan" db:"meal_plan"`
	StayDate   string    `json:"stay_date" db:"stay_date"`
	Single     *float64  `json:"single" db:"single"`
	Double     *float64  `json:"double" db:"double"`
	Triple     *float64  `json:"triple" db:"triple"`
	Quad       *float64  `json:"quad" db:"quad"`
	ExtraAdult *float64  `json:"extra_adult" db:"extra_adult"`
	ExtraChild *float64  `json:"extra_child" db:"extra_child"`
	MinStay    *int      `json:"min_stay" db:"min_stay"`
	MaxStay    *int      `json:"max_stay" db:"max_stay"`
	StopSell   bool      `json:"stop_sell" db:"stop_sell"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}
