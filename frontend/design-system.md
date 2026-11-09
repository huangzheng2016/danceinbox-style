# Design System

Native HTML controls styled once by `styles/base.css`, a small set of Vue components in `base/`, and reference pages that every new page is copied from. There is no component library and no per-page styling of shared things.

Consistency comes from copying working code, not from following rules. When you build a page, start from the closest reference page; when you need a control, use the one listed below.

## Files and ownership

| Skill path | Lands in project as | Owner |
|---|---|---|
| `frontend/styles/tokens.css`, `base.css` | `src/styles/` | Skill. Copied verbatim; to change, change the skill and re-copy. |
| `frontend/base/*` | `src/base/` | Skill, except `app.ts` and `labels.ts`. |
| `frontend/base/app.ts` | `src/base/app.ts` | Project: brand name, storage prefix, nav (with roles), nav badges, user hook. |
| `frontend/base/labels.ts` | `src/base/labels.ts` | Project: every status enum's text and tone. |
| `frontend/scaffold/*` | the project's `frontend/` | Project, from day one. |
| `frontend/assets/fonts/*` | `public/fonts/` (with licenses) | Skill. |

Updating a project to a newer skill version means re-copying `styles/` and `base/` (minus the two project files) and fixing whatever no longer compiles.

## Page patterns

Pick the row by what the page does, then copy its reference.

| Page does | Shape | Copy from |
|---|---|---|
| Manage a collection (browse, filter, create, edit, batch, delete) | List + detail split | `scaffold/src/pages/ExampleList.vue` + `ExampleFormDialog.vue` |
| One object with more than fits in the split pane (tabs, raw data, history) | Full detail page | `scaffold/src/pages/ExampleDetail.vue` |
| Agent or long-running job with streamed output | Run panel (a detail-page tab) | `scaffold/src/pages/ExampleRuns.vue` |
| Files belonging to an object | Attachments panel (a detail-page tab) | `scaffold/src/pages/ExampleAttachments.vue` |
| A task with several dependent steps (import, onboarding, setup) | Wizard page | `scaffold/src/pages/ExampleImport.vue` |
| Configure the system (one document of settings, versioned) | Settings page | `scaffold/src/pages/Settings.vue` + `SettingsHistory.vue` |
| Search a history nobody edits (audit, deliveries, job attempts) | Read-only log (list + detail) | `scaffold/src/pages/AuditList.vue` |
| What happened to one object | Timeline tab on its detail page | `scaffold/src/pages/ExampleHistory.vue` |
| Overview, dashboard | Scrolling page of cards | `scaffold/src/pages/Home.vue` |
| Sign in, error pages | Bare page, no shell | `scaffold/src/pages/Login.vue` (route `meta.bare`) |

The shell renders the page title from `app.ts` nav (or `route.meta.title`). Pages never render an `h1` or their own header.

### List + detail

- Page root is `.split` with `.has-detail` while a row is selected. Under 960px only one pane shows; the detail head has a back button (`.split-back`).
- `useList(fetcher, filters, { search })` owns filters, page, size, loading, and errors. All of it lives in the URL. Selection is a separate `useRouteQuery({ id: '' })` key. Do not keep list state in local refs.
- Toolbar order, left to right: search input (`.search`), filters (`Segmented` for a status with 2 to 5 values, `Select` or `Combobox` for more, `DateRange` for time), `.spacer`, refresh icon button, primary `新建`.
- Batch actions: a checkbox column (`th.col-check` / `td.col-check`, the cell stops click propagation) driven by `useSelection`. While anything is selected the toolbar is replaced by `.toolbar.batch`: `已选 N 项`, the actions, `取消选择`. The server reports done and skipped counts; the toast says both (`已删除 3 项，跳过 2 项`). Destructive batches confirm first.
- Sorting: `SortTh` on columns the server can sort (it needs an index), bound to `list.sort` / `list.setSort`. The server whitelists sortable fields with `db.Order`.
- Time in list columns is relative (`fmtRelative`, full time in `v-tip`); absolute times belong in detail views.
- An empty result with filters applied says `没有符合条件的<对象>` and offers `清除筛选` (`list.reset`).
- Every filter in the URL is visible. A filter that arrives from a link and has no toolbar control (`?sandbox_id=12` from a sandbox page) shows in the toolbar as `<span class="filter-chip">沙箱：构建机 A<button type="button" aria-label="清除筛选" @click="list.filters.sandbox_id = ''"><Icon name="x" :size="14" /></button></span>`, with the readable name, never the raw id. A link that sends a filter value (`?status=issue`) is only added together with support for that value on the target page; test it by clicking it.
- Picking a related object (owner, mailbox, parent ticket) is a `Combobox` with `:load` that searches by name and shows the readable code. Never a text box for a database id or an enum code; filter options come from `labelOptions(MAP)`.
- Table columns: identifying name first, state next, numbers and times right-aligned (`.num`), actions last.
- Row actions: at most one inline ghost button (usually `编辑`) plus an `ActionMenu` for everything else; a menu that would hold one item is a button instead. Destructive actions live in the menu, never inline.
- Rows are clickable (`tr.clickable` + `tabindex="0"` + Enter) and select into the detail pane. Buttons inside rows use `@click.stop`.
- Create and edit use the same form dialog component. Detail panes are read views with an `编辑` button, not inline forms, so switching rows never discards edits.
- When the pane gets crowded, link to a full detail page (`完整详情`) instead of growing the pane.

