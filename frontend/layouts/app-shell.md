# App Shell - root layout

The shell is a fixed app frame: `.shell-layout` pins to the viewport (`height: 100vh; overflow: hidden`) and NEVER scrolls. Scrolling happens only inside `.page-stage` (normal pages) or `.split-master` / `.split-detail` (split pages). The shell also owns the page title: the header renders `page-eyebrow` + `page-title` + `page-sub` (title and sub computed from the nav definition; the eyebrow is the static brand text), so pages never render their own `h1`.

Source of truth: the template `frontend/templates/MainLayout.vue` in this skill. **Do not retype this file: copy `frontend/templates/MainLayout.vue` verbatim and edit only the `EDIT:` markers** (nav arrays, brand name, storage key prefixes); fixes go upstream in the skill and get re-copied, never patched in the project (rationale: the Alignment Contract in `SKILL.md`). This document explains the contract so you can verify and extend it; every snippet below matches the template.

## Entry: `src/main.ts`

```ts
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { router } from './routes'
import './styles/tokens.css'; import './styles/base.css' // + code-editor.css only if CodeEditor / JsonField is used
import MainLayout from './MainLayout.vue' // adjust to where the template set lives (e.g. './base/MainLayout.vue')
createApp(MainLayout).use(createPinia()).use(router).mount('#app')
```

This is a minimal wiring sketch only, not the scaffolding template. Part 1 scaffolded projects MUST use the canonical `src/main.ts` from `frontend/skeleton.md`, which additionally registers `i18n` and calls `authStore.init()` ONCE before `app.mount()`.

No component library import or external font CDN. Choose the design profile once per project (see `frontend/design-profiles.md`) and use the same root attributes on every HTML entry. Copy `frontend/assets/fonts/` to `public/fonts/`, including the license files, then preload the self-hosted fonts:

```html
<html lang="zh-CN" data-shape="micro" data-surface="soft-white">
  <head>
    <link rel="preload" href="/fonts/plus-jakarta-sans-latin.woff2" as="font" type="font/woff2" crossorigin />
    <link rel="preload" href="/fonts/geist-mono-latin.woff2" as="font" type="font/woff2" crossorigin />
  </head>
</html>
```

## Structure

```html
<div class="ambient-shell">
  <div class="shell-layout" :class="{ 'is-collapsed': railCollapsed, 'nav-open': navOpen }">
    <aside class="shell-sidebar"> brand / nav-list / rail-toggle </aside>
    <main class="shell-main">
      <header class="shell-header"> nav-toggle(narrow only) / title block / actions </header>
      <div class="toast-stack"> fixed floating toasts, top-right </div>
      <div class="page-stage"><RouterView /></div>
    </main>
  </div>
  <ConfirmDialog />
</div>
```

- `.ambient-shell` is a plain `min-height: 100vh` wrapper. There is NO ambient gradient layer and NO grain layer; do not reintroduce them.
- `.shell-sidebar` and `.shell-header` are flat surfaces: profile-driven `--radius-md`, 1px `--border-soft`, NO box-shadow, NO backdrop-filter.
- `.shell-header` carries `z-index: var(--z-shell-header)`; every z-index in the app goes through the `--z-*` token ladder.
- The shell renders the `errorBus` toasts (behavior spec: `SKILL.md`) as a `position: fixed` top-right stack at `z-index: var(--z-toast)`; each toast is a floating card (`--surface-elevated`, `--shadow-floating`, 1px `--border-soft`) with a kind icon (`check-circle` / `x-circle`).

## Nav model

- `nav` is a typed array; every item AND every group carries an `icons.ts` icon and a Chinese `label`, and every item additionally carries a one-line `sub` (groups have no `sub` - see the `NavItem` / `NavGroup` interfaces in the template). `titleOnlyGroups` feed page-title computation only and never render in the sidebar.
- Active entry = exact match first, then longest prefix (excluding `/`). `pageTitle` = route meta title, else `group · item` (or just `item`), else the brand name; `pageSub` = the item's `sub`. `document.title` follows with a ` - Brand` suffix.
- Group expand state persists in localStorage as a JSON array; the group containing the active route is always treated as expanded. Sidebar collapse state persists under its own key (template default `yourapp.sidebar.collapsed`; rename per project).
- On every route change: the nav panel closes and the scroll position resets - `.page-stage` scrolls back to top, or the document itself at <= 760px where the frame's scroll lock is released and the page scrolls as a whole.

## Sidebar collapse spec (desktop, >= 961px)

Collapsing is a layout-level state (`.shell-layout.is-collapsed` via `railCollapsed = collapsed && !isNarrow` - it only applies on desktop). Every piece below is required; dropping any one of them visibly breaks the rail.

1. **Column transition, not width transition.** `.shell-layout` is `display: grid; grid-template-columns: 232px minmax(0, 1fr)` with `transition: grid-template-columns var(--duration-base) var(--ease-spring)`; `.is-collapsed` flips the first track to `72px`. Only the grid track animates. Never transition `width` on the sidebar.
2. **Labels leave the DOM.** `.nav-label`, `.brand-text`, and `.nav-chevron` render with `v-if="!railCollapsed"`. They do not shrink, fade, or wrap during the transition; they simply are not there.
3. **Collapsed item geometry is unchanged.** `.shell-layout.is-collapsed .nav-item { justify-content: center; padding: 8px; }` - same box height as expanded, icon centered. This rule (3 classes) beats `.nav-subitem { padding-left: 28px }` (1 class), so subitem icons center too.
4. **Icons have fixed boxes.** `.nav-icon` is `width: 18px; height: 18px; flex-shrink: 0` (subitem icons 15x15). No layout change may scale an icon, and flex may never compress one.
5. **Groups flatten into the rail.** `.is-collapsed .nav-group-header { display: none; }` and `.is-collapsed .nav-subitems { display: flex; }`: group headers disappear and every subitem icon shows directly in the rail.
6. **Native tooltips on the rail.** Every nav item and group header carries `:title="railCollapsed ? item.label : undefined"`.
7. **The toggle is honest.** `.rail-toggle` shows `chevron-left` expanded / `chevron-right` collapsed, with `:aria-expanded="!collapsed"` and an aria-label; the state persists in localStorage.
8. **The brand centers too.** `.shell-layout.is-collapsed .brand { justify-content: center; padding: 4px 0 14px; }` - the brand mark centers in the 72px rail instead of staying left-aligned with its expanded padding (`.brand-text` is already gone via rule 2).

