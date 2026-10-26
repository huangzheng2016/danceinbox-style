# Loading and Empty States

State feedback has a strict division of labor: `Loading` for pending, `EmptyState` / `TableEmptyRow` for loaded-but-empty. No per-page feedback markup.

## `Loading`

One component: `Loading.vue`. Key behaviors:

- **200ms delay**: when `visible` flips true, the spinner appears only after `delay` ms (default 200). Fast requests never flash a spinner. Hiding is immediate.
- **Two modes**: default `loading-overlay` (fixed centered card, `--radius-lg` + `--shadow-soft`, `pointer-events: none`, `z-index: var(--z-loading)`) and `inline` (`loading-inline`, small spinner + muted 12px label in flow).
- CSS-only spinner: 26px ring (14px inline), `--accent-soft` track + `--accent` top, 0.8s rotation. Fade transition `var(--duration-fast)` (0.18s).
- Default label `加载中…`; `role="status"` (which already implies `aria-live="polite"`).

```html
<!-- list toolbar, REQUIRED form: inline -->
<Loading inline :visible="list.loading.value" />

<!-- fullscreen overlay: route transitions, blocking loads only -->
<Loading :visible="initialLoading" />
```

## `EmptyState` - standalone empty areas

Dashed-border box (`1px dashed var(--border-strong)`, `--radius-lg`, `padding: 48px 24px`, centered). Inside a flex parent it stretches to fill the container (content vertically centered); outside one it sizes to content - either way floored at `min-height: 160px`. Icon 32px muted at 60% opacity, 14px semibold title, 12px muted hint, optional default slot for an action (e.g. a create button).

```html
<EmptyState
  v-if="!loading && items.length === 0"
  icon="inbox"
  title="暂无待审批项"
  hint="Agent 起草的提案会出现在这里"
/>
```

Props: `title` (required), `hint?`, `icon?` (defaults to `inbox`). No emoji; the icon comes from `icons.ts`.

## `TableEmptyRow` - inside tables

Renders one `<tr>` whose cell colspans the table. While loading it renders nothing. On error it renders the error detail and a `重试` button that emits `retry`; it never mislabels a failed request as an empty result. Only a successful response with `count === 0` renders `EmptyState` (160px floor, padding included):

```html
<tbody>
  <tr v-for="row in items" :key="row.id">...</tr>
  <TableEmptyRow
    :colspan="6"
    :loading="loading"
    :error="error"
    :count="items.length"
    title="暂无规则"
    @retry="refresh"
  />
</tbody>
```

Pass the fetch function to `@retry`. Keep the toolbar `.op-err` for immediate feedback; the in-table error makes the failed state visible and recoverable without a second custom error block.

## Anti-patterns

| Don't | Do |
|-------|-----|
| Fullscreen overlay on a list refresh | `Loading inline` in the toolbar; the overlay is reserved for genuinely blocking states (first paint of a heavy page) |
| Spinner placed ad hoc on the page | Toolbar slot between the 刷新 button and the `.op-err` span (toolbar spec in `SKILL.md`) |
| Per-page `加载中…` text spans, skeleton screens, or progress bars | `Loading inline` for a pending region, `EmptyState` after load |
| No delay spinner flicker | The 200ms `delay` prop already handles it |
| Hand-rolled spinner CSS | `Loading.vue` |
| `v-if="loading"` hiding the whole table | Keep the table; show `Loading inline` + `TableEmptyRow` handles the empty/loading boundary |
| Emoji as the empty-state icon | `icons.ts` icon via the component |
| `v-if="items.length === 0"` showing empty state during load | `TableEmptyRow` with `:loading` / `:error` gates |
| Blank white area when a list is empty | `EmptyState` |
| Custom dashed boxes per page | The two components |
| Jokes or mascot tone in empty copy | Neutral noun phrase (`暂无 X`); the hint says what lands here |
| Nesting `EmptyState` and `.split-detail-empty` (two dashed frames stack) | Exactly one dashed placeholder per pane; "select a row first" is `.split-detail-empty` (see `cards.md`) |
