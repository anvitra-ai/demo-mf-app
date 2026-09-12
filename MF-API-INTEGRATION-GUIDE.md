> ## Documentation Index
> Fetch the complete documentation index at: https://docs.mf-atlas.space/llms.txt
> Use this file to discover all available pages before exploring further.

# MF API INTEGRATION GUIDE

# Mutual Fund API — Integration Guide (for AI coding agents)

This document is written to be handed directly to an AI coding agent that will
implement a client integration against the Mutual Fund API. It describes the
**end-to-end flow**, the exact request/response shapes, and — most
importantly — the **asynchronous / multi-step nuances** that are easy to get
wrong if you assume any of these calls complete synchronously.

If you are the implementing agent: read the whole document before writing
code. The three flows in "Critical async nuances" below are the ones that
cause the most integration bugs — do not skip them.

## Documentation & spec links

This guide is a summary — use these as the source of truth for anything not
covered here, or if a field/status looks like it might have changed since
this was written:

| Resource                | Link                                                     | Use it for                                                                                                                                                                                                                                                              |
| ----------------------- | -------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Full documentation site | `https://docs.mf-atlas.space/`                           | Prose explanations, field-by-field tables, Mermaid sequence/state diagrams per resource. Start at `/api-documentation/prerequisites`.                                                                                                                                   |
| **OpenAPI spec**        | `https://docs.mf-atlas.space/openapi.yaml`               | Machine-readable schema for every customer-facing endpoint — feed this directly to an AI agent or a codegen tool (e.g. `openapi-generator`, `orval`) to generate typed request/response models and a client SDK instead of hand-transcribing shapes from this document. |
| Webhooks reference      | `https://docs.mf-atlas.space/api-documentation/webhooks` | Full event catalog, signature verification, retry/backoff schedule — §5 below is a summary of this page.                                                                                                                                                                |

<sub>This is the documentation host, not the API host — it doesn't take your
bearer token. Your actual `POST /api/auth/v1/token` calls go to the
sandbox/production API base URL your onboarding contact gave you (see §0
below), which is a separate host from the docs site above.</sub>

***

## 0. Conventions that apply to every call

* **Base URL**: `https://<env-host>/api/<resource>/v1` — each resource group
  is versioned independently (`/api/investors/v1`, `/api/orders/v1`, etc).
* **Auth**: `POST /api/auth/v1/token` with `{ client_id, client_secret }` →
  `access_token` (Bearer), valid for `expires_in` seconds. No refresh token —
  request a new one before expiry. Send it as `Authorization: Bearer <token>`
  on every other call.
* **Response envelope**:
  * Success: `{ "success": true, "data": { ... }, "message": "" }`
  * Error: `{ "success": false, "error": { "code": "...", "message": "...", "details": [...], "provider_remark": "...", "request_id": "..." } }`
  * **Branch on `error.code`, never on `error.message` or `provider_remark`** —
    those are human text and can change without notice.
* **Idempotency**: send `Idempotency-Key: <uuid>` on write calls, required in
  practice on `POST /api/orders/v1/`. Replaying the same key + body returns
  the original response with `Idempotency-Replayed: true`; same key with a
  different body → `409 IDEMPOTENCY_KEY_REUSED`.
* **Pagination**: list endpoints take `?limit=&cursor=`, return
  `{ data: [...], next_cursor: "..." }`. No `next_cursor` = last page.
* **PII**: `GET` responses always mask PAN/bank account numbers. See the PII
  masking doc if you need to reveal one field for support/verification UI.
* **IDs are friendly, not UUIDs**: `I_SI_IND_000042` (investor),
  `A_RT_000001` (investment account), `bnk_...` (bank account), `mnd_...`
  (mandate), `ord_...` (order), `pay_...` (payment). Store these as your
  foreign keys — never store or depend on the exchange's own internal codes.

## 1. High-level flow

