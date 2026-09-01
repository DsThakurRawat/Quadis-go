import {
  MEAL_PLAN_UPLIFT_PERCENT,
  baseRoomRateFor,
  mealUpliftFor,
  mealOffsetFor,
  computeStayBreakdown,
  gstRatePercentFor,
  GST_PERCENT_STANDARD,
  GST_PERCENT_LUXURY,
  GST_LUXURY_THRESHOLD_PER_ROOM_NIGHT,
} from '../src/lib/pricing'
import { seedProperties, seedRoomTypes } from '../src/data/seed'
import type { MealPlan } from '../src/types'

/**
 * The two rules the client set on 5 Aug 2026, locked down.
 *
 * Neither had any test coverage before: the invoice suite renders a PDF without
 * asserting a tax rate, and nothing anywhere asserted a meal price. Both numbers
 * are money the guest is charged, and both were wrong in production until this
 * change, so they get tests that fail loudly if anyone edits the constants.
 */

const PLANS: MealPlan[] = ['Room Only', 'With Breakfast', 'All Meals Included']

describe('GST — 5%, not 12% (client, 5 Aug 2026)', () => {
  it('taxes an ordinary room at 5%', () => {
    expect(GST_PERCENT_STANDARD).toBe(5)
    expect(gstRatePercentFor(2000)).toBe(5)
    expect(gstRatePercentFor(7499.99)).toBe(5)
  })

  it('keeps the 18% slab for supplies valued above the threshold', () => {
    expect(GST_PERCENT_LUXURY).toBe(18)
    expect(GST_LUXURY_THRESHOLD_PER_ROOM_NIGHT).toBe(7500)
    // The argument is GST-inclusive; the slab is defined on the value of supply.
    // 7,875 inclusive is exactly 7,500 net at the 5% rate, so the line sits there.
    expect(gstRatePercentFor(7875)).toBe(5)
    expect(gstRatePercentFor(7875.01)).toBe(18)
    expect(gstRatePercentFor(20000)).toBe(18)
  })

  it('does not push the ₹7,500 Royal Suite on All Meals into the 18% slab', () => {
    // Regression. EOK and Amar Inn Royal Suites are ₹5,000 + 50% = ₹7,500
    // inclusive, whose value of supply is ₹7,142.86 — inside the 5% band. The
    // old comparison read the inclusive ₹7,500 and over-charged 13 points of tax
    // on the group's two most expensive rooms.
    expect(gstRatePercentFor(7500)).toBe(5)
    expect(7500 / 1.05).toBeCloseTo(7142.86, 2)
  })

  /**
   * Which seeded rooms actually reach the upper slab.
   *
   * Until 1 Sep 2026 the answer was "none", and this test asserted exactly that.
   * Hotel Amaltas International changed it: its Superior is ₹6,000, and All
   * Meals adds 50%, so the quote is ₹9,000 inclusive — a value of supply of
   * ₹8,571, comfortably past the ₹7,500 line. It is the first room in the group
   * to cross it, and 18% on it is correct, not a regression.
   *
   * The assertion is therefore an exact set rather than a blanket "all 5%": it
   * still fails loudly if a rate change quietly pushes another room over, and it
   * fails just as loudly if the threshold logic starts over-taxing rooms that
   * belong in the 5% band — which is the bug the ₹7,500 Royal Suite test above
   * guards from the other side.
   */
  it('quotes every seeded room in the 5% slab except the one that genuinely clears the threshold', () => {
    const luxury: string[] = []

    for (const room of seedRoomTypes) {
      const prop = seedProperties.find((p) => p.id === room.property_id)!
      const base = baseRoomRateFor(prop.base_price, room.price_offset)
      for (const plan of PLANS) {
        const inclusive = base + mealUpliftFor(plan, base)
        if (gstRatePercentFor(inclusive) === GST_PERCENT_LUXURY) {
          luxury.push(`${prop.slug}/${room.slug}/${plan}`)
          // Never 18% on a supply that is actually inside the band.
          expect(inclusive / (1 + GST_PERCENT_STANDARD / 100))
            .toBeGreaterThan(GST_LUXURY_THRESHOLD_PER_ROOM_NIGHT)
        }
      }
    }

    expect(luxury).toEqual(['hotel-amaltas-international/superior-room/All Meals Included'])
  })

  it('puts the Amaltas Superior in the 5% slab on Room Only and Breakfast', () => {
    // Only the MAP quote crosses. ₹6,000 alone, and ₹7,500 with breakfast, are
    // both inside the band — the second by the same ₹7,142.86 margin as the
    // Royal Suite above, so the two sit on the threshold from opposite sides.
    expect(gstRatePercentFor(6000)).toBe(5)
    expect(gstRatePercentFor(7500)).toBe(5)
    expect(gstRatePercentFor(9000)).toBe(18)
  })

  it('splits a total inclusive of 5% into base + tax that add back up', () => {
    const total = 6000
    const rate = gstRatePercentFor(total)
    const base = Math.round((total / (1 + rate / 100)) * 100) / 100
    const tax = Math.round((total - base) * 100) / 100
    expect(rate).toBe(5)
    expect(base).toBeCloseTo(5714.29, 2)
    expect(tax).toBeCloseTo(285.71, 2)
    expect(Math.round((base + tax) * 100) / 100).toBe(total)
  })
})

