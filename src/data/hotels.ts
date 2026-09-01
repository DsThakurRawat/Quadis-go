import type { Hotel, BanquetVenue, City, HotelRoom, UpcomingHotel, CityFilter } from '../types.ts'
import { MEAL_PLAN_UPLIFT_PERCENT, baseRoomRateFor, mealUpliftFor } from '../lib/pricing.ts'

/**
 * The three meal plans, priced as a percentage of one room's base rate.
 *
 * Client, 5 Aug 2026 — EP no charge, CP +25%, MAP +50%, "applied automatically
 * across all hotels based on the base room rate". The percentages themselves
 * live in src/lib/pricing.ts next to the rest of the rate rules, so this file
 * holds no pricing literals of its own; it only decides what to apply them to.
 *
 * `baseRoomRate` is the property's own nightly price plus that category's
 * offset — see baseRoomRateFor(). Passing it in per hotel is what makes the
 * uplift track the property: Cladis 15's ₹1,800 Deluxe gets +₹450 of breakfast,
 * Amar Inn's ₹3,000 Deluxe gets +₹750, and both are the same 25%.
 *
 * The rupee figures these replace (₹300/₹800, ₹350/₹900, ₹450/₹1,200) are gone
 * from this file entirely rather than kept as a fallback — a stale flat number
 * behind a percentage is the bug the client reported.
 */
const mealOptionsFor = (baseRoomRate: number): HotelRoom['mealOptions'] =>
  (Object.keys(MEAL_PLAN_UPLIFT_PERCENT) as (keyof typeof MEAL_PLAN_UPLIFT_PERCENT)[]).map((plan) => ({
    plan,
    priceOffset: mealUpliftFor(plan, baseRoomRate),
  }))

/**
 * Category templates. `mealOptions` here is a placeholder priced off a ₹0 base,
 * so every entry is ₹0: the real figures cannot be known until a hotel is in
 * hand, and getHotelRooms() below recomputes them for the property being shown.
 * Nothing should read `mealOptions` off these constants directly.
 */
export const DEFAULT_ROOMS: HotelRoom[] = [
  {
    id: 'deluxe-room',
    name: 'Deluxe Room',
    description: 'Calm, refined comfort designed for effortless rest. Features plush bedding, executive workspace, and modern ensuite bath with premium bath amenities.',
    size: '240 sq ft',
    bed: 'King / Twin Beds',
    maxGuests: 2,
    basePriceOffset: 0,
    mealOptions: mealOptionsFor(0),
  },
  {
    id: 'superior-room',
    name: 'Superior Room with Balcony',
    description: 'Elevated space with private outdoor seating and expansive city views. Includes upgraded seating area, high-speed Wi-Fi, and personalized evening turndown.',
    size: '310 sq ft',
    bed: 'King Bed + Balcony',
    maxGuests: 3,
    basePriceOffset: 400,
    mealOptions: mealOptionsFor(0),
  },
  {
    id: 'royal-suite',
    name: 'Royal Suite',
    description: 'Our most luxurious sanctuary featuring separate master bedroom, private lounge and dining area, luxury soaking tub, and priority concierge check-in.',
    size: '450 sq ft',
    bed: 'Master Suite + Living Room',
    maxGuests: 4,
    basePriceOffset: 1200,
    mealOptions: mealOptionsFor(0),
  },
]

/**
 * The client's rate sheet (27 Jul 2026) settled the category question for the
 * original nine: three categories — DELUXE, SUPER and ROYAL — not the five this
 * file used to carry. All nine sell Deluxe and Super; only Downtown EOK and
 * Amar Inn sell a Royal, one key each.
 *
 * Pricing was uniform, again per the sheet: "Upper category 1000 plus in each
 * hotel". The per-hotel variation was entirely in `price` (the Deluxe rate) —
 * Super +1,000 on it and Royal +2,000 — so these are offsets, not rates, and the
 * sheet's Super and Royal columns fall out of them: Cladis 15 at 1,800 quotes
 * Super at 2,800, EOK at 3,000 quotes Royal at 5,000.
 *
 * Hotel Amaltas International (1 Sep 2026) is the first property that does not
 * follow that rule, which is why there is now a fourth template below. Read
 * "uniform" as "uniform across the nine on the sheet", not as a group-wide
 * invariant a new property can be assumed into: the tenth arrived with its own
 * rates, and the eleventh may too.
 *
 * Slugs are unchanged on purpose — `room_types.id` is derived from them and
 * `bookings.room_type_id` points at it, so renaming would strand live bookings.
 */
