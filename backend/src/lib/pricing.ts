import type { MealPlan } from '../types'

/**
 * Single source of truth for what a stay costs.
 *
 * The frontend mirrors this in src/lib/pricing.ts so the quoted price and the
 * charged price agree. If you change the rule here, change it there too.
 */

/**
 * The shape `mealOffsetFor` needs off a room row.
 *
 * `base_price` is a property column, not a room column — it is here because the
 * booking query joins the two (`SELECT rt.*, p.base_price ...`), and the meal
 * uplift is a percentage OF the room's nightly rate, which no room row carries
 * on its own. Everything is optional so both the joined row and a bare
 * RoomTypeRecord satisfy it; the fallback chain in `mealOffsetFor` covers the
 * bare case.
 */
export interface RoomMealOffsets {
  /** The property's nightly rate for its cheapest category. */
  base_price: number
  /** What this category adds to it — Super Deluxe +1000, Royal +2000. */
  price_offset: number
  /**
   * Legacy flat-rupee meal supplements, superseded by the percentages below on
   * 5 Aug 2026. Still written by the seed and the schema migration so that they
   * agree with the percentage to the rupee, and still read as a last resort by
   * `mealOffsetFor` when no base rate is available.
   */
  breakfast_offset: number
  all_meals_offset: number
}

/* ---------- Occupancy ---------- */

/**
 * Every rate on this site is quoted for two adults sharing a room. A third
 * adult is an extra bed, not a free upgrade.
 *
 * The rule, in the client's own words. They have now stated it three times, so
 * the history matters:
 *
 *   26 Jul 2026, WhatsApp:
 *     "Double occupancy room ka 40% increase hoga triple mein"
 *     "And agar teesra person adult hain only then" / "If it's child then no"
 *   27 Jul 2026, rate sheet: "EXTRA ADULT 500, CHILD 250"
 *   27 Jul 2026, answering "is the 40% the same at all hotels":
 *     "If a third person is included in the same room, a 30% extra charge will
 *      apply for the additional mattress."
 *
 * The last one wins: it is the most recent, and it is a direct answer to a
 * direct question about this number, which the rate sheet's cell was not. It
 * also confirms the model is a PERCENTAGE, not the sheet's flat rupee figure.
 *
 * A percentage tracks the room rate and the weekend surcharge automatically —
 * a triple on a surcharged Friday costs 30% more than that Friday's double,
 * not 30% more than a weekday double.
 *
 * RESOLVED 27 Jul 2026 — this paragraph used to say children were free at any
 * age pending confirmation. She confirmed, and the answer was three bands, not
 * "free": under 8 free, 8-12 at 20%, 13 and over charged as an adult. That is
 * AGENTS.md §2 rule 3 and it is what the DEFAULT_ constants below encode.
 * Do not reintroduce "children are free" — it under-bills every 8-to-12-year-old
 * and it disagrees with the schema, the seed and the live API.
 *
 * The live figures are per property, set from the admin panel and stored on
 * `properties` (`extra_adult_percent`, `child_free_under_age`, `child_percent`,
 * `adult_from_age`). These constants are only the fallbacks for a record that
 * predates those columns.
 */
export const DEFAULT_EXTRA_ADULT_PERCENT = 30

/** Adults included in the advertised nightly rate, per room. */
export const STANDARD_OCCUPANCY_PER_ROOM = 2

/**
 * Age bands, per the client on 27 Jul 2026:
 *
 *   "for extra adult 30%, 0-7 years waalo ka free rahega stay,
 *    baki 8-12 years ka 20%"
 *
 * So three bands, not the single free/charged threshold this file used to have:
 *
 *   0-7    free
 *   8-12   20% of the nightly room rate
 *   13+    charged as an adult, currently 30%
 *
 * ASSUMPTION, NOT CONFIRMED: the client specified 0-7 and 8-12 and stopped. 13
 * to 17 is read as adult, because "baki 8-12 ka 20%" implies the concession
 * ends at 12. If they meant under-18s to stay on 20%, raise DEFAULT_ADULT_FROM_AGE.
 * Flagged in docs/client-comms/.
 */
