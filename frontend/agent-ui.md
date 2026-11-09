# Agent Streaming, Thinking Fold, and Tool Fold UI

Frontend patterns and shared components for rendering streamed AI runs, multi-turn agent chats, thinking blocks, tool call cards, process groups, and subagent timelines.

Built around the shared components in `frontend/base/` (`agent.ts`, `sse.ts`, `AgentTranscript.vue`, `ToolCallCard.vue`, `MarkdownText.vue`). Read this file when building or customizing agent run panels, chat workbenches, or trace viewers.

## Core design rules

1. **Conclusions stay open; intermediate work folds.** Final answers, assessments, and actionable errors render open. Thinking (`think` / `reasoning`), tool calls (`tool`), and raw payloads collapse behind a single-line summary.
2. **Failures never hide inside a fold.** A failed tool call opens itself by default (`ToolCallCard`), or if manually collapsed, shows its first error line in the header. A collapsed process group (`思考 N 段 · 工具 M 次`) turns danger-bordered and shows the failure count and first error summary when any step inside it failed.
3. **Stream resume is lossless and idempotent.** Every event carries a monotonic `seq` (and `epoch` when runs can restart). Reconnecting sends `Last-Event-ID` (or `?after_seq=`), and the reducer ignores `seq <= currentSeq`.
4. **Coalesce high-frequency deltas to animation frames.** Never trigger a full reactive array copy or Markdown re-parse on every SSE packet when tokens arrive at 100+ chunks/s; batch deltas per `requestAnimationFrame` (or 32–50ms window) and flush immediately on structural events (`tool_call`, `tool_result`, `turn_end`, `done`).
5. **Follow the bottom until the reader scrolls up.** Auto-scroll while within 24–80px of the bottom; stop immediately when the user scrolls up and show a floating `跳到最新` button.

---

## 1. Built-in `base/` components (ready out of the box)

Every project generated from this skill already includes these in `frontend/src/base/`:

| File | Responsibility |
|---|---|
| `base/sse.ts` | `createSseParser()` + `readEventStream(url, opts)` over `fetch` (sends cookies, handles 401 refresh, reconnects with `Last-Event-ID` and exponential backoff + jitter) |
| `base/agent.ts` | Pure transcript reducer (`Transcript`, `ThinkBlock`, `TextBlock`, `ToolBlock`, `ProcessBlock`, `applyEvent`, `groupBlocks`, `toolSummary`) |
| `base/MarkdownText.vue` | Safe `markdown-it` renderer (`html: false`, external links `target="_blank" rel="noopener noreferrer"`, scrollable `.md-table`, rAF-coalesced streaming re-render, `healStreamingMarkdown` unclosed code-fence closure, blinking cursor) |
| `base/ToolCallCard.vue` | Single-line collapsible tool card with spinner/status icon, readable target summary (`toolSummary`), duration (`fmtMs`), auto-open on failure, first-line error preview when collapsed, and nested `subBlocks` timeline |
| `base/AgentTranscript.vue` | Scroll-following transcript container rendering `ThinkBlock` (collapsible thinking with character count), `ToolCallCard`, `MarkdownText`, and optional `foldProcess` grouping (`执行过程 · 思考 N 段 · 工具 M 次`) |

### Standard usage in a page (`ExampleRuns.vue`)

```html
<AgentTranscript :transcript="transcript" :fold-process="true" max-height="60vh" />
```

Event vocabulary accepted by `applyEvent(transcript, type, data, seq)`:
- `status`: `{ status: 'queued' | 'running' | 'succeeded' | 'failed' | 'canceled', error?: string }`
- `think` (or `reasoning`): `{ delta?: string, text?: string, content?: string }` — extends the current `ThinkBlock` or starts a new one.
- `text` (or `delta`): `{ delta?: string, text?: string, content?: string }` — extends the current `TextBlock` or starts a new one.
- `tool_call`: `{ id: string, name: string, args: unknown }` — appends a `running` `ToolBlock`.
- `tool_result`: `{ id: string, output?: unknown, error?: string, duration_ms?: number }` — settles the matching `ToolBlock`.
- Subagent events: any `think`, `text`, `tool_call`, or `tool_result` carrying `tool_call_id` or `parent_call_id` is folded into that parent `ToolBlock.subBlocks`.
- `done`: `{ status?: RunStatus }` — marks `done = true` and settles any still-`running` tool calls as `failed` (`未完成`).

