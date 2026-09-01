import { PropertyRecord, RoomTypeRecord } from '../types'
import { baseRoomRateFor, mealUpliftFor } from '../lib/pricing'

/**
 * `rating` is the property's Google Business Profile score, transcribed from
 * the client's own listing audit (feedback, 5 Aug 2026) — not an editorial
 * number. Eight of the nine moved, two of them down by more than half a star
 * (Cladis 15 and Amby Inn, both 3.8), so this is the figure to trust when the
 * frontend's STATIC_HOTELS disagrees. `map_link` is likewise the GMB share
 * link; Cladis 15's pointed at the wrong listing until the same audit.
 */
export const seedProperties: PropertyRecord[] = [
  { id: 'prop-2', slug: 'hotel-quadis-sector-51-noida', lat: 28.5833, lng: 77.3712, name: 'Hotel Quadis Sector 51', city: 'Noida', address: 'H-22, Hoshiarpur Village, Sector 51, Noida, Uttar Pradesh 201301', map_link: 'https://share.google/X3cBuD2gbz27Jf5Ct', phone: '+91 92173 73532', whatsapp: '+91 92173 73532', email: 'stay@quadishotels.com', base_price: 1500, rating: 4.5, is_active: true, weekend_surcharge_percent: 0, extra_adult_percent: 30, child_free_under_age: 8, child_percent: 20, adult_from_age: 13, tier: 'central', tier_label: 'Quadis Central' },
  { id: 'prop-3', slug: 'hotel-quadis-central-sector-27-noida', lat: 28.5778, lng: 77.3243, name: 'Hotel Quadis Central', city: 'Noida', address: 'D-192, E Block, Pocket E, Sector 27, Noida, Uttar Pradesh 201301', map_link: 'https://share.google/VGqI5StPFPeLyZIMO', phone: '+91 92173 73532', whatsapp: '+91 92173 73532', email: 'stay@quadishotels.com', base_price: 2500, rating: 4.5, is_active: true, weekend_surcharge_percent: 0, extra_adult_percent: 30, child_free_under_age: 8, child_percent: 20, adult_from_age: 13, tier: 'central', tier_label: 'Quadis Central' },
  { id: 'prop-4', slug: 'hotel-downtown-sector-15-noida', lat: 28.5847, lng: 77.3129, name: 'Hotel Downtown Sector 15 Noida', city: 'Noida', address: 'Metro pillar no. 33, Opposite, New Ashok Nagar Rd, Naya Bans, Naya Bans Village, Sector 15, Noida, Uttar Pradesh 201301', map_link: 'https://share.google/oTnXw9glnDyZei1tL', phone: '+91 92173 73532', whatsapp: '+91 92173 73532', email: 'stay@quadishotels.com', base_price: 2000, rating: 4.0, is_active: true, weekend_surcharge_percent: 0, extra_adult_percent: 30, child_free_under_age: 8, child_percent: 20, adult_from_age: 13, tier: 'central', tier_label: 'Quadis Central' },
  { id: 'prop-5', slug: 'hotel-cladis-sector-15-noida', lat: 28.5855, lng: 77.311, name: 'Hotel Cladis Sector 15 Noida', city: 'Noida', address: 'New Ashok Nagar Rd, opposite metro pillar no. 36, Naya Bans, Naya Bans Village, Sector 15, Noida, Uttar Pradesh 201301', map_link: 'https://share.google/1Gbjxirb5YQWy6h6D', phone: '+91 92173 73532', whatsapp: '+91 92173 73532', email: 'stay@quadishotels.com', base_price: 1800, rating: 3.8, is_active: true, weekend_surcharge_percent: 0, extra_adult_percent: 30, child_free_under_age: 8, child_percent: 20, adult_from_age: 13, tier: 'central', tier_label: 'Quadis Central' },
  { id: 'prop-6', slug: 'hotel-cladis-sector-19-noida', lat: 28.583, lng: 77.321, name: 'Hotel Cladis Sector 19 Noida', city: 'Noida', address: 'A-369, A Block, Pocket A, Sector 19, Noida, Uttar Pradesh 201301', map_link: 'https://share.google/2YthY0ZjkrW3jnT3n', phone: '+91 92173 73532', whatsapp: '+91 92173 73532', email: 'stay@quadishotels.com', base_price: 2000, rating: 4.5, is_active: true, weekend_surcharge_percent: 0, extra_adult_percent: 30, child_free_under_age: 8, child_percent: 20, adult_from_age: 13, tier: 'central', tier_label: 'Quadis Central' },
  { id: 'prop-7', slug: 'hotel-downtown-sector-51-noida', lat: 28.5815, lng: 77.375, name: 'Hotel Downtown Sector 51 Noida', city: 'Noida', address: 'House No : C-155, Sector 51, Noida, Uttar Pradesh 201304', map_link: 'https://share.google/Mwl1FiCVC8ucqXrd', phone: '+91 92173 73532', whatsapp: '+91 92173 73532', email: 'stay@quadishotels.com', base_price: 2500, rating: 4.4, is_active: true, weekend_surcharge_percent: 0, extra_adult_percent: 30, child_free_under_age: 8, child_percent: 20, adult_from_age: 13, tier: 'central', tier_label: 'Quadis Central' },
  { id: 'prop-8', slug: 'hotel-downtown-east-of-kailash', lat: 28.555, lng: 77.245, name: 'Hotel Downtown EOK', city: 'New Delhi', address: 'B-14, B Block, East of Kailash, New Delhi, Delhi 110065', map_link: 'https://share.google/3RsBzxkp8xV1e0AuY', phone: '+91 92173 73532', whatsapp: '+91 92173 73532', email: 'stay@quadishotels.com', base_price: 3000, rating: 4.5, is_active: true, weekend_surcharge_percent: 0, extra_adult_percent: 30, child_free_under_age: 8, child_percent: 20, adult_from_age: 13, tier: 'central', tier_label: 'Quadis Central' },
  { id: 'prop-9', slug: 'hotel-amby-inn-lajpat-nagar-ii', lat: 28.57, lng: 77.24, name: 'Hotel Amby Inn', city: 'New Delhi', address: 'M13, Vinoba Puri, Block M, Lajpat Nagar II, Lajpat Nagar, New Delhi, Delhi 110024', map_link: 'https://share.google/pSTT03I5OWszpSj5c', phone: '+91 92173 73532', whatsapp: '+91 92173 73532', email: 'stay@quadishotels.com', base_price: 2500, rating: 3.8, is_active: true, weekend_surcharge_percent: 0, extra_adult_percent: 30, child_free_under_age: 8, child_percent: 20, adult_from_age: 13, tier: 'central', tier_label: 'Quadis Central' },
  { id: 'prop-10', slug: 'hotel-amar-inn', lat: 28.571, lng: 77.2415, name: 'Hotel Amar Inn', city: 'New Delhi', address: 'K-102, Road, near Central Market, Block K, Lajpat Nagar II, Jal Vihar, New Delhi, Delhi 110024', map_link: 'https://share.google/IQLx35cfOmLf93S2o', phone: '+91 92173 73532', whatsapp: '+91 92173 73532', email: 'stay@quadishotels.com', base_price: 3000, rating: 4.3, is_active: true, weekend_surcharge_percent: 0, extra_adult_percent: 30, child_free_under_age: 8, child_percent: 20, adult_from_age: 13, tier: 'central', tier_label: 'Quadis Central' },
  /*
   * Added 1 Sep 2026 from the client's new-property brief. `rating` is 4.1,
   * which she gave us directly rather than via the GMB audit; `lat`/`lng` are
   * locality-level, as explained on the matching STATIC_HOTELS row in
   * src/data/hotels.ts — this file and that one must not disagree, since the
   * API record wins on the live site. `map_link` IS hers, from the brief.
   *
   * Contact columns are the group's shared reservations desk, as on every other
   * property; if this hotel takes its own line, it is a one-field change here.
   */
  { id: 'prop-11', slug: 'hotel-amaltas-international', lat: 28.5603, lng: 77.2022, name: 'Hotel Amaltas International', city: 'New Delhi', address: '6, Opposite Sukhmani Hospital, Block W, Green Park Extension, Green Park, New Delhi, Delhi 110016', map_link: 'https://share.google/VkPBBiHgI3Vpgx3y0', phone: '+91 92173 73532', whatsapp: '+91 92173 73532', email: 'stay@quadishotels.com', base_price: 4000, rating: 4.1, is_active: true, weekend_surcharge_percent: 0, extra_adult_percent: 30, child_free_under_age: 8, child_percent: 20, adult_from_age: 13, tier: 'central', tier_label: 'Quadis Central' },
]

