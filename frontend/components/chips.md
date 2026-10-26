# Status Pills and Labels

Status display has two layers, both mandatory:

1. `lib/labels.ts` - the ONLY registration point for every status / severity / risk / tier label text and its pill class. New statuses are added here, never as per-page maps.
2. The pill components `StatusPill` / `SeverityPill` / `TierPill`, which render a `.pill` span whose class and text both come from `labels.ts`.

## `.pill` base (base.css)

A 22px-high inline-flex status label: 12px `--font-display`, weight 500, normal case and letter-spacing, nowrap, `--text-secondary` on `--input-bg`, 1px `--border-soft`, `--radius-xs`, and `tabular-nums` built in. Chinese labels never use uppercase transforms. Code-style identifiers use `.pill.pill-code`, which applies `--font-mono`; `.pill-sm` uses 11px. Full declaration: `.pill` in `frontend/base.css` - do not re-declare or restyle it per page.

## Variant classes shipped in base.css

Status (`status-*`): `active`/`done`, `paused`/`queued`, `draft`, `running`, `failed`, `applied`, `enabled`/`approved`/`resolved`/`succeeded`, `pending`, `rejected`/`open`, `investigating`, `configured`/`observing`, `disabled`/`suppressed`.

Severity (`sev-*`): `info` (muted), `low` (accent), `medium` (warning), `high` (`--ctp-peach`), `critical` (danger).

Risk (`risk-*`): `low` (success), `medium` (warning), `high` / `critical` (danger tiers).

Each variant is soft-background + strong-text + a color-mixed border, all through semantic tokens, so every pill works in both themes with no extra code.

## The components

```html
<StatusPill :status="row.status" />
<SeverityPill :severity-id="row.severity_id" size="sm" />
<TierPill :tier="row.tier" />
```

- `StatusPill` - `statusLabel(status)` + `statusPillClass(status)` from `labels.ts`. Unknown/empty status falls back to the raw string / `'—'` and the `status-draft` class.
- `SeverityPill` - numeric severity id (1 提示 / 2 低危 / 3 中危 / 4 高危 / 5 紧急 / 6 致命) mapped by `severityIDLabel` / `severityIDClass`. Six ids land on the five `sev-*` classes: 1 提示 -> `sev-info`, 2 低危 -> `sev-low`, 3 中危 -> `sev-medium`, 4 高危 -> `sev-high`, and 5 紧急 + 6 致命 both -> `sev-critical` (the two top tiers merge into the danger styling; an unknown/missing id falls back to `sev-info`). `size="sm"` adds `.pill-sm` for dense table cells.
- `TierPill` - noise/signal tier via `tierLabel` / `tierPillClass`.

## Anti-patterns

| Don't | Do |
|-------|-----|
| `<span class="tag-green">成功</span>` per page, or any hardcoded pill text | Register in `labels.ts` (a new status lands there, with its class mapping, in the same change), use `StatusPill` |
| Colored dot + text hand-rolled | `.pill.status-*` |
| `style="color: green"` for state | Semantic variant class |
| English status text in a pill | Chinese label from `labels.ts` (product terms excepted) |
| `.pill` as a generic tag/badge for static metadata | Plain muted text or `.list-name`; pills are for state, not decoration |
