# Data and Cache Conventions

Rules for every service that touches the database or the cache. `biz/service/example` follows all of them; copy its shape.

## Queries

Services use only the generated, typed query layer:

```go
e := s.q.Example
row, err := e.WithContext(ctx).Where(e.ID.Eq(id), e.Status.Eq(model.StatusEnabled)).First()
```

- No string conditions (`Where("name = ?", v)`), no raw SQL, no `map[string]any` updates in services. If a query cannot be expressed with typed fields (aggregates, dialect functions), it becomes a typed function in `biz/dal/db` that services call.
- Every chain starts with `WithContext(ctx)`. Background work uses a context derived from the caller or `context.WithoutCancel`, never a bare `context.Background()` in request paths.
- Text search uses `db.Contains(field, s)`: user input is matched literally (`%` and `_` are not wildcards) and it behaves the same on MySQL and SQLite.
- Lists use `db.Page(page, size)` then `FindByPage(offset, limit)`. It defaults to page 1 and 20 rows and caps at 50, matching the frontend's page sizes.
- Sorting goes through `db.Order(sort, allowed)`: the client sends `name` or `-name`, the service lists the sortable columns (each must be indexed), anything else is `errs.Invalid`.
- Time filters are half-open, `[from, to)` in unix seconds, with 0 meaning unbounded.
- Filter in `Where`, never after `FindByPage`; post-filtering produces short pages and wrong totals.
- `IN` lists whose size the caller controls are chunked (about 500 ids per query).
- Related rows are loaded with a second query by ids (`Where(e.ID.In(ids...))`), not joins on generated models. Display names of referenced users and the like are resolved once per response (`user.Names(ctx, ids)` in the handler's `ownerNames`), never once per row.

## Not found

The service translates `gorm.ErrRecordNotFound` into `errs.NotFound("<对象>不存在")`. `gorm` errors never cross the service boundary. `db.Init` enables `TranslateError`, so unique-key violations arrive as `gorm.ErrDuplicatedKey` and become `errs.Conflict("<字段>已存在")`.

## Writes

- **Update with `UpdateSimple(field.Value(v), ...)`.** It writes zero values (`""`, `0`, `false`) and still sets `UpdatedAt`. `Updates(&model.X{...})` silently skips zero values, so a cleared field would never save; do not use it.
- **Do not use `RowsAffected` to detect existence on plain updates.** MySQL reports rows changed, not rows matched, so saving unchanged values reports 0. Check existence first (`s.Get`), then write.
- **State-dependent writes are compare-and-set.** Put the precondition in `Where` and treat `RowsAffected == 0` as "precondition did not hold":

  ```go
  res, err := e.WithContext(ctx).Where(e.ID.Eq(id), e.Status.Eq(model.StatusDisabled)).Delete()
  if err != nil || res.RowsAffected > 0 {
  	return err
  }
  // 0 rows: either gone already (success, retries are safe) or in the wrong state (errs.Conflict)
  ```

  A separate read followed by a write is a race; never do that for transitions, balances, claims, or quotas.
- **Counters change atomically:** `UpdateSimple(e.Count.Add(1))`, never read-modify-write in Go.
- **Optimistic locking** on tables edited concurrently: a `Version int64` column in the `Where` and incremented in the same `UpdateSimple`; 0 rows affected is `errs.Conflict("数据已被修改,请刷新后重试")`.
- **Validation lives in the service**, once, so every entry point (API, admin API, jobs) gets it. References are validated there too (an owner must be an existing, active user).
- **Batch operations** take at most 100 ids (validated in the IDL), apply what is allowed, and return `done` and `skipped` counts instead of failing the whole batch on one row. State rules still sit in `Where` (batch delete only removes disabled rows).

## Transactions

One form, and only the `tx` handles inside it:

```go
err := s.q.Transaction(func(tx *query.Query) error {
	if err := tx.Order.WithContext(ctx).Create(order); err != nil {
		return err
	}
	_, err := tx.Stock.WithContext(ctx).Where(tx.Stock.ID.Eq(id), tx.Stock.Left.Gte(n)).UpdateSimple(tx.Stock.Left.Sub(n))
	return err
})
```

Returning an error rolls back. Lock rows in a fixed order (by id) when one transaction touches several.

## Deletes

- Hard delete is the default. If a table needs history, add a `DeletedAt gorm.DeletedAt` column; then unique indexes on user-visible columns must include `deleted_at`, and `Unscoped()` stays out of services.
- Records that carry decisions (who approved, when) are never deleted; they change state.
- Deleting something already gone succeeds. Deleting something in the wrong state is a conflict.

## Schema changes

`AutoMigrate` at startup handles additive changes only: new tables, new nullable or defaulted columns, new indexes. Renames, type changes, `NOT NULL` on populated columns, and data backfills are explicit one-off steps reviewed like code: expand (add new), backfill, switch code, contract (drop old) in separate releases. A migration that meets data it does not expect stops and reports; it never "cleans" data on its own.

The primary key has no explicit `type:` tag so that it auto-increments on both MySQL and SQLite.

## Cache

The `cache.Cache` interface is the only way to Redis (or the in-process cache in the lite profile). Nothing outside `biz/extension/cache` imports a Redis client.

- Cache only reads that are expensive, read far more than written, and allowed to be stale for the TTL. Write the staleness bound next to the TTL constant.
- Cache closed facts only. Never cache an error, a not-found, or a still-changing value such as today's count.
- `cache.ErrMiss` means absent; any other error means the cache is down. A normal read falls back to the database and logs it. A guardrail (quota, permission) fails closed instead.
- Fill the cache with `context.WithoutCancel(ctx)` so a client disconnect does not abort the fill.
- Collapse concurrent misses of one key with `golang.org/x/sync/singleflight` in the service.
- One key builder and one TTL constant per cached read path; include a version segment (`example:v1:<id>`) and bump it when the cached shape changes.
- `TryLock` is for "only one instance does this at a time" (a scheduled job, a rebuild). It does not replace compare-and-set in the database.
