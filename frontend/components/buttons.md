# Buttons

There is no button component. `base.css` styles the bare `button` element (and `.button`) globally; variants are classes on the same element. All specs below are from `base.css`.

## Base style (what every button already is)

- Shared control shape: `border-radius: var(--radius-sm)`, `min-height: 36px`, `padding: 0 14px`, `gap: 6px`. The selected shape profile controls the actual radius.
- `font-size: var(--text-md)`, `line-height: 1.2`, `font-weight: 600`, solid `var(--accent)` background, `var(--text-on-accent)` text. Buttons, text inputs, and selects share the same 36px baseline height.
- Hover: `background: var(--accent-strong)` (a deeper cyan) PLUS `box-shadow: var(--accent-shadow)`. The hover shadow is deliberate: shadows mark interactivity, and the primary button is the one static element allowed to lift (the `.danger` hover's `--danger-shadow` is the only other hover accent - see below).
- Active: `transform: scale(0.97)`.
- Focus: `outline: 3px solid var(--accent-ring); outline-offset: 2px`.
- Disabled: `opacity: 0.45; cursor: not-allowed`, transform suppressed.
- Transitions only over `background / color / border-color / transform` with `var(--duration-fast) var(--ease-spring)`. Never `transition: all`.

```html
<button @click="create">+ 新建规则</button>
```

## Variants

### `.muted` - secondary action

```html
<button class="muted" :disabled="loading" @click="refresh">刷新</button>
```

`var(--input-bg)` background, `1px solid var(--border-soft)`, primary text. Hover: `--input-bg-hover` background, `--floating-border` border, accent text, and an explicit `box-shadow: none`. Use for 刷新, cancel, and row actions.

### `.danger` - destructive action

```html
<button class="danger" @click="onDelete">删除</button>
```

Danger text on `--danger-soft`, danger-tinted border; hover deepens and adds `--danger-shadow`. Always pair with `useConfirm`. At most one solid primary per action area; danger only for destructive ops.

### `.btn-sm` - compact (row actions, inline triggers)

```html
<button class="muted btn-sm" @click="edit(row)">编辑</button>
```

`min-height: 26px; padding: 2px 10px; font-size: var(--text-sm)`. Pagination uses this same variant instead of cancelling button height.

### `.toggle-btn` - enable/disable switch (the ONLY enable/disable control)

```html
<button
  class="toggle-btn"
  :class="{ on: rule.enabled }"
  :title="rule.enabled ? '点击停用' : '点击启用'"
  @click="toggle(rule)"
>{{ rule.enabled ? '已启用' : '已停用' }}</button>
```

`font-size: var(--text-xs); min-height: 26px; min-width: 56px; padding: 3px 12px`, muted by default; `.on` = solid `--success`. Hover only swaps the border and stays flat (`box-shadow: none`); the `.on` state keeps its green surface on hover. Used in lists and forms. Never use a checkbox or a hand-rolled switch for enable/disable.

## Row actions

Row actions live in `td.row-actions` (sticky right) inside a `.row-actions` flex container (`gap: 6px`); buttons are `muted btn-sm`. The inline/menu split, danger-in-menu, and condition-disabled `title` rules are the table spec in `SKILL.md`; the menu component is `RowActionsMenu` (see `dialogs.md`).

## Anti-patterns

| Don't | Do |
|-------|-----|
| `<a class="btn-primary">` or div buttons | Real `<button>` |
| Emoji or glyph "icons" in a button | `<Icon name="...">` |
| Gradient background, glow shadow | Solid `var(--accent)` only |
| `border-radius: 8px` | `var(--radius-sm)` from base.css; choose the shape profile once for the project |
| Checkbox styled as a switch | `.toggle-btn` |
| Two solid primaries in one action area | One primary (`+ 新建 X`), rest `muted` |
| Per-page `button` restyle or a hand-made new variant | Existing classes; the default answer is `muted` |
| Submit button still clickable mid-request | `:disabled` while the request is in flight |
| `onclick="confirm('...')"` or bare delete | `useConfirm` + `.danger` |