/**
 * A category before it is attached to a property.
 *
 * `breakfast_offset` and `all_meals_offset` are zero on every template and are
 * filled in per property by roomsFor(). They stopped being a property of the
 * category on 5 Aug 2026, when the client moved meal plans onto a percentage of
 * the base room rate: the same Deluxe template seeds a ₹1,500 room at Quadis 51
 * and a ₹3,000 one at Amar Inn, and 25% of those is not the same number.
 */
type RoomTemplate = Omit<RoomTypeRecord, 'id' | 'property_id'>

const DELUXE: RoomTemplate = {
  slug: 'deluxe-room',
  name: 'Deluxe Room',
  description: 'Calm, refined comfort designed for effortless rest. Features plush bedding and executive workspace.',
  size_sqft: '240 sq ft',
  bed_type: 'King / Twin Beds',
  max_guests: 2,
  price_offset: 0,
  breakfast_offset: 0,
  all_meals_offset: 0,
  total_units: 5,
  available_units: 5,
  is_available: true,
}

const ROYAL: RoomTemplate = {
  slug: 'royal-suite',
  name: 'Royal Suite',
  description: 'Our most luxurious sanctuary featuring separate master bedroom, private lounge and dining area.',
  size_sqft: '450 sq ft',
  bed_type: 'Master Suite + Living Room',
  max_guests: 4,
  price_offset: 2000,
  breakfast_offset: 0,
  all_meals_offset: 0,
  total_units: 1,
  available_units: 1,
  is_available: true,
}

