# Design System

Native HTML controls styled once by `styles/base.css`, a small set of Vue components in `base/`, and reference pages that every new page is copied from. There is no component library and no per-page styling of shared things.

Consistency comes from copying working code, not from following rules. When you build a page, start from the closest reference page; when you need a control, use the one listed below.

## Files and ownership

| Skill path | Lands in project as | Owner |
|---|---|---|
| `frontend/styles/tokens.css`, `base.css` | `src/styles/` | Skill. Copied verbatim; to change, change the skill and re-copy. |
| `frontend/base/*` | `src/base/` | Skill, except `app.ts` and `labels.ts`. |
| `frontend/base/app.ts` | `src/base/app.ts` | Project: brand name, storage prefix, nav, user hook. |
| `frontend/base/labels.ts` | `src/base/labels.ts` | Project: every status enum's text and tone. |
| `frontend/scaffold/*` | the project's `frontend/` | Project, from day one. |
| `frontend/assets/fonts/*` | `public/fonts/` (with licenses) | Skill. |

Updating a project to a newer skill version means re-copying `styles/` and `base/` (minus the two project files) and fixing whatever no longer compiles.

## Page patterns

Every page is one of three shapes. Pick by what the page does, then copy the reference.

| Page does | Shape | Copy from |
|---|---|---|
| Manage a collection (browse, filter, create, edit, batch, delete) | List + detail split | `scaffold/src/pages/ExampleList.vue` + `ExampleFormDialog.vue` |
| One object with more than fits in the split pane (tabs, raw data, history) | Full detail page | `scaffold/src/pages/ExampleDetail.vue` |
| Agent or long-running job with streamed output | Run panel (a detail-page tab) | `scaffold/src/pages/ExampleRuns.vue` |
| Files belonging to an object | Attachments panel (a detail-page tab) | `scaffold/src/pages/ExampleAttachments.vue` |
| A task with several dependent steps (import, onboarding, setup) | Wizard page | `scaffold/src/pages/ExampleImport.vue` |
| Overview, dashboard | Scrolling page of cards | `scaffold/src/pages/Home.vue` |
| Sign in, error pages | Bare page, no shell | `scaffold/src/pages/Login.vue` (route `meta.bare`) |

The shell renders the page title from `app.ts` nav (or `route.meta.title`). Pages never render an `h1` or their own header.

### List + detail

- Page root is `.split` with `.has-detail` while a row is selected. Under 960px only one pane shows; the detail head has a back button (`.split-back`).
- `useList(fetcher, filters, { search })` owns filters, page, size, loading, and errors. All of it lives in the URL. Selection is a separate `useRouteQuery({ id: '' })` key. Do not keep list state in local refs.
- Toolbar order, left to right: search input (`.search`), filters (`Segmented` for a status with 2 to 5 values, `<select>` or `Combobox` for more, `DateRange` for time), `.spacer`, refresh icon button, primary `新建`.
- Batch actions: a checkbox column (`th.col-check` / `td.col-check`, the cell stops click propagation) driven by `useSelection`. While anything is selected the toolbar is replaced by `.toolbar.batch`: `已选 N 项`, the actions, `取消选择`. The server reports done and skipped counts; the toast says both (`已删除 3 项，跳过 2 项`). Destructive batches confirm first.
- Sorting: `SortTh` on columns the server can sort (it needs an index), bound to `list.sort` / `list.setSort`. The server whitelists sortable fields with `db.Order`.
- Time in list columns is relative (`fmtRelative`, full time in `title`); absolute times belong in detail views.
- An empty result with filters applied says `没有符合条件的<对象>` and offers `清除筛选` (`list.reset`).
- Table columns: identifying name first, state next, numbers and times right-aligned (`.num`), actions last.
- Row actions: at most one inline ghost button (usually `编辑`) plus an `ActionMenu` for everything else. Destructive actions live in the menu, never inline.
- Rows are clickable (`tr.clickable` + `tabindex="0"` + Enter) and select into the detail pane. Buttons inside rows use `@click.stop`.
- Create and edit use the same form dialog component. Detail panes are read views with an `编辑` button, not inline forms, so switching rows never discards edits.
- When the pane gets crowded, link to a full detail page (`完整详情`) instead of growing the pane.

### Full detail page

