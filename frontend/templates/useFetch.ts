// copy verbatim; edit only the marked regions
import { getCurrentScope, onScopeDispose, ref, shallowRef, type Ref } from 'vue'
import { errorMessage } from './errors'

export interface FetchState<T> {
  data: Ref<T | null>
  error: Ref<string>
  loading: Ref<boolean>
  refresh: () => Promise<void>
  cancel: () => void
}

export interface FetchOptions {
  clearOnRefresh?: boolean
  timeoutMs?: number
}

export function useFetch<T>(
  fn: (signal: AbortSignal) => Promise<T>,
  opts: FetchOptions = {},
): FetchState<T> {
  const data = shallowRef<T | null>(null)
  const error = ref('')
  const loading = ref(false)
  let seq = 0
  let inflight: AbortController | null = null
  let inflightTimer: ReturnType<typeof setTimeout> | undefined

  const clearInflightTimer = () => {
    if (inflightTimer !== undefined) clearTimeout(inflightTimer)
    inflightTimer = undefined
  }

  const cancel = () => {
    seq++
    inflight?.abort()
    clearInflightTimer()
    inflight = null
    loading.value = false
  }

  const refresh = async () => {
    const current = ++seq
    inflight?.abort()
    clearInflightTimer()
    const controller = new AbortController()
    inflight = controller
    loading.value = true
    error.value = ''
    if (opts.clearOnRefresh) {
      data.value = null
    }
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
      const next = await fn(controller.signal)
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
  return { data, error, loading, refresh, cancel }
}