```
1. Get a token                          (sync)
2. Create investor                      (ASYNC — see §2)
3. Create investment account            (sync, needs investor to exist — not necessarily REGISTERED)
4. Create mandate (optional, for auto-debit)  (needs verification — see §3)
5. Create purchase order                (ASYNC dispatch — see §4)
6. Create payment for the order         (mode-dependent settlement — see §4)
7. Poll or (preferably) receive webhooks for every async step above
```

Steps 2–6 each return **before** the underlying work is finished. Your
integration must be built around polling and/or webhooks from day one — do
not design it as if any of these calls are synchronous request/response with
a final result in the HTTP response body.

***

## 2. Critical nuance #1 — Investor creation is asynchronous, and ends with a link the investor must open

`POST /api/investors/v1/` returns `201` **immediately** with
`status: "PENDING"`. Registering the Unique Client Code (UCC) with the
exchange happens in the background. **You cannot place an order for this
investor yet** — an investor must reach `status: "REGISTERED"` first.

### 2.1 Lifecycle

```
DRAFT --(submit:true)--> PENDING --> PROCESSING --> REGISTERED
                                            \--> REJECTED --(fix + resubmit)--> PENDING
```

* `DRAFT` — you passed `"submit": false` (e.g. a form still being filled in).
  Nothing sent to the exchange.
* `PENDING` — queued for the exchange.
* `PROCESSING` — exchange call in flight.
* `REGISTERED` — UCC registered. Terminal-success state; orders are now
  possible.
* `REJECTED` — **not discarded**. Not terminal. Fix the data and resubmit.
  Reason is in `provider_remark` / the last `provider_steps` entry.

### 2.2 How to find out registration finished

Two mechanisms — implement both, webhook first, poll as a fallback:

1. **Webhook** (preferred): `investor.registered` / `investor.rejected`.
2. **Poll**: `GET /api/investors/v1/:id`, inspect `data.status`.

Do not assume `REGISTERED` arrives within any fixed time window — build this
as an event-driven state transition in your own data model, not a blocking
wait after the `POST`.

### 2.3 The confirmation link (the actual "link submission" nuance)

Once UCC registration succeeds, **NSE emails/SMSes the investor a
confirmation link directly** — this happens outside your system, you do not
control it, and you cannot skip it. The investor must open that link and
complete whatever confirmation NSE requires there.

If you'd rather surface that link inside your own product instead of relying
on the investor to find NSE's email/SMS, fetch it yourself:

```
GET /api/investors/v1/:id?auth_link=true
```

```json theme={null}
{ "data": { "id": "I_SI_IND_000042", "status": "REGISTERED",
            "auth_link": "https://nseinvest.com/mf/51ae3912" } }
```

Notes your implementation must respect:

* `auth_link` is fetched **live** from NSE on every call with the flag set —
  it is never stored on our side. Don't cache it beyond the current response.
* It is only populated once `status` is `REGISTERED`. Calling with the flag
  before then simply omits the field — don't treat that as an error.