export const DEFAULT_CHILD_FREE_UNDER_AGE = 8
export const DEFAULT_CHILD_PERCENT = 20
export const DEFAULT_ADULT_FROM_AGE = 13

/** The occupancy policy in force for one property. */
export interface OccupancyPolicy {
  /** Percentage of the nightly room rate added per extra adult. */
  extraAdultPercent: number
  /** Below this age a guest is free. */
  childFreeUnderAge: number
  /** Percentage added per child in the concession band. */
  childPercent: number
  /** At this age and above, a guest is charged as an adult. */
  adultFromAge: number
}

/** Reads the policy off a property row, falling back for older records. */
export function policyFor(property: {
  extra_adult_percent?: number | string | null
  child_free_under_age?: number | string | null
  child_percent?: number | string | null
  adult_from_age?: number | string | null
} | null | undefined): OccupancyPolicy {
  const num = (v: unknown, fallback: number): number => {
    const n = Number(v)
    return Number.isFinite(n) && n >= 0 ? n : fallback
  }
  return {
    extraAdultPercent: num(property?.extra_adult_percent, DEFAULT_EXTRA_ADULT_PERCENT),
    childFreeUnderAge: num(property?.child_free_under_age, DEFAULT_CHILD_FREE_UNDER_AGE),
    childPercent: num(property?.child_percent, DEFAULT_CHILD_PERCENT),
    adultFromAge: num(property?.adult_from_age, DEFAULT_ADULT_FROM_AGE),
  }
}

export interface OccupancyInput {
  adults: number
  /** One entry per child, their age at check-in. */
  childAges?: number[]
  roomsCount: number
  childFreeUnderAge?: number
  adultFromAge?: number
}

/** Heads beyond the included occupancy, split by the rate each is charged at. */
export interface ChargeableGuests {
  /** Charged at extraAdultPercent. */
  extraAdults: number
  /** Charged at childPercent. */
  extraChildren: number
}

/**
 * Who has to be paid for, and at which rate.
 *
 * Two adults per room are included. Guests under childFreeUnderAge are free and
 * never consume an included place — a toddler sharing the bed does not push a
 * paying guest into a surcharge.
 *
 * The included places are filled with the most expensive heads first, so any
 * excess is drawn from the cheaper band. Two adults and a ten-year-old in one
 * room therefore pay the child rate on the child, not the adult rate: the
 * adults take the two included places. Doing it the other way round would
 * quietly overcharge every family.
 *
 * Counted across the whole booking rather than per room, so three adults in two
 * rooms sit inside the four included places and pay nothing extra.
 */
export function chargeableGuestsFor(input: OccupancyInput): ChargeableGuests {
  const freeUnder = Number.isFinite(Number(input.childFreeUnderAge))
    ? Number(input.childFreeUnderAge) : DEFAULT_CHILD_FREE_UNDER_AGE
  const adultFrom = Number.isFinite(Number(input.adultFromAge))
    ? Number(input.adultFromAge) : DEFAULT_ADULT_FROM_AGE

  const ages = Array.isArray(input.childAges) ? input.childAges.map(Number) : []
  const childBand = ages.filter((a) => a >= freeUnder && a < adultFrom).length
  const adultLike = (Number(input.adults) || 0) + ages.filter((a) => a >= adultFrom).length

  const included = STANDARD_OCCUPANCY_PER_ROOM * (Number(input.roomsCount) || 1)
  const extraAdults = Math.max(0, adultLike - included)
  const includedLeft = Math.max(0, included - adultLike)

  return { extraAdults, extraChildren: Math.max(0, childBand - includedLeft) }
}