const SUPER: RoomTemplate = {
  slug: 'super-deluxe',
  name: 'Super Deluxe',
  description: 'A larger, more considered room with an upgraded seating area, high-speed Wi-Fi and evening turndown.',
  size_sqft: '290 sq ft',
  bed_type: 'King Bed',
  max_guests: 3,
  price_offset: 1000,
  breakfast_offset: 0,
  all_meals_offset: 0,
  total_units: 4,
  available_units: 4,
  is_available: true,
}

/**
 * Amaltas International's upper category — the group's first exception to
 * "upper category 1000 plus in each hotel".
 *
 * Her brief prices the Deluxe at ₹4,000 and the Superior at ₹6,000, so the step
 * is ₹2,000. That is ROYAL's offset with SUPER's shape, and neither template
 * could be reused: SUPER would have sold it for ₹5,000, and ROYAL would have
 * priced it right while booking the guest into a "Royal Suite", because
 * `room_types.id` is derived from the slug. Mirrors SUPERIOR in the frontend's
 * src/data/hotels.ts, which must quote the same number.
 */
const SUPERIOR: RoomTemplate = {
  slug: 'superior-room',
  name: 'Superior Room',
  description: 'An elevated, more generous room with a separate seating area, high-speed Wi-Fi and evening turndown.',
  size_sqft: '310 sq ft',
  bed_type: 'King Bed',
  max_guests: 3,
  price_offset: 2000,
  breakfast_offset: 0,
  all_meals_offset: 0,
  total_units: 4,
  available_units: 4,
  is_available: true,
}

/**
 * Real key counts, from the client's rate sheet (27 Jul 2026). These replace the
 * placeholder 5/3/2 the templates above still carry as defaults — placeholders
 * that were the standing double-booking risk on this project, since a property
 * with 4 Super keys seeded as 5 sells one room it does not have.
 *
 * The sheet's own totals are 125 keys across Noida and 72 across Delhi, 197 in
 * all. Two rows could not be taken at face value; both are marked below and are
 * deliberately seeded LOW, because under-selling loses a booking while
 * over-selling loses a guest who has already paid.
 */
/**
 * `super` became optional on 1 Sep 2026 so a property can sell a Superior
 * instead of a Super Deluxe. Every pre-existing row still sets it, so nothing
 * about the nine below changed — but a plan that omits a category must not seed
 * it with zero keys, which is what a required field would have forced.
 */
interface RoomPlan { deluxe: number; super?: number; superior?: number; royal?: number }

