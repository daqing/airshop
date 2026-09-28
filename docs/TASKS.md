# AirShop Task List

Development tasks derived from the feature scope in the README, ordered by
dependency so the project can be built step by step in spare time.
For the Chinese edition, see [TASKS.zh-CN.md](TASKS.zh-CN.md).

## How to use

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

- [ ] Get local dev running: `airway db:create && airway db:migrate && go run .`, confirm the homepage renders
- [ ] ❓ Set shared column conventions: primary key type, `created_at`/`updated_at`, soft deletes or not — write them down here or in CONTRIBUTING
- [ ] Split storefront and admin layouts: add both under `app/views/layouts/` (header, nav, footer skeletons)
- [ ] Route groups: storefront routes vs `/admin` routes, plus 404/403 fallback pages
- [ ] Fix the money and stock storage format ❓ (recommend storing money as the smallest currency unit, e.g. cents) and apply it to every table from here on

## M1 Catalog

Prerequisites: M0.

### Data layer

- [ ] Migration: `categories` (name, slug, parent, sort order, enabled)
- [ ] Migration: `products` (name, slug, description, price, stock, status: active/inactive, main image)
- [ ] ❓ Decide whether SKU variants are needed (`product_variants`: options, own price and stock) — this shapes the schema
- [ ] Models and REPL registration: `Category`, `Product` (and variants)

### Admin

- [ ] Category management: list, create, edit, delete
- [ ] Product management: list (paged, filter by name/status), create, edit, activate/deactivate
- [ ] Product image upload (on top of the existing storage; multiple images, first one is the main image)

### Storefront

- [ ] Rework the homepage: product section (featured or latest items)
- [ ] Product listing page: category filter, pagination
- [ ] Product detail page: images, price, stock, description, add-to-cart button (placeholder until M4)

## M2 Users & auth (phone-number sign-up and login)

Prerequisites: M0.

- [ ] Migration: `users` (unique phone, password hash, display name, status)
- [ ] ❓ Decide the login method: password + SMS code, or code only? Start with a fake code locally (fixed value / logged to console); pick an SMS provider later
- [ ] Sign-up, sign-in pages and sign-out action
- [ ] Sessions: cookie sessions, auth middleware (protected pages redirect to login)
- [ ] Form validation and error messages (phone format, duplicate sign-up)
- [ ] Service layer: `AuthService` (sign-up/sign-in/sign-out), actions only bind parameters

## M3 Address book

Prerequisites: M2 (addresses belong to users).

- [ ] Migration: `addresses` (user ID, recipient, phone, province/city/district, street, is-default)
- [ ] Address CRUD in the account area: list, create, edit, delete, set default
- [ ] ❓ Source for province/city/district data (bundle a dataset, or start with three free-text inputs and defer)

## M4 Shopping cart

Prerequisites: M1, M2.

- [ ] ❓ Guest cart strategy: signed-in only, or guests too (session-stored, merged on login)?
- [ ] Migration: `carts` + `cart_items` (product/variant, quantity, price snapshot at add time)
- [ ] Add/update/remove/clear actions and pages
- [ ] Cart page: subtotal, stock and availability checks (show unbuyable states)
- [ ] Service layer: `CartService` (validation, totals), reused by checkout

## M5 Orders & checkout

Prerequisites: M3, M4.

- [ ] Migrations: `orders` (order number, user, address snapshot, subtotal/discount/shipping/total, status, payment method) + `order_items` (product snapshot: name, image, unit price, quantity)
- [ ] Checkout page: choose address → choose payment method → place order (amounts recomputed server-side; never trust the client)
- [ ] Order state machine: `pending → paid → shipped → completed`, plus `cancelled / refunded`; keep transitions in the service layer
- [ ] ❓ Stock deduction strategy (reserve at order time and restore on cancel, or deduct after payment) — record the answer next to the state machine
- [ ] Account area: order list (filter by status), order detail
- [ ] Cancel order (pending only), auto-cancel expired unpaid orders (scheduled task)

## M6 Payments

Prerequisites: M5.

- [ ] Gateway abstraction: `PaymentGateway` interface + a registry
- [ ] Fake gateway (one click marks the order paid) so the full flow is testable locally
- [ ] ❓ Real channels: Alipay / WeChat Pay / Stripe — pick per target market, then break into tasks
- [ ] Payment callbacks: verify signatures, idempotency, update order status
- [ ] Cashier redirect and result pages (success/failure)

## M7 Coupons

Prerequisites: M5 (money math already centralized in the service layer).

- [ ] Migrations: `coupons` (type: fixed amount / percentage, threshold, value, validity window, total count) + `coupon_redemptions` (per user and order, prevents double use)
- [ ] Admin coupon management: create, list, disable
- [ ] Apply a coupon at checkout (pick one or enter a code), integrated into order total calculation
- [ ] ❓ Coupon refund policy on cancellation/refund
- [ ] (Optional) coupon center for users to claim

## M8 Shipment tracking

Prerequisites: M5 (shipments attach to orders).

- [ ] Migration: `shipments` (order, carrier, tracking number, status, track events JSON)
- [ ] Admin shipping: enter carrier + tracking number, order moves to `shipped`
- [ ] ❓ Track-event source: a tracking service (Kuaidi100 / AfterShip / …, pick by availability) or manual entry first — break into tasks once decided
- [ ] Account area: order shipment page (timeline of events)

## M9 Admin consolidation

Prerequisites: the admin pages from M1–M8 in place.

- [ ] ❓ Admin accounts: separate `admin_users` table vs a role column on users; admin login and auth middleware
- [ ] Dashboard: orders and sales for today/recent, pending work (to ship, to refund)
- [ ] Order management: list (status filter, search), detail, ship action, refund action
- [ ] Unify admin UX across modules (pagination, filters, empty states, confirm dialogs)
- [ ] ❓ Audit trail if needed: `admin_logs`

## M10 Polish & release

Prerequisites: everything above.

- [ ] Seed data: demo categories/products and an admin account (`go run . repl` or a seed command)
- [ ] Full-flow regression: sign up → browse → add to cart → checkout → pay (fake gateway) → ship in admin → track on storefront → complete
- [ ] Error handling and logging review (5xx pages, logs on critical paths)
- [ ] `go test ./...` green; cover the core services (money math, state machine)
- [ ] Desktop packaging check: `airway desktop:init` → `wails3 task dev`
- [ ] Update the README feature list to reflect what ships

---

## Deferred (Phase 2/3, not planned yet)

- AirShop-specific features (Phase 2, planned after the core system is complete)
- AI features (Phase 3, to be planned)

## Progress log

| Milestone | Done on | Notes |
|---|---|---|
| | | |