/** Children old enough to be charged at all — the 8-12 band plus any 13+. */
export function chargeableChildren(
  childAges: number[] | undefined,
  childFreeUnderAge: number = DEFAULT_CHILD_FREE_UNDER_AGE
): number {
  if (!Array.isArray(childAges)) return 0
  return childAges.filter((age) => Number(age) >= childFreeUnderAge).length
}

/**
 * Kept for callers that only need a single count. Prefer chargeableGuestsFor —
 * this collapses two different rates into one number and will under-quote a
 * booking that includes a child in the concession band.
 */
export function extraAdultsFor(input: OccupancyInput): number {
  return chargeableGuestsFor(input).extraAdults
}

/* ---------- Nightly rate ---------- */

/** Fri and Sat nights carry the property's weekend surcharge. */
export function isWeekendNight(date: Date): boolean {
  const day = date.getUTCDay()
  return day === 5 || day === 6
}

/* ---------- Meal plans ---------- */

/**
 * Meal plans are a PERCENTAGE of the base room rate, not a flat rupee figure.
 *
 * The client, 5 Aug 2026:
 *
 *   "Please add a percentage-based pricing option for meal plans in the
 *    category settings.
 *      EP: No additional charge
 *      CP (Breakfast): +25%
 *      MAP (All Meals Included): +50%
 *    This percentage should be applied automatically across all hotels based on
 *    the base room rate. Currently, the CP and MAP prices are displaying
 *    incorrectly, so implementing percentage-based pricing will ensure accurate
 *    and consistent rates across all properties."
 *
 * EP / CP / MAP are the trade's names for the three plans this site already
 * sells as "Room Only" / "With Breakfast" / "All Meals Included", in that order.
 *
 * What this replaces: a flat supplement per room category — ₹300/₹800 on a
 * Deluxe, ₹350/₹900 on a Super Deluxe, ₹450/₹1,200 on a Royal. Flat rupees are
 * what she means by "displaying incorrectly": ₹300 of breakfast on Cladis 15's
 * ₹1,800 Deluxe is a 17% uplift, and the same ₹300 on Amar Inn's ₹3,000 Deluxe
 * is 10%. The same plan cost a different fraction at every property, which is
 * precisely the inconsistency the percentage removes.
 *
 * These are group-wide and deliberately NOT per property or per category. "The
 * same percentage across all hotels" is the instruction; a per-row override
 * would let one property drift and would have to be mirrored into the frontend
 * bundle to keep the quoted price and the charged price equal.
 */
export const MEAL_PLAN_UPLIFT_PERCENT: Record<MealPlan, number> = {
  'Room Only': 0,
  'With Breakfast': 25,
  'All Meals Included': 50,
}

/**
 * What the percentage is a percentage OF.
 *
 * The base room rate is the property's base price plus that category's offset —
 * the room's nightly rate BEFORE meals, before the weekend surcharge, before
 * any extra-bed uplift and before tax. So Downtown EOK's Super Deluxe is
 * 3,000 + 1,000 = ₹4,000, and that ₹4,000 is what CP and MAP are measured
 * against.
 *
 * Reading it any other way breaks the client's arithmetic. Measuring off the
 * property base price alone would make CP on a Royal Suite cost the same as CP
 * on a Deluxe, and measuring off the surcharged Friday rate would make the same
 * breakfast cost more on a Friday than on a Tuesday.
 *
 * The consequence, which is the point: nightly = base + meal uplift, so
 * CP = 1.25x and MAP = 1.5x the room's pre-tax nightly rate exactly.
 */
export function baseRoomRateFor(basePrice: number | string | null | undefined,
                                roomOffset: number | string | null | undefined): number {
  return (Number(basePrice) || 0) + (Number(roomOffset) || 0)
}

/**
 * The meal supplement in whole rupees for one room for one night.
 *
 * Rounded here, once, for the same reason the extra-adult uplift is rounded per
 * night: 25% of ₹1,800 is ₹450 but 25% of ₹1,899 is ₹474.75, and a hotel quotes
 * a whole number. Rounding at this point also keeps the nightly rate an integer
 * before the weekend surcharge multiplies it.
 */