const SUPER: HotelRoom = {
  id: 'super-deluxe',
  name: 'Super Deluxe',
  description: 'A larger, more considered room with an upgraded seating area, high-speed Wi-Fi and evening turndown.',
  size: '290 sq ft',
  bed: 'King Bed',
  maxGuests: 3,
  basePriceOffset: 1000,
  mealOptions: mealOptionsFor(0),
}
const DELUXE: HotelRoom = { ...DEFAULT_ROOMS[0]! }
const ROYAL: HotelRoom = { ...DEFAULT_ROOMS[2]!, basePriceOffset: 2000 }

/**
 * Amaltas International's upper category, and the first property that does not
 * follow the group's "+1,000 for the upper category" rule.
 *
 * The client's own figures (new-property brief, 1 Sep 2026) are ₹4,000 for the
 * Deluxe / Deluxe Twin and ₹6,000 for the Superior — a ₹2,000 step, which is
 * the Royal offset, not the Super Deluxe one. Reusing SUPER would have quoted
 * ₹5,000 and undercut the rate sheet by a thousand rupees a night; reusing
 * ROYAL would have got the price right and the category name wrong, and the
 * slug is what `room_types.id` is derived from, so the name a guest books under
 * would have been "Royal Suite".
 *
 * So it is its own template. Note the offset is the one thing here that is NOT
 * shared with the group, which is why it is spelt out rather than derived.
 *
 * Size and bed are the DEFAULT_ROOMS 'superior-room' figures, i.e. the group's
 * representative spec — the client sent rates and photos, not a floor plan.
 * The balcony the default template claims is deliberately dropped from both the
 * name and the description: nothing in the brief or the photograph shows one.
 */
const SUPERIOR: HotelRoom = {
  id: 'superior-room',
  name: 'Superior Room',
  description: 'An elevated, more generous room with a separate seating area, high-speed Wi-Fi and evening turndown.',
  size: '310 sq ft',
  bed: 'King Bed',
  maxGuests: 3,
  basePriceOffset: 2000,
  mealOptions: mealOptionsFor(0),
}

/**
 * Every property is listed now, so the DEFAULT_ROOMS fallback below is only
 * reached by a slug this file has never heard of. Must stay in step with
 * ROOMS_BY_SLUG in backend/src/data/seed.ts, which additionally carries the
 * key counts — a category the site offers but the API has not seeded cannot be
 * booked.
 */
const ROOMS_BY_SLUG: Record<string, HotelRoom[]> = {
  'hotel-downtown-sector-15-noida': [DELUXE, SUPER],
  'hotel-downtown-sector-51-noida': [DELUXE, SUPER],
  'hotel-cladis-sector-19-noida': [DELUXE, SUPER],
  'hotel-quadis-sector-51-noida': [DELUXE, SUPER],
  'hotel-cladis-sector-15-noida': [DELUXE, SUPER],
  'hotel-quadis-central-sector-27-noida': [DELUXE, SUPER],
  'hotel-downtown-east-of-kailash': [DELUXE, SUPER, ROYAL],
  'hotel-amby-inn-lajpat-nagar-ii': [DELUXE, SUPER],
  'hotel-amar-inn': [DELUXE, SUPER, ROYAL],
  // Superior, not Super Deluxe — see the SUPERIOR template above.
  'hotel-amaltas-international': [DELUXE, SUPERIOR],
}

