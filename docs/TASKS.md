# AirShop Task List

Development tasks derived from the feature scope in the README, ordered by
dependency so the project can be built step by step in spare time.
For the Chinese edition, see [TASKS.zh-CN.md](TASKS.zh-CN.md).

## How to use

- Every task has an ID (`T<milestone>.<n>`, e.g. T0.1); use it when
  referring to a task in commits, notes or conversations.
- Tick a task by turning `[ ]` into `[x]`. The order is a suggestion — just
  respect the **prerequisites** at the top of each milestone.
- Tasks are sized for one spare-time block (roughly 1–3 hours); bigger work
  is broken into subtasks.
- After each task, run `go test ./...`, and jot the date next to the
  milestone once it's done.
- Items marked ❓ are **open decisions**: settle the approach before
  implementing, then record the answer next to the task so it doesn't get
  relitigated.

## Workflow cheatsheet

```bash
airway generate model post         # scaffold a model
airway generate migration create_posts
airway db:migrate                  # run migrations
airway generate api admin          # scaffold an API namespace
just generate                      # regenerate *_templ.go from .templ views
go run .                           # start the server
go run . repl                      # REPL with this project's models
```

Conventions: routes and actions live in `app/api/<name>_api/`, templ views in
`app/views/`, models in `app/models/` (register them for the REPL), business
logic in `app/services/`, migrations in `db/migrate/`.

---

## M0 Foundations

Prerequisites: none (the codebase is the bare scaffold today).

- [x] T0.1 Get local dev running: `airway db:create && airway db:migrate && go run .`, confirm the homepage renders — done 2026-09-28: local Postgres on 127.0.0.1:5432, DB `airshop` created, migrations OK (`db/schema.json` written), homepage HTTP 200 (`/health` 200). DSN passed via `AIRWAY_DSN` (process env wins over `.env`); to make it permanent, set `DSN` in `.env`
- [x] T0.2 ❓ Set shared column conventions: primary key type, `created_at`/`updated_at`, soft deletes or not — settled 2026-09-28, written up in [CONVENTIONS.md](CONVENTIONS.md): auto-increment integer `id` (int64), scaffold-standard timestamps, explicit `updated_at` maintenance, hard delete with status fields as substitute, no DB-level foreign keys
- [x] T0.3 Split storefront and admin layouts: add both under `app/views/layouts/` (header, nav, footer skeletons) — done 2026-09-28: `storefront.templ` (brand header, nav, footer) and `admin.templ` (admin topbar, nav) wrap `Base`; homepage now uses Storefront; covered by `app/views/layouts/layouts_test.go`
- [x] T0.4 Route groups: storefront routes vs `/admin` routes, plus 404/403 fallback pages — done 2026-09-28: `AdminRoutes` group at `/admin` (dashboard placeholder in the admin layout), `NoRoute` renders the storefront 404 page, exported `ForbiddenHandler` renders the 403 page for future auth middleware; covered by `config/routes_test.go`
- [x] T0.5 Fix the money and stock storage format ❓ (recommend storing money as the smallest currency unit, e.g. cents) and apply it to every table from here on — settled 2026-09-28, written up in [CONVENTIONS.md](CONVENTIONS.md): money is BIGINT minor units with a `_cents` suffix (int64 in Go), single store currency, stock is non-negative INTEGER, percentages are integers 0–100

## M1 Catalog

Prerequisites: M0.

### Data layer

- [x] T1.1 Migration: `categories` (name, slug, parent, sort order, enabled) — done 2026-09-28: first migration under the T0.2/T0.5 conventions (`BIGSERIAL` id, `slug` UNIQUE, nullable `parent_id` BIGINT with no FK, `sort_order` INTEGER, `enabled` BOOLEAN, standard timestamps); up/down roundtrip verified against local Postgres via `db:migrate` + `db:rollback`
- [x] T1.2 Migration: `products` (name, slug, description, price, stock, status: active/inactive, main image) — done 2026-09-28: follows T0.2/T0.5 (`price_cents` BIGINT, `stock` INTEGER, `active` BOOLEAN like categories.enabled, `main_image` VARCHAR storage key, `description` TEXT NOT NULL DEFAULT ''); also added nullable indexed `category_id` for T1.9 category filtering; up/down roundtrip verified on local Postgres
- [x] T1.3 ❓ Decide whether SKU variants are needed (`product_variants`: options, own price and stock) — this shapes the schema — settled 2026-09-28 by David Zhang: **variants now**; `product_variants` migration created and verified (own `price_cents`/`stock`, combination `name`, active flag); sellable-unit rule (variants if any active, else the product) lives in the Product service; structured option axes deferred; see [CONVENTIONS.md](CONVENTIONS.md) "Product variants"
- [x] T1.4 Models and REPL registration: `Category`, `Product` (and variants) — done 2026-09-28: three models in `app/models/` following the scaffold style (db/json tags, `TableName()`, REPL registration); nullable `parent_id`/`category_id` as `*int64`; sellable-unit helpers in `app/services/product_service.go` per the T1.3 rule; blank import in main.go wires model registration into the binary; verified end-to-end via `go run . repl` round-tripping sample rows through all three models