export function mealUpliftFor(plan: MealPlan | undefined, baseRoomRate: number): number {
  const percent = plan ? MEAL_PLAN_UPLIFT_PERCENT[plan] ?? 0 : 0
  const base = Number(baseRoomRate)
  if (!Number.isFinite(base) || base <= 0 || percent <= 0) return 0
  return Math.round(base * (percent / 100))
}

/**
 * The meal supplement for a room row.
 *
 * `baseRate` is preferred when the caller has already computed it. Otherwise it
 * is derived from the row, which works because the booking query joins the
 * property's base_price onto the room.
 *
 * The flat `breakfast_offset` / `all_meals_offset` columns are the last resort,
 * for a row that carries no base price at all — the in-memory development store
 * holds bare RoomTypeRecords. That is not a second pricing model: the seed and
 * the schema migration both write those columns as the percentage of that
 * property's base room rate, so the fallback returns the same rupees this
 * function would have computed. If the two ever disagree the percentage is
 * right and the column is stale.
 */
export function mealOffsetFor(
  plan: MealPlan | undefined,
  room: Partial<RoomMealOffsets>,
  baseRate?: number
): number {
  const base = baseRate != null
    ? Number(baseRate)
    : room.base_price != null
      ? baseRoomRateFor(room.base_price, room.price_offset)
      : null

  if (base != null && Number.isFinite(base) && base > 0) return mealUpliftFor(plan, base)

  if (plan === 'With Breakfast') return Number(room.breakfast_offset) || 0
  if (plan === 'All Meals Included') return Number(room.all_meals_offset) || 0
  return 0
}

/* ---------- Stay total ---------- */

export interface StayPricingInput {
  basePrice: number
  roomOffset: number
  /**
   * The meal supplement in rupees for one room-night, from `mealUpliftFor` or
   * `mealOffsetFor`. Still rupees rather than a percentage because it has to be
   * added to the nightly rate before the weekend surcharge is applied to it,
   * and because a caller that has already resolved it should not resolve it
   * twice.
   */
  mealOffset: number
  weekendSurchargePercent: number
  checkIn: string
  checkOut: string
  roomsCount: number
  /**
   * Adults beyond the two included per room, from extraAdultsFor(). Each adds
   * extraAdultPercent of that night's room rate, for every night of the stay.
   */
  extraAdults?: number
  /**
   * Children in the concession band (8-12 by default), from
   * chargeableGuestsFor(). Each adds childPercent of that night's room rate.
   */
  extraChildren?: number
  /** The property's admin-set percentage uplift per extra adult. */
  extraAdultPercent?: number
  /** The property's admin-set percentage uplift per concession-band child. */
  childPercent?: number
}

export interface StayPricingBreakdown {
  nights: number
  /** Room charge for the whole stay, all rooms, weekend surcharge included. */
  roomTotal: number
  /** Extra-adult charge for the whole stay. */
  extraAdultTotal: number
  extraAdults: number
  /** Concession-band child charge for the whole stay. */
  extraChildTotal: number
  extraChildren: number
  /** The percentages applied, so callers can display and store them. */
  extraAdultPercent: number
  childPercent: number
  /**
   * Rupees per extra adult per night, averaged over the stay. Derived, not an
   * input — stored on the booking so an invoice can show a figure, and averaged
   * because a stay spanning a weekend surcharge has more than one nightly rate.
   */
  extraAdultChargePerNight: number
  total: number
}

