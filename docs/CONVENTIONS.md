# AirShop Conventions

Shared schema and column conventions for every migration and model, decided
in T0.2. They follow the Airway v0.18 scaffold behavior — where the framework
already dictates a pattern we adopt it instead of inventing our own.
For the Chinese edition, see [CONVENTIONS.zh-CN.md](CONVENTIONS.zh-CN.md).

## Primary keys

- Every table has a single surrogate key `id`, auto-increment integer:
  `BIGSERIAL PRIMARY KEY` on Postgres, `BIGINT NOT NULL AUTO_INCREMENT
  PRIMARY KEY` on MySQL, `INTEGER PRIMARY KEY AUTOINCREMENT` on SQLite
  (this is exactly what `airway generate model` emits per dialect).
- The Go type is `airwaysql.IdType` (`int64`) everywhere, and ids appear as
  integers in URLs and OpenAPI path parameters. No UUID primary keys.

## Timestamps

- Every table ends with the same two columns, as emitted by the scaffold:

  ```sql
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
  ```

- Every model carries `CreatedAt time.Time` and `UpdatedAt time.Time` with
  `db:"created_at"` / `db:"updated_at"` and `json:"created_at"` /
  `json:"updated_at"` tags.
- **Write time values with `time.Now().UTC()`.** The columns are
  `TIMESTAMP WITHOUT TIME ZONE`: pgx reads them back labeled UTC while a
  bare `time.Now()` writes the local wall clock (UTC+8 here), which shifts
  every expiry/comparison by up to eight hours. Go-side writes must use UTC
  so the stored wall clock and the read-back label agree.
- **Comparing against database-default timestamps needs the local wall
  clock.** Columns filled by `CURRENT_TIMESTAMP` (e.g. `created_at`) store
  the server-local wall time, so SQL-side comparisons (like the unpaid-order
  expiry sweep) must compute their cutoff with a bare `time.Now()`, while
  Go-side comparisons of service-written values use UTC as above.

## updated_at maintenance

The repo layer never touches `updated_at`, and the scaffolded `UpdateAction`
does not set it either — left alone the column goes stale. Therefore:

- Every `repo.UpdateByID` call must explicitly include
  `"updated_at": time.Now()`.
- Updates are centralized in the service layer wherever possible so this
  rule is enforced in one place per aggregate, not at every action.
- When scaffolding a real resource, add the `updated_at` key to the
  generated `UpdateAction` before shipping it.

## Naming

- Table names are plural snake_case (`products`, `order_items`), matching
  what `TableName()` returns in generated models.
- Columns are snake_case. Foreign-key columns are named `<entity>_id`
  (`user_id`, `order_id`) and typed `BIGINT`.

## Deletes

- **Hard delete is the default.** `repo.DeleteByID` removes the row; no
  soft-delete columns anywhere at first.
- Business needs that soft delete usually covers are handled instead by:
  - status fields — e.g. products are deactivated (`active → inactive`),
    never deleted;
  - the order state machine — orders are never deleted;
  - snapshots — `order_items` copies product name/image/price at order
    time, so deleting a product never corrupts order history.
- If a table later genuinely needs soft delete, add a nullable
  `deleted_at TIMESTAMP` to that table only, and manually filter
  `deleted_at IS NULL` in every query touching it. Soft delete is opt-in
  per table, never global.

## Foreign keys

- **No database-level `FOREIGN KEY` constraints in migrations.** Relations
  are plain `BIGINT <entity>_id` columns; referential integrity is enforced
  in the service layer.
- Rationale: matches the scaffold (which emits no FKs), keeps migrations
  portable across Postgres / MySQL / SQLite, and keeps up/down migrations
  free of dependency ordering.

## Money and stock

- **Money is `BIGINT` in minor units** of the store's single currency — cents
  for USD, 分 for CNY. Columns carry an explicit `_cents` suffix
  (`price_cents`, `subtotal_cents`, `discount_cents`, `shipping_fee_cents`,
  `total_cents`); the Go type is `int64`. Never floats, never
  `NUMERIC`/`DECIMAL`: integer minor units are exact, portable across all
  three supported databases, and safe to add and compare in SQL and Go.
- **Single store currency.** The currency is a store-wide setting, not a
  per-row column; per-row currency codes only enter the schema if
  multi-currency support is ever added (out of core scope).
- **Money math happens once, in int64.** Discounts apply to the subtotal and
  round half-up to the whole cent in a single final step; no intermediate
  rounding. Totals are always recomputed server-side.
- **Stock is `INTEGER NOT NULL DEFAULT 0`**, never negative, never
  fractional. Deduction/restock strategy is settled with T5.4.
- Percentage values (e.g. percent-off coupons) are plain integers 0–100;
  switch to basis points only if sub-percent precision is ever needed.
- Display formatting (cents → "¥12.34") lives in one small shared helper
  when the first UI needs it (M1), never inline at call sites.

## Product variants

- Variants are supported from day one (settled in T1.3, 2026-09-28). A
  product sells through **variants** when it has at least one active row in
  `product_variants`; otherwise the product row itself is the sellable unit
  (its `price_cents`/`stock` are the default price and stock).
- Which unit is sellable is decided **only in the Product service layer**
  (T1.4); cart, checkout and admin never query `product_variants` directly.
- Each variant is one sellable SKU: `product_id`, display `name` for the
  combination (e.g. "Red / M"), own `price_cents`, own `stock`, `active`,
  `sort_order`. Structured multi-axis options (an options/option-values
  table family) are deliberately deferred; when they arrive they only label
  variants and never change price/stock lookup paths.
- A variant is never deleted once referenced by a cart or order: it gets
  `active = FALSE` instead, keeping history intact.

## Decision log

- 2026-09-28 — T0.2 settled by David Zhang: hard delete with status fields
  as the substitute, no DB-level foreign keys, explicit `updated_at`
  maintenance.
- 2026-09-28 — T0.5 settled by David Zhang: money as BIGINT minor units
  (`_cents` suffix, int64 in Go), single store currency, stock as
  non-negative INTEGER, percentages as integers 0–100.
- 2026-09-28 — T1.3 settled by David Zhang: variants are supported now
  (`product_variants` table); sellable-unit decision lives in the Product
  service; structured option axes deferred; referenced variants deactivate
  instead of deleting.
- 2026-09-29 — T2.2 settled by David Zhang: sign-in is phone number + SMS
  verification code only, no passwords. `users.password_hash` stays as an
  empty-default column; the identity column is `users.phone_number`
  (VARCHAR(20), unique).
- 2026-09-29 — T2.6 pitfall fixed while closing the auth service: time
  columns are written with `time.Now().UTC()` because pgx labels
  `TIMESTAMP WITHOUT TIME ZONE` reads as UTC; local-time writes skewed
  session expiry by up to eight hours.
- 2026-09-29 — T4.1 settled by David Zhang: the cart is signed-in users
  only — no guest cart, no merge logic; add-to-cart for guests redirects
  to sign-in with a `next` back to the product page.
- 2026-09-29 — T6.3 settled by David Zhang: payment channels are WeChat Pay
  and Alipay; Stripe is not supported. The `PaymentGateway` registry is the
  extension seam — new channels implement the interface and self-register
  from `init()`, gated on their configuration being present.
