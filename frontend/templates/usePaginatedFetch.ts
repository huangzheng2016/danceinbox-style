// copy verbatim; edit only the marked regions
import { getCurrentScope, onScopeDispose, ref, shallowRef, computed } from 'vue'
import { errorMessage } from './errors'

export function usePaginatedFetch<T>(
  fetcher: (
    limit: number,
    offset: number,
    signal: AbortSignal,
  ) => Promise<{ items: T[]; total: number }>,
  initialPageSize = 20,
  opts: { timeoutMs?: number } = {},
) {
  const page = ref(1)
  const pageSize = ref(initialPageSize)
  const data = shallowRef<{ items: T[]; total: number } | null>(null)
  const loading = ref(false)
  const error = ref('')
  let seq = 0
  let inflight: AbortController | null = null
  let inflightTimer: ReturnType<typeof setTimeout> | undefined

  const items = computed(() => data.value?.items ?? [])
  const total = computed(() => data.value?.total ?? 0)

  function clearInflightTimer() {
    if (inflightTimer !== undefined) clearTimeout(inflightTimer)
    inflightTimer = undefined
  }

  function cancel() {
    seq++
    inflight?.abort()
    clearInflightTimer()
    inflight = null
    loading.value = false
  }

  async function refresh() {
    const current = ++seq
    inflight?.abort()
    clearInflightTimer()
    const controller = new AbortController()
    inflight = controller
    loading.value = true
    error.value = ''
    const offset = (page.value - 1) * pageSize.value
    const timeoutMs = opts.timeoutMs ?? 0
    let timedOut = false
    let timer: ReturnType<typeof setTimeout> | undefined
    if (timeoutMs > 0) {
      timer = setTimeout(() => {
        timedOut = true
        controller.abort()
      }, timeoutMs)
      inflightTimer = timer
    }
    try {
      const next = await fetcher(pageSize.value, offset, controller.signal)
      if (current === seq) {
        if (timedOut) error.value = `请求超时(${timeoutMs}ms)`
        else data.value = next
      }
    } catch (e) {
      if (current === seq) {
        error.value = timedOut ? `请求超时(${timeoutMs}ms)` : errorMessage(e)
      }
    } finally {
      if (timer !== undefined) clearTimeout(timer)
      if (inflightTimer === timer) inflightTimer = undefined
      if (current === seq) loading.value = false
      if (inflight === controller) inflight = null
    }
  }

  if (getCurrentScope()) onScopeDispose(cancel)
  return { items, total, page, pageSize, loading, error, refresh, cancel }
}
