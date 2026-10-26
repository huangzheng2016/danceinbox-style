# Standard Page Anatomy (non-split pages)

In this system the shell owns the page title (eyebrow + title + sub in `.shell-header`). A page NEVER renders its own `h1`, eyebrow, or page header. What a page renders is its functional content: toolbars, cards, tables, forms.

There are two page shapes:

1. **Split pages** (list + edit) - see "Page Patterns" in `SKILL.md` and the split rules in `base.css` (`.split-layout` / `.split-master` / `.split-detail`).
2. **Scrolling pages** (detail, dashboard, long content) - the page flows inside `.page-stage`, which is the only scrolling region. Content is stacked `.card` sections.

This file covers shape 2.

## Skeleton

```vue
<template>
  <div>
    <!-- optional toolbar / tabs for this page's functional area -->
    <div class="list-toolbar">
      <button>+ 新建 X</button>
      <button class="muted" :disabled="loading">刷新</button>
      <Loading inline :visible="loading" />
      <span class="muted-text" style="margin-left:auto">共 {{ total }} 条</span>
    </div>

    <section class="card">
      <div class="section-head">
        <h3>区块标题</h3>
        <p>12px muted 的一行说明</p>
      </div>
      <!-- body: table / form / grid -->
    </section>

    <section class="card">
      <!-- next functional block -->
    </section>
  </div>
</template>
```

`.card` comes from `base.css` with all its chrome baked in (full spec: `cards.md`). Do not re-declare any of it per page.

## Rules

- No page-level `h1` / page header / breadcrumb. The shell rendered it already.
- Card-level section headers are legal and encouraged: `.section-head` (flex, `align-items: end`, space-between) with an `h2`/`h3` and an optional muted 12px description `p`. This is NOT a "double subtitle" violation.
- One page carries one functional area. When a page accumulates multiple stacked cards with distinct purposes, split them into tabs (`.tabs` + `useTabs` + tab in the route query) or into separate pages instead of stacking.
- Long text / JSON inside a card is capped or viewer-rendered per `cards.md` "Long blocks" (never a hand-rolled `pre` for JSON).
- Tables inside a card scroll horizontally through the card itself (`.card:has(> table)` handles `overflow-x: auto`); bare tables outside a card must be wrapped in `<div class="table-scroll">`.
- A page with a single primary long block (JSON / raw payload / large text) lets it grow: in a split-detail context use `.grow-fill`; on a scrolling page use a generous max-height, not a hard-coded tiny one.

## Anti-patterns

| Don't | Do |
|-------|-----|
| `<h1>页面名</h1>` at the top of the page | Shell header already renders it; remove |
| `.page-card` / `.page-header` / `.page-eyebrow` inside a page | Retired pattern; use plain `.card` + `.section-head` |
| `background: white; border-radius: 12px` | `.card` class, tokens only |
| Custom card padding per page | `.card` padding is fixed at `14px 16px` |
| Multiple stacked functional cards on one page | Tabs (`.tabs` + route query) or separate pages |
| JSON hand-rendered into `<pre>` | `JsonViewer` |
| `<div class="table-scroll">` missing around a bare table | Wrap it, or the table cannot scroll horizontally |
| Making `.page-stage`'s parent scroll | Only `.page-stage` scrolls; the shell frame is fixed |
