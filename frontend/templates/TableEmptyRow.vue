<!-- copy verbatim; edit only the marked regions -->
<script setup lang="ts">
import { computed } from 'vue'
import { errorMessage } from './errors'
import EmptyState from './EmptyState.vue'
import Icon from './Icon.vue'

const props = defineProps<{
  colspan: number
  loading?: boolean
  error?: unknown
  count: number
  title: string
}>()

const emit = defineEmits<{
  (e: 'retry'): void
}>()

const errorText = computed(() => (props.error ? errorMessage(props.error) : ''))
</script>

<template>
  <tr v-if="!loading && error">
    <td :colspan="colspan" class="empty-cell">
      <div class="table-error">
        <Icon name="x" :size="16" class="table-error-icon" />
        <span class="table-error-text">加载失败:{{ errorText }}</span>
        <button type="button" class="btn-sm muted" @click="emit('retry')">重试</button>
      </div>
    </td>
  </tr>
  <tr v-else-if="!loading && count === 0">
    <td :colspan="colspan" class="empty-cell"><EmptyState :title="title" /></td>
  </tr>
</template>

<style scoped>
.empty-cell {
  height: 100%;
}
.empty-cell :deep(.empty-state-box) {
  height: 100%;
}
.table-error {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 24px 12px;
  font-size: var(--text-sm);
  color: var(--text-secondary);
}
.table-error-icon {
  color: var(--danger);
  flex-shrink: 0;
}
.table-error-text {
  overflow-wrap: anywhere;
}
</style>