/**
 * The categories a property sells, with their meal plans priced for THAT
 * property.
 *
 * The meal supplement is re-derived here rather than read off the template,
 * because from 5 Aug 2026 it is a percentage of the base room rate and the
 * templates are shared across all nine hotels — a single stored rupee figure
 * cannot be right for a ₹1,500 Deluxe and a ₹3,000 one at the same time.
 *
 * It is recomputed for `hotel.rooms` too, not just for the static templates:
 * those arrive from the API, whose room rows still carry the legacy flat
 * breakfast_offset / all_meals_offset columns. Overriding them here is what
 * guarantees the price on the page matches what the server will charge, since
 * the server derives the same percentage from the same base rate.
 */
export const getHotelRooms = (hotel: Hotel): HotelRoom[] =>
  (hotel.rooms ?? ROOMS_BY_SLUG[hotel.slug] ?? DEFAULT_ROOMS).map((room) => ({
    ...room,
    mealOptions: mealOptionsFor(baseRoomRateFor(hotel.price, room.basePriceOffset)),
  }))

import { getApiUrl } from '../config/api'

import { useState, useEffect } from 'react'

export const STATIC_HOTELS: Hotel[] = [
  { slug: 'hotel-quadis-sector-51-noida', coords: { lat: 28.5833, lng: 77.3712 }, transit: { metro: { name: 'Sector 52 Metro', value: '5 min walk' }, airport: { name: 'IGI Airport T3', value: '32 km · 55 min' }, rail: { name: 'New Delhi Railway Station', value: '24 km' }, landmark: { name: 'Sector 51 Market', note: 'dining & retail' } }, name: 'Hotel Quadis Sector 51', area: 'Sector 51', city: 'Noida', address: 'H-22, Hoshiarpur Village, Sector 51, Noida, Uttar Pradesh 201301', price: 1500, rating: 4.5 },
  { slug: 'hotel-quadis-central-sector-27-noida', coords: { lat: 28.5778, lng: 77.3243 }, transit: { metro: { name: 'Sector 18 Metro', value: '10 min walk' }, airport: { name: 'IGI Airport T3', value: '28 km · 45 min' }, rail: { name: 'Nizamuddin Railway Station', value: '15 km' }, landmark: { name: 'Atta Market', note: 'dining & retail' } }, name: 'Hotel Quadis Central', area: 'Sector 27', city: 'Noida', address: 'D-192, E Block, Pocket E, Sector 27, Noida, Uttar Pradesh 201301', price: 2500, rating: 4.5 },
  { slug: 'hotel-downtown-sector-15-noida', coords: { lat: 28.5847, lng: 77.3129 }, transit: { metro: { name: 'Sector 15 Metro', value: '2 min walk' }, airport: { name: 'IGI Airport T3', value: '26 km · 40 min' }, rail: { name: 'Nizamuddin Railway Station', value: '13 km' }, landmark: { name: 'Sector 15 Indian Oil', note: 'Metro Pillar 33' } }, name: 'Hotel Downtown Sector 15 Noida', area: 'Sector 15', city: 'Noida', address: 'Metro pillar no. 33, Opposite, New Ashok Nagar Rd, Naya Bans, Naya Bans Village, Sector 15, Noida, Uttar Pradesh 201301', price: 2000, rating: 4.0 },
  { slug: 'hotel-cladis-sector-15-noida', coords: { lat: 28.5855, lng: 77.311 }, transit: { metro: { name: 'Sector 15 Metro', value: '4 min walk' }, airport: { name: 'IGI Airport T3', value: '26 km · 40 min' }, rail: { name: 'Nizamuddin Railway Station', value: '13 km' }, landmark: { name: 'Naya Bans Village', note: 'neighbourhood' } }, name: 'Hotel Cladis Sector 15 Noida', area: 'Sector 15', city: 'Noida', address: 'New Ashok Nagar Rd, opposite metro pillar no. 36, Naya Bans, Naya Bans Village, Sector 15, Noida, Uttar Pradesh 201301', price: 1800, rating: 3.8 },
  { slug: 'hotel-cladis-sector-19-noida', coords: { lat: 28.583, lng: 77.321 }, transit: { metro: { name: 'Sector 16 Metro', value: '8 min walk' }, airport: { name: 'IGI Airport T3', value: '27 km · 45 min' }, rail: { name: 'Nizamuddin Railway Station', value: '14 km' }, landmark: { name: 'Indo Gulf Hospital', note: 'landmark' } }, name: 'Hotel Cladis Sector 19 Noida', area: 'Sector 19', city: 'Noida', address: 'A-369, A Block, Pocket A, Sector 19, Noida, Uttar Pradesh 201301', price: 2000, rating: 4.5 },
  { slug: 'hotel-downtown-sector-51-noida', coords: { lat: 28.5815, lng: 77.375 }, transit: { metro: { name: 'Sector 52 Metro', value: '10 min walk' }, airport: { name: 'IGI Airport T3', value: '33 km · 55 min' }, rail: { name: 'New Delhi Railway Station', value: '25 km' }, landmark: { name: 'Kendriya Vihar', note: 'neighbourhood' } }, name: 'Hotel Downtown Sector 51 Noida', area: 'Sector 51', city: 'Noida', address: 'House No : C-155, Sector 51, Noida, Uttar Pradesh 201304', price: 2500, rating: 4.4 },
  { slug: 'hotel-downtown-east-of-kailash', coords: { lat: 28.555, lng: 77.245 }, transit: { metro: { name: 'Kailash Colony Metro', value: '5 min walk' }, airport: { name: 'IGI Airport T3', value: '18 km · 35 min' }, rail: { name: 'Nizamuddin Railway Station', value: '4 km' }, landmark: { name: 'ISKCON Temple', note: 'landmark' } }, name: 'Hotel Downtown EOK', area: 'East of Kailash', city: 'New Delhi', address: 'B-14, B Block, East of Kailash, New Delhi, Delhi 110065', price: 3000, rating: 4.5 },
  { slug: 'hotel-amby-inn-lajpat-nagar-ii', coords: { lat: 28.57, lng: 77.24 }, transit: { metro: { name: 'Lajpat Nagar Metro', value: '3 min walk' }, airport: { name: 'IGI Airport T3', value: '19 km · 35 min' }, rail: { name: 'Nizamuddin Railway Station', value: '5 km' }, landmark: { name: 'Central Market', note: 'dining & retail' } }, name: 'Hotel Amby Inn', area: 'Lajpat Nagar', city: 'New Delhi', address: 'M13, Vinoba Puri, Block M, Lajpat Nagar II, Lajpat Nagar, New Delhi, Delhi 110024', price: 2500, rating: 3.8 },
  { slug: 'hotel-amar-inn', coords: { lat: 28.571, lng: 77.2415 }, transit: { metro: { name: 'Lajpat Nagar Metro', value: '4 min walk' }, airport: { name: 'IGI Airport T3', value: '19 km · 35 min' }, rail: { name: 'Nizamuddin Railway Station', value: '5 km' }, landmark: { name: 'Jal Vihar', note: 'neighbourhood' } }, name: 'Hotel Amar Inn', area: 'Lajpat Nagar', city: 'New Delhi', address: 'K-102, Road, near Central Market, Block K, Lajpat Nagar II, Jal Vihar, New Delhi, Delhi 110024', price: 3000, rating: 4.3 },
  /*
   * Added 1 Sep 2026 from the client's new-property brief. Three fields on this
   * record are NOT from her, and each is deliberately conservative rather than
   * invented to look complete:
   *
   *  - `coords` are locality-level for Green Park Extension, not the building.
   *    Her GMB share link (share.google/VkPBBiHgI3Vpgx3y0) resolves only to a
   *    knowledge-graph id, /g/11h347q0bq, with no coordinates exposed, so the
   *    pin is good to a few hundred metres and no better. It is accurate enough
   *    for NcrLocatorMap, which is schematic and relative; replace it with the
   *    real pin before anyone relies on it for directions.
   *  - `transit` carries the two facts that ARE known — Green Park is the metro
   *    that serves this locality, and "opposite Sukhmani Hospital" is her own
   *    address line. Walk times, the airport and the railway station are
   *    omitted rather than estimated: per HotelTransit, a fact we have not
   *    verified is simply not shown, and a wrong walk time is worse than a
   *    missing one.
   *  - `rating` was NOT in the brief and the GMB share link would not serve it
   *    (it resolves only to knowledge-graph id /g/11h347q0bq, and Google blocks
   *    automated reads of the panel). 4.1 is the figure the client gave us
   *    directly on 1 Sep 2026, so it is her number like every other rating in
   *    this array — not a placeholder. Worth re-checking against the GMB
   *    listing at the next audit, as the 5 Aug one moved eight of nine.
   */
  { slug: 'hotel-amaltas-international', coords: { lat: 28.5603, lng: 77.2022 }, transit: { metro: { name: 'Green Park Metro', note: 'Yellow Line' }, landmark: { name: 'Sukhmani Hospital', note: 'opposite' } }, name: 'Hotel Amaltas International', area: 'Green Park Extension', city: 'New Delhi', address: '6, Opposite Sukhmani Hospital, Block W, Green Park Extension, Green Park, New Delhi, Delhi 110016', price: 4000, rating: 4.1 },
]

