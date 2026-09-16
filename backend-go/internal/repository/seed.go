package repository

import (
	"math"
	"quadis-backend-go/internal/domain"
)

func ptrString(s string) *string   { return &s }
func ptrFloat64(f float64) *float64 { return &f }
func ptrInt(i int) *int             { return &i }

var SeedProperties = []domain.PropertyRecord{
	{
		ID:                      "prop-2",
		Slug:                    "hotel-quadis-sector-51-noida",
		Lat:                     ptrFloat64(28.5833),
		Lng:                     ptrFloat64(77.3712),
		Name:                    "Hotel Quadis Sector 51",
		City:                    domain.CityNoida,
		Address:                 "H-22, Hoshiarpur Village, Sector 51, Noida, Uttar Pradesh 201301",
		MapLink:                 ptrString("https://share.google/X3cBuD2gbz27Jf5Ct"),
		Phone:                   "+91 92173 73532",
		WhatsApp:                "+91 92173 73532",
		Email:                   "stay@quadishotels.com",
		BasePrice:               1500,
		Rating:                  4.5,
		IsActive:                true,
		WeekendSurchargePercent: 0,
		ExtraAdultPercent:       30,
		ChildFreeUnderAge:       8,
		ChildPercent:            ptrFloat64(20),
		AdultFromAge:            ptrInt(13),
		Tier:                    ptrString("central"),
		TierLabel:               ptrString("Quadis Central"),
	},
	{
		ID:                      "prop-3",
		Slug:                    "hotel-quadis-central-sector-27-noida",
		Lat:                     ptrFloat64(28.5778),
		Lng:                     ptrFloat64(77.3243),
		Name:                    "Hotel Quadis Central",
		City:                    domain.CityNoida,
		Address:                 "D-192, E Block, Pocket E, Sector 27, Noida, Uttar Pradesh 201301",
		MapLink:                 ptrString("https://share.google/VGqI5StPFPeLyZIMO"),
		Phone:                   "+91 92173 73532",
		WhatsApp:                "+91 92173 73532",
		Email:                   "stay@quadishotels.com",
		BasePrice:               2500,
		Rating:                  4.5,
		IsActive:                true,
		WeekendSurchargePercent: 0,
		ExtraAdultPercent:       30,
		ChildFreeUnderAge:       8,
		ChildPercent:            ptrFloat64(20),
		AdultFromAge:            ptrInt(13),
		Tier:                    ptrString("central"),
		TierLabel:               ptrString("Quadis Central"),
	},
	{
		ID:                      "prop-4",
		Slug:                    "hotel-downtown-sector-15-noida",
		Lat:                     ptrFloat64(28.5847),
		Lng:                     ptrFloat64(77.3129),
		Name:                    "Hotel Downtown Sector 15 Noida",
		City:                    domain.CityNoida,
		Address:                 "Metro pillar no. 33, Opposite, New Ashok Nagar Rd, Naya Bans, Naya Bans Village, Sector 15, Noida, Uttar Pradesh 201301",
		MapLink:                 ptrString("https://share.google/oTnXw9glnDyZei1tL"),
		Phone:                   "+91 92173 73532",
		WhatsApp:                "+91 92173 73532",
		Email:                   "stay@quadishotels.com",
		BasePrice:               2000,
		Rating:                  4.0,
		IsActive:                true,
		WeekendSurchargePercent: 0,
		ExtraAdultPercent:       30,
		ChildFreeUnderAge:       8,
		ChildPercent:            ptrFloat64(20),
		AdultFromAge:            ptrInt(13),
		Tier:                    ptrString("central"),
		TierLabel:               ptrString("Quadis Central"),
	},
	{
		ID:                      "prop-5",
		Slug:                    "hotel-cladis-sector-15-noida",
		Lat:                     ptrFloat64(28.5855),
		Lng:                     ptrFloat64(77.311),
		Name:                    "Hotel Cladis Sector 15 Noida",
		City:                    domain.CityNoida,
		Address:                 "New Ashok Nagar Rd, opposite metro pillar no. 36, Naya Bans, Naya Bans Village, Sector 15, Noida, Uttar Pradesh 201301",
		MapLink:                 ptrString("https://share.google/1Gbjxirb5YQWy6h6D"),
		Phone:                   "+91 92173 73532",
		WhatsApp:                "+91 92173 73532",
		Email:                   "stay@quadishotels.com",
		BasePrice:               1800,
		Rating:                  3.8,
		IsActive:                true,
		WeekendSurchargePercent: 0,
		ExtraAdultPercent:       30,
		ChildFreeUnderAge:       8,
		ChildPercent:            ptrFloat64(20),
		AdultFromAge:            ptrInt(13),
		Tier:                    ptrString("central"),
		TierLabel:               ptrString("Quadis Central"),
	},
	{
		ID:                      "prop-6",
		Slug:                    "hotel-cladis-sector-19-noida",
		Lat:                     ptrFloat64(28.583),
		Lng:                     ptrFloat64(77.321),
		Name:                    "Hotel Cladis Sector 19 Noida",
		City:                    domain.CityNoida,
		Address:                 "A-369, A Block, Pocket A, Sector 19, Noida, Uttar Pradesh 201301",
		MapLink:                 ptrString("https://share.google/2YthY0ZjkrW3jnT3n"),
		Phone:                   "+91 92173 73532",
		WhatsApp:                "+91 92173 73532",
		Email:                   "stay@quadishotels.com",
		BasePrice:               2000,
		Rating:                  4.5,
		IsActive:                true,
		WeekendSurchargePercent: 0,
		ExtraAdultPercent:       30,
		ChildFreeUnderAge:       8,
		ChildPercent:            ptrFloat64(20),
		AdultFromAge:            ptrInt(13),
		Tier:                    ptrString("central"),
		TierLabel:               ptrString("Quadis Central"),
	},
	{
		ID:                      "prop-7",
		Slug:                    "hotel-downtown-sector-51-noida",
		Lat:                     ptrFloat64(28.5815),
		Lng:                     ptrFloat64(77.375),
		Name:                    "Hotel Downtown Sector 51 Noida",
		City:                    domain.CityNoida,
		Address:                 "House No : C-155, Sector 51, Noida, Uttar Pradesh 201304",
		MapLink:                 ptrString("https://share.google/Mwl1FiCVC8ucqXrd"),
		Phone:                   "+91 92173 73532",
		WhatsApp:                "+91 92173 73532",
		Email:                   "stay@quadishotels.com",
		BasePrice:               2500,
		Rating:                  4.4,
		IsActive:                true,
		WeekendSurchargePercent: 0,
		ExtraAdultPercent:       30,
		ChildFreeUnderAge:       8,
		ChildPercent:            ptrFloat64(20),
		AdultFromAge:            ptrInt(13),
		Tier:                    ptrString("central"),
		TierLabel:               ptrString("Quadis Central"),
	},
	{
		ID:                      "prop-8",
		Slug:                    "hotel-downtown-east-of-kailash",
		Lat:                     ptrFloat64(28.555),
		Lng:                     ptrFloat64(77.245),
		Name:                    "Hotel Downtown EOK",
		City:                    domain.CityNewDelhi,
		Address:                 "B-14, B Block, East of Kailash, New Delhi, Delhi 110065",
		MapLink:                 ptrString("https://share.google/3RsBzxkp8xV1e0AuY"),
		Phone:                   "+91 92173 73532",
		WhatsApp:                "+91 92173 73532",
		Email:                   "stay@quadishotels.com",
		BasePrice:               3000,
		Rating:                  4.5,
		IsActive:                true,
		WeekendSurchargePercent: 0,
		ExtraAdultPercent:       30,
		ChildFreeUnderAge:       8,
		ChildPercent:            ptrFloat64(20),
		AdultFromAge:            ptrInt(13),
		Tier:                    ptrString("central"),
		TierLabel:               ptrString("Quadis Central"),
	},
	{
		ID:                      "prop-9",
		Slug:                    "hotel-amby-inn-lajpat-nagar-ii",
		Lat:                     ptrFloat64(28.57),
		Lng:                     ptrFloat64(77.24),
		Name:                    "Hotel Amby Inn",
		City:                    domain.CityNewDelhi,
		Address:                 "M13, Vinoba Puri, Block M, Lajpat Nagar II, Lajpat Nagar, New Delhi, Delhi 110024",
		MapLink:                 ptrString("https://share.google/pSTT03I5OWszpSj5c"),
		Phone:                   "+91 92173 73532",
		WhatsApp:                "+91 92173 73532",
		Email:                   "stay@quadishotels.com",
		BasePrice:               2500,
		Rating:                  3.8,
		IsActive:                true,
		WeekendSurchargePercent: 0,
		ExtraAdultPercent:       30,
		ChildFreeUnderAge:       8,
		ChildPercent:            ptrFloat64(20),
		AdultFromAge:            ptrInt(13),
		Tier:                    ptrString("central"),
		TierLabel:               ptrString("Quadis Central"),
	},
	{
		ID:                      "prop-10",
		Slug:                    "hotel-amar-inn",
		Lat:                     ptrFloat64(28.571),
		Lng:                     ptrFloat64(77.2415),
		Name:                    "Hotel Amar Inn",
		City:                    domain.CityNewDelhi,
		Address:                 "K-102, Road, near Central Market, Block K, Lajpat Nagar II, Jal Vihar, New Delhi, Delhi 110024",
		MapLink:                 ptrString("https://share.google/IQLx35cfOmLf93S2o"),
		Phone:                   "+91 92173 73532",
		WhatsApp:                "+91 92173 73532",
		Email:                   "stay@quadishotels.com",
		BasePrice:               3000,
		Rating:                  4.3,
		IsActive:                true,
		WeekendSurchargePercent: 0,
		ExtraAdultPercent:       30,
		ChildFreeUnderAge:       8,
		ChildPercent:            ptrFloat64(20),
		AdultFromAge:            ptrInt(13),
		Tier:                    ptrString("central"),
		TierLabel:               ptrString("Quadis Central"),
	},
	{
		ID:                      "prop-11",
		Slug:                    "hotel-amaltas-international",
		Lat:                     ptrFloat64(28.5603),
		Lng:                     ptrFloat64(77.2022),
		Name:                    "Hotel Amaltas International",
		City:                    domain.CityNewDelhi,
		Address:                 "6, Opposite Sukhmani Hospital, Block W, Green Park Extension, Green Park, New Delhi, Delhi 110016",
		MapLink:                 ptrString("https://share.google/VkPBBiHgI3Vpgx3y0"),
		Phone:                   "+91 92173 73532",
		WhatsApp:                "+91 92173 73532",
		Email:                   "stay@quadishotels.com",
		BasePrice:               4000,
		Rating:                  4.1,
		IsActive:                true,
		WeekendSurchargePercent: 0,
		ExtraAdultPercent:       30,
		ChildFreeUnderAge:       8,
		ChildPercent:            ptrFloat64(20),
		AdultFromAge:            ptrInt(13),
		Tier:                    ptrString("central"),
		TierLabel:               ptrString("Quadis Central"),
	},
}

