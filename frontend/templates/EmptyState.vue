<!-- copy verbatim; edit only the marked regions -->
<script setup lang="ts">
import Icon from './Icon.vue'
import type { IconName } from './icons'

withDefaults(
  defineProps<{
    title: string
    hint?: string
    icon?: IconName
  }>(),
  { hint: '', icon: 'inbox' },
)
</script>

<template>
  <div class="empty-state-box">
    <Icon
      :name="icon"
      :size="32"
      class="empty-state-box-icon"
    />
    <div class="empty-state-box-title">{{ title }}</div>
    <div
      v-if="hint"
      class="empty-state-box-hint"
    >{{ hint }}</div>
    <div
      v-if="$slots.default"
      class="empty-state-box-extra"
    >
      <slot />
    </div>
  </div>
</template>

<style scoped>
.empty-state-box {
  display: flex;
  flex: 1 1 auto;
  min-height: 160px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 48px 24px;
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius-lg);
  text-align: center;
}

.empty-state-box-icon {
  color: var(--text-muted);
  opacity: 0.6;
}

.empty-state-box-title {
  font-size: var(--text-base);
  font-weight: 600;
  color: var(--text-secondary);
}

.empty-state-box-hint {
  font-size: var(--text-sm);
  color: var(--text-muted);
}

.empty-state-box-extra {
  margin-top: 4px;
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
