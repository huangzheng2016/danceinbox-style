# Dialogs, Confirms, and Overflow Menus

All modal/menu behavior in the system goes through three components plus one shared stack. Drawers in ANY direction (right/left/top/bottom slide-over) are banned sitewide; small anchored overlays use the floating-panel pattern (`RowActionsMenu`, and the narrow-screen nav panel in the shell).

## `dialogStack` - the shared modal stack

`composables/dialogStack.ts`: a symbol stack (`pushDialog` / `removeDialog` / `isTopDialog`) plus a counted body scroll lock (`lockBodyScroll` / `unlockBodyScroll`, toggling `body.dialog-scroll-locked`). Every floating layer (BaseDialog, ConfirmDialog, RowActionsMenu, the shell nav panel) registers itself here, which is what makes **Esc close only the topmost layer**.

## `BaseDialog` - the only dialog shell

Used for small create forms and one-off results. Contract:

- Props: `open: boolean`, `title: string`, `width?: string` (default `520px`). Emits `close`.
- `Teleport` to `body`; overlay is a plain `--overlay-mask` (no blur), `z-index: var(--z-dialog)`; clicking the overlay itself closes (`@click.self`).
- The dialog surface is flat: `--radius-md`, 1px `--floating-border`, `box-shadow: var(--shadow-dialog)` (a floating layer - the legal shadow case), no backdrop-filter.
- `role="dialog" aria-modal="true"`, focus trap on Tab/Shift-Tab, focus moves to the first focusable element on open and returns to the previously focused element on close.
- Esc closes ONLY if this dialog is the top of `dialogStack`.
- Body scroll locks while open (via `dialogStack`).
- Close button is `class="muted base-dialog-close"` wrapping `<Icon name="x" :size="14" />` with `aria-label="关闭"`. A bare `x` glyph is a red flag.
- Layout: `base-dialog-head` (title `h3` 16px + close), `base-dialog-body` (scrolls, `gap: 8px`), optional `base-dialog-footer` slot (right-aligned actions).
- Enter animation: fade + `translateY(8px) scale(0.97)`.

```html
<BaseDialog :open="creating" title="新建规则" @close="creating = false">
  <!-- form fields (.field-label + bare inputs + JsonField) -->
  <template #footer>
    <button class="muted" @click="creating = false">取消</button>
    <button :disabled="saving" @click="save">保存</button>
  </template>
</BaseDialog>
```

## `useConfirm` + `ConfirmDialog` - the only confirmation UI

- `ConfirmDialog` is mounted ONCE inside the `MainLayout.vue` template (a sibling of `.shell-layout` inside `.ambient-shell` - there is no `App.vue`; the app entry is `createApp(MainLayout)`). `z-index: var(--z-confirm)`, above BaseDialog.
- Pages call the composable and await a boolean:

```ts
const confirm = useConfirm()
async function onDelete(row: Rule) {
  if (!(await confirm({ message: `确定删除规则「${row.name}」吗？`, danger: true, confirmText: '删除' }))) return
  // ...
}
```

- Promise-based; opening a new confirm resolves the pending one with `false`.
- Defaults: title `确认`, confirm `确定`, cancel `取消`. `danger: true` turns the confirm button solid danger.
- Keyboard behavior applies ONLY while focus is inside the dialog (the handler checks `focusInside()`): Enter confirms (except when focus is already on a button), Esc cancels, overlay click cancels; all routed through `dialogStack` so nested confirm-over-dialog behaves. Unlike `BaseDialog`, `ConfirmDialog` has NO focus trap - if Tab moves focus out of the dialog, Esc/Enter stop responding until focus returns inside.
- Every destructive action goes through this. Native `confirm()` / `alert()` are never shippable.

## `RowActionsMenu` - overflow menu for row actions

- Trigger: `class="muted btn-sm"` button with `<Icon name="dots" :size="14" />` and `title="更多操作"`.
- Items: `{ label, icon?, danger?, disabled?, title?, onSelect }[]`. Danger items render in danger color and always live in the menu (never inline in the row).
- Menu is teleported to `body`, `role="menu"` / items `role="menuitem"`, fixed-positioned under the trigger, `z-index: var(--z-menu)` (9500), min-width 156px, surface `--surface-sticky` + `--shadow-floating`.
- Closes on outside mousedown, on Esc (only when top of stack), and repositions on scroll.
- Usage rule: the inline-vs-menu split for row actions follows the table spec in `SKILL.md`.
- This button-anchored, JS-positioned, maskless pattern is THE floating-panel interaction: the narrow-screen shell nav panel (`frontend/layouts/app-shell.md`) uses the same contract.

```html
<RowActionsMenu :items="[
  { label: '立即评估', icon: 'play', onSelect: () => evaluate(row) },
  { label: '复制 ID', icon: 'copy', onSelect: () => copyText(row.id) },
  { label: '删除', icon: 'trash-2', danger: true, onSelect: () => onDelete(row) },
]" />
```

## Anti-patterns

| Don't | Do |
|-------|-----|
| Hand-rolled modal (div + fixed overlay) | `BaseDialog` |
| Native `confirm()` / `alert()` | `useConfirm` / `errorBus` |
| Drawer in any direction | Banned sitewide; `BaseDialog`, the split detail pane, or a floating panel |
| Bare `x` close glyph | `<Icon name="x">` |
| Esc closing every open layer at once | `dialogStack` top-only semantics (free via the components) |
| Dropdown menu hand-positioned in a `td` | `RowActionsMenu` (teleport + reposition on scroll) |
| Blurred overlay (`backdrop-filter`) behind a dialog | Plain `--overlay-mask`; blur does not exist in this system |