/**
 * Prices each night individually so weekend nights can carry the surcharge,
 * multiplies by room count, then adds the extra-adult uplift. Returns whole
 * paise-accurate rupees.
 *
 * The uplift is a percentage of *that night's* room rate, accumulated inside the
 * same loop. That matters when a stay straddles a weekend: a Friday triple is
 * 30% above the surcharged Friday double, not 30% above a weekday rate.
 *
 * It is charged per extra adult and NOT multiplied by room count — one extra
 * person occupies one extra bed, however many rooms the booking spans.
 *
 * How the meal uplift composes with the rest, since three percentages now meet
 * in one line and it would be easy to double-count:
 *
 *   base       = property base_price + category price_offset
 *   mealOffset = MEAL_PLAN_UPLIFT_PERCENT × base, resolved by the caller
 *   nightly    = base + mealOffset                    (25%/50% applied ONCE)
 *   rate       = nightly × (1 + weekendSurcharge)     (Fri/Sat only)
 *   extra bed  = extraAdultPercent × rate             (per head, per night)
 *
 * The meal uplift is taken off `base` alone, so it is never a percentage of a
 * percentage. The weekend surcharge and the extra-bed charge then run on the
 * meal-inclusive `nightly`, which is the behaviour the flat supplements already
 * had — a third adult on All Meals is fed, so charging their 30% on the
 * meal-inclusive rate is the intended reading and not a regression.
 */
export function computeStayBreakdown(input: StayPricingInput): StayPricingBreakdown {
  const nightly =
    (Number(input.basePrice) || 0) + (Number(input.roomOffset) || 0) + (Number(input.mealOffset) || 0)
  const surcharge = Number(input.weekendSurchargePercent) || 0

  const extraAdults = Math.max(0, Number(input.extraAdults) || 0)
  const extraChildren = Math.max(0, Number(input.extraChildren) || 0)
  const rooms = Number(input.roomsCount) || 1

  const pct = (v: unknown, fallback: number): number => {
    const n = Number(v)
    return Number.isFinite(n) && n >= 0 ? n : fallback
  }
  const extraAdultPercent = pct(input.extraAdultPercent, DEFAULT_EXTRA_ADULT_PERCENT)
  const childPercent = pct(input.childPercent, DEFAULT_CHILD_PERCENT)

  const end = new Date(input.checkOut)
  const cursor = new Date(input.checkIn)

  /**
   * The uplift for one extra adult on a night at `rate`, in whole rupees.
   *
   * Rounded per night rather than at the end: 30% of ₹1,799 is ₹539.70, and a
   * hotel quotes ₹640. Leaving it unrounded surfaced totals like "₹2,238.6" in
   * the guest's summary, which reads as a bug rather than a price.
   */
  const upliftPerAdult = (rate: number) => Math.round(rate * (extraAdultPercent / 100))
  const upliftPerChild = (rate: number) => Math.round(rate * (childPercent / 100))

  let roomTotal = 0
  let extraAdultTotal = 0
  let extraChildTotal = 0
  let nights = 0
  while (cursor < end) {
    const rate = isWeekendNight(cursor) ? nightly * (1 + surcharge / 100) : nightly
    roomTotal += rate
    extraAdultTotal += upliftPerAdult(rate) * extraAdults
    extraChildTotal += upliftPerChild(rate) * extraChildren
    nights += 1
    cursor.setUTCDate(cursor.getUTCDate() + 1)
  }

  // Never bill a zero-night stay as free. Callers reject checkout <= check-in
  // before reaching here, so this is a floor rather than a real code path.
  if (nights === 0) {
    roomTotal = nightly
    extraAdultTotal = upliftPerAdult(nightly) * extraAdults
    extraChildTotal = upliftPerChild(nightly) * extraChildren
    nights = 1
  }

  const round = (n: number) => Math.round(n * 100) / 100
  const roomCharge = round(roomTotal * rooms)
  const extraCharge = round(extraAdultTotal)
  const childCharge = round(extraChildTotal)

  return {
    nights,
    roomTotal: roomCharge,
    extraAdultTotal: extraCharge,
    extraAdults,
    extraChildTotal: childCharge,
    extraChildren,
    extraAdultPercent,
    childPercent,
    extraAdultChargePerNight: extraAdults > 0 ? round(extraCharge / (extraAdults * nights)) : 0,
    total: round(roomCharge + extraCharge + childCharge),
  }
}