- Route `/<feature>/:id` with `meta.parent` set to the list path; `usePageTitle(() => item.name)` makes the shell header read `示例 / 订单同步` with a link back.
- Top: an `Alert` for any state the user must know about (disabled, read-only, expiring), with the fixing action inside it. Then a `.toolbar` with the status pill and relative update time on the left, actions on the right.
- Sections go in `Tabs` bound to a `useRouteQuery` key so reload and links keep the tab. Raw payloads go in a `CodeBlock` tab.
- Deleting from the detail page navigates back to the list.

### Scrolling page

- Root `.page` (a vertical stack). Optional `.stats` row of stat cards, then one `.card` per block.
- A card that needs a title uses `.card-head` (title + optional `.card-desc` on the left, actions on the right).
- More than one card is fine. Use `Tabs` only when blocks are alternatives the user switches between, not to hide length.

## Which control

| Need | Use |
|---|---|
| Button | `<button>` (secondary). Add `.primary` for the single main action of an area, `.danger` only inside a confirm, `.ghost` for low-emphasis row and toolbar actions, `.btn-sm` / `.btn-icon` for size and shape. |
| Link styled as a button | `<RouterLink class="btn ...">` |
| Icon | `<Icon name="..." />` from `icons.ts`. Add missing icons there (Lucide paths). No emoji, no glyph characters as icons. |
| On/off setting | `Switch` (`role="switch"`). Never a checkbox, never a button whose label is the state. |
| One of 2 to 5 options, all visible | `Segmented` (status filter, view switch, granularity) |
| One of a short fixed list | native `<select>` |
| One of many, or from the server, or several | `Combobox` (`:options` or `:load`, `multiple`). Its model holds `{ value, label }` so saved values show their label. |
| Multiple independent flags | checkboxes in `label.check` |
| Date / time field | native `input type="date" / "datetime-local"`, converted with `toInputValue` / `fromInputValue` |
| Time filter | `DateRange` (presets + custom), converted with `rangeBounds` in the API module |
| Status, severity, any enum | `StatusPill` with a map from `labels.ts` |
| Persistent message about the current page or object | `Alert` (info / success / warning / danger, optional action) |
| Result of an action | `toast.success(...)` / `toast.error('动作失败', e)` |
| Overflow actions, user menu | `ActionMenu` |
| Any other anchored floating panel | `Popover` (menus and selects are built on it; do not position panels by hand) |
| Modal form or result | `BaseDialog` |
| Yes/no before a destructive or irreversible action | `useConfirm()` |
| Leaving a form with unsaved edits | `useDirtyGuard` (dialogs: route every close through `confirmDiscard`; pages: `{ route: true }`) |
| Id, key, address to paste elsewhere | `CopyText` |
| Code, logs, JSON payload | `CodeBlock` (objects become highlighted JSON; long content folds) |
| Progress, quota, usage | `ProgressBar` (`tone="auto"` turns warning/danger near the limit) |
| Sortable column | `SortTh` |
| Row selection for batch actions | `useSelection` + `.toolbar.batch` |
| Table loading / error / empty | `TableState` as the last row of `tbody` |
| Region loading | `Loading` (block) or `Loading inline` beside a control |
| Empty region | `EmptyState` |
| Alternative views of one object | `Tabs`, with the active key bound to a `useRouteQuery` key |
| Resizable split | `SplitResizer` between master and detail |
| Pagination | `Pagination` wired to `useList` |
| Any data fetch | `useFetch` (single resource) or `useList` (collection) |
| Copy from a menu item | `copyText` (toasts the result) |
| One-line hint on hover | native `title` |
| Several lines or structure on hover (metric definition, profile) | `HoverCard`; the same information must also be reachable without hovering |
| Show Markdown (model output, notes) | `MarkdownText` (raw HTML is not rendered; links open safely in a new tab) |
| Write Markdown | `MarkdownEditor` (编辑 / 预览) |
| Write code or SQL | `CodeEditor` |
| Write JSON | `JsonField`, validated with `jsonObjectError` in the form rules |
| Chart | `Chart` with an ECharts option; register new chart types only in `base/echarts.ts` |
| Pick or drop files | `FileUpload`; send them with `useUpload` (progress, cancel, retry) |
| Re-fetch while something is in progress | `usePolling` (never `setInterval`) |
| Live output of a run | `readEventStream` + `applyEvent` + `AgentTranscript` / `ToolCallCard` |
| Multi-step flow | `useWizard` + `Steps` |
| Parse a CSV in the browser | `parseCsv` |
| Sizes, durations, counts | `fmtBytes`, `fmtMs`, `fmtNumber` (`format.ts`) |

