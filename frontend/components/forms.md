# Forms - inputs, selects, labels, validation

All form controls are styled globally by `base.css`. There are no form component classes to remember: a bare `<input>`, `<textarea>`, or `<select>` is already the designed control. Page code only adds layout (`.row`, `.edit-grid`, `.field-label`) and validation feedback.

## What base.css already gives every input

Text-like inputs (`input` without a type, `text/number/password/email/search/url/tel/date/time/datetime-local`) and `select`:

- `display: block; width: 100%; min-height: 36px; padding: 0 14px`
- `border-radius: var(--radius-sm)`, `background: var(--input-bg)`, `border: 1px solid var(--border-soft)` (flat: no shadow); the shape profile owns the radius
- The `date` / `time` / `datetime-local` family additionally gets `line-height: 32px` so text stays vertically centered under custom fonts
- Hover: `--input-bg-hover` + `--border-strong`
- Focus: accent border + `var(--focus-ring)` (3px `--accent-ring` outline)
- Disabled: `opacity: 0.5`
- Placeholder: `--text-muted`

`textarea`: same surface treatment, `border-radius: var(--radius-md)`, `min-height: 80px`, `resize: vertical`.

`input[type='checkbox']` / `radio`: `accent-color: var(--accent)` and nothing else. For enable/disable use `.toggle-btn`, never a styled checkbox.

## Selects

The bare `<select>` shares the input block above and adds:

- `appearance: none` with a data-URI SVG chevron (`padding-right: 36px` on the control, chevron at `right 14px center`, 11px). The chevron recolors on focus (accent) and has dedicated `:root.dark` variants; `::-ms-expand` is hidden.
- `option` colors pinned to `--app-bg` / `--text-primary` so the native flyout does not flash white in dark mode; `:checked` option uses `--accent-soft`.

```html
<select v-model="status">
  <option value="">全部</option>
  <option value="enabled">已启用</option>
  <option value="disabled">已停用</option>
</select>
```

## Labels

Two label styles exist, both in base.css:

- Bare `<label>`: block, 12px `--font-display`, weight 600, normal case and letter-spacing, muted. Chinese labels never use an uppercase transform.
- `.field-label`: block, 12px muted, `margin-bottom: 4px`. Use this for form fields in detail panes and dialogs.

`--font-mono` is for data and code identifiers, not ordinary Chinese labels. Required fields add `<span class="required-mark">*</span>` (danger-colored) and validate before submit, with the message shown next to the field.

```html
<label class="field-label" for="rule-name">名称 <span class="required-mark">*</span></label>
<input id="rule-name" v-model="form.name" />
<span v-if="errs.name" class="op-err">{{ errs.name }}</span>
```

## Layout helpers

- `.row` - horizontal flex row (`gap: 8px`, wraps); direct `input`/`select`/`textarea` children become `width: auto`, direct `label` children become inline. This is the filter-bar pattern.
- `.list-toolbar` - the list-page toolbar; its inputs/selects stay `width: auto` and only `input.toolbar-search` grows (`flex: 1 1 240px; max-width: 480px`).
- `.edit-grid` - label/value two-column form grid (see `cards.md`).
- `.check-inline` - inline checkbox row (`display: inline-flex; gap: 6px; font-size: var(--text-sm)`).

## JSON and code inputs

- JSON text areas use the `JsonField` component: live `JSON.parse` validation, error surfaced as `JSON 格式错误:...` below the field, `update:error` event.
- Script/SQL inputs use `CodeEditor` (editable, highlighted). See `code.md`.

## Date and time inputs

Always the native `input[type='datetime-local']` (or `date` / `time`), styled by the shared base.css block above. Never build page-local date pickers, calendar popups, or composite hour/minute fields, and never restyle or resize the input per page: no inline `width` / `height` / `min-height` / `line-height` on it and no page `<style>` rules targeting it. Relative ranges (最近 1 小时 / 24 小时 / 7 天 / 30 天) use a `<select>` preset, never a date input; inside `.list-toolbar` / `.row` the base.css rules already keep it `width: auto`, so no inline style is needed. `input[type='month']` is the sanctioned picker for monthly periods; it sits outside the 36px shared block and keeps native styling with `style="width:auto"`. Form fields bind the `datetime-local` string with `v-model` and convert via `parseInputTs` on submit.

## Validation timing

Error display timing follows `useFormValidation` (`submitted` flag + per-field `blur` / `show` / `reset`):

- Dialogs and pages open clean: no required or format errors on first render.
- An error appears only after the first submit attempt (`submitted`) or that field's first blur, and clears as soon as it is fixed.
- `update:error` (from `JsonField` and friends) stays the instant validation-state channel; gate its display behind `submitted`.

## Anti-patterns

| Don't | Do |
|-------|-----|
| `.my-input { border-radius: 8px; border: 1px solid #ddd }` | Bare `<input>`, base.css owns it |
| Icon inside a custom "search shell" | `input.toolbar-search` or a plain input; no custom shells |
| `<label>名称:</label>` with default browser styling | `.field-label` (or the shared bare label) |
| Checkbox as an enable/disable switch | `.toggle-btn` |
| Plain textarea for JSON | `JsonField` |
| Validation via `alert()` | Inline `.op-err` next to the field |
| Toolbar filters each on their own line | `.list-toolbar` keeps them `width: auto` |
| Page-local `select { background-image: ... }` overrides | Nothing; the global chevron + dark variants already exist |
| `width: 100%` or a fixed pixel width on a filter select | Nothing; `.list-toolbar` / `.row` already set `width: auto` |
| Filter select detached from its `<label>` | Sibling `<label>` + control on one line (toolbar spec in `SKILL.md`) |
| `<select multiple>` | Raise the design question; no multi-select exists in the system |
| Hand-rolled dropdown panel (div + v-show) to prettify the OS flyout | Native `<select>`; richer popovers are `RowActionsMenu` / `HoverPopover` territory, not selects |
| Stale filtered options after the search input is cleared | Refetch on a ~250ms debounce (timer cleared on unmount); clearing the search refetches the unfiltered options |
| Installing a select component library | Banned; the global style is the component |
| Page-local date picker / calendar popup, or per-page sizing on a date input | Native `date` / `time` / `datetime-local` input; relative ranges are `<select>` presets |
| Submit disabled until the form validates | `:disabled` only while in flight; errors surface via `useFormValidation` timing |
| Op feedback detached from the submit actions | `.op-ok` / `.op-err` below the form action row |
| Raw `e.message` shown to the user | `'动作名失败:' + errorMessage(e)` |