* Build a UI/notification step that surfaces this link to the investor (or
  confirms they received NSE's own email/SMS) as part of onboarding — this is
  a real manual step in the flow, not just an API nuance.

### 2.4 Rejection → retry

* If the **data was wrong**: `PATCH /api/investors/v1/:id` with corrected
  fields + `"submit": true` → re-queues as a fresh registration attempt.
* If the **data was fine** and the failure was transient (exchange
  temporarily unavailable): `POST /api/investors/v1/:id/retry` (no body) —
  re-sends the stored request as-is. `409` if already `REGISTERED`.

### 2.5 Bank account status is a second, independent async status

Each `bank_accounts[]` entry has its own `status` (`ADDED` → `PENDING` →
whatever NSE settles on), tracked **separately** from the investor's overall
UCC `status`. An investor can be `REGISTERED` while a specific bank account is
still `PENDING` in NSE's back office. If your flow depends on a specific bank
account being usable (e.g. before creating a mandate against it), don't infer
that from investor `status` alone — check that account's own `status`, and
call `POST /api/investors/v1/sync` if you suspect it's stale.

### 2.6 Minimum implementation checklist for this step

* [ ] Persist investor locally with your own status field mirroring ours.
* [ ] Subscribe to `investor.registered` / `investor.rejected` webhooks.
* [ ] Fallback poll job for investors stuck `PENDING`/`PROCESSING` beyond a
  sane timeout (also covers lost webhook deliveries — or call
  `POST /api/investors/v1/sync` to reconcile against NSE directly).
* [ ] Surface the NSE confirmation link/email step to the investor as a
  required manual action before considering onboarding "done".
* [ ] Handle `REJECTED` as a correctable state in your UI, not a dead end.

***

## 3. Critical nuance #2 — Mandate creation needs investor confirmation before it's usable

A mandate authorizes the exchange to auto-debit an investor's bank account up
to a ceiling `amount`. Creating one is `POST /api/mandates/v1/`, but **the
mandate is not necessarily active the instant you create it** — it depends on
`type`:

| `type`     | On create                                         | Becomes usable when                                                                                                                                             |
| ---------- | ------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ENACH`    | `REGISTERED` (or `REJECTED` if declined outright) | Immediately, once `REGISTERED` — **but** the bank still has to approve the eNACH registration; NSE also emails/SMSes the investor a confirmation link for this. |
| `PHYSICAL` | `PENDING`                                         | You collect a signed paper form, scan it, and `POST /api/mandates/v1/:id/image`; on acceptance it moves to `REGISTERED`.                                        |

### 3.1 The verification link, again

Just like investor UCC confirmation, mandate creation returns (best-effort,
may be absent) an `auth_link`:

```json theme={null}
{ "data": { "id": "mnd_4a1b2c", "status": "REGISTERED",
            "auth_link": "https://nseinvest.com/mf/51ae3912" } }
```

* Absent for `PHYSICAL` mandates, or if it just isn't ready yet at creation
  time.
* If missing, poll `GET /api/mandates/v1/:id` — it retries fetching the link
  while it's outstanding.
* Redirect the investor to this link so **they** can approve the eNACH
  registration with their bank — this is investor action you cannot automate
  away. Treat "mandate created" and "mandate approved by investor's bank" as
  two different milestones in your own UI/state machine.

### 3.2 Don't use a mandate until you've confirmed it's active

Before paying an order with a mandate, the mandate must be `REGISTERED` *and*
its ceiling must cover the payment. Two checks happen server-side at payment
time and will reject you if you skip client-side verification:

| Condition                               | Response                         |
| --------------------------------------- | -------------------------------- |
| Mandate not `REGISTERED`                | `422 MANDATE_NOT_ACTIVE`         |
| Payment total exceeds mandate `amount`  | `422 INSUFFICIENT_MANDATE_LIMIT` |
| Mandate belongs to a different investor | `400 VALIDATION_FAILED`          |

`GET /api/mandates/v1/:id` **refreshes status live from the exchange** before
returning — always poll this endpoint rather than trusting a locally cached
status when you're about to use the mandate for a payment.

### 3.3 Minimum implementation checklist for this step

* [ ] After creating an `ENACH` mandate, surface `auth_link` (or poll for it)
  and drive the investor to approve it with their bank.
* [ ] Track mandate status independently from investor status — don't gate
  UI on investor `REGISTERED` alone.
* [ ] Before submitting a mandate-funded payment, `GET` the mandate fresh (not
  from local cache) to confirm `REGISTERED` and sufficient `amount`.
* [ ] For `PHYSICAL` mandates, build the upload step
  (`POST /api/mandates/v1/:id/image`) and handle its own accept/reject
  (`422 UCC_REJECTED`) outcome.

***

## 4. Critical nuance #3 — Purchase order creation is two calls: order, then payment (which itself branches by mode)

Placing a lumpsum purchase is **always two separate API calls**, not one:

```
Step 1: POST /api/orders/v1/    → creates + validates + queues the order (202)
Step 2: POST /api/payments/v1/  → references the order(s) and settles funds
```

An order with no payment against it never gets units allotted. Do not treat
step 1's `202` as "purchase complete."

### 4.0 Look up the scheme first

`GET /api/schemes/v1/:code` (or `GET /api/schemes/v1/?search=...` to find the
code) returns whether the scheme currently allows the transaction type you
intend (`purchase_allowed`, `sip_allowed`, ...), plus `min_purchase`,
`purchase_multiple`, `min_additional` and `min_redemption_units` — validate
the order amount client-side against these before submitting, since an
unknown or disallowed `scheme_code` is rejected synchronously with
`400 VALIDATION_FAILED`. Served entirely from ingested master data, so it's
cheap to call on every quote/checkout screen render.

### 4.1 Order creation is itself async

```json theme={null}
POST /api/orders/v1/
{
  "order_type": "LUMPSUM_PURCHASE",
  "investor_id": "I_SI_IND_000042",
  "investment_account_id": "A_RT_000001",
  "scheme_code": "128SDGP",
  "amount": 25000,
  "purchase_type": "FRESH",
  "bank_account_id": "bnk_9f21c8",
  "mandate_id": "mnd_4a1b2c",
  "client_ref": "ord-5521"
}
```

Returns `202` with `status: "PENDING"` the moment it's validated and
persisted — dispatch to the exchange happens in the background:

```
PENDING → SUBMITTED → ACCEPTED → ALLOTTED
              \-> REJECTED / FAILED         (CANCELLED possible from PENDING/SUBMITTED/ACCEPTED)
```

`ALLOTTED` is set later still, by an asynchronous status-sync job that polls
the exchange and fills in `nav`, `units`, `allotment_date` — this can lag
well behind order acceptance. Poll `GET /api/orders/v1/:id` or listen for
`order.accepted` / `order.rejected` / `order.failed` webhooks; check
`GET /api/orders/v1/:id/events` for the full transition trail if you need to
show history.

Prerequisites that must already be true or the order is rejected
synchronously before the `202`:

* Investor must be `REGISTERED` (else `422 KYC_INCOMPLETE`).
* `investment_account_id` must exist and belong to the investor.
* If `mandate_id` given, it's checked for usability immediately
  (`422 MANDATE_NOT_ACTIVE` / `INSUFFICIENT_MANDATE_LIMIT`) — this is a
  preview check, not a substitute for the payment-time check in §3.2.

Always send `Idempotency-Key` on this call — it's the one endpoint in the API
where idempotency is actively enforced, because retrying a failed request
without it risks a duplicate purchase.

### 4.2 Payment settlement branches by `mode` — only one is "fire and forget"

```json theme={null}
POST /api/payments/v1/
{
  "investor_id": "I_SI_IND_000042",
  "investment_account_id": "A_RT_000001",
  "order_ids": ["ord_1a2b3c4d"],
  "mode": "MANDATE",
  "mandate_id": "mnd_4a1b2c"
}
```

| `mode`               | What you must do after `201 INITIATED`                                                                                                                                                   |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `MANDATE`            | Nothing — auto-debit is triggered against the registered mandate. Requires the mandate to already be `REGISTERED` (§3).                                                                  |
| `UPI` / `NETBANKING` | Response includes `payment_url` — **you must redirect/hand this to the investor** to complete payment on a hosted page. Nothing settles until they do.                                   |
| `CHEQUE`             | Physical cheque details registered (`cheque: { number, date }`, date within T−90…T+3 days); funds clear out of band — no further API call needed from you, but settlement isn't instant. |
| `NEFT_RTGS`          | Investor transfers funds outside the API; **you must then call** `POST /api/payments/v1/:id/utr` with the bank's UTR once available, or settlement never completes.                      |

Regardless of mode, final settlement is asynchronous and arrives via:

* Webhooks `payment.captured` / `payment.failed` / `payment.refunded`, and
* Each linked order's `payment_status` field (`INITIATED → CAPTURED / FAILED / REFUNDED`)
  updates in step — poll `GET /api/orders/v1/:id` or `GET /api/payments/v1/:id`
  if you're not on webhooks yet.

Only once payment is `CAPTURED` does NAV get applied and units allotted
(order moves to `ALLOTTED`).

### 4.3 Minimum implementation checklist for this step

* [ ] Never treat `POST /api/orders/v1/`'s `202` as final — track `status`
  through to `ALLOTTED`/`REJECTED`/`FAILED` via webhook + poll fallback.
* [ ] Always send a unique `Idempotency-Key` per logical order submission;
  reuse the same key only when deliberately retrying the identical
  request.
* [ ] Branch your post-payment UI/flow by `mode`:
  * `MANDATE`/`CHEQUE` → just wait for the webhook/poll.
  * `UPI`/`NETBANKING` → redirect the investor to `payment_url` and only then
    wait.
  * `NEFT_RTGS` → collect the UTR from the investor and call
    `POST /api/payments/v1/:id/utr` yourself, or the payment never settles.
* [ ] Subscribe to `payment.captured`/`payment.failed`/`payment.refunded` and
  `order.accepted`/`order.rejected`/`order.failed`.
* [ ] Before defaulting to `mode: MANDATE`, re-fetch the mandate (§3.2) to
  confirm it's actually `REGISTERED` and its `amount` covers this order —
  don't rely on a status you cached earlier in the session.

***

## 5. Webhooks vs polling — recommended pattern

All three flows above (investor, mandate, order+payment) share the same
shape: an initial synchronous acknowledgement, then an asynchronous outcome.
Build one reusable mechanism rather than three.

### 5.1 Subscribe

```json theme={null}
POST /api/webhooks/v1/subscriptions
{ "url": "https://yourapp.example.com/hooks/mf-atlas",
  "events": ["investor.*", "order.*", "payment.*", "mandate.*"] }
```

Returns `{ id, url, events, secret }` — `secret` is shown **once**, at
creation; store it, it's what you use to verify `X-MF-Signature` on every
delivery and cannot be retrieved later. `url` must be `https://` and resolve
to a public address (no `localhost`/private/loopback ranges — enforced at
subscribe time and again on every delivery).

### 5.2 Event catalog

| Event                                                                   | `data` shape                               |
| ----------------------------------------------------------------------- | ------------------------------------------ |
| `investor.registered` / `investor.rejected`                             | `{ investor_id, status, provider_remark }` |
| `mandate.registered` / `mandate.rejected` / `mandate.cancelled`         | `{ mandate_id, status }`                   |
| `order.accepted` / `order.rejected` / `order.failed` / `order.allotted` | `{ order_id, status, provider_remark }`    |
| `payment.captured` / `payment.failed` / `payment.refunded`              | `{ payment_id, status }`                   |

Every payload is the usual `{ success, data }` envelope. Treat it as a
trigger to `GET /api/.../:id` for the authoritative record — it is
intentionally thin, not the full resource.

### 5.3 Delivery, signing, retries

Each delivery is a `POST` to your `url` with headers `X-MF-Event` (event
name), `X-MF-Delivery` (unique per attempt — use it to de-duplicate),
`X-MF-Signature` (`t=<unix>,v1=<hex HMAC-SHA256("<t>.<raw body>", secret)>`).
Respond `2xx` within 10s or it's treated as a failure.

Failed deliveries retry with backoff — **1m, 5m, 25m, 2h, 12h** — then are
marked `PARKED` (no further automatic retries; replay manually with
`POST /api/webhooks/v1/deliveries/:id/retry`). Delivery is **at-least-once**
and can arrive out of order — de-duplicate on `X-MF-Delivery`, and always
trust the resource's own `status` field over delivery order.

```
Node.js verification:
const [tPart, vPart] = header.split(",");
const t = tPart.split("=")[1], sig = vPart.split("=")[1];
const expected = crypto.createHmac("sha256", secret)
  .update(`${t}.${rawBody}`).digest("hex");
crypto.timingSafeEqual(Buffer.from(sig), Buffer.from(expected)); // must use the RAW body
```

### 5.4 Build pattern

1. Register one webhook endpoint that dispatches by `X-MF-Event`.
2. On each event, update the local record's status and trigger whatever
   downstream action depends on it (e.g. unlock "place order" once investor
   is `REGISTERED`; unlock "mark account funded" once payment is
   `CAPTURED`).
3. Run a low-frequency reconciliation poll (e.g. every 15–30 min) over any
   record still in a non-terminal status past a reasonable SLA, in case a
   webhook delivery was ever `PARKED` or lost. For investors, prefer
   `POST /api/investors/v1/sync` for bulk reconciliation over N individual
   `GET`s; for other resources, list `GET /api/webhooks/v1/deliveries?status=PARKED`
   and replay.
4. Never build a synchronous "wait and block" loop against these endpoints —
   design your UI to show a pending/processing state and update it when the
   webhook/poll resolves.

Full reference: [Webhooks](https://docs.mf-atlas.space/api-documentation/webhooks) (see "Documentation & spec links" at the top of this file).

***

## 6. Error handling

Branch only on `error.code` (see the table below); `error.message` and
`provider_remark` are human strings for logs/support tickets, not for control
flow.

| `code`                       | HTTP | Meaning                                                       |
| ---------------------------- | ---- | ------------------------------------------------------------- |
| `VALIDATION_FAILED`          | 400  | Request shape or business rule failed locally.                |
| `UNAUTHORIZED`               | 401  | Missing / invalid / expired token.                            |
| `FORBIDDEN`                  | 403  | Valid token, missing scope.                                   |
| `NOT_PROVISIONED`            | 403  | No active account for these credentials.                      |
| `NOT_FOUND`                  | 404  | Not found within your account.                                |
| `IDEMPOTENCY_KEY_REUSED`     | 409  | Same key, different body.                                     |
| `INVALID_STATE`              | 409  | Illegal state transition (e.g. cancelling an allotted order). |
| `INVALID_SCHEME`             | 422  | Scheme doesn't allow this transaction.                        |
| `KYC_INCOMPLETE`             | 422  | Investor is not `REGISTERED` yet.                             |
| `MANDATE_NOT_ACTIVE`         | 422  | Mandate isn't usable.                                         |
| `INSUFFICIENT_MANDATE_LIMIT` | 422  | Debit exceeds the mandate ceiling.                            |
| `UCC_REJECTED`               | 422  | The exchange rejected an investor/mandate registration.       |
| `ORDER_REJECTED`             | 422  | The exchange rejected the order.                              |
| `PAYMENT_REJECTED`           | 422  | The exchange rejected the payment.                            |
| `RATE_LIMITED`               | 429  | Back off; `Retry-After` is set.                               |
| `PROVIDER_UNAVAILABLE`       | 502  | Exchange unreachable or erroring; safe to retry.              |
| `INTERNAL`                   | 500  | Unexpected server error.                                      |

Retry policy: `PROVIDER_UNAVAILABLE` and `RATE_LIMITED` (respecting
`Retry-After`) are safe to retry with backoff. Anything else (`VALIDATION_FAILED`,
`*_REJECTED`, `KYC_INCOMPLETE`, etc.) requires fixing the input or the
upstream state (e.g. waiting for investor registration) before retrying —
don't blind-retry these.

***

## 7. Suggested build order for the implementing agent

0. Fetch `https://docs.mf-atlas.space/openapi.yaml` (see "Documentation & spec
   links" above) and generate request/response types or a client stub from it
   before writing any hand-rolled structs — it's kept in sync with the live
   API on every deploy, this document is not.
1. Token exchange + auth wrapper (attach bearer, refresh before expiry).
2. Investor create/read/patch/retry + webhook handlers + reconciliation poll
   (§2). Build and test the `REJECTED → fix → resubmit` loop explicitly.
3. Investment account create/read (simple, synchronous, low risk).
4. Mandate create/read + `auth_link` surfacing + physical-mandate image
   upload (§3).
5. Order create + payment create, with the mode-branching logic in §4.2 and
   full webhook coverage for both resources.
6. Idempotency-key generation/storage on every order submission.
7. Central error-code → retry/no-retry policy (§6), reused everywhere.

Do not consider the integration "done" after step 5 compiles and returns
`201`/`202` — every flow above ends with an asynchronous, often
investor-facing, confirmation step. The integration is only correct once
those confirmations are wired to real state transitions in your own system.
