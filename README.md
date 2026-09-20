# Quadis Hotels — Production Architecture & High-Performance Go Engine

> **Enterprise-grade hospitality booking platform, AI concierge, and OTA channel manager engine powering Quadis Hotels across Delhi NCR.**

---

## 1. System Overview & Architecture

Quadis Hotels operates a hybrid production architecture consisting of:
1. **Frontend SPA**: Vite + React 18 + Vanilla CSS, high-performance static bundle served via Nginx with sub-second mobile rendering.
2. **Backend Engine (`backend-go`)**: High-concurrency Go (Golang) microservice built with the [Chi](https://github.com/go-chi/chi) router, PostgreSQL (`pgxpool`), and concurrent background workers.
3. **Channel Manager OTA Gateway**: Full implementation of ResAvenue's *OTA API Specification v2.0* bridging Quadis inventory, pricing, and reservations across external channels.
4. **Resilient AI Concierge**: Multi-tiered LLM routing (Google Gemini 2.5/2.0 Flash with automatic Groq Llama failover).
5. **Payment Gateway & Comms**: Razorpay idempotent webhook engine + Meta WhatsApp Cloud API automated guest vouchers.

```
                            ┌────────────────────────────────────────────────────────┐
                            │                    GUEST / BROWSER                     │
                            └───────────┬────────────────────────────────┬───────────┘
                                        │ HTTPS                          │ HTTPS
                                        ▼                                ▼
                            ┌───────────────────────┐        ┌───────────────────────┐
                            │  React 18 SPA (dist)  │        │   Nginx Reverse Proxy │
                            └───────────────────────┘        └───────────┬───────────┘
                                                                         │
                                       ┌─────────────────────────────────┴─────────────────────────────────┐
                                       │ HTTP Loopback (Port 3001)                                         │
                                       ▼                                                                   ▼
┌────────────────────────────────────────────────────────────────────────┐       ┌───────────────────────────────────┐
│                     Quadis Go Backend (`backend-go`)                    │       │        ResAvenue Channel Manager  │
│                                                                        │       │             (OTA Extranet)        │
│  ┌───────────────────────┐  ┌────────────────────┐  ┌────────────────┐ │       └─────────────────┬─────────────────┘
│  │ Bookings & Holds      │  │ Dynamic Pricing    │  │ ResAvenue OTA  │◄├─────────────────────────┘
│  │ State Machine Engine  │  │ & GST Calculator   │  │ Channel Gateway│ │  POST /api/ota/* (Fetch, Update, Pull)
│  └──────────┬────────────┘  └─────────┬──────────┘  └────────┬───────┘ │  POST /push (OTA_HotelResNotifRQ)
│             │                         │                      │         │
│  ┌──────────┴────────────┐  ┌─────────┴──────────┐  ┌────────┴───────┐ │
│  │ AI Concierge Service  │  │ Payments & Webhook │  │ WhatsApp Comms │ │
│  │ (Gemini + Groq)       │  │ (Razorpay Gateway) │  │ (Meta Cloud API│ │
│  └───────────────────────┘  └────────────────────┘  └────────────────┘ │
└──────────────────────────────────────┬─────────────────────────────────┘
                                       │
                                       ▼
                     ┌───────────────────────────────────┐
                     │ PostgreSQL Database (Local Socket) │
                     │  - Properties & Room Inventory    │
                     │  - Confirmed & Held Bookings      │
                     │  - Room Inventory Days & Overrides│
                     │  - Room Rate Days & Surcharges    │
                     │  - Audit Logs & Site Content      │
                     └───────────────────────────────────┘
```

---

## 2. Directory Structure

```
Quadis-go/
├── backend-go/                      # High-performance Go backend service
│   ├── cmd/
│   │   ├── server/                  # Server entrypoint (`main.go`)
│   │   └── verify_compat/           # Parity and verification harness
│   ├── internal/
│   │   ├── api/                     # HTTP router, middleware, rate-limiters
│   │   │   ├── handlers/            # REST & OTA HTTP controllers
│   │   │   ├── middleware/          # JWT/Cookie session auth & admin guards
│   │   │   └── router.go            # Chi route registrations
│   │   ├── config/                  # Environment & SSM parameter configuration
│   │   ├── domain/                  # Core entities, state definitions & models
│   │   ├── repository/              # Persistence layer
│   │   │   ├── interfaces.go        # Decoupled storage interfaces
│   │   │   ├── postgres/            # Production pgx PostgreSQL implementation
│   │   │   ├── memory/              # High-speed in-memory store for unit tests
│   │   │   └── seed.go              # Seed data for properties & room types
│   │   └── service/                 # Business logic services
│   │       ├── ai/                  # Gemini & Groq multi-model concierge
│   │       ├── auth/                # scrypt password hashing & session management
│   │       ├── booking/             # Hold engine & state machine
│   │       ├── invoice/             # PDF voucher generation
│   │       ├── ota/                 # ResAvenue OTA API service & sync workers
│   │       ├── payment/             # Razorpay API client & signature verifier
│   │       └── pricing/             # Complex room, occupancy, meal & GST pricing
│   ├── pkg/
│   │   └── dateutil/                # Date manipulation, nights calculations
│   ├── go.mod
│   └── go.sum
├── deploy/                          # Production deployment scripts (Zero Port 22)
│   ├── build-artifact.sh            # Packages production bundle
│   ├── cutover.sh                   # DNS cutover & SSL certification checks
│   ├── install.sh                   # Target EC2 installer and SSM extractor
│   ├── push.sh                      # S3 upload & AWS SSM Session execution
│   └── nginx/                       # Production Nginx site configuration
├── src/                             # React 18 frontend source code
├── public/                          # Optimized WebP assets, logos, manifests
└── render.yaml                      # Cloud deployment specification
```

---

## 3. Key Architectural Decisions

### 3.1 Migration to Go Backend (`backend-go`)
- **Resource Footprint**: The Go microservice runs with an RSS memory footprint of **under 35 MB** (compared to ~180 MB on Node.js/TypeScript), eliminating memory starvation on AWS `t3.medium` instances.
- **True Concurrency**: Concurrent background workers (`StartHoldCleanupWorker` and `StartChannelSyncWorker`) run on dedicated Goroutines with ticker channels, ensuring zero lag on high-traffic booking days.
- **Compile-Time Integrity**: Strict compile-time typing prevents subtle JavaScript runtime errors like null-coercion in booking calculations or unhandled promise rejections during webhook delivery.

### 3.2 Booking Lifecycle & Soft-Hold State Machine
Bookings transition through a strict, deterministic state machine:

```
                  ┌──────────────────────┐
                  │   Guest Initiates    │
                  │   Booking Request    │
                  └──────────┬───────────┘
                             │
                             ▼
                  ┌──────────────────────┐
                  │   PENDING_PAYMENT    │ ◄── 15-Minute Soft Hold Locked
                  └─────┬──────────┬─────┘     Inventory Deducted
         Payment        │          │
        Succeeds        │          │ Expired without payment (Worker triggers)
                        ▼          ▼
             ┌─────────────┐    ┌─────────────┐
             │  CONFIRMED  │    │   EXPIRED   │ ──► Inventory Released
             └──────┬──────┘    └─────────────┘
      Customer/     │
      Desk Cancels  ▼
             ┌─────────────┐
             │  CANCELLED  │ ──► Inventory Released
             └─────────────┘
```

- **Soft Hold Lock**: When a guest initiates checkout, inventory is held for **15 minutes**. Other guests cannot claim the held rooms.
- **Hold Cleanup Goroutine**: Runs periodically (default: every 60s) and transitions stale holds older than 15 minutes to `EXPIRED`, restoring room availability atomically without database deadlocks.
- **Idempotent Payment Confirmation**: When the Razorpay webhook arrives, the transaction checks if the booking is already `CONFIRMED`. If confirmed, it executes as a no-op to prevent duplicate notifications or corrupted room counts.

### 3.3 Dynamic Multi-Tier Pricing Engine
Pricing is computed down to the individual night and adheres to the client's verified occupancy rules:
1. **Base Nightly Formula**:
   $$\text{Base Nightly} = \text{Property Base Price} + \text{Room Category Offset} + \text{Meal Plan Offset}$$
   - **Meal Plan Uplifts**:
     - Room Only (EP): $+0\%$
     - With Breakfast (CP): $+25\%$ of $(\text{Base} + \text{Room Offset})$
     - All Meals Included (MAP): $+50\%$ of $(\text{Base} + \text{Room Offset})$
2. **Weekend Surcharge**: Automatically adds the property's surcharge (typically $10\text{--}25\%$) to Friday and Saturday room nights.
3. **Occupancy & Child Concessions**:
   - Included adults per room: **2**.
   - Extra Adult ($13+$ years): $+30\%$ of room rate per adult per night.
   - Children Under 8: **Free of charge** ($0\%$).
   - Children 8–12: $+20\%$ concession rate per child per night.
4. **GST Threshold Calculation**:
   - Evaluated on the net **Value of Supply** (tax-exclusive nightly room tariff):
     $$\text{Value of Supply} = \frac{\text{Gross Rate Per Room Night}}{1.05}$$
   - **Standard Slab**: $5\%$ if Value of Supply $\le ₹7,500$.
   - **Luxury Slab**: $18\%$ if Value of Supply $> ₹7,500$.
5. **Channel Manager Nightly Overrides**:
   - If ResAvenue has pushed explicit rates for specific dates (`room_rate_days`), the pushed rate supersedes the base formula and weekend surcharges for those exact dates.

---

## 4. ResAvenue Channel Manager (OTA API v2.0)

Quadis acts as an **OTA (Online Travel Agent)** from the perspective of ResAvenue's *OTA API Guide v2.0*. The site exposes full two-way synchronization:

### 4.1 Numerical Code Mapping Scheme
ResAvenue requires short numerical codes ($\le 10$ characters). Codes are **derived deterministically** rather than hardcoded:

| Dimension | Formula / Derivation | Example | Code |
|---|---|---|---|
| **Hotel Code** | Extracted from `prop-{N}` | `prop-7` (Downtown Sec 51) | `7` |
| **Room Code (`InvTypeCode`)** | $\text{Hotel} \times 100 + \text{Category Index}$ | Deluxe Room ($1$) at Hotel $7$ | `701` |
| **Rate Plan Code (`RatePlanCode`)**| $\text{Room} \times 10 + \text{Plan Index}$ | CP Plan ($2$) on Room `701` | `7012` |

**Category Indices**: `deluxe-room: 1`, `super-deluxe: 2`, `superior-room: 3`, `royal-suite: 4`.  
**Plan Indices**: `EP (Room Only): 1`, `CP (With Breakfast): 2`, `MAP (All Meals): 3`.

### 4.2 Endpoint Specifications

Base Path: `https://www.quadishotels.com/api/ota`

#### 1. Property Details (`OTA_HotelDetailsRQ`)
- **Route**: `POST /api/ota/property-details`
- **Purpose**: Returns hotel information, active rooms, capacities, and rate plans.
- **Request Payload**:
  ```json
  {
    "OTA_HotelDetailsRQ": {
      "POS": { "Username": "...", "Password": "...", "ID_Context": "QUADIS" },
      "HotelCode": 7,
      "Target": "Production",
      "Version": "2.0"
    }
  }
  ```
- **Response (`OTA_HotelDetailsRS`)**: Echoes hotel metadata, rooms, base/max occupancy, and 3-year rolling validity rate plans.

#### 2. Inventory Fetch (`OTA_HotelInventoryRQ`)
- **Route**: `POST /api/ota/inventory/fetch`
- **Purpose**: Fetches per-night sellable inventory, CTA, CTD, cutoff, and stop-sell flags.
- **Request Payload**:
  ```json
  {
    "OTA_HotelInventoryRQ": {
      "POS": { "Username": "...", "Password": "...", "ID_Context": "QUADIS" },
      "HotelCode": 7,
      "InvCodes": [701, 702],
      "Start": "2026-10-01",
      "End": "2026-10-15"
    }
  }
  ```
- **Response (`OTA_HotelInventoryRS`)**: Returns available units (free = total - holds - booked) per night per room code.

#### 3. Inventory Update (`OTA_HotelInvCountNotifRQ`)
- **Route**: `POST /api/ota/inventory/update`
- **Purpose**: Channel manager updates sellable inventory, stop-sell, closed-to-arrival (CTA), closed-to-departure (CTD), and booking cutoff.
- **Request Payload**:
  ```json
  {
    "OTA_HotelInvCountNotifRQ": {
      "POS": { "RequestorID": { "User": "...", "Password": "...", "ID_Context": "QUADIS" } },
      "HotelCode": 7,
      "InvTypeCode": 701,
      "StartDate": "2026-10-01",
      "EndDate": "2026-10-07",
      "InvCount": 4,
      "StopSell": false,
      "CloseOnArrival": false,
      "CloseOnDeparture": false,
      "CutOff": 1
    }
  }
  ```
- **Response**: Standard Status/Remark envelope (`OTA_HotelInvCountNotifRS`).

#### 4. Rate Fetch (`OTA_HotelRateRQ`)
- **Route**: `POST /api/ota/rates/fetch`
- **Purpose**: Reads active per-night rates for single, double, triple, quad occupancy, and extra guest charges.
- **Request Payload**:
  ```json
  {
    "OTA_HotelRateRQ": {
      "POS": { "Username": "...", "Password": "...", "ID_Context": "QUADIS" },
      "HotelCode": 7,
      "RateCodes": [7011, 7012],
      "Start": "2026-10-01",
      "End": "2026-10-10"
    }
  }
  ```
- **Response (`OTA_HotelRateRS`)**: Returns per-night breakdown of `Single`, `Double`, `Triple`, `Quad`, `ExtraPax`, `ExtraChild`, `MinStay`, and `StopSell`.

#### 5. Rate Update (`OTA_HotelRateAmountNotifRQ`)
- **Route**: `POST /api/ota/rates/update`
- **Purpose**: Channel manager sets per-night rates and occupancy pricing.
- **Request Payload**:
  ```json
  {
    "OTA_HotelRateAmountNotifRQ": {
      "POS": { "RequestorID": { "User": "...", "Password": "...", "ID_Context": "QUADIS" } },
      "RatePlanCode": 7012,
      "StartDate": "2026-10-01",
      "EndDate": "2026-10-05",
      "Single": 2200,
      "Double": 2200,
      "Triple": 2860,
      "ExtraAdult": 660,
      "ExtraChild": 440,
      "MinStay": 1,
      "StopSell": false
    }
  }
  ```
- **Response**: Standard Status/Remark envelope (`OTA_HotelRateAmountNotifRS`).

#### 6. Booking Pull (`OTA_HotelResNotifRQ`)
- **Route**: `POST /api/ota/bookings/pull`
- **Purpose**: Channel manager pulls reservations created or modified within a date window.
- **Security Rule**: ResAvenue must supply credentials (`POS` or HTTP Basic Auth) because guest phone numbers, emails, and names are returned.
- **Response (`OTA_HotelResNotifRS`)**: Array of reservations wrapped in the standard ResAvenue schema. Pulling marks pending rows as `SENT`.

#### 7. Single Unified Dispatcher URL
- **Route**: `POST /api/ota/resavenue` (or `POST /api/ota/`)
- **Purpose**: If ResAvenue is configured with a single webhook URL for all messages, the dispatcher automatically inspects the JSON root key and routes to the appropriate internal controller.

#### 8. Live Mapping Code Table
- **Route**: `GET /api/ota/codes`
- **Auth**: HTTP Basic Auth.
- **Output**: Full hierarchical JSON listing every Hotel, Room Type, InvTypeCode, and RatePlanCode across all 9 properties.

---

## 5. Complete REST API Reference

| Domain | Method | Path | Auth | Description |
|---|---|---|---|---|
| **Health** | `GET` | `/healthz` | None | Lightweight load-balancer health check |
| **Health** | `GET` | `/api/health` | None | Deep health check (DB ping, storage, memory) |
| **Properties** | `GET` | `/api/properties` | None | List all properties with rooms, ratings, images |
| **Properties** | `GET` | `/api/properties/{slug}` | None | Get detailed property data and amenities |
| **Content** | `GET` | `/api/content` | None | Fetch site copy, promo banners, FAQ content |
| **Bookings** | `POST` | `/api/bookings/initiate` | None (Rate Limited) | Create 15-min soft hold & compute invoice breakdown |
| **Bookings** | `GET` | `/api/bookings/{code}` | Optional Phone | Get booking status & voucher details |
| **Bookings** | `GET` | `/api/bookings/{code}/invoice` | None | Download formatted booking invoice PDF |
| **Payments** | `POST` | `/api/payments/create-order` | None | Create Razorpay order for an active hold |
| **Payments** | `POST` | `/api/payments/payment-link` | None | Generate standard Razorpay payment link |
| **Webhooks** | `POST` | `/api/webhooks/razorpay` | HMAC SHA256 | Razorpay payment confirmation webhook |
| **Webhooks** | `POST` | `/api/webhooks/whatsapp-staff` | None | Meta WhatsApp inbound webhook |
| **Enquiries** | `POST` | `/api/enquiries` | None (Rate Limited) | Submit banquet/corporate event enquiry |
| **Auth** | `POST` | `/api/auth/register` | None (Rate Limited) | Create guest user account |
| **Auth** | `POST` | `/api/auth/login` | None (Rate Limited) | Guest login (sets HTTP-only cookie) |
| **Auth** | `GET` | `/api/auth/me` | User Session | Get authenticated guest profile |
| **Admin** | `POST` | `/api/admin/auth` | None (Rate Limited) | Admin PIN authentication |
| **Admin** | `GET` | `/api/admin/dashboard` | Admin Session | Metrics: bookings, revenues, leads, occupancy |
| **Admin** | `PATCH`| `/api/admin/room-availability`| Admin Session | Instant toggle room sold-out / active status |
| **Admin** | `PATCH`| `/api/admin/surcharge` | Admin Session | Update property weekend surcharge |
| **AI Concierge**| `POST` | `/api/ai/chat` | None (Rate Limited) | Conversational hotel concierge query |
| **ResAvenue** | `POST` | `/api/ota/*` | POS / Basic Auth | Channel manager endpoints (detailed in §4) |

---

## 6. Security Posture & Hardening

1. **Zero Port 22 Access**: The production EC2 instance has port 22 closed at the AWS Security Group level (`sg-0b19531290e391e17`). Access and deployments occur strictly over AWS SSM Session Manager.
2. **SSM Parameter Store Secrets**:
   - Zero credentials or API keys exist in git.
   - `deploy/install.sh` queries AWS SSM Parameter Store (`/quadis/*`) at deploy time and writes an isolated, root-owned file (`/etc/quadis/api.env`, chmod 0640).
3. **Timing-Safe Authentication**:
   - Channel manager credentials use `crypto/subtle.ConstantTimeCompare` to defend against side-channel timing attacks.
4. **Multi-Tier Rate Limiting**:
   - **General API**: 120 req / 15 min.
   - **Authentication (`/api/auth`, `/api/admin/auth`)**: 10 req / 15 min.
   - **ResAvenue Gateway (`/api/ota`)**: 1,000 req / 15 min bucket (accommodating year-round bulk rate updates).
   - **CSP Violations (`/api/csp-report`)**: 30 req / 15 min.
5. **Raw Body Webhook Validation**:
   - Razorpay signatures are verified using the unparsed raw request buffer (`X-Razorpay-Signature`).

---

## 7. Local Development & Testing

### Prerequisites
- Go 1.22+
- Node.js 20+ & npm
- PostgreSQL (optional: in-memory store activates automatically when `DATABASE_URL` is omitted)

### 7.1 Running the Go Backend

```bash
cd backend-go

# 1. Copy environment template
cp .env.example .env

# 2. Run test suite
go test -v ./...

# 3. Start API server (defaults to port 3001)
go run ./cmd/server
```

### 7.2 Running the Frontend

```bash
# In repository root
npm install
npm run dev
```
The frontend starts at `http://localhost:5173` and proxies `/api/*` to the Go backend.

---

## 8. Deployment & CI/CD Pipeline

Production deployments use the assembled artifact workflow:

```bash
# 1. Build immutable deployable artifact
./deploy/build-artifact.sh

# 2. Upload artifact to private S3 bucket and trigger SSM deployment on EC2
./deploy/push.sh
```

### Deployment Guarantees
- The artifact builder enforces strict blacklisting: `client-assets/`, `.agents/`, `docs/`, `.env`, and test keys are omitted from the archive.
- `install.sh` runs migrations, extracts SSM parameters, reloads systemd unit `quadis-api.service`, and validates Nginx configuration before restarting.
- Nginx provides an ACME webroot challenge bypass (`^~ /.well-known/acme-challenge/`), ensuring Let's Encrypt certificates renew automatically without manual intervention.