### Full detail page

- Route `/<feature>/:id` with `meta.parent` set to the list path; `usePageTitle(() => item.name)` makes the shell header read `示例 / 订单同步` with a link back.
- Top: an `Alert` for any state the user must know about (disabled, read-only, expiring), with the fixing action inside it. Then a `.toolbar` with the status pill and relative update time on the left, actions on the right. An action lives in one place: if the alert carries `启用`, the toolbar does not repeat it. No alert for the normal state (`当前运行正常`).
- Show the identifier people say out loud (`SEC-12`, an order number) next to the status, with `CopyText`; when other screens refer to the object by its database id, show both (`SEC-12（#12）`).
- Sections go in `Tabs` bound to a `useRouteQuery` key so reload and links keep the tab. Raw payloads go in a `CodeBlock` tab.
- Deleting from the detail page navigates back to the list.
- Links to another object's page (a ticket's run, an audit entry's target) use `linkFrom(path, route)` from `backLink.ts`. The shell's back link then returns to the exact page, tab and filters the user left, instead of the target's list.

### Settings page

- One document, one form, one primary: `.save-bar` at the bottom holds `撤销修改` and `保存` and says when there are unsaved edits; `保存` is disabled with `v-tip="没有修改"` until something changes. Leaving with edits asks first (`useDirtyGuard` with `{ route: true }`).
- Group fields in cards by what they affect (公告, 运行, 附件), not by type. Every field is read by a service on each use, so a save applies at once; anything that needs a restart is config, not a setting.
- Each save names the version it was edited from. A 409 opens a dialog with a `DiffView` of the saved version against the draft and three ways out: `继续编辑`, `放弃我的修改`, `用我的修改覆盖`. Never overwrite silently, never discard the draft silently.
- The line above the form says where the values come from: `版本 3 · admin保存于 2 小时前`, or `尚未保存过，当前为默认值` for version 0.
- People who may read but not save get the same page inside `<fieldset :disabled>`, an `Alert` saying who can change it, and no save bar.
- `历史版本` lists every version; selecting one shows its diff against the previous version and `恢复此版本` (a confirm, then a new version, so a restore can itself be undone).

### Read-only log

- The list + detail split without create, edit or batch. The detail is the selected row; a change renders as `DiffView` with field names from `labels.ts` (`AUDIT_FIELDS`) and values through a formatter (units, enums, 不限), never as JSON.
- Filters people choose are controls (操作人 `Combobox`, 操作 and 对象类型 `Select`, `DateRange`); a filter that only arrives from a link (one object) is a `.filter-chip`. Names travel in the URL beside ids (`actor_name`, `target_name`) so a shared link reads correctly without another lookup.
- Link to the object only while it still has a page (not after a delete).
- An object's own history is a `Timeline` tab on its detail page (`ExampleHistory.vue`), with a link into the full log filtered to it for those who may see the log.

### Scrolling page

- Root `.page` (a vertical stack). Optional `.stats` row of stat cards, then one `.card` per block. The stat cards are the markup in `Home.vue` (`.stats > .card` with `.stat-label` / `.stat-value`); a hand-made `.stat` block has no styles and its numbers run into the labels (`7342` for `734` + `2`).
- Each figure appears once per screen. No stat card that repeats a count shown in a card title, a list total or a banner below it; no banner restating the stat cards.
- Sections with nothing in them are not rendered: a work queue with 0 items, an "attention" block with no issues, a chart with no series. If every section is empty, show one `EmptyState` for the page (`暂无待处理事项`), not six empty cards. Before any data exists (nothing connected, nothing imported), hide the all-zero stats and show the first step instead.
- A card that needs a title uses `.card-head` (title + optional `.card-desc` on the left, actions on the right). `.card-desc` is for information the title cannot carry (a time window, a scope); it never restates the title.
- More than one card is fine. Use `Tabs` only when blocks are alternatives the user switches between, not to hide length.