### Admin

- [x] T1.5 Category management: list, create, edit, delete — done 2026-09-28: `services/category_service.go` (validation: required name, slug format/uniqueness, parent exists, no self-parent, delete blocked while children exist; `updated_at` maintained per T0.2); server-rendered admin pages under `/admin/categories` in the admin layout; verified end-to-end with live server (create parent+child, edit form prefill, update, duplicate-slug 422, self-parent 422, delete-protection redirect, delete)
- [x] T1.6 Product management: list (paged, filter by name/status), create, edit, activate/deactivate — done 2026-09-28: `services/product_admin_service.go` (name/slug validation with auto-slug from name, category existence, non-negative price/stock; `updated_at` per T0.2) plus `services/money.go` shared price formatting (T0.5); admin pages under `/admin/products` in the admin layout; verified end-to-end (create with auto slug, edit prefill, price update, ILIKE name search, status filter, combined filter, deactivate/activate, duplicate-slug/bad-price/missing-name 422s, pagination with 21 products)
- [ ] T1.6a Variant management: edit variants on the product form (add/remove rows: name, price, stock, active, sort order) — added during T1.6: required by the T1.3 variants decision but missing from the original list
- [x] T1.7 Product image upload (on top of the existing storage; multiple images, first one is the main image) — done 2026-09-28: `product_images` table + model; multipart upload on the product edit page (multi-file, 5 MB cap, jpg/png/webp/gif only) stored via the airway storage layer under `products/`; first image is the main one with `products.main_image` kept in sync (product saves preserve it); make-main reorders and delete cascades promote the next image; verified end-to-end (upload ×2, disk + served URL, make-main, delete-main promotion, delete-last clears main, non-image rejected)

### Storefront

- [x] T1.8 Rework the homepage: product section (featured or latest items) — done 2026-09-29: storefront homepage replaces the scaffold welcome page; latest 8 active products as cards (main-image thumb or placeholder, formatted price, link to `/products/<slug>`); `services.LatestProducts` verified by an env-gated integration test (`AIRWAY_PG_TEST_DSN`, runs the real migrations on a scratch DB); homepage E2E verified (empty state, active-only, newest first, formatted prices)
- [x] T1.9 Product listing page: category filter, pagination — done 2026-09-29: `/products` storefront page (12/page, active only) with category chips (enabled categories only, active chip highlighted, unknown slug falls back to All) and Prev/Next pagination; product cards extracted into a shared `views/catalog.Grid` reused by the homepage; `services.StorefrontListProducts` powers filter + count; E2E verified (all view, category filter, page 2, inactive excluded, disabled category hidden, unknown category fallback)
- [x] T1.10 Product detail page: images, price, stock, description, add-to-cart button (placeholder until M4) — done 2026-09-29: `/products/<slug>` (active products only; unknown or inactive slug renders the storefront 404 page); gallery with main image + thumbnails (CSS placeholder when imageless), price block or variant radios per the sellable-unit rule, stock label, description, disabled add-to-cart button; covered by view tests and E2E (simple product, variant product via seeded rows, both 404 cases)

## M2 Users & auth (phone-number sign-up and login)

Prerequisites: M0.