type roomPlan struct {
	deluxe   int
	super    int
	superior int
	royal    int
}

var keysBySlug = map[string]roomPlan{
	"hotel-downtown-sector-15-noida":      {deluxe: 24, super: 4},
	"hotel-downtown-sector-51-noida":      {deluxe: 6, super: 3},
	"hotel-cladis-sector-19-noida":        {deluxe: 10, super: 2},
	"hotel-quadis-sector-51-noida":        {deluxe: 25, super: 3},
	"hotel-cladis-sector-15-noida":        {deluxe: 24, super: 7},
	"hotel-quadis-central-sector-27-noida": {deluxe: 11, super: 6},
	"hotel-downtown-east-of-kailash":      {deluxe: 23, super: 6, royal: 1},
	"hotel-amby-inn-lajpat-nagar-ii":      {deluxe: 20, super: 3},
	"hotel-amar-inn":                      {deluxe: 12, super: 6, royal: 1},
	"hotel-amaltas-international":          {deluxe: 6, superior: 2},
}

var deluxeTemplate = domain.RoomTypeRecord{
	Slug:        "deluxe-room",
	Name:        "Deluxe Room",
	Description: "Calm, refined comfort designed for effortless rest. Features plush bedding and executive workspace.",
	SizeSqft:    "240 sq ft",
	BedType:     "King / Twin Beds",
	MaxGuests:   2,
	PriceOffset: 0,
	IsAvailable: true,
}