## Which control

| Need | Use |
|---|---|
| Button | `<button>` (secondary). Add `.primary` for the single main action of an area, `.danger` only inside a confirm, `.ghost` for low-emphasis row and toolbar actions, `.btn-sm` / `.btn-icon` for size and shape. |
| Link styled as a button | `<RouterLink class="btn ...">` |
| Icon | `<Icon name="..." />` from `icons.ts`. Add missing icons there (Lucide paths). No emoji, no glyph characters as icons. |
| On/off setting | `Switch` (`role="switch"`). Never a checkbox, never a button whose label is the state. |
| One of 2 to 5 options, all visible | `Segmented` (status filter, view switch, granularity) |
| One of a short fixed list | `Select` (string or number model, `disabledReason` per option, type-ahead). Never a native `<select>`: its open list is drawn by the OS and matches nothing else |
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
| A duration or size field | `UnitInput` (`kind="duration"` stores seconds, `kind="bytes"` stores bytes; the user picks 分钟 / 小时 / MB) |
| Events of one object, newest first | `Timeline` (one line per event, details in its slot) |
| A count beside a nav entry for work waiting on the user | `loadNavBadges` in `app.ts` (polled by the shell, hidden at 0, explained by `tip`) |
| A nav entry only some roles may use | `role` on the `NavItem`, plus the same `meta.role` on its route |
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
| One-line hint on hover (disabled reason, full time, short explanation) | `v-tip="text"` from `tooltip.ts`. Icon-only buttons (`.btn-icon` + `aria-label`) and overflowing `.truncate` text get a tip automatically. Never the `title` attribute |
| Several lines or structure on hover (metric definition, profile) | `HoverCard`; the same information must also be reachable without hovering |
| What changed between two versions, or server vs draft on a save conflict | `DiffView` with `labels` for every field path; raw JSON stays in a `CodeBlock` tab |
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
| Sizes, durations, counts, ratios | `fmtBytes`, `fmtMs`, `fmtNumber`, `fmtRatio` (`format.ts`); the helper adds the unit |
| Quota where 0 or -1 means unlimited | `fmtLimit` (`不限`) |
| Enum text outside a pill, enum filter options | `fmtLabel(MAP, v)`, `labelOptions(MAP)` |
| Link to another object's page | `linkFrom(path, route)` (`backLink.ts`), so the header's back link returns here |
| URL filter with no toolbar control | `.filter-chip` with a clear button |
| Secondary line in a table cell | `.cell-sub` |

Not in the system: drawers, WYSIWYG rich text, collaborative editing, resumable or chunked uploads, maps. If a feature truly needs one, add it to `base/` in the skill first so every project gets the same one.

## Streaming and agent runs

