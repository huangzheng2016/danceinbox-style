# Code, JSON, Markdown, and Hover Content

The system splits code display into exactly two components.

## `CodeEditor` - editable code input

Transparent `<textarea>` stacked over a highlighted `<pre>` overlay (styles in `code-editor.css`, tokens from `lib/codeHighlight.ts`, a regex tokenizer):

- Props: `modelValue`, `lang?: 'sql' | 'json' | 'text'`, `readonly?`, `placeholder?`, `rows?` (default 8), `minHeight?`, `maxHeight?`.
- The textarea keeps text transparent with a visible caret (`caret-color`); the `pre` behind renders colored tokens (`tok-k` keyword / `tok-s` string / `tok-n` number / `tok-l` literal / `tok-c` comment) and scroll-syncs via `transform: translate(...)`.
- Focus ring follows the input spec (`--focus-ring` on `:focus-within`).
- Use for: SQL/JSON and other code editing forms. For JSON specifically, prefer `JsonField` below.

## `CodeViewer` - read-only code display

CodeMirror 6, configured once and reused:

- Props: `code`, `lang?: 'go' | 'sql' | 'text'`, `maxHeight?`, `minHeight?`. The union is extensible: a project may add more grammar-backed values (e.g. a domain language routed to an existing grammar) as long as each one maps to exactly one language extension in the shared setup.
- Line numbers ALWAYS on; read-only + non-editable; line wrapping on.
- `lang="go"` (and any project-added Go-family alias) goes through the Go grammar (`@codemirror/lang-go`); `lang="sql"` goes through a small custom `StreamLanguage`; highlighting via `classHighlighter` mapped to the `cmt-*` token classes in `base.css`.
- The editor surface matches inputs (`--input-bg` + `--border-soft` + `--radius-md`); `grow-fill` variant fills a split-detail pane.

Decision rule: editing -> `CodeEditor`; display -> `CodeViewer`.

## `JsonViewer` - read-only JSON

- Default view is a field list (key + scalar value, nested objects expandable with count badges); a toggle switches to the highlighted raw view. The mode persists in `localStorage['<app>.jsonviewer.mode']` (`list` / `raw`).
- Non-JSON input degrades to raw text display.
- Any JSON payload shown to the user goes through this component.
- Large raw JSON/payload blocks default to collapsed via the `collapsedSummary` prop: the viewer wraps in a `<details>` that starts closed and shows the summary line. Omit the prop only for small JSON that should stay visible.

## `JsonField` - JSON input

- `CodeEditor`-style overlay (lang `json`) + live `JSON.parse` validation on every input.
- Emits `update:error`; renders `JSON 格式错误:...` in `.op-err` below the field. `allowEmpty` controls whether blank is valid (default true).
- The required JSON input for every form.

## `MarkdownText` - agent/user markdown

markdown-it (`html: false, linkify: true, breaks: true`); links forced `target="_blank" rel="noopener noreferrer"`; tables wrapped in a scroll container. Use for agent-generated markdown and long user-authored text. Never `v-html` raw content anywhere else.

## `HoverPopover` - rich hover content

- Hover-intent popover (open delay via `delay` prop, 200ms close delay), fixed-positioned above the trigger, clamped to the viewport, `width` prop (default 360), `align: 'left' | 'right'`.
- Shadow is `var(--shadow-tooltip)` - the lightest floating shadow, reserved for this hover layer (menus/dialogs use `--shadow-floating` / `--shadow-dialog`; never hand-roll a value).
- Use for multi-line or structured hover content. Native `title` is only for one-liners (and is REQUIRED on condition-disabled buttons and truncated cells).

## Anti-patterns

| Don't | Do |
|-------|-----|
| Installing Monaco / a second CodeMirror setup | `CodeEditor` / `CodeViewer` |
| JSON.stringify into `<pre>` | `JsonViewer` |
| Plain textarea for JSON | `JsonField` |
| `v-html` on API content | `MarkdownText` |
| `title="...multi-line paragraph..."` | `HoverPopover` |
| `CodeViewer` for an edit form | `CodeEditor` |
