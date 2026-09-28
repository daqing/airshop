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

## Decision log

- 2026-09-28 — T0.2 settled by David Zhang: hard delete with status fields
  as the substitute, no DB-level foreign keys, explicit `updated_at`
  maintenance.