const KEYS_BY_SLUG: Record<string, RoomPlan> = {
  // Noida — sheet total 125.
  'hotel-downtown-sector-15-noida': { deluxe: 24, super: 4 },  // 28
  'hotel-downtown-sector-51-noida': { deluxe: 6, super: 3 },   // 9
  'hotel-cladis-sector-19-noida': { deluxe: 10, super: 2 },    // 12
  // Confirmed by the client on WhatsApp, 27 Jul 2026, resolving the sheet's two
  // unreadable rows: "Quadis central me 11 deluxe or 6 super deluxe h" and "Or
  // quadis 51 me 25 deluxe 3 super deluxe". They also clarified the sheet
  // itself — "Total vala column Total Rooms ka h" — so where a row's TOTAL and
  // its category breakdown disagreed, the TOTAL was right.
  //
  // That settles Quadis 51 at 28, not the 9 its 6 + 3 breakdown implied, and
  // these two figures are what make the group total land on the sheet's stated
  // 197 keys exactly.
  'hotel-quadis-sector-51-noida': { deluxe: 25, super: 3 },         // 28
  'hotel-cladis-sector-15-noida': { deluxe: 24, super: 7 },         // 31
  'hotel-quadis-central-sector-27-noida': { deluxe: 11, super: 6 }, // 17

  // Delhi — sheet total 72.
  'hotel-downtown-east-of-kailash': { deluxe: 23, super: 6, royal: 1 }, // 30
  'hotel-amby-inn-lajpat-nagar-ii': { deluxe: 20, super: 3 },           // 23
  'hotel-amar-inn': { deluxe: 12, super: 6, royal: 1 },                  // 19

  // Added 1 Sep 2026. NOT from a rate sheet: the client's brief gives the two
  // categories and their rates and says nothing about how many keys of each the
  // property has. These are therefore placeholders of exactly the kind the note
  // above calls the standing double-booking risk on this project, and they are
  // set LOW on purpose, for the reason given there — under-selling loses a
  // booking, over-selling loses a guest who has already paid. Ask her for the
  // key count and replace both figures before this property goes live.
  'hotel-amaltas-international': { deluxe: 6, superior: 2 },             // 8 (UNCONFIRMED)
}

/**
 * Category list per property, derived from the key plan rather than repeated —
 * a hotel sells exactly the categories it has keys for. Must stay in step with
 * ROOMS_BY_SLUG in the frontend's src/data/hotels.ts.
 */
const roomsFor = (plan: RoomPlan, basePrice: number): RoomTemplate[] => {
  /**
   * Key count and meal supplements, both of which depend on the property.
   *
   * The meal columns are written as the percentage of this room's base rate
   * (property base_price + the category's price_offset) that the client set on
   * 5 Aug 2026 — CP 25%, MAP 50%. They are stored rather than left at zero
   * because the columns are still read directly by the concierge when it quotes
   * "breakfast adds ₹x", and by the in-memory development store, which holds
   * bare room rows with no property price on them for mealOffsetFor() to work
   * from. Deriving them from the same helper the pricing library uses is what
   * keeps the stored rupees and the live percentage the same number.
   */
  const forProperty = (t: RoomTemplate, n: number): RoomTemplate => {
    const baseRate = baseRoomRateFor(basePrice, t.price_offset)
    return {
      ...t,
      total_units: n,
      available_units: n,
      breakfast_offset: mealUpliftFor('With Breakfast', baseRate),
      all_meals_offset: mealUpliftFor('All Meals Included', baseRate),
    }
  }
  return [
    forProperty(DELUXE, plan.deluxe),
    ...(plan.super ? [forProperty(SUPER, plan.super)] : []),
    ...(plan.superior ? [forProperty(SUPERIOR, plan.superior)] : []),
    ...(plan.royal ? [forProperty(ROYAL, plan.royal)] : []),
  ]
}

/** Reached only by a slug not in KEYS_BY_SLUG, which today is none of them. */
const DEFAULT_PLAN: RoomPlan = { deluxe: 5, super: 3 }

export const seedRoomTypes: RoomTypeRecord[] = seedProperties.flatMap((prop) =>
  roomsFor(KEYS_BY_SLUG[prop.slug] ?? DEFAULT_PLAN, prop.base_price).map((t) => ({
    ...t,
    id: `room-${prop.id}-${t.slug}`,
    property_id: prop.id,
  }))
)