## Narrow-screen nav: floating panel, not a drawer (<= 960px)

Under 960px the rail and the manual collapse disappear entirely. The sidebar leaves the layout flow and becomes a button-anchored floating panel (the same interaction family as `RowActionsMenu`). Drawers in ANY direction are banned; this panel is the only narrow-nav pattern.

- **Layout**: `.shell-layout` collapses to one column (`grid-template-columns: minmax(0, 1fr)`) with `padding: 14px; gap: 14px`, and `.shell-main` tightens to `height: calc(100vh - 28px)` (2 x 14px padding). `.rail-toggle` is hidden; a `.shell-nav-toggle` (menu icon, `aria-expanded`) appears inside `.shell-header`.
- **Position**: computed in JS from the toggle button rect, exactly like the row-actions menu: `top = rect.bottom + 8`, `left = clamp(rect.left, 14, vw - width - 14)` with `width = min(300px, vw - 96px)`, `maxHeight = viewport height - top - 14`. Recomputed on scroll (capture-phase listener).
- **Visibility**: default state in this band is `visibility: hidden; opacity: 0; transform: translateY(-6px); pointer-events: none` (with `visibility` on a delayed transition); `.shell-layout.nav-open .shell-sidebar` flips to visible with an opacity + transform spring transition. There is NO mask/overlay: the page behind stays visible and outside clicks close the panel.
- **Floating chrome**: `z-index: var(--z-nav-panel)` (8900, below dialogs at 9000) and `box-shadow: var(--shadow-floating)` - floating layers are the legal shadow case.
- **Close paths, all four**: outside mousedown (capture phase, toggle excluded), Esc only when the panel is top of `dialogStack` (`pushDialog` on open), route change, and leaving the narrow band (`matchMedia('(max-width: 960px)')` change event).
- **Focus**: first focusable element inside the panel is focused on open; on close, focus returns to the toggle if it was inside. The aside gets `aria-label` + `tabindex="-1"` in narrow mode.
- **No manual collapse here**: `railCollapsed = collapsed && !isNarrow`, so the panel always shows full labels; the 232/72 rail spec above is desktop-only.

## Other breakpoints

- 760px: the fixed frame releases its scroll lock - `.shell-layout` and `.shell-main` become `height: auto` (with `min-height: 100vh` / `min-height: calc(100vh - 28px)` fallbacks) plus `overflow: visible`, while `.page-stage` (including `.page-stage:has(.split-layout)`) only flips to `overflow: visible` - the template sets no `height` on it there, since it never had a fixed height to release. The document scrolls as a whole, pairing with the base.css split-layout stacking fallbacks.
- 640px: padding/gap shrink to 10px and the `.shell-main` fallback tightens to `min-height: calc(100vh - 20px)` (2 x 10px padding). `.page-title` keeps the shared `--text-xl` size; no off-scale 18px override. This block comes after the 760px block in the template so its `.shell-main` override wins. (The nav stays the floating panel.)

## Clickable rows: `.row-link`

`base.css` ships `.row-link` for rows whose entire surface is the click target (a `<tr>` or list row that navigates/opens on click): `cursor: pointer`, a `--surface-soft` hover background, and a `:focus-visible` outline (`--accent-ring` with a negative offset so it stays inside the row). When a whole row is clickable, put `class="row-link"` on it (with `tabindex="0"` and an Enter/Space keydown handler for non-link elements) - never an inline `style="cursor: pointer"` or a per-page hover rule.

## Checklist for this file

- [ ] `MainLayout.vue` is the verbatim copy of `frontend/templates/MainLayout.vue` with only `EDIT:` regions customized
- [ ] NO ambient gradient layer, NO grain layer, NO backdrop-filter anywhere in the shell; sidebar and header stay flat (`--radius-md`, 1px border, no default shadow)
- [ ] `.shell-layout` is `height: 100vh; overflow: hidden`; only `.page-stage` / split columns scroll
- [ ] Desktop collapse contract intact (all 8 points above)
- [ ] Narrow band (<=960px) uses the floating nav panel: button-anchored JS position, no mask, four close paths, focus return; NO drawer
- [ ] Every nav item and group in the EDIT nav arrays has an `icons.ts` icon and a Chinese label (items also a one-line `sub`)
- [ ] No page renders its own `h1`; the shell header owns the title block
- [ ] Theme toggle writes `document.documentElement.classList` + `localStorage['theme']`; shape and surface attributes match the recorded project profile on every HTML entry
- [ ] All z-index values come from the `--z-*` token ladder
- [ ] All three responsive blocks (960px panel / 760px / 640px) are present, in that source order
- [ ] Both Plus Jakarta Sans and Geist Mono are self-hosted and preloaded from every HTML entry; no external font CDN
- [ ] This shell is the ONLY sidebar/nav in the app; pages never hand-roll their own