Not in the system: drawers, WYSIWYG rich text, collaborative editing, resumable or chunked uploads, maps. If a feature truly needs one, add it to `base/` in the skill first so every project gets the same one.

## Streaming and agent runs

- Output that is produced over time (model answers, job logs) streams over SSE from a route ending in `/stream`. The backend stores every event with a sequence number; the client resumes with `Last-Event-ID`, so a reload or dropped connection replays exactly what was missed.
- Event types: `status` (`{status, error}`), `text` (`{delta}`, Markdown), `tool_call` (`{id, name, args}`), `tool_result` (`{id, output | error, duration_ms}`), `done` (`{status}`). `applyEvent` folds them into a transcript; it ignores events it has already applied.
- The transcript follows new output while the reader is at the bottom and stops following once they scroll up (with a 跳到最新 button). Text shows a cursor while streaming. Tool calls are one line, collapsed; a failed one opens itself.
- 停止 is shown only while the run is active and cancels it on the server; the run ends as 已停止, not as a failure.
- Status of things that change slowly (a run list, a job queue) is polled with `usePolling` only while something is active, never with a fixed `setInterval`.

## Uploads

- `FileUpload` checks type and size before sending and reports each rejected file by name. The server repeats every check: it is the authority, the client only saves a failed upload.
- Each file shows progress, can be canceled while uploading, and offers 重试 after a failure. Finished items leave the queue on their own; failures stay until handled.
- Downloads go through the API's `download()` helper so an expired session refreshes instead of showing a JSON error.

## Wizards

- One reactive state object for all steps; going back never loses input. `useWizard` gates 下一步 on the current step's `validate` and lets the user jump back to any visited step.
- Check everything that can be checked before the final step, and show it (a preview table with a 问题 column). The last step before submitting summarizes what will happen (`将导入 N 行，跳过 M 行`).
- The result step reports partial success precisely (which rows failed and why) and locks the earlier steps. Leaving mid-flow asks first (`useDirtyGuard` with `{ route: true }`).

## Charts

- ECharts only, through `Chart`: colors come from `--chart-1..8`, text and axes from the theme tokens, and it redraws on light/dark switch and resizes with its container. It loads on first use.
- Every chart has a title (`.card-head`), an empty state instead of empty axes, and a tooltip. Bars for counts over time, a donut only for a few parts of a whole; no 3D, no dual axes.

## Interaction rules

- **One primary per area.** A toolbar, a dialog footer, and a detail head each have at most one `.primary`. Everything else is secondary or ghost.
- **Button placement.** Dialog footer: `取消` then the primary, right-aligned. Detail head: back (narrow only), title, actions on the right.
- **In-flight.** A submit button is disabled while its request runs and its label switches to the progressive form (`保存中`). Do not pre-disable a submit button because the form is invalid; show the errors on submit instead.
- **Disabled needs a reason.** A disabled action carries a `title` that says why (`title="运行中的任务不能删除"`).
- **Every mutation gives feedback.** Success: a short toast in past tense (`已保存`, `已删除`). Failure: `toast.error('保存失败', e)`, which renders the backend `msg`. Never a silent failure, never `alert()`. Toasts stack bottom-right so they never cover headers or toolbars.
- **Unsaved edits are never discarded silently.** Closing a dirty form (X, Esc, mask, 取消) or navigating away asks first.
- **Destructive actions confirm** with a message that names the object and the consequence (`删除「订单同步」后不可恢复。`). The confirm focuses `取消`.
- **Lists stay put.** A refresh keeps the current rows on screen; only the first load shows the loading row. A failed load shows the error and a `重试` button, never an empty state.
- **URL is the state.** Filters, page, selection, and active tab survive reload, back/forward, and sharing a link.
- **Session expiry** redirects to login with `next`, and login returns there. This is handled in `api/http.ts`; pages do nothing.

## Forms

