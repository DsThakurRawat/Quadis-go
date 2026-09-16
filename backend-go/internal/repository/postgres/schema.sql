-- Quadis Hotels PostgreSQL Database Schema
-- Phase 1: Core API & Database Foundation

CREATE TABLE IF NOT EXISTS properties (
  id VARCHAR(64) PRIMARY KEY,
  slug VARCHAR(64) UNIQUE NOT NULL,
  name VARCHAR(128) NOT NULL,
  city VARCHAR(32) NOT NULL,
  address TEXT NOT NULL,
  -- Google Maps share link. Present in the seed data, on PropertyRecord and in
  -- the admin-editable whitelist since day one, but the column was never
  -- created — so against real Postgres (never against the in-memory store) a
  -- property edit carrying map_link failed with "column does not exist".
  map_link TEXT,
  phone VARCHAR(20) NOT NULL,
  whatsapp VARCHAR(20) NOT NULL,
  email VARCHAR(128) NOT NULL,
  base_price NUMERIC(10, 2) NOT NULL,
  rating NUMERIC(3, 2) DEFAULT 4.50,
  is_active BOOLEAN DEFAULT TRUE,
  weekend_surcharge_percent NUMERIC(5, 2) DEFAULT 0.00,
  -- Occupancy policy, per property and set from the admin panel.
  --
  -- Rates are quoted for two adults per room. A third ADULT adds
  -- extra_adult_percent of that night's room rate. Client, 27 Jul 2026: "If a
  -- third person is included in the same room, a 30% extra charge will apply
  -- for the additional mattress."
  --
  -- Children are NOT free at any age. That was her first answer and it was
  -- superseded on 27 Jul by the three bands below; the defaults on the next
  -- three columns are the live rule (AGENTS.md §2 rule 3).
  extra_adult_percent NUMERIC(5, 2) NOT NULL DEFAULT 30.00,
  -- Three age bands, per the client 27 Jul 2026: 0-7 free, 8-12 at
  -- child_percent, adult_from_age and above charged as an adult.
  child_free_under_age INTEGER NOT NULL DEFAULT 8,
  child_percent NUMERIC(5, 2) NOT NULL DEFAULT 20.00,
  adult_from_age INTEGER NOT NULL DEFAULT 13,
  -- Null until a real coordinate is confirmed for the property. The UI falls
  -- back to an address search rather than showing an invented pin.
  lat NUMERIC(10, 7),
  lng NUMERIC(10, 7),
  place_id VARCHAR(128),
  tier VARCHAR(32) DEFAULT 'central',
  tier_label VARCHAR(64) DEFAULT 'Quadis Central'
);

CREATE TABLE IF NOT EXISTS room_types (
  id VARCHAR(64) PRIMARY KEY,
  property_id VARCHAR(64) REFERENCES properties(id) ON DELETE CASCADE,
  slug VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  description TEXT,
  size_sqft VARCHAR(32),
  bed_type VARCHAR(64),
  max_guests INTEGER NOT NULL DEFAULT 2,
  price_offset NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
  breakfast_offset NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
  all_meals_offset NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
  total_units INTEGER NOT NULL DEFAULT 5,
  available_units INTEGER NOT NULL DEFAULT 5,
  is_available BOOLEAN DEFAULT TRUE,
  CONSTRAINT check_available_positive CHECK (available_units >= 0 AND available_units <= total_units)
);

