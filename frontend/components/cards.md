# Cards and Content Blocks

Cards are the `.card` class from `base.css`. There is no card component and no per-page card CSS.

## `.card`

Already carries: `display: flex; flex-direction: column; gap: 10px; padding: 14px 16px; margin-bottom: 12px; border-radius: var(--radius-md)`, `background: var(--surface-strong)`, `border: 1px solid var(--border-soft)`. The selected shape profile owns the radius. That is the entire chrome: NO box-shadow, NO backdrop-filter. Depth comes from hover states and floating layers, not from the card.

```html
<section class="card">
  <div class="section-head">
    <h3>命中规则</h3>
    <p>最近 24 小时按规则聚合</p>
  </div>
  <!-- body -->
</section>
```

- Headings carry `margin: 0 0 8px` globally (base.css), so a card's first heading already starts flush - no card-scoped override needed.
- A card that directly contains a `table` scrolls it horizontally on its own (`.card:has(> table)`); the table itself keeps `flex: 0 0 auto`.

## `.section-head` - card-level section header

`.section-head` (base.css, do not re-declare) is a wrapping flex row (`align-items: end`, space-between, `gap: 8px`) whose `h2` / `h3` / `p` margins are zeroed; the `p` is a muted 12px description line. That description line is the legal way to explain a block; the page title itself still belongs to the shell only. For a form-section heading without a description row, use the shared `.section-title` (12px semibold secondary).

## `.detail-head` - split-detail pane header

`display: flex; align-items: center; justify-content: space-between` with `h3 { margin: 0 }`. The pane's title on the left, its actions on the right inside `.detail-actions` (`display: flex; align-items: center; gap: 6px; flex-wrap: wrap` - the shared home for the pane's buttons/menus). Op feedback (`.op-ok` / `.op-err`, both 12px) sits BELOW the form action row, not in the header.

## `.split-detail-empty` - unselected placeholder

Dashed `--border-soft` border, `--radius-md`, centered muted text, `min-height: 200px`. Shown in `.split-detail` when no row is selected:

```html
<div v-if="!selected" class="split-detail-empty">从左侧选择一条规则查看详情</div>
```

## Grouped detail content

Use `.kv-group` to stack related key-value rows with a hairline separator between groups. `.group-title` is the small muted group heading. Use `.kv-group.kv-group-compact` only for short, non-wrapping metadata that should flow horizontally; do not recreate these classes in a page.

## `.edit-grid` - aligned edit forms

Two-column grid (`var(--edit-grid-label-width, 120px) 1fr`, `gap: 8px 12px`); labels are 12px muted `--font-display`, controls are `width: 100%`. Use for structured edit forms in detail panes. Mono is reserved for data and code identifiers.

## Long blocks

- Long text/JSON is capped with max-height + scroll, or rendered via `JsonViewer` / `CodeViewer`.
- Inside `.split-detail`, a SINGLE primary long block gets `.grow-fill` (`flex: 1 1 auto; min-height: 160px`) so it fills the pane instead of wasting tall viewports on a hard-coded max-height. Never apply `.grow-fill` when multiple long blocks coexist.
- `.code-block` exists for small inline snippets (max-height 300px).

## Anti-patterns

| Don't | Do |
|-------|-----|
| `.my-card { border-radius: 12px; background: #fff }` or any per-page card chrome (padding / radius / shadow / background) | `.card` - the only card |
| Page-level `h1` + subtitle inside a page | Shell header owns it; `.section-head` for block titles |
| Hand-rendered JSON in a `<pre>` | `JsonViewer` |
| `.grow-fill` on every block | Only the single primary long block in a detail pane |
| Stacking 3+ unrelated cards on one page | `.tabs` + `useTabs` + route query (one card = one functional block) |
