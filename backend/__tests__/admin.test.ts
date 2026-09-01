import request from 'supertest'
import { createApp } from '../src/app'
import { db } from '../src/db'
import { seedProperties } from '../src/data/seed'

describe('Phase 3: Owner Switchboard & WhatsApp Quick-Commands Suite', () => {
  const app = createApp()
  const ADMIN_PIN = process.env.ADMIN_PIN || '998877'
  let authToken: string
  let targetRoomId: string
  let targetRoomSlug: string
  let targetPropertySlug: string

  beforeAll(async () => {
    db.useInMemory = true
    db.initializeInMemorySeed()
    const props = await db.getPropertiesWithRooms()
    const firstProp = props[0]
    targetPropertySlug = firstProp.property.slug
    targetRoomId = firstProp.rooms[0].id
    targetRoomSlug = firstProp.rooms[0].slug
  })

  it('POST /api/admin/auth validates PIN code and issues management token', async () => {
    const resFail = await request(app).post('/api/admin/auth').send({ pin: '000000' })
    expect(resFail.status).toBe(401)
    expect(resFail.body.success).toBe(false)

    const resOk = await request(app).post('/api/admin/auth').send({ pin: ADMIN_PIN })
    expect(resOk.status).toBe(200)
    expect(resOk.body.success).toBe(true)
    expect(resOk.body.token).toBeDefined()
    authToken = resOk.body.token
  })

  it('GET /api/admin/dashboard returns daily glance summary, inventory, and leads', async () => {
    const res = await request(app).get('/api/admin/dashboard').set('Authorization', authToken)
    expect(res.status).toBe(200)
    expect(res.body.success).toBe(true)
    expect(res.body.data.metrics).toBeDefined()
    expect(typeof res.body.data.metrics.todayCheckIns).toBe('number')
    expect(typeof res.body.data.metrics.pendingHolds).toBe('number')
    expect(typeof res.body.data.metrics.pendingEnquiries).toBe('number')
    expect(typeof res.body.data.metrics.todayRevenue).toBe('number')
    expect(Array.isArray(res.body.data.properties)).toBe(true)
    // Off the seed, not a literal — the dashboard's contract is that it
    // lists every active property, whatever that number is this month.
    expect(res.body.data.properties.length).toBe(seedProperties.filter((p) => p.is_active).length)
  })

  it('PATCH /api/admin/room-availability toggles inventory availability ([ Sold Out ] vs [ Available ])', async () => {
    // Mark sold out
    const resSold = await request(app)
      .patch('/api/admin/room-availability')
      .set('Authorization', authToken)
      .send({ roomTypeId: targetRoomId, isAvailable: false })

    expect(resSold.status).toBe(200)
    expect(resSold.body.success).toBe(true)
    expect(resSold.body.data.is_available).toBe(false)

    // Verify in DB
    const roomAfter = await db.getRoomTypeById(targetRoomId)
    expect(roomAfter?.is_available).toBe(false)

    // Mark back available
    const resAvail = await request(app)
      .patch('/api/admin/room-availability')
      .set('Authorization', authToken)
      .send({ roomTypeId: targetRoomId, isAvailable: true })

    expect(resAvail.status).toBe(200)
    expect(resAvail.body.data.is_available).toBe(true)
  })

  it('PATCH /api/admin/surcharge applies +15% weekend/seasonal surcharge globally', async () => {
    const res = await request(app)
      .patch('/api/admin/surcharge')
      .set('Authorization', authToken)
      .send({ surchargePercent: 15, propertyId: 'all' })

    expect(res.status).toBe(200)
    expect(res.body.success).toBe(true)
    expect(res.body.data[0].property.weekend_surcharge_percent).toBe(15)

    // Revert back to 0
    await request(app)
      .patch('/api/admin/surcharge')
      .set('Authorization', authToken)
      .send({ surchargePercent: 0, propertyId: 'all' })
  })

  it('POST /api/admin/payment-link generates instant deposit link for walk-ins/inquiries', async () => {
    const payload = {
      phone: '9811223344',
      amount: 12000,
      guestName: 'Rohit Sharma (VIP Walk-in)',
      description: 'Grand Ballroom Token Deposit',
      propertyName: 'Hotel Quadis Sector 51',
    }

    const res = await request(app)
      .post('/api/admin/payment-link')
      .set('Authorization', authToken)
      .send(payload)

    expect(res.status).toBe(200)
    expect(res.body.success).toBe(true)
    expect(res.body.data.paymentLinkId).toBeDefined()
    expect(res.body.data.shortUrl).toContain('/pay-enquiry/')
    expect(res.body.data.enquiryId).toMatch(/^enq_/)
  })

  it('POST /api/webhooks/whatsapp-staff processes staff text commands (Sold out / Available / Glance)', async () => {
    // 1. Check Glance command
    const resGlance = await request(app)
      .post('/api/webhooks/whatsapp-staff')
      .send({ from: '919217373532', text: 'glance' })

    expect(resGlance.status).toBe(200)
    expect(resGlance.body.success).toBe(true)
    expect(resGlance.body.reply).toContain('Quadis Hotels Daily Glance')

    // 2. Check Sold out command via text
    const resSold = await request(app)
      .post('/api/webhooks/whatsapp-staff')
      .send({ from: '919217373532', text: `Sold out ${targetRoomSlug.replace(/-/g, ' ')}` })

    expect(resSold.status).toBe(200)
    expect(resSold.body.success).toBe(true)
    expect(resSold.body.reply).toContain('as *SOLD OUT*')

    // Verify room is actually sold out
    const roomCheck = await db.getRoomTypeById(targetRoomId)
    expect(roomCheck?.is_available).toBe(false)

    // 3. Check Available command via text
    const resAvail = await request(app)
      .post('/api/webhooks/whatsapp-staff')
      .send({ from: '919217373532', text: `Available ${targetRoomSlug.replace(/-/g, ' ')}` })

    expect(resAvail.status).toBe(200)
    expect(resAvail.body.success).toBe(true)
    expect(resAvail.body.reply).toContain('as *AVAILABLE*')
  })
})
