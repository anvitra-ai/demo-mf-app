# Mutual Fund Demo App

A sample client for the [mf-atlas.space](https://docs.mf-atlas.space) Mutual Fund
API: register an investor, open an investment account, register an eNACH
mandate, place a purchase order, and pay for it &mdash; walking through the exact
async flow described in [`MF-API-INTEGRATION-GUIDE.md`](./MF-API-INTEGRATION-GUIDE.md).

- **Frontend**: plain HTML/CSS/JS, no build step, no framework.
- **Backend**: Go (stdlib `net/http`), holds the mf-atlas credentials and proxies
  a simplified JSON API to the frontend.
- **Database**: MongoDB, mirrors mf-atlas resources locally so the UI has
  something to read while async registration/settlement is in flight.

There is **no authentication on this app itself** &mdash; it's a local sample. The
only credentials involved are your mf-atlas API credentials, kept server-side.

## What's implemented

- Investors: register, view, live-refresh, retry (`REJECTED` &rarr; retry),
  resubmit-with-corrections, reconciliation ("sync"), NSE confirmation link
  ("auth_link") surfacing.
- Investment accounts: create, list.
- Mandates: register (`ENACH` only), live status refresh, bank-approval link
  surfacing.
- Schemes: search, used to sanity-check an order amount before submitting.
- Orders: place a `LUMPSUM_PURCHASE`, live-refresh status.
- Payments: initiate with mode branching (`MANDATE` / `UPI` / `NETBANKING` /
  `CHEQUE` / `NEFT_RTGS`), UTR submission for `NEFT_RTGS`.
- A webhook receiver (`POST /hooks/mf-atlas`) with `X-MF-Signature` verification
  and `X-MF-Delivery` de-duplication.
- A background reconciliation poller that re-checks any investor / mandate /
  order / payment stuck in a non-terminal state, so the app works correctly
  even without webhooks configured.

**Not implemented** (kept out of scope for this sample): joint holders,
nominees, guardian, demat, FATCA, EUIN fields on investor registration;
document upload; eKYC initiation; physical mandate paper-form upload; SIP /
XSIP / STP / SWP / switch / redemption order types; order cancel/pause; PII
unmask. The nested request shapes for `primary_holder` / `address` /
`bank_accounts` aren't fully published in mf-atlas's docs at the time of
writing, so this app uses a reasonable, commonly-used shape for them &mdash; check
the OpenAPI spec (`https://docs.mf-atlas.space/openapi.yaml`) if a field looks
rejected that shouldn't be.

## Prerequisites

- Go 1.22+
- Docker (to run MongoDB via the provided `docker-compose.yml`) &mdash; or any
  MongoDB instance you already have (local install, Atlas, etc).
- mf-atlas.space credentials: `client_id`, `client_secret`, and the
  sandbox/production **API** base URL from your onboarding contact (this is
  *not* `docs.mf-atlas.space`, which is only the documentation site).

## Setup

1. Copy the env template and fill in your mf-atlas credentials:

   ```bash
   cp .env.example backend/.env
   # edit backend/.env: set MF_ATLAS_BASE_URL, MF_ATLAS_CLIENT_ID, MF_ATLAS_CLIENT_SECRET
   ```

2. Start MongoDB:

   ```bash
   docker compose up -d
   ```

3. Run the backend (it also serves the frontend, so this is the only process
   you need):

   ```bash
   cd backend
   go run .
   ```

4. Open [http://localhost:8080](http://localhost:8080).

Without valid mf-atlas credentials, the server still starts (with a warning
logged), and the UI still loads &mdash; but every action that calls mf-atlas will
show an error banner (`NOT_CONFIGURED` / `UNAUTHORIZED`). This is expected;
fill in `backend/.env` to actually exercise the flows.

## Using the app

The tabs follow the guide's build order:

1. **Investors** &mdash; register one. It's created `PENDING`; NSE registration
   happens in the background. Use **Sync now** or **Refresh status** on the
   detail view to check progress, and **Fetch NSE confirmation link** once
   `REGISTERED` (the investor must open that link with NSE to finish
   onboarding &mdash; this app can only surface it, not complete it for them).
2. **Accounts** &mdash; create an investment account for a `REGISTERED` (or even
   still-`PENDING`) investor.
3. **Mandates** &mdash; register an eNACH mandate; the investor's bank still has
   to approve it (link shown once available).
4. **Schemes** &mdash; search for a scheme code to use when placing an order.
5. **Orders & Payments** &mdash; place a purchase order, then immediately pay for
   it. Payment UI branches by mode: `UPI`/`NETBANKING` show a link to open;
   `NEFT_RTGS` asks for a UTR once you have one; `MANDATE`/`CHEQUE` just wait
   for settlement.

Lists refresh automatically every ~8s so async status changes (driven by the
webhook receiver and/or the reconciliation poller) show up without a manual
click; "Refresh" buttons force an immediate live check against mf-atlas.

## Webhooks (optional)

The app works via the reconciliation poller alone (default: every 20s, see
`RECONCILE_INTERVAL_SECONDS`). If you'd rather see near-instant updates via
webhooks:

1. Expose your local server publicly, e.g. with [ngrok](https://ngrok.com):

   ```bash
   ngrok http 8080
   ```

2. Register a subscription pointing at your tunnel's `/hooks/mf-atlas` path
   (mf-atlas requires a public `https://` URL &mdash; `localhost` is rejected):

   ```bash
   curl -X POST "$MF_ATLAS_BASE_URL/api/webhooks/v1/subscriptions" \
     -H "Authorization: Bearer <token from POST /api/auth/v1/token>" \
     -H "Content-Type: application/json" \
     -d '{
       "url": "https://<your-ngrok-subdomain>.ngrok.app/hooks/mf-atlas",
       "events": ["investor.*", "order.*", "payment.*", "mandate.*"]
     }'
   ```

3. The response's `secret` is shown **once** &mdash; copy it into
   `MF_ATLAS_WEBHOOK_SECRET` in `backend/.env` and restart the backend, so
   incoming deliveries can be signature-verified.

## Project layout

```
backend/            Go module (net/http API + static frontend server)
  internal/
    config/          env var loading
    mfatlas/          mf-atlas API client (auth, resources, webhook signing)
    store/            MongoDB repositories
    models/           local mirror document types
    api/               HTTP handlers (our own JSON API + webhook receiver)
    reconcile/        background polling job
frontend/            vanilla HTML/CSS/JS, no build step
docker-compose.yml    local MongoDB
.env.example
```

## Development notes

- `cd backend && go build ./... && go vet ./...` to check the backend compiles
  cleanly.
- The backend serves `../frontend` by default when run from `backend/` via
  `go run .`; override with the `FRONTEND_DIR` env var if needed.
- All upstream error handling branches on `error.code` (never on
  `error.message`/`provider_remark`), per the integration guide's §6 &mdash; see
  `internal/mfatlas/client.go`.
