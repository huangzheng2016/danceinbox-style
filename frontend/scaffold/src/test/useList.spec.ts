import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { useList, type Page } from '@/base/useList'
import { useFetch } from '@/base/useFetch'

type Params = { q: string; status: string; page: number; size: number; sort: string }

function setup(fetcher: (p: Params, s: AbortSignal) => Promise<Page<number>>, initial = '/') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { render: () => null } }],
  })
  let list!: ReturnType<typeof useList<number, { q: string; status: string }>>
  const C = defineComponent({
    setup() {
      list = useList(fetcher, { q: '', status: '' }, { search: 'q' })
      return () => h('div')
    },
  })
  return router.push(initial).then(async () => {
    await router.isReady()
    mount(C, { global: { plugins: [router] } })
    await flushPromises()
    return { router, list }
  })
}

describe('useList', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('fetches with defaults, then reflects filter changes in the URL and resets page', async () => {
    const fetcher = vi.fn(async () => ({ items: [1, 2], total: 100 }))
    const { router, list } = await setup(fetcher, '/?page=3')
    expect(fetcher).toHaveBeenLastCalledWith(
      { q: '', status: '', page: 3, size: 20, sort: '' },
      expect.any(AbortSignal),
    )
    list.filters.status = 'enabled'
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ status: 'enabled' })
    expect(fetcher).toHaveBeenLastCalledWith(
      { q: '', status: 'enabled', page: 1, size: 20, sort: '' },
      expect.any(AbortSignal),
    )
  })

  it('debounces the search key only', async () => {
    const fetcher = vi.fn(async () => ({ items: [], total: 0 }))
    const { router, list } = await setup(fetcher)
    const calls = fetcher.mock.calls.length
    list.filters.q = 'a'
    await nextTick()
    list.filters.q = 'ab'
    await flushPromises()
    expect(fetcher.mock.calls.length).toBe(calls)
    vi.advanceTimersByTime(300)
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ q: 'ab' })
    expect(fetcher.mock.calls.length).toBe(calls + 1)
  })

  it('normalizes an unsupported page size and follows back/forward', async () => {
    const fetcher = vi.fn(async () => ({ items: [], total: 0 }))
    const { router, list } = await setup(fetcher, '/?size=37&status=x')
    expect(list.size.value).toBe(20)
    expect(list.filters.status).toBe('x')
    await router.replace({ query: { status: 'y' } })
    await flushPromises()
    expect(list.filters.status).toBe('y')
  })

  it('steps back when the current page no longer exists', async () => {
    let total = 41
    const fetcher = vi.fn(async () => ({ items: [], total }))
    const { router, list } = await setup(fetcher, '/?page=3')
    total = 40
    await list.refresh()
    await flushPromises()
    expect(router.currentRoute.value.query.page).toBe('2')
  })
})

describe('useFetch', () => {
  it('drops a stale response and surfaces the backend msg', async () => {
    let resolveFirst!: (v: string) => void
    const fn = vi
      .fn()
      .mockImplementationOnce(() => new Promise<string>((r) => (resolveFirst = r)))
      .mockImplementationOnce(async () => 'second')
      .mockImplementationOnce(async () => Promise.reject({ response: { data: { code: 1004, msg: '记录不存在' } } }))
    const s = useFetch(fn)
    const first = s.refresh()
    await s.refresh()
    resolveFirst('first')
    await first
    expect(s.data.value).toBe('second')
    await s.refresh()
    expect(s.error.value).toBe('记录不存在')
  })
})