- Output that is produced over time (model answers, job logs) streams over SSE from a route ending in `/stream`. The backend stores every event with a sequence number; the client resumes with `Last-Event-ID`, so a reload or dropped connection replays exactly what was missed.
- Event types: `status` (`{status, error}`), `think` / `reasoning` (`{delta}`, collapsible reasoning), `text` (`{delta}`, Markdown), `tool_call` (`{id, name, args}`), `tool_result` (`{id, output | error, duration_ms}`), `done` (`{status}`). Events carrying `tool_call_id` or `parent_call_id` nest inside the parent tool's `subBlocks`. `applyEvent` folds them into a transcript; it ignores events it has already applied.
- The transcript follows new output while the reader is at the bottom and stops following once they scroll up (with a 跳到最新 button). Text shows a cursor while streaming and heals unclosed code fences (`healStreamingMarkdown`) so streaming code blocks do not flicker. Thinking blocks collapse by default with character count (`思考过程 · N 字`). Tool calls are one line with a readable target summary (`toolSummary`), collapsed; a failed one opens itself and keeps its first error line visible if manually closed.
- Pass `:fold-process="true"` on `AgentTranscript` when a run performs multi-step thinking and tool chains before its conclusion: consecutive intermediate blocks collapse into one summary bar (`执行过程 · 思考 N 段 · 工具 M 次`), which opens automatically if any step inside failed. For advanced multi-turn chat workbenches, rAF delta batching, lazy subagent run streams, and HITL confirmation cards, see `frontend/agent-ui.md`.
- 停止 is shown only while the run is active and cancels it on the server; the run ends as 已停止, not as a failure.
- Status of things that change slowly (a run list, a job queue) is polled with `usePolling` only while something is active, never with a fixed `setInterval`. A failed poll keeps what is on screen and shows the error beside it; it never swaps loaded data for an error state.
- Model output is for people. System prompts are written in Chinese, as Markdown with sections, and ask for Chinese output with short bullet points and bold lead-ins; the page renders it with `MarkdownText`. If old output arrives as one long paragraph, fix the prompt, not the page.
- A conclusion (summary, verdict, next step) is shown open. Intermediate steps (thinking, tool calls, raw payloads) collapse into one summary line (`思考 2 段 · 工具 5 次`), and a failure in them opens itself.
- Long text with structure the source already has (an email with quoted replies, a forwarded chain, a transcript) is split into blocks along that structure, newest first, each with its own header. One `<pre>` with the whole thread is unreadable.

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
- Plot series in the order the API returns them (time series come oldest first; do not `.reverse()` them). Leave out a series that has no data in the selected range instead of showing it empty in the legend. Tooltip values use the `format.ts` helpers.
- Never paginate a graph or cut names short to fit it: grow the layout or offer a table view. A problem node hidden on page 2 of a topology is worse than a dense picture.

## Interaction rules

- **One primary per area.** A toolbar, a dialog footer, and a detail head each have at most one `.primary`. Everything else is secondary or ghost.
- **Button placement.** Dialog footer: `取消` then the primary, right-aligned. Detail head: back (narrow only), title, actions on the right.
- **In-flight.** A submit button is disabled while its request runs and its label switches to the progressive form (`保存中`). Do not pre-disable a submit button because the form is invalid; show the errors on submit instead.
- **Disabled needs a reason.** A disabled action carries `v-tip` with the reason (`v-tip="'运行中的任务不能删除'"`); a disabled option or menu item uses `disabledReason` / `title` on its item. Disabled buttons still receive hover for this.
- **Hover is feedback, not information.** Every clickable thing changes on hover (buttons, rows, options, menu items); disabled things do not. Nothing important exists only on hover: touch never shows it.
- **Every mutation gives feedback.** Success: a short toast in past tense (`已保存`, `已删除`). Failure: `toast.error('保存失败', e)`, which renders the backend `msg`. Never a silent failure, never `alert()`. Toasts stack bottom-right so they never cover headers or toolbars.
- **Unsaved edits are never discarded silently.** Closing a dirty form (X, Esc, mask, 取消) or navigating away asks first.
- **Confirm only what can hurt.** Destructive or irreversible actions confirm with a message that names the object and the consequence (`删除「订单同步」后不可恢复。`), focused on `取消`. Ordinary ones (retry, accept, refresh) do not. Do not offer a control that then refuses (a checkbox on rows that cannot be batched).
- **Lists stay put.** A refresh keeps the current rows on screen; only the first load shows the loading row. A failed load shows the error and a `重试` button, never an empty state.
- **URL is the state.** Filters, page, selection, and active tab survive reload, back/forward, and sharing a link. Panels that share a route preserve each other's keys.
- **Results stay attached to their request.** When an action's result is shown under a form the user can keep editing (a probe, a preview, a test send), the result shows the inputs it was run with, not the current form values.
- **Panels do not move while the user types.** A section's open state is its own state, not derived from whether its fields are empty.
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
- Most fields need no hint: the label says what to enter. A `.field-hint` is for a constraint the user cannot guess (`每行一个地址`, `0 表示不限`, a unit), written as a short phrase, never a restatement of the label (`给任务看的说明` under 描述) or an explanation of how the system works. A placeholder may show an example value (`ubuntu:24.04`), not instructions.
- A hint does not make a field `.span-2`; only long inputs (textarea, editors) span both columns.
- A `Switch` row is not wrapped in a `<label>` together with its hint, so clicking the text does not flip it.
- Server-side validation errors come back as the response `msg` and surface through the save toast.

## Tables