/**
 * Card order requested by the client (Change Order #2, item 4). Applied to both
 * the static list and the API response — the API returns seed order, so sorting
 * only one of them makes the grid re-shuffle the moment the fetch lands.
 */
export const HOTEL_DISPLAY_ORDER: readonly string[] = [
  'hotel-amar-inn',                        // Hotel Amar Inn
  'hotel-downtown-east-of-kailash',       // Hotel Downtown EOK
  'hotel-downtown-sector-51-noida',       // Hotel Downtown Sec 51
  'hotel-downtown-sector-15-noida',       // Hotel Downtown Sec 15
  'hotel-quadis-central-sector-27-noida', // Hotel Quadis Central
  'hotel-cladis-sector-19-noida',         // Hotel Cladis Sector 19
  'hotel-cladis-sector-15-noida',         // Hotel Cladis Sector 15
  'hotel-quadis-sector-51-noida',         // Hotel Quadis 51
  'hotel-amby-inn-lajpat-nagar-ii',       // Hotel Amby Inn
  // Added 1 Sep 2026. The client ranked the nine above and has not said where
  // the tenth belongs, so it is pinned last rather than slotted into her order
  // on a guess. Listed explicitly all the same: an unranked slug sorts to the
  // end anyway, but only by falling through orderOf()'s MAX_SAFE_INTEGER, and
  // an eleventh property would then share that rank and order arbitrarily.
  'hotel-amaltas-international',          // Hotel Amaltas International
]