- [x] T2.1 Migration: `users` (unique phone, password hash, display name, status) — done 2026-09-29: `phone` VARCHAR(20) UNIQUE (E.164 fits), `password_hash` VARCHAR(255) NOT NULL DEFAULT '' (stays empty under a code-only login; algorithm decided in T2.2), `display_name` VARCHAR(255) DEFAULT '', `status` VARCHAR(20) DEFAULT 'active' (active/disabled, room to extend); standard timestamps; up/down roundtrip verified on local Postgres
- [x] T2.2 ❓ Decide the login method — settled 2026-09-29 by David Zhang: **phone number + SMS verification code only, no passwords**. Local dev uses a fake code (fixed value / logged to console); an SMS provider is deferred. The `password_hash` column stays (default empty) for future needs. Column renamed `phone` → `phone_number` in the T2.1 migration per the same decision.
- [x] T2.3 Sign-up, sign-in pages and sign-out action — done 2026-09-29: sign-in/registration merged into one phone+code flow (`/signin` two-step form, new users auto-created as "User <last4>"); `POST /signin/code` (code logged to console per T2.2, in-memory store, 5 min TTL, single-use), `POST /signin`, `POST /signout` (destroys the session row and clears the cookie); session machinery landed early with this task (`sessions` table + token cookie, T2.4's middleware still open); disabled accounts rejected. Covered by service tests and a full E2E (bad phone, wrong code, log-scraped code sign-in, auto sign-up, sign-out, second sign-in reuses the user)
- [x] T2.4 Sessions: cookie sessions, auth middleware (protected pages redirect to login) — done 2026-09-29: session write side landed with T2.3 (`sessions` table + httpOnly token cookie); `middlewares.LoadUser` resolves the cookie on every request, `RequireUser` bounces guests to `/signin?next=<path>` with safe same-site post-login redirect, `/account` protected page (profile + sign-out), storefront nav switches between Sign in and the account link per user. Covered by layouts test (nav states) and E2E (guest redirect, next round-trip, signed-in nav, account page, sign-out).
- [x] T2.5 Form validation and error messages (phone format, duplicate sign-up) — done 2026-09-29: mostly landed with T2.3 (phone normalization/format with user-facing errors, auto sign-up makes duplicate sign-up impossible by design and the unique constraint backs it, error banner on the sign-in page via templ auto-escaping); this task added a dedicated blank-code message (`ErrCodeRequired`) before any store lookup; airway's `lib/validation` (required/email only) was evaluated and passed over in favor of the custom phone validation
- [x] T2.6 Service layer: `AuthService` (sign-up/sign-in/sign-out), actions only bind parameters — done 2026-09-29: all auth logic lives in `services/auth_service.go` (phone normalization, code send/verify with single-use + TTL + pruning, sign-up-or-sign-in, session create/resolve/destroy with expired-row self-cleaning); actions only bind form values and orchestrate redirects. Audited while closing: fixed a real timezone bug (local-time writes vs pgx UTC-labeled reads skewed session expiry up to 8 h — all time writes now use UTC, recorded in CONVENTIONS); full flow covered by an env-gated integration test (`AIRWAY_PG_TEST_DSN`)

## M3 Address book

Prerequisites: M2 (addresses belong to users).

- [x] T3.1 Migration: `addresses` (user ID, recipient, phone, province/city/district, street, is-default) — done 2026-09-29: `user_id` BIGINT NOT NULL with an index (no FK per T0.2), region columns as VARCHAR defaults-empty so T3.3's data-source decision can land later without a migration, `is_default` BOOLEAN; single-default-per-user is enforced in the service layer (T3.2) since partial unique indexes are not portable across the three databases; up/down roundtrip verified on local Postgres
- [x] T3.2 Address CRUD in the account area: list, create, edit, delete, set default — done 2026-09-29: `services/address_service.go` (recipient/phone/street validation, ownership enforced everywhere so cross-user access reads as not-found, first address forced default, explicit default clears the previous one, deleting the default promotes the newest remaining, `updated_at` in UTC per T2.6); `/account/addresses` pages under `RequireUser` (cards with DEFAULT badge, two-step form, region fields as free text until T3.3); env-gated integration test covers default semantics and cross-user isolation; E2E verified (create/list/edit/set-default/delete-promotion, validation 422)
- [ ] T3.3 ❓ Source for province/city/district data (bundle a dataset, or start with three free-text inputs and defer)

## M4 Shopping cart

Prerequisites: M1, M2.

- [ ] T4.1 ❓ Guest cart strategy: signed-in only, or guests too (session-stored, merged on login)?
- [ ] T4.2 Migration: `carts` + `cart_items` (product/variant, quantity, price snapshot at add time)
- [ ] T4.3 Add/update/remove/clear actions and pages
- [ ] T4.4 Cart page: subtotal, stock and availability checks (show unbuyable states)
- [ ] T4.5 Service layer: `CartService` (validation, totals), reused by checkout

## M5 Orders & checkout

Prerequisites: M3, M4.

- [ ] T5.1 Migrations: `orders` (order number, user, address snapshot, subtotal/discount/shipping/total, status, payment method) + `order_items` (product snapshot: name, image, unit price, quantity)
- [ ] T5.2 Checkout page: choose address → choose payment method → place order (amounts recomputed server-side; never trust the client)
- [ ] T5.3 Order state machine: `pending → paid → shipped → completed`, plus `cancelled / refunded`; keep transitions in the service layer
- [ ] T5.4 ❓ Stock deduction strategy (reserve at order time and restore on cancel, or deduct after payment) — record the answer next to the state machine
- [ ] T5.5 Account area: order list (filter by status), order detail
- [ ] T5.6 Cancel order (pending only), auto-cancel expired unpaid orders (scheduled task)

## M6 Payments

Prerequisites: M5.

- [ ] T6.1 Gateway abstraction: `PaymentGateway` interface + a registry
- [ ] T6.2 Fake gateway (one click marks the order paid) so the full flow is testable locally
- [ ] T6.3 ❓ Real channels: Alipay / WeChat Pay / Stripe — pick per target market, then break into tasks
- [ ] T6.4 Payment callbacks: verify signatures, idempotency, update order status
- [ ] T6.5 Cashier redirect and result pages (success/failure)

## M7 Coupons

Prerequisites: M5 (money math already centralized in the service layer).

- [ ] T7.1 Migrations: `coupons` (type: fixed amount / percentage, threshold, value, validity window, total count) + `coupon_redemptions` (per user and order, prevents double use)
- [ ] T7.2 Admin coupon management: create, list, disable
- [ ] T7.3 Apply a coupon at checkout (pick one or enter a code), integrated into order total calculation
- [ ] T7.4 ❓ Coupon refund policy on cancellation/refund
- [ ] T7.5 (Optional) coupon center for users to claim

## M8 Shipment tracking

Prerequisites: M5 (shipments attach to orders).

- [ ] T8.1 Migration: `shipments` (order, carrier, tracking number, status, track events JSON)
- [ ] T8.2 Admin shipping: enter carrier + tracking number, order moves to `shipped`
- [ ] T8.3 ❓ Track-event source: a tracking service (Kuaidi100 / AfterShip / …, pick by availability) or manual entry first — break into tasks once decided
- [ ] T8.4 Account area: order shipment page (timeline of events)

## M9 Admin consolidation

Prerequisites: the admin pages from M1–M8 in place.

- [ ] T9.1 ❓ Admin accounts: separate `admin_users` table vs a role column on users; admin login and auth middleware
- [ ] T9.2 Dashboard: orders and sales for today/recent, pending work (to ship, to refund)
- [ ] T9.3 Order management: list (status filter, search), detail, ship action, refund action
- [ ] T9.4 Unify admin UX across modules (pagination, filters, empty states, confirm dialogs)
- [ ] T9.5 ❓ Audit trail if needed: `admin_logs`

## M10 Polish & release

Prerequisites: everything above.

- [ ] T10.1 Seed data: demo categories/products and an admin account (`go run . repl` or a seed command)
- [ ] T10.2 Full-flow regression: sign up → browse → add to cart → checkout → pay (fake gateway) → ship in admin → track on storefront → complete
- [ ] T10.3 Error handling and logging review (5xx pages, logs on critical paths)
- [ ] T10.4 `go test ./...` green; cover the core services (money math, state machine)
- [ ] T10.5 Desktop packaging check: `airway desktop:init` → `wails3 task dev`
- [ ] T10.6 Update the README feature list to reflect what ships

---

## Deferred (Phase 2/3, not planned yet)

- AirShop-specific features (Phase 2, planned after the core system is complete)
- AI features (Phase 3, to be planned)

## Progress log

| Milestone | Done on | Notes |
|---|---|---|
| | | |
