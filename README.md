# Quadis Hotels — Production Architecture & High-Performance Go Engine

> **Enterprise-grade hospitality booking platform, AI concierge, and OTA channel manager engine powering Quadis Hotels across Delhi NCR.**

---

## Table of Contents

1. [System Overview & Architecture](#1-system-overview--architecture)
2. [Software Engineering Decision Choices (The "Why")](#2-software-engineering-decision-choices-the-why)
   - [2.1 Migration to Go Backend (`backend-go`)](#21-migration-to-go-backend-backend-go)
   - [2.2 Router Selection: Why Chi?](#22-router-selection-why-chi)
   - [2.3 Clean Architecture & Domain-Driven Design (DDD)](#23-clean-architecture--domain-driven-design-ddd)
   - [2.4 Dual Repository Pattern (PostgreSQL + In-Memory Store)](#24-dual-repository-pattern-postgresql--in-memory-store)
3. [Core Business Engines & Implementation Details (The "How")](#3-core-business-engines--implementation-details-the-how)
   - [3.1 Booking State Machine & 15-Minute Soft-Hold Engine](#31-booking-state-machine--15-minute-soft-hold-engine)
   - [3.2 Multi-Tier Dynamic Pricing Engine (`pricing.go`)](#32-multi-tier-dynamic-pricing-engine-pricinggo)
   - [3.3 ResAvenue Channel Manager (OTA API v2.0) Gateway](#33-resavenue-channel-manager-ota-api-v20-gateway)
   - [3.4 Payment Gateway & Idempotent Webhook Engine (Razorpay)](#34-payment-gateway--idempotent-webhook-engine-razorpay)
   - [3.5 Multi-Provider AI Concierge Engine (Gemini + Groq)](#35-multi-provider-ai-concierge-engine-gemini--groq)
   - [3.6 Automated Guest Comms (Meta WhatsApp Cloud API)](#36-automated-guest-comms-meta-whatsapp-cloud-api)
   - [3.7 Authentication & Cryptography (scrypt + Admin PIN)](#37-authentication--cryptography-scrypt--admin-pin)
4. [Complete REST & OTA API Reference](#4-complete-rest--ota-api-reference)
5. [Security, Governance & Hardening](#5-security-governance--hardening)
6. [Local Development, Testing & Verification](#6-local-development-testing--verification)
7. [Production Deployment Pipeline (Zero Port 22)](#7-production-deployment-pipeline-zero-port-22)

---

## 1. System Overview & Architecture

Quadis Hotels is a full-stack hospitality platform serving 9+ properties across Noida and New Delhi. The system operates as a unified architecture running on AWS EC2 (`t3.medium`, Amazon Linux 2023):

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
┌────────────────────────────────────────────────────────────────────────────────┐       ┌───────────────────────────────────┐
│                           Quadis Go Backend (`backend-go`)                     │       │        ResAvenue Channel Manager  │
│                                                                                │       │             (OTA Extranet)        │
│  ┌───────────────────────┐  ┌────────────────────┐  ┌────────────────────────┐ │       └─────────────────┬─────────────────┘
│  │ Bookings & Holds      │  │ Dynamic Pricing    │  │ ResAvenue OTA Gateway  │◄├─────────────────────────┘
│  │ State Machine Engine  │  │ & GST Calculator   │  │ (Details/Inv/Rates/Pull│ │  POST /api/ota/* (Fetch, Update, Pull)
│  └──────────┬────────────┘  └─────────┬──────────┘  └────────┬───────────────┘ │  POST /push (OTA_HotelResNotifRQ)
│             │                         │                      │                 │
│  ┌──────────┴────────────┐  ┌─────────┴──────────┐  ┌────────┴───────────────┐ │
│  │ AI Concierge Service  │  │ Payments & Webhook │  │ WhatsApp Comms Worker  │ │
│  │ (Gemini + Groq)       │  │ (Razorpay Gateway) │  │ (Meta Cloud API)       │ │
│  └───────────────────────┘  └────────────────────┘  └────────────────────────┘ │
└──────────────────────────────────────┬─────────────────────────────────────────┘
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

## 2. Software Engineering Decision Choices (The "Why")

### 2.1 Migration to Go Backend (`backend-go`)
- **Memory Footprint on `t3.medium`**: The production EC2 host co-locates Nginx, PostgreSQL, and the application backend within a 4 GiB memory envelope. The Node.js/TypeScript backend consumed **~180 MB RSS** with garbage collection spikes; the Go backend operates consistently under **35 MB RSS**, completely eliminating out-of-memory kernel reaps.
- **Concurrent Goroutine Workers**: Hospitality systems require background schedulers (soft-hold expiry sweepers, channel-sync retry loops). In Node.js, these compete on a single-threaded event loop and can delay incoming HTTP request processing. In Go, workers run on lightweight, isolated Goroutines without blocking web traffic.
- **Strict Compile-Time Typing**: Financial transactions and occupancy pricing calculations in TypeScript risk subtle `undefined`/`NaN` coercion bugs. Go guarantees strict compile-time type safety across monetary amounts, room counts, and nullable rate overrides.
- **Single Static Binary**: Go compiles to a standalone, zero-dependency executable. This completely eliminates native ABI mismatch issues (such as `sharp` or `bcrypt` bindings breaking when built across varying Node ABI versions).

### 2.2 Router Selection: Why Chi?
Instead of heavy web frameworks (like Gin or Fiber), we selected **[Chi](https://github.com/go-chi/chi)**:
1. **100% `net/http` Compatibility**: Chi is built strictly on Go's standard `http.Handler` and `http.ResponseWriter`. Handlers, middleware, and third-party libraries require no adapter layers or proprietary contexts.
2. **Composable Sub-Routers**: Allows modular prefix routing (`api.Route("/ota", ...)`, `api.Route("/admin", ...)`, `api.Route("/bookings", ...)`).
3. **Hierarchical Middleware Scoping**: Rate limiting can be tailored per sub-router (e.g., standard endpoints: 120 req/15m; auth endpoints: 10 req/15m; OTA endpoints: 1000 req/15m to handle bulk annual rate updates).
4. **Zero Memory Allocation Overhead**: Chi's radix tree router allocates zero heap memory during route matching.

### 2.3 Clean Architecture & Domain-Driven Design (DDD)
The codebase enforces strict separation of concerns across layered boundaries:

```
┌────────────────────────────────────────────────────────────┐
│ 1. cmd/server/main.go (Composition Root & Wire-up)         │
└────────────────────────────┬───────────────────────────────┘
                             │ Dependency Injection
                             ▼
┌────────────────────────────────────────────────────────────┐
│ 2. internal/api/ (Transport Layer: HTTP Handlers, Routers) │
└────────────────────────────┬───────────────────────────────┘
                             │ Calls
                             ▼
┌────────────────────────────────────────────────────────────┐
│ 3. internal/service/ (Business Logic, Pricing, State Mach.)│
└────────────────────────────┬───────────────────────────────┘
                             │ Calls Interface
                             ▼
┌────────────────────────────────────────────────────────────┐
│ 4. internal/repository/ (Persistence: Postgres & Memory)   │
└────────────────────────────┬───────────────────────────────┘
                             │ Uses
                             ▼
┌────────────────────────────────────────────────────────────┐
│ 5. internal/domain/ (Pure Domain Models & Entities)        │
└────────────────────────────────────────────────────────────┘
```

- **Dependency Inversion**: Handlers never touch SQL or database connection pools directly. They interact solely with Domain Services or Repository interfaces.
- **Domain Purity**: `internal/domain` contains zero HTTP, database, or third-party framework dependencies.

### 2.4 Dual Repository Pattern (PostgreSQL + In-Memory Store)
`internal/repository/interfaces.go` defines the storage contract. We implemented two distinct repositories:
1. **`postgres.PostgresStore`**: Production engine utilizing `pgxpool` with parameterized queries, connection pooling, and transactional isolation.
2. **`memory.MemoryStore`**: Thread-safe, RWMutex-backed in-memory store pre-seeded with all 9 properties and room types.
   - **Why this was built**: Enables **100% of API endpoints, pricing suites, and hold state machines to be unit-tested in 0.07 seconds** without spinning up external database containers or requiring mocks.

---

## 3. Core Business Engines & Implementation Details (The "How")

### 3.1 Booking State Machine & 15-Minute Soft-Hold Engine

#### State Machine Progression
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

#### How Concurrency & Double-Booking Prevention Works
When a booking is initiated (`POST /api/bookings/initiate`):
1. The repository calculates available units for the requested room type and date range:
   $$\text{Available Units} = \text{Room Capacity} - \text{Active Confirmed Units} - \text{Active Held Units}$$
2. A hold is only valid if $\text{created\_at} > \text{NOW}() - 15\text{ minutes}$.
3. If $\text{Available Units} < \text{Requested Rooms}$, the hold is rejected immediately with HTTP 409 Conflict.
4. If available, a `PENDING_PAYMENT` record is written, locking inventory against competing requests.

#### Hold Expiry Worker (`StartHoldCleanupWorker`)
- Runs as a background Goroutine with a configurable ticker (default: 60s).
- Executes an atomic database sweep:
  ```sql
  UPDATE bookings
  SET booking_status = 'EXPIRED'
  WHERE booking_status = 'PENDING_PAYMENT'
    AND created_at < NOW() - INTERVAL '15 minutes';
  ```
- Inventory is automatically restored in real-time without locking rows or disrupting active checkouts.

---

### 3.2 Multi-Tier Dynamic Pricing Engine (`pricing.go`)

The pricing engine computes tariffs down to the individual night and guest composition.

#### 1. Baseline Nightly Rate & Meal Plans
$$\text{Base Tariff} = \text{Property Base Price} + \text{Room Category Offset}$$

Meal plans are derived as a strict percentage of the base room rate (verified client rule):
- **EP (Room Only)**: $+0\%$
- **CP (With Breakfast)**: $+25\%$ of Base Tariff
- **MAP (All Meals Included)**: $+50\%$ of Base Tariff

#### 2. Weekend Surcharges
The engine checks each night using `dateutil.IsWeekendNight(night)`. If the night is a Friday or Saturday, the property's configured weekend surcharge percentage (typically $10\text{--}25\%$) is added to the room tariff.

#### 3. Occupancy Rules & Child Concessions
- **Included Adults**: 2 adults per room.
- **Extra Adult ($13+$ years)**: $+30\%$ of the room rate per adult per night.
- **Infants & Toddlers (Under 8 years)**: **Free of charge** ($0\%$).
- **Children (8 to 12 years)**: Concession rate of $+20\%$ of the room rate per child per night.

#### 4. Indian GST Compliance (Value of Supply Threshold)
Indian tax law mandates GST based on the net **Value of Supply** (tax-exclusive nightly room tariff):
$$\text{Value of Supply} = \frac{\text{Gross Rate Per Room Night}}{1 + \text{Standard GST Rate}} = \frac{\text{Gross Rate}}{1.05}$$

- **Standard Slab ($5\%$)**: Applied when $\text{Value of Supply} \le ₹7,500$.
- **Luxury Slab ($18\%$)**: Applied when $\text{Value of Supply} > ₹7,500$.
- Evaluated per room night rather than on the total invoice, ensuring legal compliance on multi-room, multi-night stays.

#### 5. Channel Manager Overrides (`NightOverrides`)
If ResAvenue pushes custom nightly rates (`room_rate_days`), the engine applies the pushed rate for those exact dates, superseding base calculations and weekend surcharges while preserving extra guest pricing where specified.

---

### 3.3 ResAvenue Channel Manager (OTA API v2.0) Gateway

Quadis implements all 7 messages defined in ResAvenue's *OTA API Guide v2.0*. In this architecture, **Quadis acts as the OTA**.

#### Stable Numerical Code Mapping
ResAvenue maps entities using short numeric codes ($\le 10$ digits). Quadis derives codes deterministically so they never drift:

| Level | Derivation Formula | Example | Code |
|---|---|---|---|
| **Hotel Code** | Extracted from `prop-{N}` | `prop-7` (Downtown Sec 51) | `7` |
| **Room Code (`InvTypeCode`)** | $\text{Hotel Code} \times 100 + \text{Category Index}$ | Deluxe Room ($1$) at Hotel $7$ | `701` |
| **Rate Plan Code (`RatePlanCode`)** | $\text{Room Code} \times 10 + \text{Meal Plan Index}$ | CP Breakfast ($2$) on Room `701` | `7012` |

**Category Indices**: `deluxe-room: 1`, `super-deluxe: 2`, `superior-room: 3`, `royal-suite: 4`.  
**Meal Plan Indices**: `EP (Room Only): 1`, `CP (With Breakfast): 2`, `MAP (All Meals): 3`.

#### Supported OTA Message Workflows

```
┌─────────────────────────────────┐                       ┌──────────────────────────────────┐
│   ResAvenue Channel Manager     │                       │     Quadis Go OTA Controller     │
└────────────────┬────────────────┘                       └────────────────┬─────────────────┘
                 │                                                         │
                 │ 1. OTA_HotelDetailsRQ (Fetch Hotel/Rooms/RatePlans)     │
                 ├────────────────────────────────────────────────────────►│
                 │◄────────────────────────────────────────────────────────┤
                 │    OTA_HotelDetailsRS                                   │
                 │                                                         │
                 │ 2. OTA_HotelInventoryRQ (Fetch Inventory / Restrictions)│
                 ├────────────────────────────────────────────────────────►│
                 │◄────────────────────────────────────────────────────────┤
                 │    OTA_HotelInventoryRS                                 │
                 │                                                         │
                 │ 3. OTA_HotelInvCountNotifRQ (Update Inventory/StopSell) │
                 ├────────────────────────────────────────────────────────►│
                 │◄────────────────────────────────────────────────────────┤
                 │    OTA_HotelInvCountNotifRS (Success/Failure)           │
                 │                                                         │
                 │ 4. OTA_HotelRateRQ (Fetch Rates per Night)              │
                 ├────────────────────────────────────────────────────────►│
                 │◄────────────────────────────────────────────────────────┤
                 │    OTA_HotelRateRS                                      │
                 │                                                         │
                 │ 5. OTA_HotelRateAmountNotifRQ (Update Rates/Occupancy)  │
                 ├────────────────────────────────────────────────────────►│
                 │◄────────────────────────────────────────────────────────┤
                 │    OTA_HotelRateAmountNotifRS (Success/Failure)         │
                 │                                                         │
                 │ 6. OTA_HotelResNotifRQ (Pull Bookings Window)           │
                 ├────────────────────────────────────────────────────────►│
                 │◄────────────────────────────────────────────────────────┤
                 │    OTA_HotelResNotifRS (Reservations Array)             │
                 │                                                         │
                 │ 7. OTA_HotelResNotifRQ (Real-time Push from Quadis)     │
                 │◄────────────────────────────────────────────────────────┤
                 │    Quadis PushBooking Worker                            │
```

#### Authentication & Timing-Safe Comparison
Credentials arrive either in the request body `POS` envelope or via HTTP Basic Auth:
```json
"POS": {
  "Username": "resavenue_user",
  "Password": "secure_password",
  "ID_Context": "QUADIS"
}
```
Validation uses `crypto/subtle.ConstantTimeCompare` on username, password, and `ID_Context="QUADIS"`, eliminating side-channel timing attacks.

#### Two-Way Reservation Sync
1. **Pull Flow (`/api/ota/bookings/pull`)**: ResAvenue queries changed reservations within a date window. Delivers confirmed and cancelled bookings. Delivered records are atomically marked `cm_sync_status = 'SENT'`.
2. **Push Flow (`PushBooking` & `StartChannelSyncWorker`)**:
   - Upon payment capture, the system fires an asynchronous Goroutine to push `OTA_HotelResNotifRQ` to ResAvenue's configured webhook URL.
   - If the push fails (e.g. timeout or 5xx from ResAvenue), the record is marked `FAILED` with error diagnostics.
   - `StartChannelSyncWorker` retries failed and pending pushes in the background.

#### Unified Message Dispatcher
If ResAvenue's platform can only target a single webhook URL, both `POST /api/ota/resavenue` and `POST /api/ota/` act as intelligent dispatchers:
- Inspects top-level JSON keys (`OTA_HotelDetailsRQ`, `OTA_HotelInventoryRQ`, `OTA_HotelInvCountNotifRQ`, etc.).
- Automatically delegates execution to the correct internal controller.

---

### 3.4 Payment Gateway & Idempotent Webhook Engine (Razorpay)

- **Order Creation (`POST /api/payments/create-order`)**: Generates Razorpay order IDs linked directly to active booking holds.
- **Raw-Body HMAC SHA-256 Signature Verification**:
  Nginx passes the raw request buffer. The handler computes:
  $$\text{Expected Signature} = \text{HMAC-SHA256}(\text{raw\_body}, \text{RAZORPAY\_WEBHOOK\_SECRET})$$
  Evaluated using constant-time comparison against `X-Razorpay-Signature`.
- **Idempotent Capture**: Webhooks can be delivered multiple times by payment providers. If the target booking is already `CONFIRMED`, the webhook returns HTTP 200 immediately without executing duplicate operations.
- **Fire-and-Forget Asynchronous Notifications**:
  Upon confirmation, independent Goroutines fire:
  1. WhatsApp Guest Receipt Voucher
  2. WhatsApp Hotel Owner Alert
  3. ResAvenue Channel Reservation Push

---

### 3.5 Multi-Provider AI Concierge Engine (Gemini + Groq)

The concierge provides 24/7 natural-language assistance regarding amenities, room pricing, local metro stations, banquet capacities, and check-in policies:
1. **Primary Provider: Google Gemini**:
   - Rotates through models: `gemini-2.5-flash`, `gemini-2.5-flash-lite`, `gemini-2.0-flash`.
   - Supports key pooling across multiple API keys (`GEMINI_API_KEYS`).
2. **Automatic Fallback Provider: Groq**:
   - If Gemini returns rate limits (429) or transient 5xx errors, the request instantly fails over to Groq's high-speed Llama inference (`llama-3.3-70b-versatile` / `llama-3.1-8b-instant`).
3. **Grounding & Audit Logging**:
   - Prompt context is dynamically injected with structured property metadata and verified policies.
   - All guest interactions are logged to `chat_logs` for sentiment analysis and booking conversion tracking.

---

### 3.6 Automated Guest Comms (Meta WhatsApp Cloud API)

- Directly integrated with Meta's Graph API.
- Generates instant booking confirmation cards containing:
  - Guest Name & Booking Reference Code
  - Property Name, Address & Google Maps Navigation Link
  - Room Category & Meal Plan Selection
  - Check-In & Check-Out Timestamps
  - Authoritative Breakdown of Tariff, Taxes & Balance Paid

---

### 3.7 Authentication & Cryptography (scrypt + Admin PIN)

- **User Password Hashing**: Utilizes **Node-compatible scrypt** ($N=16384, r=8, p=1, \text{keyLen}=64$), allowing seamless user account portability between Node.js and Go backends without password resets.
- **Admin PIN Verification**: Scrypt-hashed with per-record salts; verified in constant time.
- **Session Tokens**: 256-bit cryptographically secure pseudorandom tokens stored in HTTP-only, secure, `SameSite=Lax` cookies.

---

## 4. Complete REST & OTA API Reference

### Public & Guest Endpoints
| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/healthz` | None | Lightweight load-balancer health check |
| `GET` | `/api/health` | None | Deep health check (DB ping, storage, memory) |
| `GET` | `/api/properties` | None | List all properties with rooms, ratings, images |
| `GET` | `/api/properties/{slug}`| None | Get detailed property data and amenities |
| `GET` | `/api/content` | None | Fetch site copy, banners, and FAQs |
| `POST`| `/api/bookings/initiate`| Rate Limited | Create 15-min soft hold & compute price |
| `GET` | `/api/bookings/{code}` | Phone (Optional)| Retrieve booking status & voucher details |
| `GET` | `/api/bookings/{code}/invoice` | None | Download formatted booking invoice PDF |
| `POST`| `/api/payments/create-order` | None | Create Razorpay order for an active hold |
| `POST`| `/api/payments/payment-link` | None | Generate standard Razorpay payment link |
| `POST`| `/api/enquiries` | Rate Limited | Submit banquet / corporate enquiry |
| `POST`| `/api/auth/register` | Rate Limited | Guest registration |
| `POST`| `/api/auth/login` | Rate Limited | Guest login |
| `GET` | `/api/auth/me` | User Session | Get authenticated guest profile |
| `POST`| `/api/ai/chat` | Rate Limited | Natural language AI concierge query |

### Webhook Endpoints
| Method | Path | Auth | Description |
|---|---|---|---|
| `POST`| `/api/webhooks/razorpay` | HMAC-SHA256 | Razorpay payment confirmation webhook |
| `POST`| `/api/webhooks/whatsapp-staff` | None | Meta WhatsApp inbound status webhook |
| `POST`| `/api/csp-report` | Rate Limited | Content Security Policy violation reports |

### Admin Endpoints (Guarded by PIN Session)
| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/api/admin/auth` | Rate Limited | Admin PIN sign-in |
| `GET` | `/api/admin/dashboard` | Admin Session | Aggregated revenues, bookings, leads |
| `PATCH`| `/api/admin/room-availability`| Admin Session | Toggle room availability / sold-out |
| `PATCH`| `/api/admin/surcharge` | Admin Session | Update property weekend surcharge |
| `POST` | `/api/admin/payment-link` | Admin Session | Generate custom quotation payment link |
| `PUT` | `/api/admin/content` | Admin Session | Update CMS website copy |

### ResAvenue Channel Manager (OTA Gateway)
| Method | Path | Supported Messages | Description |
|---|---|---|---|
| `POST` | `/api/ota/property-details` | `OTA_HotelDetailsRQ` | Fetch hotel rooms and rolling 3-year rate plans |
| `POST` | `/api/ota/inventory/fetch` | `OTA_HotelInventoryRQ` | Read sellable inventory and night restrictions |
| `POST` | `/api/ota/inventory/update`| `OTA_HotelInvCountNotifRQ`| Set inventory, stop-sell, CTA, CTD, cutoff |
| `POST` | `/api/ota/rates/fetch` | `OTA_HotelRateRQ` | Fetch per-night rates across all occupancies |
| `POST` | `/api/ota/rates/update` | `OTA_HotelRateAmountNotifRQ`| Set rates, extra adult/child, and min/max stay |
| `POST` | `/api/ota/bookings/pull` | `OTA_HotelResNotifRQ` | Pull confirmed/cancelled bookings window |
| `POST` | `/api/ota/resavenue` | *Any OTA message* | Unified single-URL message dispatcher |
| `GET` | `/api/ota/codes` | HTTP Basic Auth | Retrieve numerical mapping table for all hotels |

---

## 5. Security, Governance & Hardening

1. **Zero Port 22 Access**: Port 22 is disabled at the AWS EC2 Security Group level (`sg-0b19531290e391e17`). Access occurs exclusively via AWS Systems Manager (SSM) Session Manager.
2. **Zero Plaintext Secrets in Repositories**:
   - Zero credentials, database passwords, or payment secrets exist in git.
   - Production secrets reside in AWS SSM Parameter Store (`/quadis/*`).
   - `deploy/install.sh` fetches parameters at deploy time and writes an isolated, root-owned file (`/etc/quadis/api.env`, permissions `0640`, group `quadis`).
3. **Strict Rate Limiting Policies**:
   - **General API**: 120 req / 15 minutes.
   - **Authentication**: 10 req / 15 minutes.
   - **ResAvenue OTA**: 1,000 req / 15 minutes (to accommodate annual rate uploads).
   - **CSP Violations**: 30 req / 15 minutes.
4. **Content Security Policy (CSP)**: Strict headers enforced by Nginx and reported to `/api/csp-report`.
5. **PostgreSQL Socket Isolation**: Database listens strictly on loopback (`127.0.0.1:5432`), completely unreachable from the public internet.

---

## 6. Local Development, Testing & Verification

### Prerequisites
- **Go**: Version 1.22+ installed
- **Node.js**: Version 20+ & npm

### 6.1 Running the Go Backend
```bash
cd backend-go

# 1. Initialize environment template
cp .env.example .env

# 2. Run unit & integration test suites
go test -v ./...

# 3. Compile and launch API server
go run ./cmd/server
```
*Note: When `DATABASE_URL` is omitted in development, the application automatically mounts the high-speed in-memory store (`MemoryStore`) with complete mock data.*

### 6.2 Running the Frontend SPA
```bash
# In repository root
npm install
npm run dev
```
The React development server runs at `http://localhost:5173` and proxies API requests to `http://localhost:3001`.

---

## 7. Production Deployment Pipeline (Zero Port 22)

Deployments follow an assembled, immutable artifact process:

```bash
# 1. Assemble and build deployment artifact (stripping source code, credentials, test files)
./deploy/build-artifact.sh

# 2. Upload tarball to S3 and trigger AWS SSM RunCommand execution on EC2 instance
./deploy/push.sh
```

### Deployment Workflow Details
1. **`build-artifact.sh`**: Compiles the Go binary (`backend-go/cmd/server`), bundles the React frontend (`dist/`), and validates that no sensitive folders (`client-assets/`, `.agents/`, `docs/`, `.env`) are packaged.
2. **`push.sh`**: Uploads `quadis-<sha>.tar.gz` to the private AWS S3 bucket (`quadis-hotel-photos/deploy/`) and dispatches an SSM Command to the live instance (`i-0d126c49ffdfe1668`).
3. **`install.sh` (Runs on Target)**:
   - Unpacks bundle to `/opt/quadis/api`.
   - Queries AWS SSM Parameter Store (`/quadis/*`) to generate `/etc/quadis/api.env`.
   - Atomically deploys frontend static files to `/var/www/quadis`.
   - Reloads systemd service `quadis-api.service`.
   - Validates Nginx configurations and preserves ACME renewal paths (`^~ /.well-known/acme-challenge/`).
