# ResAvenue channel manager — OTA API integration

Written 11 Sep 2026 against ResAvenue's *OTA API Guide v2.0* (22 Sep 2020,
Infibeam Avenues). The guide is written from the channel manager's side; in
its vocabulary **this site is the OTA**. Everything below is implemented in
`backend/src/routes/ota.ts`, `backend/src/lib/channel.ts`,
`backend/src/services/ChannelManagerService.ts` and
`backend/src/workers/channelSync.ts`, and covered by
`backend/__tests__/ota.test.ts`.

## What this does

ResAvenue can now:

| Guide section | Message | What it does here | Our URL |
|---|---|---|---|
| II | `OTA_HotelDetailsRQ` | Read a hotel's rooms and rate plans | `POST /api/ota/property-details` |
| III | `OTA_HotelInventoryRQ` | Read sellable units and restrictions per night | `POST /api/ota/inventory/fetch` |
| IV | `OTA_HotelInvCountNotifRQ` | **Set** units, stop-sell, closed arrival/departure, cut-off per night | `POST /api/ota/inventory/update` |
| V | `OTA_HotelRateRQ` | Read rates per night | `POST /api/ota/rates/fetch` |
| VI | `OTA_HotelRateAmountNotifRQ` | **Set** rates, extra-guest charges, min/max stay, stop-sell per night | `POST /api/ota/rates/update` |
| VII.i | `OTA_HotelResNotifRQ` (pull) | Read reservations confirmed or cancelled in a date window | `POST /api/ota/bookings/pull` |
| VII.ii | `OTA_HotelResNotifRQ` (push) | **We send** each confirmed / cancelled reservation to their URL | `RESAVENUE_PUSH_URL` |

`PUT` works everywhere `POST` does, as the guide allows. There is also a single
dispatcher, `POST /api/ota/resavenue` (or `/api/ota`), that routes on the root
key of the body, for a channel manager that can only be configured with one
URL.

What ResAvenue sets **changes what the site sells**: the booking engine reads
the per-night inventory and rate tables ahead of `total_units` and the computed
nightly rate, and the checkout's payment step now shows the server's total so
"Pay ₹X" is the ₹X Razorpay takes. A room or night ResAvenue has never touched
behaves exactly as before.

## Before ResAvenue can connect

1. **Choose credentials** and set them on the server. The endpoints answer
   `503` until both are set. They let the channel manager rewrite our
   inventory and rates, so treat them like the admin PIN: long, random, and
   not in this repo.
   ```
   RESAVENUE_OTA_USERNAME=<username ResAvenue will send>
   RESAVENUE_OTA_PASSWORD=<password ResAvenue will send>
   RESAVENUE_OTA_ID_CONTEXT=          # only if ResAvenue assigns one
   ```
   On the production box these go where the other secrets go (SSM
   `/quadis/*`, read by the systemd unit — see AGENTS.md §3b).
2. **Give ResAvenue** the username, password, the base URL
   `https://www.quadishotels.com/api/ota/` (or the dispatcher URL), and the
   code table from step 3.
3. **Get the code table.** ResAvenue maps hotels, rooms and rate plans by
   numeric code. Ours are derived, not stored, so they never change:
   ```
   curl -u "$USER:$PASS" https://www.quadishotels.com/api/ota/codes
   ```
   The scheme, if someone needs it by hand: hotel code = the number in the
   property id (`prop-7` → **7**); room code = hotel × 100 + category
   (Deluxe 1, Super Deluxe 2, Superior 3, Royal Suite 4 — so Deluxe at
   `prop-7` is **701**); rate plan = room × 10 + plan (EP 1, CP 2, MAP 3 — so
   breakfast on that Deluxe is **7012**). The site's three meal plans are its
   three rate plans per room.
4. **For push**, ask ResAvenue for their push URL and any header they want on
   it:
   ```
   RESAVENUE_PUSH_URL=https://<their endpoint>
   RESAVENUE_PUSH_AUTHORIZATION=        # e.g. "Bearer …", sent verbatim; optional
   RESAVENUE_TARGET=Production          # or the test value they give
   ```
   Until the URL is set nothing is pushed; confirmed bookings queue as
   `cm_sync_status = PENDING` and the pull endpoint serves them. Both channels
   can be on at once.

## Authentication

Every message carries credentials in its body, in whichever of the guide's two
shapes that message uses:

```json
"POS": { "Username": "…", "Password": "…", "ID_Context": "…" }
"POS": { "RequestorID": { "User": "…", "Password": "…", "ID_Context": "" } }
```

HTTP Basic auth is accepted in place of either. `ID_Context` is only checked
when `RESAVENUE_OTA_ID_CONTEXT` is set; the guide's own update samples send it
empty.

**One deliberate departure from the guide:** the booking-pull sample has no
credentials at all. That reply contains guest names, phones and emails, so it
is not served unauthenticated here. ResAvenue must include a `POS` block or
Basic auth on the pull too.

## Reply conventions

- Success replies follow the guide's samples field for field, including its
  oddities: the pull reply is rooted at `OTA_HotelResNotifRQ` (the request
  name) and wraps each reservation in its own `{ "HotelReservation": [...] }`.
- The fetch replies echo the `POS` block back with **`Password` blank**.
- Update replies are the guide's `Status: "Success" | "Failure"` + `Remark`.
- A failure on a fetch, which the guide does not define, uses the same pair
  under that message's `RS` root, e.g.
  `{ "OTA_HotelInventoryRS": { "Status": "Failure", "Remark": "Unknown InvCode \"999\" …" } }`.