- Wrap every table in `.table-wrap`; it scrolls and keeps the header sticky.
- `td.num` for numbers and times, `.truncate` for long text (it shows the full value on hover when cut off), `td.wrap` when a column must wrap.
- `th.col-actions` / `td.row-actions` keep actions visible while the table scrolls horizontally. Put the action buttons in a single `<span>` inside the cell.
- Empty values render as `—`, timestamps via `time.ts`, enums via `StatusPill`. No raw ids as the primary column, no `null`, no `undefined`.
- A secondary line in a cell (an email under a name) is `<span class="cell-sub">`, which breaks onto its own line. Two inline elements written next to each other inside a `nowrap` cell render glued together (`adminadmin@example.test`).

## Copy

- UI text is Chinese. Product and protocol terms stay as they are (Webhook, Cron, token, ID).
- Buttons are verbs or verb + object: `新建`, `保存`, `删除`, `启用`, `重试`. Not `确定` when a specific verb exists.
- Chinese text uses full-width punctuation (，。？：（）), toasts included (`保存失败：名称已存在`).
- A space separates Chinese from Latin words and numbers, interpolated ones included: `版本 3 · admin 保存于 2 小时前`, `已选 3 项`. A value glued to the next word (`admin保存于`) is the common slip; `lint-ui` catches it in templates.
- Toasts: `已<动作>` for success, `<动作>失败：<原因>` for failure. No exclamation marks, no trailing period.
- Empty states: `暂无<对象>`, or `没有符合条件的<对象>` when filters are applied.
- Unexplained failures say what failed and show the backend message. Do not guess a cause.
- Sentinels never reach the screen: `-1` or `0` meaning unlimited renders `不限` (`fmtLimit`), unknown counts render `—`. A `0` id is "none": `未关联工单`, `未使用模板`, never `工单 0`, `#0`, `v0`. A `0` timestamp is `—` (`time.ts` does this), never 1970.
- A unit belongs to its formatter (`fmtMs`, `fmtBytes`, `fmtRatio`), never to the template: `{{ fmtMs(x) }}`, not `{{ x }} ms`, which shows `— ms` the moment the value is missing.
- Placeholders are short: `输入名称`, `搜索名称或邮箱`, not `为这份配置起一个容易识别的名称`.

### Write for the person at the screen

Pages are read by operators, not by the people who wrote the spec. The most common failure in generated projects is copy taken from design documents, schemas and code: true, precise, and useless to the reader. Before a page is done, read every string on it as someone who has never seen the code.

| Never on screen | Instead |
|---|---|
| Milestone or task codes, internal component names (`阶段二路由快照`, `网关身份`, `Agent 舰队`) | Nothing, or the plain effect (`由当前登录账号记录`) |
| Protocol commands, field and struct names (`只读 EXAMINE/BODY.PEEK`, `ReceivedAt`, `text_body`) | What it means for the user (`全程只读，不修改服务器上的邮件`, `邮件正文`) |
| Raw enum codes and role codes (`ticket.close`, `deny / timeout`, `analyst/admin`) | Labels from `labels.ts` (`关闭工单`, `超时阻断`, `分析人员`) |
| Chinese with an English tail in brackets (`轮询间隔（Poll interval）`, `注册令牌 (Bootstrap Token)`) | Chinese only. A technical identifier the user needs (to paste into a terminal) goes in its own `<code>` or `CopyText` |
| Raw units and exact internals in brackets (`15 分 0 秒（900,000 ms）`, `（67,108,864 字节）`) | One formatted value (`15 分钟`, `64 MB`) |
| Defensive notes and disclaimers (`未收到的数据不会显示为零`, `AI 不会自动关单`, `读取不会写审计`) | Nothing. State facts the user acts on, not what the system promises not to do |
| "不是 X 而是 Y", explanations of how it works (`命令直接执行，不经过模型`) | The instruction itself (`通过参数指定环境变量名`) |
| A sentence describing what the page is for, above the page | Nothing. The title and the content say it |
| Programmer voice (`不会伪造签名响应`, `worker 不会抢走任务`, `安全整数`) | Operator voice (`版本号需为正整数`) |
| Courtesy and filler (`请联系管理员`, `您可以…`) | The fact (`当前账号没有权限`) |
| Transport errors (`rpc error: code = …`, `Failed to fetch`, `Request failed with status code 500`) | The backend's Chinese message; `errorMessage` strips transport text, the backend maps errors (see `backend/workflow.md` Errors) |
| Validation that recites rules from code (`名称必须匹配 ^[a-z][a-z0-9_]*$`) | The rule in words (`名称须以小写字母开头，仅含小写字母、数字和下划线`) |

The same concept has one name everywhere: in nav, titles, buttons, labels, toasts and backend messages. Status and action words come only from `labels.ts`; a page never keeps its own map.