```html
<form class="form" novalidate v-on="v.form" @submit.prevent="submit">
  <label class="field">
    <span class="field-label required">名称</span>
    <input v-model="form.name" name="name" :aria-invalid="!!v.error('name')" />
    <span v-if="v.error('name')" class="field-error">{{ v.error('name') }}</span>
  </label>
</form>
```

- Layout: `.form` (vertical stack). Use `.form-grid` (two columns, `.span-2` for full width) only for forms with many short fields.
- Validation: `useFormValidation(() => rules)`, its listeners on the `<form>`, each field identified by `name`. An error appears after the first submit, or once the user has typed in the field and left it; passing through a field (autofocus, tabbing) shows nothing. It clears as soon as the value is fixed.
- Hints go in `.field-hint` under the control, not in placeholders.
- Server-side validation errors come back as the response `msg` and surface through the save toast.

## Tables

- Wrap every table in `.table-wrap`; it scrolls and keeps the header sticky.
- `td.num` for numbers and times, `.truncate` for long text (add a `title` with the full value when it matters), `td.wrap` when a column must wrap.
- `th.col-actions` / `td.row-actions` keep actions visible while the table scrolls horizontally. Put the action buttons in a single `<span>` inside the cell.
- Empty values render as `—`, timestamps via `time.ts`, enums via `StatusPill`. No raw ids as the primary column, no `null`, no `undefined`.

## Copy

- UI text is Chinese. Product and protocol terms stay as they are (Webhook, Cron, token, ID).
- Buttons are verbs or verb + object: `新建`, `保存`, `删除`, `启用`, `重试`. Not `确定` when a specific verb exists.
- Chinese text uses full-width punctuation (，。？). The one convention: failure toasts join action and reason with a half-width colon (`保存失败:名称已存在`).
- Toasts: `已<动作>` for success, `<动作>失败:<原因>` for failure. Halfwidth colon, no exclamation marks, no trailing period.
- Empty states: `暂无<对象>`, or `没有符合条件的<对象>` when filters are applied.
- Unexplained failures say what failed and show the backend message. Do not guess a cause.
- Sentinels never reach the screen: `-1` meaning unlimited renders `不限`, unknown counts render `—`.

## Tokens

Pages use tokens (`var(--...)`) for every color, radius, shadow, size, and duration. No hex or rgba literals in pages, no raw `z-index`, no off-scale font sizes.

- Text: `--text`, `--text-secondary`, `--text-muted`. Surfaces: `--bg` (app), `--surface` (cards, sidebar, header), `--surface-muted` (table head, code), `--surface-hover`, `--surface-raised` (inputs, menus, dialogs).
- Status: `--success`, `--warning`, `--danger`, `--info`, each with a `-soft` background.
- Scale: `--text-xs..2xl` (11 to 28px), `--space-1..6` (4 to 24px), `--control-h` 32px / `--control-h-sm` 26px.
- Dark mode is the `.dark` class on `<html>`, set before first paint by the script in `index.html` (saved choice, else the OS preference) and toggled by the shell.

To rebrand, edit only the "brand" block at the top of `tokens.css` (accent colors and the two radii) in the skill, then re-copy.

## Page-local styles

A page's `<style scoped>` may lay out its own content (grid areas, a chart's height). It may not restyle shared classes or native controls, set colors or radii outside tokens, or nudge one control's padding or line-height to make it align. If something shared looks wrong, fix `base.css` or the component in the skill.

## Review checklist

Before calling a page done, check it in light and dark, at desktop width and at 390px:

1. Built from the matching reference page; toolbar order and action placement match it.
2. At most one `.primary` per area; destructive actions are in a menu and confirmed.
3. Every mutation shows a success toast or an error toast with the backend message.
4. Loading, error (with retry), and empty states all render; a refresh does not blank the table.
5. Filters, sort, page, selection, and tab are in the URL; reload restores the view.
6. No `h1` in the page, no hex colors, no raw z-index, no inline styles that restate a class.
7. Enums render through `StatusPill`, times through `time.ts`, empty values as `—`.
8. Keyboard: rows open with Enter, menus and dialogs close with Esc (only the topmost layer), focus returns to the trigger.
9. Closing a form with edits asks first; a dialog opened from a dialog appears on top.
10. Streams survive a reload; uploads can be canceled and retried; nothing polls while idle.