const orderOf = (slug: string): number => {
  const i = HOTEL_DISPLAY_ORDER.indexOf(slug)
  // Anything the client has not ranked sorts to the end rather than the front.
  return i === -1 ? Number.MAX_SAFE_INTEGER : i
}

export const inDisplayOrder = (list: Hotel[]): Hotel[] =>
  [...list].sort((a, b) => orderOf(a.slug) - orderOf(b.slug))

export const HOTELS: Hotel[] = inDisplayOrder(STATIC_HOTELS)

let cachedHotels: Hotel[] | null = null
let fetchPromise: Promise<void> | null = null
const listeners = new Set<() => void>()

export function useHotels(): Hotel[] {
  const [hotels, setHotels] = useState<Hotel[]>(cachedHotels || HOTELS)

  useEffect(() => {
    if (cachedHotels) return
    if (!fetchPromise) {
      fetchPromise = fetch(getApiUrl('properties'))
        .then(res => res.json())
        .then(json => {
          if (json.success && Array.isArray(json.data)) {
            // Merge over the static record rather than replacing it. The API is
            // authoritative for operational data (price, surcharge), but
            // editorial content — transit facts, hand-written area labels — only
            // lives here. Replacing outright silently blanked the "Getting here"
            // panel the moment the API responded.
            const staticBySlug = new Map(STATIC_HOTELS.map((s) => [s.slug, s]))

            cachedHotels = inDisplayOrder(json.data.map((h: any): Hotel => {
              const base = staticBySlug.get(h.slug)
              const derivedArea = h.name.includes('Sector')
                ? `Sector ${h.name.split('Sector ')[1]}`
                : h.name.split(' ').slice(-2).join(' ')

              return {
                ...base,
                slug: h.slug,
                name: h.name,
                area: base?.area ?? derivedArea,
                city: h.city as City,
                address: h.address,
                coords:
                  h.lat != null && h.lng != null
                    ? { lat: Number(h.lat), lng: Number(h.lng), placeId: h.place_id ?? undefined }
                    : base?.coords,
                // Coerce every numeric the API hands back.
                //
                // Postgres NUMERIC arrives as a string unless a type parser is
                // registered, and one missing coercion took the entire site
                // down: `rating.toFixed(1)` threw and React unmounted, so every
                // page rendered blank. The backend now parses these properly,
                // but this stays as the second line of defence — a string
                // reaching `hotel.price + offset` concatenates instead of
                // adding, which misquotes a room rather than crashing, and is
                // therefore the more dangerous failure of the two.
                price: Number(h.base_price),
                weekendSurchargePercent: Number(h.weekend_surcharge_percent ?? 0),
                // Postgres returns NUMERIC as a string, so coerce rather than
                // letting "500" reach the arithmetic as a string.
                extraAdultPercent: h.extra_adult_percent != null ? Number(h.extra_adult_percent) : undefined,
                childFreeUnderAge: h.child_free_under_age != null ? Number(h.child_free_under_age) : undefined,
                childPercent: h.child_percent != null ? Number(h.child_percent) : undefined,
                adultFromAge: h.adult_from_age != null ? Number(h.adult_from_age) : undefined,
                rating: Number(h.rating),
                // Admin-uploaded photography. Carried through the merge so
                // `imagesForHotel` can prefer it over the bundled glob; absent
                // on an API that predates the feature, which just means the
                // bundled images keep being used.
                images: Array.isArray(h.images) ? h.images : undefined,
              }
            }))
            listeners.forEach(l => l())
          }
        })
        .catch(e => {
          console.error('Failed to fetch hotels', e)
          // Clear the in-flight promise so a later mount can retry. Leaving it
          // set pinned the whole session to static data after one flaky load.
          fetchPromise = null
        })
    }

    const listener = () => {
      if (cachedHotels) setHotels(cachedHotels)
    }
    listeners.add(listener)
    return () => { listeners.delete(listener) }
  }, [])

  return hotels
}