## Saying less

A screen earns attention by showing each thing once. Generated pages tend to add: a banner, then stat cards, then a card title with the same numbers; a subtitle under every card; a hint under every field. Remove until removing more would lose information.

- **Once per screen.** A count, a status or an action appears in one place. Toolbar `共 N 项` duplicates `Pagination`; an alert button duplicates the toolbar button; a card `<h2>` duplicates the shell title; a header link duplicates the same link in a tab.
- **Nothing empty.** Zero-item sections, `已加载 3 / 3`, `上限：未配置`, empty legend entries and placeholder rows of `—` are hidden (see Scrolling page).
- **Nothing obvious.** No subtitle that restates its title, no hint that restates its label, no legend explaining what grey means.
- **Folding is not simplifying.** Do not collapse a conclusion, a summary, or the action the user came for into `<details>`. Collapse only bulky raw material (payloads, logs, intermediate steps) behind a one-line summary, and never put a single button inside a fold: a rarely used destructive action goes into the head's `ActionMenu`.

## Tokens

Pages use tokens (`var(--...)`) for every color, radius, shadow, size, and duration. No hex or rgba literals in pages, no raw `z-index`, no off-scale font sizes.

- Text: `--text`, `--text-secondary`, `--text-muted`. Surfaces: `--bg` (app), `--surface` (cards, sidebar, header), `--surface-muted` (table head, code), `--surface-hover`, `--surface-raised` (inputs, menus, dialogs).
- Status: `--success`, `--warning`, `--danger`, `--info`, each with a `-soft` background.
- Scale: `--text-xs..2xl` (11 to 28px), `--space-1..6` (4 to 24px), `--control-h` 32px / `--control-h-sm` 26px.
- Dark mode is the `.dark` class on `<html>`, set before first paint by the script in `index.html` (saved choice, else the OS preference) and toggled by the shell.

To rebrand, edit only the "brand" block at the top of `tokens.css` (accent colors and the two radii) in the skill, then re-copy.

## Page-local styles

Every class in a template is defined in `styles/` or in that component's `<style>`. A class that exists nowhere (`.stat` instead of `.stats > .card`, `.active` where the stylesheet says `.on`, `.kv-group`) fails silently: text runs together, a toggle never shows as on. `pnpm check` runs `scripts/lint-ui.mjs`, which reports undefined classes along with the copy mistakes it can detect. Outside the verbatim `base/` and `styles/` it also rejects literal colors, raw `z-index`, page selectors that restyle a shared class, `toLocale*`/`toFixed` formatting, `MAP[x]?.text` lookups and local copies of a `labels.ts` map; mark a deliberate exception with `lint-ui-ignore` on that line.

A page's `<style scoped>` may lay out its own content (grid areas, a chart's height). It may not restyle shared classes or native controls, set colors or radii outside tokens, or nudge one control's padding or line-height to make it align. If something shared looks wrong, fix `base.css` or the component in the skill.

## Review checklist

Before calling a page done, check it in light and dark, at desktop width and at 390px:

1. Built from the matching reference page; toolbar order and action placement match it.
2. At most one `.primary` per area; destructive actions are in a menu and confirmed.
3. Every mutation shows a success toast or an error toast with the backend message.
4. Loading, error (with retry), and empty states all render; a refresh does not blank the table.
5. Filters, sort, page, selection, and tab are in the URL; reload restores the view.
6. No `h1` in the page, no hex colors, no raw z-index, no inline styles that restate a class.
7. Keyboard: rows open with Enter, menus and dialogs close with Esc (only the topmost layer), focus returns to the trigger.
8. Closing a form with edits asks first; a dialog opened from a dialog appears on top.
9. Streams survive a reload; uploads can be canceled and retried; nothing polls while idle.
10. Read every string as the operator (see "Write for the person at the screen"): no codes, field names, English tails, disclaimers, or explanations of how it works.
11. Each count, status and action appears once; sections with zero items are hidden; with no data at all the page shows one empty state.
12. Enums render through `StatusPill`, times through `time.ts`, units through `format.ts`. Open the page with missing and zero values (no relation, unlimited quota, never run): `—`, `不限`, `未关联`, never `0`, `#0`, `v0`, `— ms` or 1970.
13. Every link into this page and out of it has been clicked: filters it sends are supported and visible, and back returns where it came from.
14. `pnpm check` passes, including `lint-ui`.
15. A settings field is read by a service per use; if it needs a restart it is config. A change a person makes leaves an audit entry with readable before/after values.