describe('Meal plans — percentage of the base room rate (client, 5 Aug 2026)', () => {
  it('is EP 0 / CP 25 / MAP 50', () => {
    expect(MEAL_PLAN_UPLIFT_PERCENT['Room Only']).toBe(0)
    expect(MEAL_PLAN_UPLIFT_PERCENT['With Breakfast']).toBe(25)
    expect(MEAL_PLAN_UPLIFT_PERCENT['All Meals Included']).toBe(50)
  })

  it('measures the percentage off base price + category offset, not base price alone', () => {
    // Downtown EOK: ₹3,000 base, Super Deluxe +₹1,000 => ₹4,000.
    expect(baseRoomRateFor(3000, 1000)).toBe(4000)
    expect(mealUpliftFor('With Breakfast', 4000)).toBe(1000)
    expect(mealUpliftFor('All Meals Included', 4000)).toBe(2000)

    // If it were measured off the ₹3,000 base alone, CP on a Super Deluxe would
    // cost the same as CP on a Deluxe. It must not.
    expect(mealUpliftFor('With Breakfast', baseRoomRateFor(3000, 0)))
      .not.toBe(mealUpliftFor('With Breakfast', baseRoomRateFor(3000, 1000)))
  })

  it('makes CP exactly 1.25x and MAP exactly 1.5x the nightly rate, at every hotel', () => {
    for (const room of seedRoomTypes) {
      const prop = seedProperties.find((p) => p.id === room.property_id)!
      const base = baseRoomRateFor(prop.base_price, room.price_offset)

      const nightly = (plan: MealPlan) => base + mealUpliftFor(plan, base)

      expect(nightly('Room Only')).toBe(base)
      expect(nightly('With Breakfast')).toBe(base * 1.25)
      expect(nightly('All Meals Included')).toBe(base * 1.5)
    }
  })

  it('applies the same percentage at every property', () => {
    // The real invariant is that no two properties share a slug — `room_types.id`
    // is derived from it and a collision would cross two hotels' inventory. This
    // used to read `toBe(9)`, which asserted the size of the group instead and
    // failed the day a tenth hotel was added.
    const slugs = new Set(seedProperties.map((p) => p.slug))
    expect(slugs.size).toBe(seedProperties.length)

    for (const room of seedRoomTypes) {
      const prop = seedProperties.find((p) => p.id === room.property_id)!
      const base = baseRoomRateFor(prop.base_price, room.price_offset)
      expect((room.breakfast_offset / base) * 100).toBe(25)
      expect((room.all_meals_offset / base) * 100).toBe(50)
    }
  })

  it('resolves the same figure from a joined row, an explicit base rate and the seeded columns', () => {
    for (const room of seedRoomTypes) {
      const prop = seedProperties.find((p) => p.id === room.property_id)!
      const base = baseRoomRateFor(prop.base_price, room.price_offset)

      for (const plan of PLANS) {
        const fromJoin = mealOffsetFor(plan, { ...room, base_price: prop.base_price })
        const fromExplicit = mealOffsetFor(plan, room, base)
        const fromColumns = mealOffsetFor(plan, room) // in-memory: no base_price
        expect(fromJoin).toBe(fromExplicit)
        expect(fromColumns).toBe(fromExplicit)
      }
    }
  })
})

describe('Meal uplift composes with the other percentages without double-counting', () => {
  const base = 4000 // EOK Super Deluxe
  const stay = {
    basePrice: 3000,
    roomOffset: 1000,
    weekendSurchargePercent: 20,
    checkIn: '2026-09-07', // Mon
    checkOut: '2026-09-08',
    roomsCount: 1,
  }

  it('applies the meal percentage once, not to itself', () => {
    const total = computeStayBreakdown({
      ...stay,
      mealOffset: mealUpliftFor('All Meals Included', base),
    }).total
    expect(total).toBe(6000) // 4000 * 1.5, not 4000 * 1.5 * 1.5
  })

  it('lets the weekend surcharge run on the meal-inclusive rate', () => {
    const friday = computeStayBreakdown({
      ...stay,
      checkIn: '2026-09-11', // Fri
      checkOut: '2026-09-12',
      mealOffset: mealUpliftFor('All Meals Included', base),
    }).total
    expect(friday).toBe(7200) // 6000 * 1.2
  })

  it('charges the extra adult on the meal-inclusive rate, once per night per head', () => {
    const b = computeStayBreakdown({
      ...stay,
      mealOffset: mealUpliftFor('All Meals Included', base),
      extraAdults: 1,
      extraAdultPercent: 30,
    })
    expect(b.roomTotal).toBe(6000)
    expect(b.extraAdultTotal).toBe(1800) // 30% of 6000
    expect(b.total).toBe(7800)
  })

  it('does not multiply the extra bed by the room count', () => {
    const b = computeStayBreakdown({
      ...stay,
      roomsCount: 3,
      mealOffset: mealUpliftFor('With Breakfast', base),
      extraAdults: 1,
      extraAdultPercent: 30,
    })
    expect(b.roomTotal).toBe(15000) // 5000 * 3
    expect(b.extraAdultTotal).toBe(1500) // one extra bed, not three
  })

  it('charges nothing extra for Room Only', () => {
    expect(computeStayBreakdown({ ...stay, mealOffset: mealUpliftFor('Room Only', base) }).total)
      .toBe(4000)
  })
})
