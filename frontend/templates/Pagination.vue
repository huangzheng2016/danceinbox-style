<!-- copy verbatim; edit only the marked regions -->
<script setup lang="ts">
import { computed, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    total: number
    page: number
    pageSize: number
    // EDIT: the backend clamps page_size at MaxLimit = 50, so options above 50
    // silently drop rows; extend this list only after raising MaxLimit there
    sizes?: number[]
  }>(),
  {
    sizes: () => [20, 50],
  },
)

const emit = defineEmits<{
  (e: 'update:page', v: number): void
  (e: 'update:pageSize', v: number): void
}>()

watch(
  () => props.pageSize,
  (size) => {
    if (props.sizes.includes(size)) return
    const nextSize = props.sizes.find((allowed) => allowed >= size) ?? props.sizes[props.sizes.length - 1]
    emit('update:pageSize', nextSize)
    if (props.page !== 1) emit('update:page', 1)
  },
  { immediate: true },
)

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
const canPrev = computed(() => props.page > 1)
const canNext = computed(() => props.page < totalPages.value)

const fromIdx = computed(() => props.total === 0 ? 0 : (props.page - 1) * props.pageSize + 1)
const toIdx = computed(() => Math.min(props.page * props.pageSize, props.total))

function go(p: number) {
  const clamped = Math.max(1, Math.min(totalPages.value, p))
  if (clamped !== props.page) emit('update:page', clamped)
}

function changeSize(e: Event) {
  const v = parseInt((e.target as HTMLSelectElement).value, 10)
  if (Number.isFinite(v) && v > 0) {
    emit('update:pageSize', v)
    if (props.page !== 1) emit('update:page', 1)
  }
}

function onJump(e: Event) {
  const input = e.target as HTMLInputElement
  const v = parseInt(input.value, 10)
  if (Number.isFinite(v)) {
    go(v)
  } else {
    input.value = String(props.page)
  }
}
</script>

<template>
  <div v-if="total > 0" class="pagination">
    <span class="pg-info muted-text">
      {{ fromIdx }} - {{ toIdx }} / 共 {{ total }} 条
    </span>
    <div class="pg-controls">
      <button
        class="pg-btn btn-sm muted"
        :disabled="!canPrev"
        :title="canPrev ? '上一页' : '已经是第一页'"
        @click="go(page - 1)"
      >‹ 上一页</button>
      <span class="pg-page">
        第
        <input
          type="number"
          class="pg-input"
          :min="1"
          :max="totalPages"
          :value="page"
          aria-label="跳转到页码"
          @change="onJump"
        />
        / {{ totalPages }} 页
      </span>
      <button
        class="pg-btn btn-sm muted"
        :disabled="!canNext"
        :title="canNext ? '下一页' : '已经是最后一页'"
        @click="go(page + 1)"
      >下一页 ›</button>
    </div>
    <span class="pg-size muted-text">
      每页
      <select :value="pageSize" class="pg-select" aria-label="每页条数" @change="changeSize">
        <option v-for="s in sizes" :key="s" :value="s">{{ s }}</option>
      </select>
      条
    </span>
  </div>
</template>

<style scoped>
.pagination {
  display: flex;
  align-items: center;
  gap: 6px 12px;
  padding: 8px 0;
  font-size: var(--text-sm);
}
.pg-info {
  flex: 1 1 0;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  font-family: var(--font-display);
  white-space: nowrap;
}
.pg-controls {
  flex: 0 1 auto;
  min-width: 0;
  overflow-x: auto;
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: nowrap;
}
.pg-btn {
  white-space: nowrap;
}
.pg-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.pg-page {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}
.pg-input {
  width: 56px;
  padding: 2px 6px;
  min-height: 26px;
  text-align: center;
  font-family: var(--font-mono);
  font-size: var(--text-sm);
}
.pg-size {
  flex: 1 1 0;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  white-space: nowrap;
}
.pg-select {
  width: 64px;
  padding: 2px 6px;
  min-height: 26px;
  font-size: var(--text-sm);
}
</style>
