import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createSseParser, readEventStream } from '@/base/sse'
import { applyEvent, emptyTranscript } from '@/base/agent'
import { parseCsv } from '@/base/csv'
import { highlight, jsonObjectError } from '@/base/highlight'
import { fmtBytes, fmtMs } from '@/base/format'
import { usePolling } from '@/base/usePolling'
import { useWizard } from '@/base/useWizard'
import { useUpload } from '@/base/useUpload'

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('SSE parser', () => {
  it('handles events split across chunks, CRLF, multi-line data and comments', () => {
    const feed = createSseParser()
    expect(feed('id: 1\nevent: text\nda')).toEqual([])
    expect(feed('ta: {"delta":"a"}\n\n: ping comment\n\nid: 2\r\nevent: done\r\ndata: x\r\ndata: y\r\n\r\n')).toEqual([
      { id: '1', event: 'text', data: '{"delta":"a"}' },
      { id: '2', event: 'done', data: 'x\ny' },
    ])
  })
})

function streamResponse(chunks: string[]) {
  const enc = new TextEncoder()
  const body = new ReadableStream<Uint8Array>({
    start(c) {
      for (const chunk of chunks) c.enqueue(enc.encode(chunk))
      c.close()
    },
  })
  return new Response(body, { status: 200, headers: { 'Content-Type': 'text/event-stream' } })
}

describe('readEventStream', () => {
  it('reconnects with Last-Event-ID after a dropped connection and stops at the final event', async () => {
    const calls: (string | null)[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(async (_url: string, init: RequestInit) => {
        calls.push(new Headers(init.headers).get('Last-Event-ID'))
        return calls.length === 1
          ? streamResponse(['id: 1\nevent: text\ndata: {}\n\n', 'id: 2\nevent: text\ndata: {}\n\n'])
          : streamResponse(['id: 3\nevent: done\ndata: {}\n\n'])
      }),
    )
    const seen: string[] = []
    await readEventStream('/s', { onEvent: (e) => seen.push(e.id), isFinal: (e) => e.event === 'done' })
    expect(seen).toEqual(['1', '2', '3'])
    expect(calls).toEqual([null, '2'])
  })

  it('gives up on an error onError rejects', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response(JSON.stringify({ code: 1004, msg: '运行记录不存在' }), { status: 404 })),
    )
    await expect(readEventStream('/s', { onEvent: () => {}, onError: () => false })).rejects.toThrow('运行记录不存在')
  })
})

describe('agent transcript', () => {
  it('merges text, tracks tools, ignores replayed events', () => {
    let t = emptyTranscript()
    const events: [string, Record<string, unknown>, number][] = [
      ['status', { status: 'running' }, 1],
      ['text', { delta: '你' }, 2],
      ['text', { delta: '好' }, 3],
      ['tool_call', { id: 't1', name: 'lookup', args: { q: 1 } }, 4],
      ['text', { delta: '!' }, 3],
      ['tool_result', { id: 't1', output: { ok: true }, duration_ms: 12 }, 5],
      ['text', { delta: '完成' }, 6],
      ['done', { status: 'succeeded' }, 7],
    ]
    for (const [type, data, seq] of events) t = applyEvent(t, type, data, seq)
    expect(t.status).toBe('succeeded')
    expect(t.done).toBe(true)
    expect(t.blocks).toEqual([
      { kind: 'text', text: '你好' },
      {
        kind: 'tool',
        id: 't1',
        name: 'lookup',
        args: { q: 1 },
        status: 'succeeded',
        output: { ok: true },
        error: undefined,
        durationMs: 12,
      },
      { kind: 'text', text: '完成' },
    ])
  })

  it('marks tools still running at the end as unfinished', () => {
    let t = applyEvent(emptyTranscript(), 'tool_call', { id: 't1', name: 'x', args: {} }, 1)
    t = applyEvent(t, 'done', { status: 'canceled' }, 2)
    expect(t.blocks[0]).toMatchObject({ status: 'failed', error: '未完成' })
  })
})

describe('parseCsv', () => {
  it('handles quotes, embedded commas and newlines, BOM, CRLF and blank lines', () => {
    expect(parseCsv('\uFEFF名称,配置\r\n"a,b","{""k"": 1}"\r\n\r\n"多\n行",\n')).toEqual([
      ['名称', '配置'],
      ['a,b', '{"k": 1}'],
      ['多\n行', ''],
    ])
  })
})

