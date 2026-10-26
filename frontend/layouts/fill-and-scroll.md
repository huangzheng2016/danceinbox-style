# Fill viewport + internal scroll

## When to use

When a page must fill the remaining viewport and scroll one inner region, instead of letting the whole page scroll. Typical cases:

- Graph/map canvas (the canvas must take the remaining height)
- Split list+detail pages (the standard pattern: `.split-master` and `.split-detail` scroll independently)
- Long lists with a sticky toolbar and a bottom-pinned Pagination

## Anti-pattern: vh subtraction

```css
/* never do this */
.my-canvas { height: calc(100vh - 160px); }
```

The subtrahend is never right: header height changes, padding changes, a banner appears. One layout edit and you re-audit every `100vh -` in the file.

## The model in this system

The shell already establishes the fixed frame (see `app-shell.md`):

- `.shell-layout` is `height: 100vh; overflow: hidden`.
- `.shell-main` is a flex column with a fixed `height: calc(100vh - 40px)` and `overflow: hidden`.
- `.page-stage` is `flex: 1 1 auto; min-height: 0` and scrolls (`overflow-y: auto`) for normal pages.
- On split pages, `.page-stage:has(.split-layout)` locks to `overflow: hidden` and the two columns scroll instead.

So page code never touches `100vh` at all. The shell body owns its two `calc(100vh - ...)` expressions (`.shell-sidebar` and `.shell-main`, both subtracting the 40px shell padding). The only other viewport caps are floating layers: `BaseDialog` uses `max-height: calc(100vh - 48px)` with a `calc(100dvh - 48px)` follow-up (dynamic-viewport browsers track the mobile URL bar, older ones keep the `100vh` fallback), plus the narrow-shell and nav-panel caps whose per-breakpoint values live in `app-shell.md`. Page-internal elements fill space with flex, never with vh math.

## Split pages: the canonical fill pattern

`base.css` ships the whole thing; a page only writes:

```html
<div class="split-layout">
  <div class="split-master">
    <!-- toolbar, table; Pagination is sticky at the bottom automatically -->
  </div>
  <SplitResizer app-prefix="yourapp" storage-key="feature" />
  <div class="split-detail">
    <!-- or .split-detail-empty when nothing is selected -->
  </div>
</div>
```

Import `SplitResizer` from the copied template set. Place it directly between the master and detail, use a stable feature-specific `storage-key`, and replace `yourapp` with the project's storage prefix.

What base.css and the shared resizer do for you:

- `.split-layout` - `display: flex; flex: 1 1 auto; min-height: 0; gap: 12px`.
- `.split-master` - fixed share (`flex: 0 0 clamp(520px, 40vw, 720px)`, overridable via `--split-master-width`), `overflow: auto`; its `.pagination` child is `position: sticky; bottom: 0`.
- `SplitResizer` - drag with pointer capture, use Left/Right (2%) or Shift+Left/Right (10%), and double-click to reset. It persists `<app>.split.width.<key>`, enforces master/detail minimums, exposes `role="separator"`, and hides at <=960px.
- `.split-detail` - `flex: 1 1 auto; overflow-y: auto; display: flex; flex-direction: column`; every direct child stays `flex-shrink: 0`; a single primary long block can take `.grow-fill` (`flex: 1 1 auto; min-height: 160px`) to fill the remaining pane height.
- In the 761-960px band the split stacks vertically while the frame stays fixed: `.split-layout` becomes a column, `.split-master` takes `flex: 0 1 40%` (still internally scrolling), `.split-detail` takes the rest.
- Under 760px the split also stacks, but the frame releases its scroll lock and the whole document flows.

## Custom fill regions (canvas pages)

For non-split fill (a canvas, a terminal, a log viewer), the same flex rules apply:

```css
.fill-row {
  flex: 1;
  min-height: 0;      /* the key line: lets the child shrink below content size */
  display: flex;
}
.scroll-child {
  flex: 1 1 0;        /* basis 0, grows into remaining space; NOT 1 1 auto */
  min-height: 0;
  overflow-y: auto;   /* scrolls itself, never propagates */
}
```

- `min-height: 0` on EVERY nested flex layer. Miss one and the whole chain stops working.
- `flex: 1 1 0`, not `flex: 1 1 auto`: an auto basis sizes to content and 50 results blow the parent open.
- Direct children of `.split-detail` keep `flex-shrink: 0` so flex never squashes `overflow: auto` blocks like `pre`.

## Checklist

- [ ] No `calc(100vh - N)` anywhere in page code (the shell body has its two; the only other occurrences are the dialog and narrow-shell viewport caps listed above)
- [ ] The fill region is `flex: 1` + `min-height: 0`
- [ ] The scrolling child is `flex: 1 1 0` + `min-height: 0` + `overflow: auto`
- [ ] Every intermediate flex layer also has `min-height: 0`
- [ ] Split pages use `.split-layout` / `.split-master` / `.split-detail` from base.css, not a local re-implementation
- [ ] `.split-detail` children are `flex-shrink: 0`; a single long block uses `.grow-fill` (never when multiple long blocks coexist)
- [ ] The window itself does not scroll (except under the 760px fallback)

## Verify

```js
// DevTools console
({
  doc: document.documentElement.scrollHeight,
  view: window.innerHeight,
  overflow: document.documentElement.scrollHeight - window.innerHeight,
})
// expected: overflow === 0
```

If `overflow > 0`, the page can scroll as a whole: find the innermost element that outgrows its parent and check `min-height: 0` on every layer above it.
