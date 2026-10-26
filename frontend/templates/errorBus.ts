// copy verbatim; edit only the marked regions
import { ref } from 'vue'

export type ErrorBusKind = 'err' | 'ok'

export interface ErrorBusToast {
  id: number
  text: string
  kind: ErrorBusKind
}

const OK_DURATION_MS = 4000
const ERR_DURATION_MS = 8000

const toasts = ref<ErrorBusToast[]>([])
const timers = new Map<number, ReturnType<typeof setTimeout>>()
let nextId = 1

function dismiss(id: number) {
  const timer = timers.get(id)
  if (timer !== undefined) {
    clearTimeout(timer)
    timers.delete(id)
  }
  toasts.value = toasts.value.filter((t) => t.id !== id)
}

function schedule(toast: ErrorBusToast) {
  const existing = timers.get(toast.id)
  if (existing !== undefined) clearTimeout(existing)
  timers.set(
    toast.id,
    setTimeout(() => dismiss(toast.id), toast.kind === 'ok' ? OK_DURATION_MS : ERR_DURATION_MS),
  )
}

function show(text: string, k: ErrorBusKind = 'err') {
  const existing = toasts.value.find((t) => t.text === text && t.kind === k)
  if (existing) {
    schedule(existing)
    return
  }
  const toast: ErrorBusToast = { id: nextId, text, kind: k }
  nextId += 1
  toasts.value = [...toasts.value, toast]
  schedule(toast)
}

function clear() {
  for (const timer of timers.values()) clearTimeout(timer)
  timers.clear()
  toasts.value = []
}

export const errorBus = { toasts, show, clear, dismiss }