export function computeStayTotal(input: StayPricingInput): number {
  return computeStayBreakdown(input).total
}

/* ---------- GST ---------- */

/**
 * GST on accommodation, SAC 996311.
 *
 * The client, 5 Aug 2026: "our gst is 5%, so please replace 12% with 5%".
 *
 * That reads like a mistake and is not. Until the September 2025 rate revision
 * a room under ₹7,500 a night was taxed at 12%; the revision retired that slab
 * and taxes the same rooms at 5%, charged without input tax credit. The upper
 * slab was left at 18%. So the client is quoting the rate her accountant is
 * actually filing at, and the 12% this codebase shipped with is simply out of
 * date — it was correct when it was written.
 *
 * The 18% slab stays, and as of 1 Sep 2026 it is no longer hypothetical — one
 * room in the group now reaches it. Hotel Amaltas International's Superior is
 * ₹6,000 before meals, and All Meals Included adds 50%, so it quotes ₹9,000
 * inclusive: a value of supply of ₹8,571, past the line. It is the only such
 * combination in the portfolio; see the exact-set assertion in
 * backend/__tests__/pricing.test.ts, which is what will catch the next one.
 *
 * The near miss is worth keeping in view too. A Royal Suite at East of Kailash
 * or Amar Inn is ₹5,000 before meals, so All Meals puts it at exactly ₹7,500 —
 * the threshold to the rupee, and INSIDE the 5% band once converted to a value
 * of supply. Under the flat supplements it retired it was ₹6,200 and nowhere
 * near. So the upper slab must neither be deleted, which would under-collect on
 * the bookings worth the most, nor applied a rupee too early.
 *
 * These three constants are duplicated in src/lib/pricing.ts, which is the
 * frontend's mirror of this file. The duplication is deliberate — the checkout
 * summary must show the same rate the invoice charges, and the frontend cannot
 * import from backend/. Change one, change the other; there is no build step
 * that will catch a divergence.
 */
export const GST_LUXURY_THRESHOLD_PER_ROOM_NIGHT = 7500
export const GST_PERCENT_STANDARD = 5
export const GST_PERCENT_LUXURY = 18

/**
 * The slab a booking falls in.
 *
 * `inclusiveRatePerRoomNight` is the booking total divided by nights and rooms,
 * which is what both callers pass and is INCLUSIVE of GST — every price this
 * site quotes is what the guest actually pays, and the invoice back-calculates
 * the tax out of it.
 *
 * The slab, however, is defined on the VALUE OF SUPPLY — the figure net of GST —
 * so the two have to be converted before they can be compared. This used to
 * compare the inclusive rupees straight against ₹7,500, which was wrong, and was
 * harmless only for as long as nothing came near the threshold.
 *
 * The meal percentages ended that on 5 Aug 2026. A Royal Suite on All Meals is
 * ₹5,000 + 50% = exactly ₹7,500 inclusive, whose value of supply is ₹7,142.86 —
 * comfortably inside the 5% band, but the old comparison read the ₹7,500 and
 * moved it to 18%. That is a 13-point over-charge on the group's two most
 * expensive rooms, invisible until someone books one.
 *
 * Testing at the standard rate is the right way round: if a supply is not
 * luxury, it is taxed at 5% and dividing by 1.05 recovers its true value. In
 * inclusive terms the line therefore sits at ₹7,875, not ₹7,500.
 *
 * `>` rather than `>=` because the rule is "above ₹7,500 per unit per day"; a
 * supply valued at exactly ₹7,500 stays in the lower band.
 */
export function gstRatePercentFor(inclusiveRatePerRoomNight: number): number {
  const inclusive = Number(inclusiveRatePerRoomNight) || 0
  const valueOfSupply = inclusive / (1 + GST_PERCENT_STANDARD / 100)
  return valueOfSupply > GST_LUXURY_THRESHOLD_PER_ROOM_NIGHT
    ? GST_PERCENT_LUXURY
    : GST_PERCENT_STANDARD
}