-- Guest accounts. password_hash holds a salted scrypt digest, never plaintext.
-- Photography uploaded from the admin panel.
--
-- Photos used to be resolved by a build-time glob over public/images, which
-- meant changing one required a developer, a rebuild and a redeploy. Rows here
-- take precedence over that glob, so an upload is live without shipping code.
--
-- The file itself lives in object storage; only its URL is kept. Deleting a
-- property takes its photo rows with it, but not the stored objects - those are
-- removed explicitly, so a mis-click cannot orphan the originals.
CREATE TABLE IF NOT EXISTS property_images (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  property_id VARCHAR(64) NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
  -- Full-size (long edge 1600px) and a 400px thumbnail for grids.
  url TEXT NOT NULL,
  thumb_url TEXT,
  -- Storage key, so a delete can remove the object and not just the row.
  storage_key TEXT NOT NULL,
  alt_text VARCHAR(255),
  -- Lowest sorts first; the first image is the property's hero.
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_property_images_property
  ON property_images (property_id, sort_order);

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  full_name VARCHAR(128) NOT NULL,
  email VARCHAR(190) UNIQUE NOT NULL,
  phone VARCHAR(20),
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- The staff PIN for /admin, so the client can change it herself.
--
-- It lives here and NOT in site_content, which is the obvious-looking home for
-- a single key/value: GET /api/content is public and unauthenticated (verified
-- 200 on production), so a hash parked there would be published to the world.
-- PUT /api/admin/content also writes arbitrary keys, so it could be overwritten
-- by anyone already inside the panel.
--
-- One row, id 'primary'. Multiple staff logins would be a separate table with
-- names against them — the client has been asked how many people use the panel
-- and has not answered, so this deliberately does not guess.
CREATE TABLE IF NOT EXISTS admin_credentials (
  id VARCHAR(32) PRIMARY KEY DEFAULT 'primary',
  pin_hash TEXT NOT NULL,
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS bookings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  booking_code VARCHAR(16) UNIQUE NOT NULL,
  -- Null for guest checkout; set when a signed-in guest books.
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  property_id VARCHAR(64) REFERENCES properties(id),
  room_type_id VARCHAR(64) REFERENCES room_types(id),
  guest_name VARCHAR(128) NOT NULL,
  guest_phone VARCHAR(20) NOT NULL,
  guest_email VARCHAR(128),
  company_name VARCHAR(128),
  gstin VARCHAR(32),
  check_in DATE NOT NULL,
  check_out DATE NOT NULL,
  rooms_count INTEGER NOT NULL DEFAULT 1,
  -- guests_count stays as the headcount total (adults + children) so older rows
  -- and the owner WhatsApp alert keep working. The split below is what prices
  -- the stay: adults beyond two per room pay the extra-bed charge.
  guests_count INTEGER NOT NULL DEFAULT 2,
  adults_count INTEGER NOT NULL DEFAULT 2,
  children_count INTEGER NOT NULL DEFAULT 0,
  -- One age per child, so "under 12 stays free" can be recomputed on any row
  -- rather than trusting a number the client sent.
  child_ages JSONB NOT NULL DEFAULT '[]'::jsonb,
  extra_adults INTEGER NOT NULL DEFAULT 0,
  total_amount NUMERIC(10, 2) NOT NULL,
  payment_mode VARCHAR(32) NOT NULL DEFAULT 'INSTANT_FULL_PAYMENT',
  payment_status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
  razorpay_order_id VARCHAR(64),
  razorpay_payment_id VARCHAR(64),
  razorpay_payment_link_id VARCHAR(64),
  booking_status VARCHAR(32) NOT NULL DEFAULT 'PENDING_PAYMENT',
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- A payment may only ever settle one booking. Razorpay retries webhooks on
-- timeout, so without this a replayed payment.captured can confirm twice.
CREATE UNIQUE INDEX IF NOT EXISTS bookings_razorpay_payment_id_key
  ON bookings (razorpay_payment_id) WHERE razorpay_payment_id IS NOT NULL;

-- Availability is per night, not a single counter per room type.
--
-- One row per room type, per night, per booking. A stay from the 1st to the 3rd
-- occupies the nights of the 1st and 2nd — the checkout date is not a night.
-- Without this, a booking in December made rooms unbookable in March.
CREATE TABLE IF NOT EXISTS room_night_holds (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  room_type_id VARCHAR(64) NOT NULL REFERENCES room_types(id) ON DELETE CASCADE,
  booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  stay_date DATE NOT NULL,
  units INTEGER NOT NULL CHECK (units > 0)
);

CREATE INDEX IF NOT EXISTS room_night_holds_lookup
  ON room_night_holds (room_type_id, stay_date);
CREATE UNIQUE INDEX IF NOT EXISTS room_night_holds_unique
  ON room_night_holds (booking_id, stay_date);

CREATE INDEX IF NOT EXISTS idx_bookings_status_created ON bookings(booking_status, created_at);
CREATE INDEX IF NOT EXISTS idx_bookings_code ON bookings(booking_code);

CREATE TABLE IF NOT EXISTS enquiries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  enquiry_type VARCHAR(32) NOT NULL,
  property_id VARCHAR(64) REFERENCES properties(id),
  guest_name VARCHAR(128) NOT NULL,
  guest_phone VARCHAR(20) NOT NULL,
  guest_email VARCHAR(128),
  event_date DATE,
  guest_count INTEGER,
  message TEXT,
  status VARCHAR(32) NOT NULL DEFAULT 'NEW',
  razorpay_payment_link_id VARCHAR(64),
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_enquiries_status_created ON enquiries(status, created_at);

-- Editable marketing copy.
--
-- Page text used to live only in the JSX, so changing a headline meant a code
-- change and a redeploy. Components read a key from here and fall back to the
-- string they shipped with, so an empty table renders exactly today's site and
-- the admin can override any block without a deploy.
CREATE TABLE IF NOT EXISTS site_content (
  key VARCHAR(128) PRIMARY KEY,
  value TEXT NOT NULL,
  -- Free-text grouping for the admin UI ("home", "about", "footer"…).
  section VARCHAR(64) NOT NULL DEFAULT 'general',
  label VARCHAR(190) NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_site_content_section ON site_content(section);

CREATE TABLE IF NOT EXISTS chat_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id VARCHAR(64) NOT NULL,
  user_message TEXT NOT NULL,
  bot_response TEXT NOT NULL,
  tools_invoked JSONB,
  handoff_triggered BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_logs_session ON chat_logs(session_id);

-- ---------------------------------------------------------------------------
-- Migrations for databases created before a column existed.
--
-- Every CREATE TABLE above is IF NOT EXISTS, which means an existing database
-- silently keeps its old shape. These run on every boot and are no-ops once
-- applied, so a redeploy onto a live database picks the new columns up.
-- ---------------------------------------------------------------------------

ALTER TABLE bookings ADD COLUMN IF NOT EXISTS adults_count INTEGER NOT NULL DEFAULT 2;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS children_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS child_ages JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS extra_adults INTEGER NOT NULL DEFAULT 0;

-- The uplift applied to each extra bed, frozen onto the booking at the time it
-- was made: the percentage in force, and the rupees per extra adult per night it
-- worked out to. Without these, an admin repricing the property later would
-- change what an already-issued invoice appears to say.
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS extra_adult_percent NUMERIC(5, 2) NOT NULL DEFAULT 0;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS extra_adult_charge NUMERIC(10, 2) NOT NULL DEFAULT 0;

ALTER TABLE properties ADD COLUMN IF NOT EXISTS map_link TEXT;

ALTER TABLE properties ADD COLUMN IF NOT EXISTS extra_adult_percent NUMERIC(5, 2) NOT NULL DEFAULT 30.00;
ALTER TABLE properties ADD COLUMN IF NOT EXISTS child_free_under_age INTEGER NOT NULL DEFAULT 8;
ALTER TABLE properties ADD COLUMN IF NOT EXISTS child_percent NUMERIC(5, 2) NOT NULL DEFAULT 20.00;
ALTER TABLE properties ADD COLUMN IF NOT EXISTS adult_from_age INTEGER NOT NULL DEFAULT 13;

-- Databases seeded before 27 Jul carry the old single-threshold policy, where
-- every under-18 was free. Move them onto the three-band rule the client gave.
-- Scoped to the old default so a hotel that has since set its own is untouched.
UPDATE properties SET child_free_under_age = 8 WHERE child_free_under_age = 18;

-- Databases created against the earlier flat-rupee model carry a redundant
-- properties.extra_adult_charge column. Harmless if present; dropped so the
-- schema has exactly one definition of the policy.
ALTER TABLE properties DROP COLUMN IF EXISTS extra_adult_charge;

-- The client's listing audit (5 Aug 2026) restated the Google rating on eight
-- of the nine properties, two of them by more than half a star, and found
-- Cladis 15 pointing at the wrong GMB listing entirely.
--
-- seedProperties carries the corrected figures, but seedPostgres is
-- INSERT ... ON CONFLICT (id) DO NOTHING, so it cannot touch a row that already
-- exists: without these statements every live database would go on quoting the
-- old numbers no matter how many times we redeploy.
--
-- Each one is scoped to the value it replaces, following the
-- child_free_under_age correction above. That makes them idempotent, and it
-- means a rating the client has since set herself from the dashboard is left
-- alone rather than being reverted on the next boot.
UPDATE properties SET rating = 4.5 WHERE slug = 'hotel-quadis-sector-51-noida'   AND rating = 4.6;
UPDATE properties SET rating = 4.0 WHERE slug = 'hotel-downtown-sector-15-noida' AND rating = 4.4;
UPDATE properties SET rating = 3.8 WHERE slug = 'hotel-cladis-sector-15-noida'   AND rating = 4.4;
UPDATE properties SET rating = 4.5 WHERE slug = 'hotel-cladis-sector-19-noida'   AND rating = 4.3;
UPDATE properties SET rating = 4.4 WHERE slug = 'hotel-downtown-sector-51-noida' AND rating = 4.5;
UPDATE properties SET rating = 4.5 WHERE slug = 'hotel-downtown-east-of-kailash' AND rating = 4.6;
UPDATE properties SET rating = 3.8 WHERE slug = 'hotel-amby-inn-lajpat-nagar-ii' AND rating = 4.5;
UPDATE properties SET rating = 4.3 WHERE slug = 'hotel-amar-inn'                 AND rating = 4.4;

UPDATE properties
   SET map_link = 'https://share.google/1Gbjxirb5YQWy6h6D'
 WHERE slug = 'hotel-cladis-sector-15-noida'
   AND map_link = 'https://share.google/nHWsuom2pwTNGRgfY';

-- Meal plans moved from flat rupees to a percentage of the base room rate on
-- the client's instruction, 5 Aug 2026: "EP: No additional charge / CP
-- (Breakfast): +25% / MAP (All Meals Included): +50% ... applied automatically
-- across all hotels based on the base room rate."
--
-- The percentages themselves are NOT stored. They live in
-- backend/src/lib/pricing.ts (mirrored in src/lib/pricing.ts) and are applied
-- at quote time, which is what makes them track a base_price the admin changes
-- later. These two columns stay as the rupee equivalent, recomputed here, for
-- the two consumers that still read them directly: the concierge's "breakfast
-- adds ₹x" answer, and the in-memory store's room rows, which carry no property
-- price for the pricing library to work from.
--
-- The base room rate is the property's base_price plus this category's
-- price_offset — the same definition baseRoomRateFor() uses — so Downtown EOK's
-- Super Deluxe is measured against 3,000 + 1,000 = ₹4,000 and gets ₹1,000 of
-- breakfast, not ₹350.
--
-- Deliberately NOT scoped to the old seeded figures, which is where the
-- corrections above and this one part company.
--
-- Those scope themselves to the value they replace so that a rating the client
-- has since set herself is not reverted on the next boot. That is right when the
-- column is authoritative. This column is not: after 5 Aug 2026 the percentage
-- decides what the guest is charged, so a hand-edited supplement of ₹500 does
-- not buy anyone a ₹500 breakfast — it just leaves a number in the database that
-- disagrees with the ₹750 actually billed, and the concierge reads this column
-- when it answers "what does breakfast cost". Recomputing every row on every
-- boot is what makes that disagreement impossible.
--
-- Still idempotent: the second run computes the same figures as the first.
--
-- Bookings are untouched. bookings.total_amount is frozen at the moment the
-- hold is taken, so nothing already sold is repriced by this — only what a new
-- quote costs from here on.
UPDATE room_types rt
   SET breakfast_offset = ROUND((p.base_price + rt.price_offset) * 0.25),
       all_meals_offset = ROUND((p.base_price + rt.price_offset) * 0.50)
  FROM properties p
 WHERE p.id = rt.property_id
   AND (rt.breakfast_offset, rt.all_meals_offset)
       IS DISTINCT FROM (ROUND((p.base_price + rt.price_offset) * 0.25),
                         ROUND((p.base_price + rt.price_offset) * 0.50));

-- ---------------------------------------------------------------------------
-- ResAvenue channel manager (backend/src/routes/ota.ts)
--
-- ResAvenue treats this site as an OTA: it pushes inventory, restrictions and
-- rates per night, and takes reservations back. Those pushes land in the two
-- tables below, which the booking engine reads ahead of total_units and the
-- computed nightly rate. A room or night with no row here behaves exactly as
-- it did before the integration existed.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS room_inventory_days (
  room_type_id VARCHAR(64) NOT NULL REFERENCES room_types(id) ON DELETE CASCADE,
  stay_date DATE NOT NULL,
  -- Sellable units that night. NULL keeps room_types.total_units as the ceiling.
  inv_count INTEGER CHECK (inv_count IS NULL OR inv_count >= 0),
  stop_sell BOOLEAN NOT NULL DEFAULT FALSE,
  close_on_arrival BOOLEAN NOT NULL DEFAULT FALSE,
  close_on_departure BOOLEAN NOT NULL DEFAULT FALSE,
  -- Fewest days before arrival a booking may be made. 0 = no limit.
  cut_off INTEGER NOT NULL DEFAULT 0 CHECK (cut_off >= 0),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (room_type_id, stay_date)
);

CREATE TABLE IF NOT EXISTS room_rate_days (
  room_type_id VARCHAR(64) NOT NULL REFERENCES room_types(id) ON DELETE CASCADE,
  -- 'Room Only' | 'With Breakfast' | 'All Meals Included' — the three plans the
  -- site sells, each a rate plan to ResAvenue (see lib/channel.ts).
  meal_plan VARCHAR(32) NOT NULL,
  stay_date DATE NOT NULL,
  -- Every amount nullable: an update carrying only NumberOfGuests 1 and 2
  -- leaves triple and quad on the site's computed figure.
  single NUMERIC(10, 2),
  double NUMERIC(10, 2),
  triple NUMERIC(10, 2),
  quad NUMERIC(10, 2),
  extra_adult NUMERIC(10, 2),
  extra_child NUMERIC(10, 2),
  min_stay INTEGER CHECK (min_stay IS NULL OR min_stay >= 0),
  max_stay INTEGER CHECK (max_stay IS NULL OR max_stay >= 0),
  stop_sell BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (room_type_id, meal_plan, stay_date)
);

-- The plan the guest chose. It priced the hold from the start but was never
-- kept, and the reservation sent to ResAvenue has to name a RatePlanCode.
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS meal_plan VARCHAR(32);

-- "What changed since" for the booking pull. created_at cannot answer it: a
-- cancellation changes a row weeks after it was created.
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Delivery state of the Confirm / Cancel owed to the channel manager. See
-- BookingRecord.cm_sync_status in types.ts for the four values.
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS cm_sync_status VARCHAR(16) NOT NULL DEFAULT 'NONE';
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS cm_res_status VARCHAR(10);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS cm_synced_at TIMESTAMPTZ;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS cm_attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS cm_last_error TEXT;

CREATE INDEX IF NOT EXISTS idx_bookings_cm_sync ON bookings (cm_sync_status) WHERE cm_sync_status IN ('PENDING', 'FAILED');
CREATE INDEX IF NOT EXISTS idx_bookings_property_updated ON bookings (property_id, updated_at);
