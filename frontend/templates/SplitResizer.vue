<!-- copy verbatim; edit only the marked regions -->
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

const props = defineProps<{ storageKey: string; appPrefix: string }>()

const MIN_PCT = 18
const MAX_PCT = 80
const MIN_MASTER_PX = 320
const MIN_DETAIL_PX = 360

const root = ref<HTMLDivElement | null>(null)
const currentPct = ref(40)
const dragging = ref(false)

let parent: HTMLElement | null = null
let startX = 0
let startMasterW = 0

const storeKey = computed(() => `${props.appPrefix}.split.width.${props.storageKey}`)

function masterEl(): HTMLElement | null {
  return (root.value?.previousElementSibling as HTMLElement | null) ?? null
}

function containerWidth(): number {
  return parent ? parent.getBoundingClientRect().width : 0
}

function clampPct(v: number): number {
  const w = containerWidth()
  let lo = MIN_PCT
  let hi = MAX_PCT
  if (w > 0) {
    lo = Math.max(lo, (MIN_MASTER_PX / w) * 100)
    hi = Math.min(hi, ((w - MIN_DETAIL_PX) / w) * 100)
    if (lo > hi) {
      lo = MIN_PCT
      hi = MAX_PCT
    }
  }
  return Math.min(hi, Math.max(lo, v))
}

function measurePct(): number {
  const m = masterEl()
  const w = containerWidth()
  if (!m || w <= 0) return currentPct.value
  return (m.getBoundingClientRect().width / w) * 100
}

function applyPct(p: number) {
  parent?.style.setProperty('--split-master-width', `${p}%`)
  currentPct.value = p
}

function persist(p: number) {
  localStorage.setItem(storeKey.value, String(Math.round(p * 10) / 10))
}

function onPointerDown(e: PointerEvent) {
  if (e.button !== 0 || !parent) return
  const m = masterEl()
  if (!m) return
  dragging.value = true
  startX = e.clientX
  startMasterW = m.getBoundingClientRect().width
  root.value?.setPointerCapture(e.pointerId)
  e.preventDefault()
}

function onPointerMove(e: PointerEvent) {
  if (!dragging.value) return
  const w = containerWidth()
  if (w <= 0) return
  applyPct(clampPct(((startMasterW + e.clientX - startX) / w) * 100))
}

function onPointerUp(e: PointerEvent) {
  if (!dragging.value) return
  dragging.value = false
  root.value?.releasePointerCapture(e.pointerId)
  persist(currentPct.value)
}

function onPointerCancel() {
  dragging.value = false
}

function reset() {
  localStorage.removeItem(storeKey.value)
  parent?.style.removeProperty('--split-master-width')
  requestAnimationFrame(() => {
    currentPct.value = measurePct()
  })
}

function onKeydown(e: KeyboardEvent) {
  const step = e.shiftKey ? 10 : 2
  let delta: number
  if (e.key === 'ArrowLeft') delta = -step
  else if (e.key === 'ArrowRight') delta = step
  else return
  e.preventDefault()
  const p = clampPct(measurePct() + delta)
  applyPct(p)
  persist(p)
}

onMounted(() => {
  parent = root.value?.parentElement ?? null
  const saved = parseFloat(localStorage.getItem(storeKey.value) ?? '')
  if (Number.isFinite(saved)) {
    applyPct(clampPct(saved))
  } else {
    currentPct.value = measurePct()
  }
})
</script>

<template>
  <div
    ref="root"
    class="split-resizer"
    :class="{ dragging }"
    role="separator"
    aria-label="调整列表和详情宽度"
    aria-orientation="vertical"
    :aria-valuenow="Math.round(currentPct)"
    :aria-valuemin="MIN_PCT"
    :aria-valuemax="MAX_PCT"
    tabindex="0"
    title="双击恢复默认"
    @pointerdown="onPointerDown"
    @pointermove="onPointerMove"
    @pointerup="onPointerUp"
    @pointercancel="onPointerCancel"
    @dblclick="reset"
    @keydown="onKeydown"
  ></div>
</template>

<style scoped>
.split-resizer {
  flex-shrink: 0;
  align-self: stretch;
  width: 5px;
  border-radius: var(--radius-xs);
  background: var(--border-soft);
  cursor: col-resize;
  touch-action: none;
  user-select: none;
}
.split-resizer:hover,
.split-resizer.dragging {
  background: var(--accent);
}
.split-resizer:focus-visible {
  outline: 3px solid var(--accent-ring);
  outline-offset: 1px;
}
@media (max-width: 960px) {
  .split-resizer {
    display: none;
  }
}
</style>