- HTTP status: **200** for business failures (so the channel manager reads the
  Remark instead of retrying a transport error), **401** for bad credentials,
  **503** when credentials are not configured, **429** past 1,000 requests per
  15 minutes per IP.
- Updates are all-or-nothing per message: one bad entry rejects the message
  and the Remark names it (`RateAmountMessages[1]: …`).
- Every update merges: a field the message omits keeps its stored value. A
  message that only flips `StopSell` does not reset the count.
- Dates are `YYYY-MM-DD`. `End` / `ToDate` may be omitted (defaults to
  `Start`; for the pull, to today). A range is capped at 400 days.
- `Days` follows the guide (`"True"`/`"False"` strings, absent = `"True"`,
  `Weds`/`Thur` spellings accepted).

## What the figures mean

**Inventory fetch** `InvCount` is what is still *sellable* that night: the
ceiling (ResAvenue's count if set, else the room's keys) minus every unit held
— paid bookings and live 15-minute soft holds alike.

**Rate fetch**, with nothing pushed, reports the site's own pricing: `Double`
is base price + category offset + the plan's uplift (EP 0 / CP 25% / MAP 50%),
with the weekend surcharge on Fri/Sat; `Single` = `Double` (every rate here is
for two adults); `ExtraPax` is the property's extra-adult percentage of that
night's rate and `ExtraChild` the child percentage; `Triple`/`Quad` add one
and two `ExtraPax`. `MinStay` 1, `MaxStay` 30.

**Rate update** stores whatever it sends, per plan per night: `NumberOfGuests`
1–4 → single/double/triple/quad; `GuestType` containing "Adult" → extra adult,
"Child" → extra child. Fields it does not send stay on the computed value. The
weekend surcharge is **not** applied on top of a pushed rate — the Friday
figure ResAvenue sends is Friday's rate.

**Booking engine** (both `POST /api/bookings/initiate` and the concierge's
availability) honours, in this order: closed arrival on check-in, closed
departure on check-out, stop-sell on any night (room-level or that rate
plan's), cut-off days before arrival, min/max stay from the arrival night's
rate row, then the per-night ceiling minus holds. A pushed rate replaces the
nightly figure and the per-head charges on the nights it covers; a stay that
straddles a pushed range pays the pushed figure on those nights and the site's
own on the rest.

## Reservations

`UniqueID.ID` is our booking code (`QD-…`). `ResStatus` is `Confirm` when a
paid hold becomes CONFIRMED and `Cancel` when a confirmed booking is cancelled;
unpaid holds, expiries and failed payments never reach ResAvenue. `Modify` is
never sent — the site has no amendment flow. `PayAtHotel` is `N` (paid online
through Razorpay). Amounts are GST-inclusive with `TaxType: "Inclusive"` and
`TotalTax` back-calculated at the slab the per-room-night rate falls in;
`Commission` is 0. Per-night `Amount`s are the total spread evenly, the last
night absorbing the rounding. `CreateDateTime` is IST.

**Push** fires from the Razorpay webhook the moment a payment is captured, and
`workers/channelSync.ts` retries anything `PENDING` or `FAILED` every five
minutes, up to 12 attempts, after which the row stays `FAILED` with
`cm_last_error` for a human. Delivery is at-least-once; ResAvenue keys on the
booking code. A pull marks what it returned as `SENT` without counting an
attempt.

Columns added to `bookings` for this: `meal_plan`, `updated_at`,
`cm_sync_status` (NONE/PENDING/SENT/FAILED), `cm_res_status`, `cm_synced_at`,
`cm_attempts`, `cm_last_error`. New tables: `room_inventory_days`,
`room_rate_days`. All applied by `schema.sql` on boot, idempotently.

## Trying it locally

```bash
cd backend
RESAVENUE_OTA_USERNAME=demo RESAVENUE_OTA_PASSWORD=demo npx ts-node src/server.ts

curl -s localhost:3001/api/ota/property-details -H 'Content-Type: application/json' -d '{
  "OTA_HotelDetailsRQ": { "POS": { "Username": "demo", "Password": "demo", "ID_Context": "" },
                          "EchoToken": "1", "HotelCode": "2" } }'

curl -s localhost:3001/api/ota/rates/update -H 'Content-Type: application/json' -d '{
  "OTA_HotelRateAmountNotifRQ": {
    "EchoToken": "2", "Target": "Production", "HotelCode": "2",
    "POS": { "RequestorID": { "User": "demo", "Password": "demo", "ID_Context": "" } },
    "RateAmountMessages": [{
      "StatusApplicationControl": { "InvTypeCode": "201", "RatePlanCode": "2011",
                                    "Start": "2027-03-10", "End": "2027-03-12" },
      "Rates": { "BaseByGuestAmts": [{ "Amount": "1700", "NumberOfGuests": "2" }],
                 "AdditionalGuestAmounts": [{ "GuestType": "ExtraAdult1", "Amount": "200" }],
                 "MinStay": "1", "MaxStay": "10", "StopSell": "False" } }] } }'
```

## Open questions for ResAvenue

- Their push URL, and whether it wants an `Authorization` header.
- Whether they will send a `POS` block (or Basic auth) on the booking pull.
- Whether a `Target` other than `Production` is used for testing.
- Whether they will assign an `ID_Context`.
- Whether they need one URL per message or can use the dispatcher.
