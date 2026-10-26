<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { app } from './app'

// Sits between .split-master and .split-detail. Drag, arrow keys (Shift for bigger
// steps), double-click to reset. The width persists per storageKey.
const props = defineProps<{ storageKey: string }>()

const MIN_MASTER = 320
const MIN_DETAIL = 360
const key = `${app.storagePrefix}.split.${props.storageKey}`

const el = ref<HTMLElement | null>(null)
const pct = ref(42)
let dragStartX = 0
let dragStartW = 0
let dragging = false

const container = () => el.value?.parentElement ?? null
const master = () => el.value?.previousElementSibling as HTMLElement | null

function clamp(p: number) {
  const w = container()?.clientWidth ?? 0
  if (w <= MIN_MASTER + MIN_DETAIL) return p
  return Math.min(((w - MIN_DETAIL) / w) * 100, Math.max((MIN_MASTER / w) * 100, p))
}

function apply(p: number, save = false) {
  pct.value = clamp(p)
  container()?.style.setProperty('--split-master-width', `${pct.value}%`)
  if (save) {
    try {
      localStorage.setItem(key, pct.value.toFixed(1))
    } catch {}
  }
}

function measure() {
  const w = container()?.clientWidth ?? 0
  return w ? ((master()?.offsetWidth ?? 0) / w) * 100 : pct.value
}

function onDown(e: PointerEvent) {
  if (e.button !== 0) return
  dragging = true
  dragStartX = e.clientX
  dragStartW = master()?.offsetWidth ?? 0
  el.value?.setPointerCapture(e.pointerId)
}

function onMove(e: PointerEvent) {
  const w = container()?.clientWidth
  if (dragging && w) apply(((dragStartW + e.clientX - dragStartX) / w) * 100)
}

function onUp() {
  if (!dragging) return
  dragging = false
  apply(pct.value, true)
}

function onKey(e: KeyboardEvent) {
  const step = e.shiftKey ? 10 : 2
  if (e.key === 'ArrowLeft') apply(measure() - step, true)
  else if (e.key === 'ArrowRight') apply(measure() + step, true)
  else return
  e.preventDefault()
}

function reset() {
  try {
    localStorage.removeItem(key)
  } catch {}
  container()?.style.removeProperty('--split-master-width')
  pct.value = measure()
}

onMounted(() => {
  let saved = NaN
  try {
    saved = parseFloat(localStorage.getItem(key) ?? '')
  } catch {}
  if (Number.isFinite(saved)) apply(saved)
  else pct.value = measure()
})
</script>

<template>
  <div
    ref="el"
    class="split-resizer"
    role="separator"
    aria-orientation="vertical"
    aria-label="调整列表宽度"
    :aria-valuenow="Math.round(pct)"
    tabindex="0"
    title="拖动调整宽度，双击恢复默认"
    @pointerdown="onDown"
    @pointermove="onMove"
    @pointerup="onUp"
    @pointercancel="onUp"
    @dblclick="reset"
    @keydown="onKey"
  />
</template>

<style scoped>
.split-resizer {
  flex: 0 0 6px;
  margin: 0 calc((var(--space-3) + 6px) / -2);
  cursor: col-resize;
  touch-action: none;
  border-radius: 3px;
}
.split-resizer:hover,
.split-resizer:focus-visible {
  background: var(--accent-soft);
  outline: none;
}
@media (max-width: 960px) {
  .split-resizer {
    display: none;
  }
}
</style>