export const UPCOMING_HOTELS: UpcomingHotel[] = [
  { name: 'Rishikesh', location: 'Rishikesh, Uttarakhand', image: '/images/upcoming/rishikesh.webp', badge: 'COMING SOON' },
  { name: 'Agra', location: 'Agra, Uttar Pradesh', image: '/images/upcoming/agra.webp', badge: 'COMING SOON' },
  { name: 'Chandigarh', location: 'Chandigarh, Punjab', image: '/images/upcoming/chandigarh.webp', badge: 'COMING SOON' },
  { name: 'Dehradun', location: 'Dehradun, Uttarakhand', image: '/images/upcoming/dehradun.webp', badge: 'COMING SOON' },
  { name: 'Faridabad', location: 'Faridabad, Haryana', image: '/images/upcoming/faridabad.webp', badge: 'COMING SOON' },
  { name: 'Gurgaon', location: 'Gurgaon, Haryana', image: '/images/upcoming/gurgaon.webp', badge: 'COMING SOON' },
  // Manesar was dropped at the client's request (July 2026): with it the
  // Destinations grid ran to nine tiles and wrapped onto a second row.
  // DestinationsGrid renders the two live cities plus every entry here that
  // is not already live (New Delhi is), so this list must stay at seven for
  // the grid to hold one line.
  { name: 'New Delhi', location: 'New Delhi', image: '/images/upcoming/delhi.webp', badge: 'COMING SOON' },
]

