package domain

import "time"

type City string

const (
	CityNoida    City = "Noida"
	CityNewDelhi City = "New Delhi"
)

type PropertyRecord struct {
	ID                      string    `json:"id" db:"id"`
	Slug                    string    `json:"slug" db:"slug"`
	Name                    string    `json:"name" db:"name"`
	City                    City      `json:"city" db:"city"`
	Address                 string    `json:"address" db:"address"`
	MapLink                 *string   `json:"map_link,omitempty" db:"map_link"`
	Phone                   string    `json:"phone" db:"phone"`
	WhatsApp                string    `json:"whatsapp" db:"whatsapp"`
	Email                   string    `json:"email" db:"email"`
	BasePrice               float64   `json:"base_price" db:"base_price"`
	Rating                  float64   `json:"rating" db:"rating"`
	IsActive                bool      `json:"is_active" db:"is_active"`
	WeekendSurchargePercent float64   `json:"weekend_surcharge_percent" db:"weekend_surcharge_percent"`
	ExtraAdultPercent       float64   `json:"extra_adult_percent" db:"extra_adult_percent"`
	ChildFreeUnderAge       int       `json:"child_free_under_age" db:"child_free_under_age"`
	ChildPercent            *float64  `json:"child_percent,omitempty" db:"child_percent"`
	AdultFromAge            *int      `json:"adult_from_age,omitempty" db:"adult_from_age"`
	Lat                     *float64  `json:"lat,omitempty" db:"lat"`
	Lng                     *float64         `json:"lng,omitempty" db:"lng"`
	PlaceID                 *string          `json:"place_id,omitempty" db:"place_id"`
	Tier                    *string          `json:"tier,omitempty" db:"tier"`
	TierLabel               *string          `json:"tier_label,omitempty" db:"tier_label"`
	Rooms                   []RoomTypeRecord `json:"rooms,omitempty" db:"-"`
}

type PropertyImageRecord struct {
	ID          string     `json:"id" db:"id"`
	PropertyID  string     `json:"property_id" db:"property_id"`
	URL         string     `json:"url" db:"url"`
	ThumbURL    *string    `json:"thumb_url,omitempty" db:"thumb_url"`
	StorageKey  string     `json:"storage_key" db:"storage_key"`
	AltText     *string    `json:"alt_text,omitempty" db:"alt_text"`
	SortOrder   int        `json:"sort_order" db:"sort_order"`
	CreatedAt   *time.Time `json:"created_at,omitempty" db:"created_at"`
}