---

## 2. Streaming rendering & frame coalescing

### Healing unclosed code fences during streaming (`MarkdownText.vue`)

While a model streams a fenced code block (```` ```go ````), the closing ```` ``` ```` has not arrived yet. Without fence healing, `markdown-it` renders the half-written block as plain paragraph text until the closing fence arrives, causing layout jumps.

`healStreamingMarkdown(source)` counts fence lines (`^\s*(`{3,}|~{3,})`) and, when an opening fence is still unclosed during streaming, appends a matching closing fence before passing the string to `markdown-it`:

```ts
export function healStreamingMarkdown(src: string): string {
  let openMarker = ''
  for (const line of src.split('\n')) {
    const m = /^\s*(`{3,}|~{3,})/.exec(line)
    if (!m) continue
    const fence = m[1]
    if (!openMarker) {
      openMarker = fence
    } else if (fence[0] === openMarker[0] && fence.length >= openMarker.length) {
      openMarker = ''
    }
  }
  return openMarker ? `${src}\n${openMarker}` : src
}
```

> **Heavy streaming alternative**: If a project renders very large streaming code/diff outputs and explicitly opts into `markstream-vue`, wrap `<MarkdownRender :content="text" :is-dark="dark" :final="!live" :smooth-streaming="live" html-policy="escape" mode="chat" />` in a single `StreamMarkdown.vue` component styled with CSS tokens. For standard projects, `base/MarkdownText.vue` requires zero extra dependencies.

### Frame-batched event ingestion

When a backend pushes raw token deltas without server-side batching:
- Buffer consecutive `text` / `think` / `subagentDelta` strings in a non-reactive pending queue.
- Schedule a single flush on `requestAnimationFrame` (or flush immediately if buffered characters exceed `64 KiB` or a non-delta event arrives).
- Inside `flush()`, fold all buffered fragments into the transcript state once and trigger a single Vue reactive update.

---

## 3. Thinking fold (`ThinkBlock` / `ThinkingBlock`)

Models with reasoning (`think` / `reasoning_content`) emit long chains of thought before calling tools or answering.

Interaction contract:
- **Header line**: `<Icon name="chevron-right" />` (rotates 90° when open) + `思考中` (with spinner while this block is actively receiving chunks) or `思考过程` + character count (`{{ fmtNumber(b.text.length) }} 字` in `--text-muted`).
- **Default state**:
  - Collapsed by default (`open = false`) so the user sees tool progress and the final answer without scrolling past walls of reasoning.
  - Clicking the header toggles `open`. Once the user manually toggles it (`toggled !== null`), their choice is preserved while new chunks stream in.
- **Body**: Rendered with `<MarkdownText :source="b.text" :streaming="isStreamingBlock" />` inside a muted container (`max-height: 280px; overflow-y: auto; font-size: var(--text-sm); color: var(--text-secondary)`).

---

## 4. Tool call fold & two-level hierarchy (`ToolCallCard` + `ProcessBlock`)

### Level 1: Single tool call card (`ToolCallCard.vue`)

- **Collapsed single-line header (34px)**:
  1. Status indicator: spinner (`running`), green check (`succeeded`), or red cross (`failed`).
  2. Tool name (`.tool-name.mono`).
  3. **Smart target summary** (`toolSummary(tool.args)`): instead of raw `{"command":"ls -la"}` braces when a common target field is present, extract the first non-empty string among `command`, `cmd`, `path`, `file_path`, `query`, `sql`, `url`, `name`, `id` (truncated to 80 chars), falling back to compact `JSON.stringify(args)`.
  4. Subagent badge (when `tool.subBlocks?.length`): `子任务 · N 步`.
  5. Duration (`fmtMs(tool.durationMs)`, right-aligned `.tabular`).
  6. Chevron icon.
- **Auto-open & never-hide-errors rule**:
  - `open` defaults to `tool.status === 'failed'`.
  - If the user manually collapses a failed tool (`!open && tool.status === 'failed'`), show a one-line truncated `.tool-err-line` preview beneath the header so the failure reason remains visible at a glance.
- **Expanded body (`.tool-body`)**:
  - `参数`: `<CodeBlock :value="tool.args" :max-lines="12" />` (hidden when `args` is empty `{}`).
  - `子任务过程` (when `tool.subBlocks?.length`): renders the nested subagent blocks (thinking, tool calls, output).
  - `错误` (`failed`) or `结果` (`succeeded`): error text in `--danger` or `<CodeBlock :value="tool.output" :max-lines="12" />`.
- **Domain-specific tool views (`ToolCallDetail.vue` pattern)**:
  - When a project has a primary domain tool such as `execute` (sandbox shell), specialize its body: show `$ <command>` in a mono block, split `stdout` and `[stderr]` into separate labeled panels, and render non-zero `exitCode` as a danger badge instead of dumping one raw string.

### Level 2: Process group fold (`foldProcess` / `TranscriptProcess` / `ChatTurn`)

When an agent performs a multi-step investigation (e.g. 3 thinking blocks + 8 tool calls) before writing its answer, showing 11 separate rows above the answer pushes the conclusion off-screen (`RunTranscript.vue`, `ChatTurn.vue`).

`groupBlocks(blocks, true)` groups consecutive `think` and `tool` blocks into a `ProcessBlock`:
- **Summary bar**:
  - `执行过程`
  - `思考 {{ g.thinking }} 段` (when `thinking > 0`)
  - `工具 {{ g.tools }} 次` (when `tools > 0`)
  - `运行中 {{ g.running }}` (when `running > 0`)
  - `失败 {{ g.failed }}` in `--danger` (when `failed > 0`)
- **Default state**: Collapsed (`open = false`), **except** when `g.failed > 0` (auto-opens so the failed step is immediately visible).
- **Body**: Vertical stack with a left border hairline (`border-left: 2px solid var(--border)`), rendering each `ThinkBlock` and `ToolCallCard` in order.

---

## 5. Subagent timeline (`SubagentTimeline`)

Two ways subagent work arrives on the frontend:

1. **In-stream bucketed events (`tool_call_id` / `parent_call_id`)**:
   - Events emitted by a child agent carry the parent tool call's ID (`tool_call_id` or `parentCallId`).
   - `applyEvent` routes those events into `parentToolBlock.subBlocks`.
   - Expanding the parent `ToolCallCard` renders those `subBlocks` inline.
2. **Independent child run (`childRunId` lazy subscription, `SubagentTimeline.vue`)**:
   - When a subagent runs as its own persisted run (`parent_run_id = currentRunId`), do **not** open SSE streams for all child runs on page load.
   - Bind the child stream composable to `computed(() => (open.value ? props.childRunId : ''))` so the SSE connection / trace fetch starts only when the operator expands that subagent row.

---

## 6. Long message preview & history windowing

In multi-turn chat sessions or long transcripts:
- **Earlier turns preview (`MessageBlock.vue` / `MessageBubble.vue`)**: Keep the latest assistant output fully expanded (`latestOutput = true`). Earlier user prompts or assistant outputs that exceed 3–5 lines clamp with `max-height: 5.25em; overflow: hidden` (measured via `ResizeObserver`) and offer an `展开全文` / `收起` button.
- **Windowed rendering (`RunTranscript.vue` / `ChatItems.vue`)**: Slice rendered groups/turns to the newest `visibleCount` (e.g. `100` or `200` items). Render a top button `显示更早的记录（还有 N 段）` that increments `visibleCount` while preserving `scrollTop` (`el.scrollTop = el.scrollHeight - prevHeight + prevTop`) so expanding history never yanks the viewport.
- **Interactive HITL cards**:
  - Render pending confirmation/question requests at the bottom of the transcript with `.alert` styling, risk pill, command/diff preview, and explicit scope buttons (`允许一次`, `本会话允许`, `永久允许`, `拒绝`).
  - For production-tagged targets, require typing the target environment name before enabling the approve buttons.
