// The agent run transcript: what the backend's run events (biz/service/run)
// build up to. Pure functions, so the same reducer serves live streams, replays,
// and tests.

export type RunStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'canceled'

export interface ThinkBlock {
  kind: 'think'
  text: string
}

export interface TextBlock {
  kind: 'text'
  text: string
}

export interface ToolBlock {
  kind: 'tool'
  id: string
  name: string
  args: unknown
  status: 'running' | 'succeeded' | 'failed'
  output?: unknown
  error?: string
  durationMs?: number
  subBlocks?: Block[]
}

export type Block = ThinkBlock | TextBlock | ToolBlock

export interface ProcessBlock {
  kind: 'process'
  key: string
  items: (ThinkBlock | ToolBlock)[]
  thinking: number
  tools: number
  running: number
  failed: number
}

export type DisplayBlock = Block | ProcessBlock

export interface Transcript {
  status: RunStatus
  error: string
  blocks: Block[]
  // last applied event id, for resuming
  seq: number
  done: boolean
}

export const emptyTranscript = (): Transcript => ({ status: 'queued', error: '', blocks: [], seq: 0, done: false })

type Payload = Record<string, unknown>

const TARGET_KEYS = ['command', 'cmd', 'path', 'file_path', 'query', 'sql', 'url', 'name', 'id'] as const

// Temporarily closes an unclosed fenced code block while streaming Markdown so
// half-streamed code blocks do not flicker as plain text.
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

// Extracts a concise, human-readable summary of tool arguments for a 1-line header.
export function toolSummary(args: unknown): string {
  if (args && typeof args === 'object' && !Array.isArray(args)) {
    const obj = args as Record<string, unknown>
    for (const k of TARGET_KEYS) {
      const v = obj[k]
      if (typeof v === 'string' && v.trim()) {
        const s = v.trim()
        return s.length > 80 ? `${s.slice(0, 80)}…` : s
      }
    }
  }
  const s = JSON.stringify(args ?? {})
  if (s === '{}') return ''
  return s.length > 80 ? `${s.slice(0, 80)}…` : s
}

function appendBlockEvent(blocks: Block[], type: string, data: Payload): Block[] {
  const last = blocks[blocks.length - 1]
  const chunk = String(data.delta ?? data.text ?? data.content ?? '')
  switch (type) {
    case 'think':
    case 'reasoning':
      if (!chunk) return blocks
      if (last?.kind === 'think') return [...blocks.slice(0, -1), { ...last, text: last.text + chunk }]
      return [...blocks, { kind: 'think', text: chunk }]
    case 'text':
    case 'delta':
      // consecutive chunks extend one block; a tool call or thinking block in between starts a new one
      if (last?.kind === 'text') return [...blocks.slice(0, -1), { ...last, text: last.text + chunk }]
      return [...blocks, { kind: 'text', text: chunk }]
    case 'tool_call':
      return [
        ...blocks,
        {
          kind: 'tool',
          id: String(data.id ?? data.call_id ?? ''),
          name: String(data.name ?? ''),
          args: data.args ?? data.arguments,
          status: 'running',
        },
      ]
    case 'tool_result': {
      const id = String(data.id ?? data.call_id ?? '')
      const hasErr = Boolean(data.error) || data.is_error === true || data.ok === false
      return blocks.map((b) =>
        b.kind === 'tool' && b.id === id
          ? {
              ...b,
              status: hasErr ? 'failed' : 'succeeded',
              output: data.output ?? data.content,
              error: data.error
                ? String(data.error)
                : hasErr && typeof data.content === 'string'
                  ? data.content
                  : undefined,
              durationMs: data.duration_ms != null ? Number(data.duration_ms) : b.durationMs,
            }
          : b,
      )
    }
    default:
      return blocks
  }
}

// Applies one event. Events at or below the current seq are ignored, so a
// replay that overlaps what was already shown changes nothing.
export function applyEvent(t: Transcript, type: string, data: Payload, seq: number): Transcript {
  if (seq && seq <= t.seq) return t
  const next = { ...t, seq: seq || t.seq, blocks: t.blocks }
  const parentId = String(data.tool_call_id ?? data.parent_call_id ?? data.parentCallId ?? '')
  if (
    parentId &&
    (type === 'think' ||
      type === 'reasoning' ||
      type === 'text' ||
      type === 'delta' ||
      type === 'tool_call' ||
      type === 'tool_result')
  ) {
    next.blocks = t.blocks.map((b) =>
      b.kind === 'tool' && b.id === parentId ? { ...b, subBlocks: appendBlockEvent(b.subBlocks ?? [], type, data) } : b,
    )
    return next
  }
  switch (type) {
    case 'think':
    case 'reasoning':
    case 'text':
    case 'delta':
    case 'tool_call':
    case 'tool_result':
      next.blocks = appendBlockEvent(t.blocks, type, data)
      break
    case 'status':
      next.status = data.status as RunStatus
      next.error = String(data.error ?? '')
      break
    case 'done':
      next.status = (data.status as RunStatus) ?? next.status
      next.done = true
      // a canceled or crashed run leaves its open tool calls unfinished
      next.blocks = t.blocks.map((b) =>
        b.kind === 'tool' && b.status === 'running' ? { ...b, status: 'failed', error: '未完成' } : b,
      )
      break
  }
  return next
}

// Groups consecutive intermediate blocks (think + tool) into a single ProcessBlock
// when foldProcess is enabled, so long tool chains fold behind one summary line.
export function groupBlocks(blocks: Block[], foldProcess = false): DisplayBlock[] {
  if (!foldProcess) return blocks
  const out: DisplayBlock[] = []
  let cur: ProcessBlock | null = null
  const flush = () => {
    if (!cur) return
    out.push(cur)
    cur = null
  }
  blocks.forEach((b, i) => {
    if (b.kind === 'text') {
      flush()
      out.push(b)
      return
    }
    if (!cur) {
      cur = { kind: 'process', key: `p-${i}`, items: [], thinking: 0, tools: 0, running: 0, failed: 0 }
    }
    cur.items.push(b)
    if (b.kind === 'think') cur.thinking++
    else {
      cur.tools++
      if (b.status === 'running') cur.running++
      else if (b.status === 'failed') cur.failed++
    }
  })
  flush()
  return out
}

export const isActive = (s: RunStatus) => s === 'queued' || s === 'running'
