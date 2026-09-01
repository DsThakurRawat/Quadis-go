import request from 'supertest'
import { createApp } from '../src/app'
import { db } from '../src/db'
import { seedProperties, seedRoomTypes } from '../src/data/seed'

/**
 * Read inventory off the seed rather than hardcoding it. These assertions are
 * about holds decrementing availability, not about how many keys a property
 * has — hardcoding the count meant every correction to the client's rate sheet
 * broke tests that had nothing to do with rates.
 */
const keysOf = (slug: string): number =>
  seedRoomTypes.find((r) => r.id === `room-prop-2-${slug}`)!.total_units

const app = createApp()

describe('Phase 1: Core API & Reservation Soft Hold Tests', () => {
  beforeEach(() => {
    // Reset in-memory database to initial state before each test
    db.initializeInMemorySeed()
  })

  test('GET /api/properties returns every seeded active property', async () => {
    // Counted off the seed for the same reason keysOf() is: this test is about
    // the endpoint returning the active set in seed order, not about how many
    // hotels the group happens to run. It was pinned at 9 and broke the day a
    // tenth property was added, which told us nothing about the endpoint.
    const activeCount = seedProperties.filter((p) => p.is_active).length
    const res = await request(app).get('/api/properties')
    expect(res.status).toBe(200)
    expect(res.body.success).toBe(true)
    expect(res.body.count).toBe(activeCount)
    expect(res.body.data[0].slug).toBe('hotel-quadis-sector-51-noida')
  })

  test('GET /api/properties/:slug returns property and room types', async () => {
    const res = await request(app).get('/api/properties/hotel-quadis-sector-51-noida')
    expect(res.status).toBe(200)
    expect(res.body.success).toBe(true)
    expect(res.body.data.name).toBe('Hotel Quadis Sector 51')
    expect(Array.isArray(res.body.data.rooms)).toBe(true)
    // Deluxe and Super. Per the client's rate sheet only Downtown EOK and Amar
    // Inn carry a third (Royal) category.
    expect(res.body.data.rooms.length).toBe(2)
  })

  test('POST /api/bookings/initiate creates a 15-minute soft hold and decrements available units', async () => {
    const payload = {
      propertySlug: 'hotel-quadis-sector-51-noida',
      roomTypeSlug: 'deluxe-room',
      checkIn: '2026-11-12',
      checkOut: '2026-11-14',
      roomsCount: 2,
      guestsCount: 4,
      guestName: 'Rajesh Kumar',
      guestPhone: '9876543210',
      guestEmail: 'rajesh@example.com',
    }

    const res = await request(app).post('/api/bookings/initiate').send(payload)
    expect(res.status).toBe(201)
    expect(res.body.success).toBe(true)
    // Codes are `QD-` + 8 Crockford base32 symbols (I/L/O/U excluded), not the
    // old 4-digit suffix, which both collided and could be enumerated.
    expect(res.body.data.booking_code).toMatch(/^QD-[0-9A-HJKMNP-TV-Z]{8}$/)
    expect(res.body.data.booking_status).toBe('PENDING_PAYMENT')
    expect(res.body.data.rooms_count).toBe(2)

    // Inventory is per night: booking 2 rooms drops these dates by 2 ...
    const deluxeKeys = keysOf('deluxe-room')
    const roomId = res.body.data.room_type_id
    expect(await db.getAvailableUnits(roomId, '2026-11-12', '2026-11-14')).toBe(deluxeKeys - 2)

    // ... while unrelated dates are untouched. A single counter used to let one
    // booking block every other date in the calendar.
    expect(await db.getAvailableUnits(roomId, '2027-03-10', '2027-03-12')).toBe(deluxeKeys)
  })

  test('a full room type on one set of dates is still sellable on other dates', async () => {
    const base = {
      propertySlug: 'hotel-quadis-sector-51-noida',
      roomTypeSlug: 'super-deluxe', // 3 keys at this property
      guestsCount: 2,
      guestName: 'Date Isolation',
      guestPhone: '9876543210',
      roomsCount: 2,
    }

    // Take 2 of the 3 for 20–22 Nov, leaving too few for another pair.
    const soldOut = await request(app)
      .post('/api/bookings/initiate')
      .send({ ...base, checkIn: '2026-11-20', checkOut: '2026-11-22' })
    expect(soldOut.status).toBe(201)

    // Same dates again must fail...
    const clash = await request(app)
      .post('/api/bookings/initiate')
      .send({ ...base, checkIn: '2026-11-20', checkOut: '2026-11-22' })
    expect(clash.status).toBe(400)

    // ...but a stay that starts the morning the others check out must succeed.
    // Checkout day is not a night, so 22nd onward is free.
    const adjacent = await request(app)
      .post('/api/bookings/initiate')
      .send({ ...base, checkIn: '2026-11-22', checkOut: '2026-11-24' })
    expect(adjacent.status).toBe(201)

    // And an unrelated month is obviously unaffected.
    const distant = await request(app)
      .post('/api/bookings/initiate')
      .send({ ...base, checkIn: '2027-05-01', checkOut: '2027-05-03' })
    expect(distant.status).toBe(201)
  })

  test('POST /api/bookings/initiate prevents double booking when room units are sold out', async () => {
    const payload = {
      propertySlug: 'hotel-quadis-sector-51-noida',
      roomTypeSlug: 'super-deluxe',
      checkIn: '2026-11-12',
      checkOut: '2026-11-14',
      roomsCount: 4, // Super Deluxe has 3 keys here, so asking for 4 must fail
      guestsCount: 4,
      guestName: 'Ananya Sharma',
      guestPhone: '9123456780',
    }

    const res = await request(app).post('/api/bookings/initiate').send(payload)
    expect(res.status).toBe(400)
    expect(res.body.success).toBe(false)
    expect(res.body.error).toContain('Only 3 units available')
  })

  test('GET /api/bookings/:code retrieves booking correctly', async () => {
    const payload = {
      propertySlug: 'hotel-quadis-sector-51-noida',
      roomTypeSlug: 'super-deluxe',
      checkIn: '2026-12-01',
      checkOut: '2026-12-03',
      roomsCount: 1,
      guestsCount: 2,
      guestName: 'Vikram Singh',
      guestPhone: '9988776655',
    }

    const initRes = await request(app).post('/api/bookings/initiate').send(payload)
    const code = initRes.body.data.booking_code

    const lookupRes = await request(app).get(`/api/bookings/${code}?phone=9988776655`)
    expect(lookupRes.status).toBe(200)
    expect(lookupRes.body.success).toBe(true)
    expect(lookupRes.body.data.guest_name).toBe('Vikram Singh')
  })

  test('POST /api/bookings/initiate rejects when check-out date is equal to or before check-in date', async () => {
    const payload = {
      propertySlug: 'hotel-quadis-sector-51-noida',
      roomTypeSlug: 'deluxe-room',
      checkIn: '2026-11-14',
      checkOut: '2026-11-12', // Check-out before check-in
      roomsCount: 1,
      guestsCount: 2,
      guestName: 'Invalid Date Tester',
      guestPhone: '9876543210',
    }

    const res = await request(app).post('/api/bookings/initiate').send(payload)
    expect(res.status).toBe(400)
    expect(res.body.success).toBe(false)
  })

  test('POST /api/bookings/initiate rejects when check-in date is in the past', async () => {
    const payload = {
      propertySlug: 'hotel-quadis-sector-51-noida',
      roomTypeSlug: 'deluxe-room',
      checkIn: '2020-01-01',
      checkOut: '2020-01-03',
      roomsCount: 1,
      guestsCount: 2,
      guestName: 'Past Date Tester',
      guestPhone: '9876543210',
    }

    const res = await request(app).post('/api/bookings/initiate').send(payload)
    expect(res.status).toBe(400)
    expect(res.body.success).toBe(false)
  })
})