export const CITIES: City[] = ['Noida', 'New Delhi']
export const CITY_FILTERS: readonly CityFilter[] = ['All', 'Noida', 'New Delhi']

// Banquet venues — §4/§6.4.
//
// Capacities are the client's own figures (feedback, 5 Aug 2026), not the
// representative specs this list shipped with: Downtown Sector 51 seats 40,
// Downtown EOK 80 and Amby Inn 65. They are an order of magnitude below the
// placeholders they replace, which is the point — the old numbers were
// quoting halls the group does not have. `hallArea` is deliberately left
// alone: the client restated capacity only, and a floor area invented to match
// the new headcount would be the same guess that produced the old numbers.
// Cladis was removed on the client's instruction, 28 Jul 2026: "cladis me
// banquet hall nahi h" — the property has no banquet hall, so the venue never
// existed. The Banquet Halls photo set they sent the same morning contains
// only the three below, which corroborates it. This list drives the banquets
// grid, the header dropdown and the /banquets/:slug routes, so removing the
// entry retires the page everywhere at once.
export const BANQUETS: BanquetVenue[] = [
  { slug: 'banquets-at-hotel-amby-inn', name: 'Banquets at Hotel Amby Inn', area: 'Lajpat Nagar', city: 'New Delhi', capacity: 65, hallArea: '4,200 sq ft', catering: 'Veg & Non-veg', parking: 'Valet available' },
  { slug: 'banquets-at-hotel-downtown-eok', name: 'Banquets at Hotel Downtown EOK', area: 'East of Kailash', city: 'New Delhi', capacity: 80, hallArea: '3,600 sq ft', catering: 'Veg & Non-veg', parking: 'Valet available' },
  { slug: 'banquets-at-hotel-downtown-sector-51', name: 'Banquets at Hotel Downtown Sector 51', area: 'Sector 51', city: 'Noida', capacity: 40, hallArea: '5,200 sq ft', catering: 'Veg & Non-veg', parking: 'On-site parking' },
  // Added 1 Sep 2026. Unlike the three above, every field here is the client's
  // own — 80 guests, 3,600 sq ft, veg and non-veg, valet — so nothing on this
  // row is a representative spec. `catering` and `parking` are worded to match
  // the existing rows rather than quoted from her brief, because BanquetsList
  // and banquetSeo() render these strings verbatim and a fourth phrasing of
  // "Vegetarian & Non-Vegetarian Options" would read as a different offering.
  //
  // PHOTOGRAPHY IS INTERIM. She sent six hotel photographs and no hall, so
  // public/images/banquets/banquets-at-hotel-amaltas-international/ holds this
  // property's own facade, reception and a Deluxe room — copies of the files in
  // its hotels/ folder. That is deliberate and it is not what should ship long
  // term: they are the right building but they are not the banquet hall.
  //
  // The alternative was worse. banquetImages() falls back to the shared
  // restaurant/dining pool when a venue has no folder, and that pool's lead
  // image is a rooftop terrace belonging to another property — so the page
  // advertised a venue this hotel does not have. Wrong-building beats
  // wrong-venue, but only until she sends hall photos: drop them into that
  // folder, delete these three, and nothing in the code changes.
  { slug: 'banquets-at-hotel-amaltas-international', name: 'Banquets at Hotel Amaltas International', area: 'Green Park Extension', city: 'New Delhi', capacity: 80, hallArea: '3,600 sq ft', catering: 'Veg & Non-veg', parking: 'Valet available' },
]

// ₹1,899 / night  (Indian comma grouping)
export const inr = (n: number): string => '₹' + Number(n).toLocaleString('en-IN')
export const priceNight = (n: number): string => `${inr(n)} / night`
export const stars = (r: number): string => `★ ${r.toFixed(1)}`