describe('highlight and format', () => {
  it('escapes HTML before adding token spans', () => {
    const out = highlight('{"<img src=x onerror=alert(1)>": 1}', 'json')
    expect(out).not.toContain('<img')
    expect(out).toContain('&lt;img')
  })

  it.each([
    ['', ''],
    ['{"a":1}', ''],
    ['[1]', '需要 JSON 对象，例如 {"key": "value"}'],
  ])('jsonObjectError(%s)', (input, want) => {
    expect(jsonObjectError(input)).toBe(want)
  })

  it('formats sizes and durations', () => {
    expect([fmtBytes(512), fmtBytes(1536), fmtBytes(20 << 20), fmtBytes(null)]).toEqual([
      '512 B',
      '1.5 KB',
      '20 MB',
      '—',
    ])
    expect([fmtMs(850), fmtMs(1200), fmtMs(184_000)]).toEqual(['850 ms', '1.2 s', '3 分 4 秒'])
  })
})

describe('usePolling', () => {
  it('waits for the previous run, backs off on errors, pauses while hidden', async () => {
    vi.useFakeTimers()
    vi.spyOn(Math, 'random').mockReturnValue(0.5)
    const enabled = ref(true)
    let calls = 0
    let fail = false
    let release: () => void = () => {}
    const fn = vi.fn(() => {
      calls++
      return new Promise<void>((resolve, reject) => (release = () => (fail ? reject(new Error()) : resolve())))
    })
    mount(defineComponent({ setup: () => (usePolling(fn, { interval: 1000, enabled }), () => h('div')) }))

    await vi.advanceTimersByTimeAsync(1000)
    expect(calls).toBe(1)
    await vi.advanceTimersByTimeAsync(5000)
    expect(calls).toBe(1) // slow request: no overlap
    release()
    await vi.advanceTimersByTimeAsync(1000)
    expect(calls).toBe(2)

    fail = true
    release()
    await vi.advanceTimersByTimeAsync(1000)
    expect(calls).toBe(2) // backed off to 2x
    await vi.advanceTimersByTimeAsync(1000)
    expect(calls).toBe(3)

    fail = false
    release()
    Object.defineProperty(document, 'hidden', { value: true, configurable: true })
    await vi.advanceTimersByTimeAsync(10_000)
    expect(calls).toBe(3)
    Object.defineProperty(document, 'hidden', { value: false, configurable: true })
    document.dispatchEvent(new Event('visibilitychange'))
    await flushPromises()
    expect(calls).toBe(4) // refreshes as soon as visible

    release()
    enabled.value = false
    await nextTick()
    await vi.advanceTimersByTimeAsync(10_000)
    expect(calls).toBe(4)
  })
})

describe('useWizard', () => {
  it('gates forward moves, keeps backward moves free, jumps only to visited steps', async () => {
    let ok = false
    const w = useWizard([
      { key: 'a', label: 'A', validate: () => ok },
      { key: 'b', label: 'B' },
      { key: 'c', label: 'C' },
    ] as const)
    expect(await w.next()).toBe(false)
    ok = true
    await w.next()
    await w.next()
    expect(w.current.value.key).toBe('c')
    w.goTo(0)
    expect(w.index.value).toBe(0)
    w.goTo(2)
    expect(w.index.value).toBe(2)
    w.reset()
    w.goTo(2)
    expect(w.index.value).toBe(0)
  })
})

describe('useUpload', () => {
  it('reports progress, supports cancel and retry', async () => {
    const pending: { resolve: () => void; reject: (e: unknown) => void; signal: AbortSignal }[] = []
    let up!: ReturnType<typeof useUpload<string>>
    mount(
      defineComponent({
        setup() {
          up = useUpload<string>((file, { signal, onProgress }) => {
            onProgress(0.5)
            return new Promise((resolve, reject) => {
              pending.push({ resolve: () => resolve(file.name), reject, signal })
              signal.addEventListener('abort', () => reject(Object.assign(new Error(), { name: 'CanceledError' })))
            })
          })
          return () => h('div')
        },
      }),
    )
    up.add([new File(['x'], 'a.txt'), new File(['y'], 'b.txt'), new File(['z'], 'c.txt')])
    expect(up.items.value.map((i) => i.status)).toEqual(['uploading', 'uploading', 'waiting'])
    expect(up.items.value[0].progress).toBe(0.5)

    up.cancel(up.items.value[1].id)
    await flushPromises()
    expect(up.items.value[1].status).toBe('canceled')
    expect(up.items.value[2].status).toBe('uploading') // a slot freed up

    up.retry(up.items.value[1].id)
    await flushPromises()
    expect(up.items.value[1].status).toBe('waiting')
    pending[0].resolve()
    await flushPromises()
    expect(up.items.value[0].status).toBe('done')
    expect(up.items.value[1].status).toBe('uploading')
  })
})