var superTemplate = domain.RoomTypeRecord{
	Slug:        "super-deluxe",
	Name:        "Super Deluxe",
	Description: "A larger, more considered room with an upgraded seating area, high-speed Wi-Fi and evening turndown.",
	SizeSqft:    "290 sq ft",
	BedType:     "King Bed",
	MaxGuests:   3,
	PriceOffset: 1000,
	IsAvailable: true,
}

var superiorTemplate = domain.RoomTypeRecord{
	Slug:        "superior-room",
	Name:        "Superior Room",
	Description: "An elevated, more generous room with a separate seating area, high-speed Wi-Fi and evening turndown.",
	SizeSqft:    "310 sq ft",
	BedType:     "King Bed",
	MaxGuests:   3,
	PriceOffset: 2000,
	IsAvailable: true,
}

var royalTemplate = domain.RoomTypeRecord{
	Slug:        "royal-suite",
	Name:        "Royal Suite",
	Description: "Our most luxurious sanctuary featuring separate master bedroom, private lounge and dining area.",
	SizeSqft:    "450 sq ft",
	BedType:     "Master Suite + Living Room",
	MaxGuests:   4,
	PriceOffset: 2000,
	IsAvailable: true,
}

func BuildSeedRoomTypes() []domain.RoomTypeRecord {
	var allRooms []domain.RoomTypeRecord

	for _, prop := range SeedProperties {
		plan, exists := keysBySlug[prop.Slug]
		if !exists {
			plan = roomPlan{deluxe: 5, super: 3}
		}

		makeRoom := func(tmpl domain.RoomTypeRecord, units int) domain.RoomTypeRecord {
			baseRate := prop.BasePrice + tmpl.PriceOffset
			r := tmpl
			r.ID = "room-" + prop.ID + "-" + tmpl.Slug
			r.PropertyID = prop.ID
			r.TotalUnits = units
			r.AvailableUnits = units
			r.BreakfastOffset = math.Round(baseRate * 0.25)
			r.AllMealsOffset = math.Round(baseRate * 0.50)
			return r
		}

		if plan.deluxe > 0 {
			allRooms = append(allRooms, makeRoom(deluxeTemplate, plan.deluxe))
		}
		if plan.super > 0 {
			allRooms = append(allRooms, makeRoom(superTemplate, plan.super))
		}
		if plan.superior > 0 {
			allRooms = append(allRooms, makeRoom(superiorTemplate, plan.superior))
		}
		if plan.royal > 0 {
			allRooms = append(allRooms, makeRoom(royalTemplate, plan.royal))
		}
	}

	return allRooms
}
