<script setup lang="ts">
import { computed, ref } from 'vue'
import { toolSummary, type ToolBlock } from './agent'
import CodeBlock from './CodeBlock.vue'
import Icon from './Icon.vue'
import MarkdownText from './MarkdownText.vue'
import { fmtMs, fmtNumber } from './format'

// One tool call in an agent transcript. Collapsed to a single line by default;
// a failure opens itself so the error is never hidden, and if manually closed
// keeps its first error line visible.
const props = defineProps<{ tool: ToolBlock }>()

const toggled = ref<boolean | null>(null)
const open = computed(() => toggled.value ?? props.tool.status === 'failed')

const summary = computed(() => toolSummary(props.tool.args))
const errorHeadline = computed(() => {
  if (open.value || props.tool.status !== 'failed' || !props.tool.error) return ''
  return props.tool.error.trim().split(/\r?\n/, 1)[0] ?? ''
})
</script>

<template>
  <div class="tool" :class="tool.status">
    <button type="button" class="tool-head" :aria-expanded="open" @click="toggled = !open">
      <span v-if="tool.status === 'running'" class="tool-spinner" aria-label="运行中" />
      <Icon v-else :name="tool.status === 'succeeded' ? 'check-circle' : 'x-circle'" :size="14" class="tool-icon" />
      <span class="tool-name mono">{{ tool.name }}</span>
      <span class="tool-args mono">{{ summary }}</span>
      <span v-if="tool.subBlocks?.length" class="tool-sub-badge"
        >子任务 · {{ fmtNumber(tool.subBlocks.length) }} 步</span
      >
      <span v-if="tool.durationMs != null" class="tool-time tabular">{{ fmtMs(tool.durationMs) }}</span>
      <Icon name="chevron-right" :size="14" class="tool-chevron" />
    </button>
    <p v-if="errorHeadline" class="tool-err-line text-danger truncate">{{ errorHeadline }}</p>
    <div v-if="open" class="tool-body">
      <div class="field-label">参数</div>
      <CodeBlock :value="tool.args ?? {}" :max-lines="12" />
      <template v-if="tool.subBlocks?.length">
        <div class="field-label">子任务过程</div>
        <div class="tool-sub-list">
          <template v-for="(sb, idx) in tool.subBlocks" :key="idx">
            <div v-if="sb.kind === 'think'" class="tool-sub-think text-sm">
              <span class="field-label">思考</span>
              <MarkdownText :source="sb.text" />
            </div>
            <MarkdownText v-else-if="sb.kind === 'text'" :source="sb.text" />
            <ToolCallCard v-else-if="sb.kind === 'tool'" :tool="sb" />
          </template>
        </div>
      </template>
      <template v-if="tool.status === 'failed'">
        <div class="field-label">错误</div>
        <p class="text-danger">{{ tool.error }}</p>
      </template>
      <template v-else-if="tool.status === 'succeeded'">
        <div class="field-label">结果</div>
        <CodeBlock :value="tool.output ?? null" :max-lines="12" />
      </template>
    </div>
  </div>
</template>

<style scoped>
.tool {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-raised);
}
.tool.failed {
  border-color: color-mix(in srgb, var(--danger) 35%, transparent);
}
.tool-head {
  width: 100%;
  height: 34px;
  justify-content: flex-start;
  gap: var(--space-2);
  padding: 0 10px;
  font-weight: 400;
  background: transparent;
  border: 0;
}
.tool-head:hover {
  background: var(--surface-hover);
}
.tool-name {
  font-weight: 600;
}
.tool-args {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text-muted);
  text-align: left;
}
.tool-sub-badge {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}
.tool-time {
  color: var(--text-muted);
  font-size: var(--text-sm);
}
.succeeded .tool-icon {
  color: var(--success);
}
.failed .tool-icon {
  color: var(--danger);
}
.tool-chevron {
  color: var(--text-muted);
  transition: transform var(--dur-fast) var(--ease);
}
.tool-head[aria-expanded='true'] .tool-chevron {
  transform: rotate(90deg);
}
.tool-spinner {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  border: 2px solid var(--accent-soft);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
.tool-err-line {
  padding: 0 10px 6px 32px;
  font-size: var(--text-xs);
}
.tool-body {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-2) 10px 10px;
  border-top: 1px solid var(--border);
}
.tool-sub-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding-left: var(--space-2);
  border-left: 2px solid var(--border);
}
.tool-sub-think {
  color: var(--text-secondary);
}
</style>
