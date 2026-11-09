<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { groupBlocks, type ProcessBlock, type Transcript } from './agent'
import { fmtNumber } from './format'
import Icon from './Icon.vue'
import Loading from './Loading.vue'
import MarkdownText from './MarkdownText.vue'
import ToolCallCard from './ToolCallCard.vue'

// Renders a run transcript: thinking blocks, tool calls, and Markdown text in order.
// Follows new output while the reader is at the bottom; once they scroll up it stays
// put and offers a jump back instead of yanking them down.
const props = withDefaults(defineProps<{ transcript: Transcript; maxHeight?: string; foldProcess?: boolean }>(), {
  maxHeight: undefined,
  foldProcess: false,
})

const box = ref<HTMLElement | null>(null)
const following = ref(true)
const openMap = ref<Record<string, boolean>>({})

const displayBlocks = computed(() => groupBlocks(props.transcript.blocks, props.foldProcess))

function isThinkOpen(key: string): boolean {
  return openMap.value[key] ?? false
}

function isProcessOpen(g: ProcessBlock): boolean {
  return openMap.value[g.key] ?? g.failed > 0
}

function toggleKey(key: string, current: boolean) {
  openMap.value[key] = !current
}

function atBottom() {
  const el = box.value
  return !el || el.scrollHeight - el.scrollTop - el.clientHeight < 24
}

function onScroll() {
  following.value = atBottom()
}

function toBottom() {
  box.value?.scrollTo({ top: box.value.scrollHeight })
  following.value = true
}

watch(
  () => props.transcript,
  async () => {
    if (!following.value) return
    await nextTick()
    box.value?.scrollTo({ top: box.value.scrollHeight })
  },
)
onMounted(toBottom)

const running = () => !props.transcript.done

// A run that failed because a tool failed already shows that error on the tool card.
const runError = () => {
  const t = props.transcript
  if (!t.done || !t.error) return ''
  return t.blocks.some((b) => b.kind === 'tool' && b.error === t.error) ? '' : t.error
}
</script>

<template>
  <div class="transcript-wrap">
    <div ref="box" class="transcript" :style="{ maxHeight }" @scroll="onScroll">
      <template v-for="(b, i) in displayBlocks" :key="i">
        <MarkdownText
          v-if="b.kind === 'text'"
          :source="b.text"
          :streaming="running() && i === displayBlocks.length - 1"
        />
        <div v-else-if="b.kind === 'think'" class="think">
          <button
            type="button"
            class="think-head"
            :aria-expanded="isThinkOpen(`t-${i}`)"
            @click="toggleKey(`t-${i}`, isThinkOpen(`t-${i}`))"
          >
            <Icon name="chevron-right" :size="14" class="think-chevron" />
            <span>{{ running() && i === displayBlocks.length - 1 ? '思考中' : '思考过程' }}</span>
            <span class="text-muted text-sm tabular">{{ fmtNumber(b.text.length) }} 字</span>
          </button>
          <div v-if="isThinkOpen(`t-${i}`)" class="think-body">
            <MarkdownText :source="b.text" :streaming="running() && i === displayBlocks.length - 1" />
          </div>
        </div>
        <div v-else-if="b.kind === 'process'" class="process" :class="{ 'process-failed': b.failed > 0 }">
          <button
            type="button"
            class="process-head"
            :aria-expanded="isProcessOpen(b)"
            @click="toggleKey(b.key, isProcessOpen(b))"
          >
            <Icon name="chevron-right" :size="14" class="think-chevron" />
            <span>执行过程</span>
            <span v-if="b.thinking > 0" class="text-muted text-sm">思考 {{ fmtNumber(b.thinking) }} 段</span>
            <span v-if="b.tools > 0" class="text-muted text-sm">工具 {{ fmtNumber(b.tools) }} 次</span>
            <span v-if="b.running > 0" class="text-muted text-sm">运行中 {{ fmtNumber(b.running) }}</span>
            <span v-if="b.failed > 0" class="text-danger text-sm">失败 {{ fmtNumber(b.failed) }}</span>
          </button>
          <div v-if="isProcessOpen(b)" class="process-body">
            <template v-for="(item, j) in b.items" :key="j">
              <div v-if="item.kind === 'think'" class="think">
                <button
                  type="button"
                  class="think-head"
                  :aria-expanded="isThinkOpen(`${b.key}-${j}`)"
                  @click="toggleKey(`${b.key}-${j}`, isThinkOpen(`${b.key}-${j}`))"
                >
                  <Icon name="chevron-right" :size="14" class="think-chevron" />
                  <span>思考过程</span>
                  <span class="text-muted text-sm tabular">{{ fmtNumber(item.text.length) }} 字</span>
                </button>
                <div v-if="isThinkOpen(`${b.key}-${j}`)" class="think-body">
                  <MarkdownText :source="item.text" />
                </div>
              </div>
              <ToolCallCard v-else :tool="item" />
            </template>
          </div>
        </div>
        <ToolCallCard v-else :tool="b" />
      </template>
      <Loading v-if="running() && transcript.blocks.length === 0" inline label="等待输出" :delay="0" />
      <p v-if="runError()" class="text-danger transcript-error"><Icon name="x-circle" :size="14" />{{ runError() }}</p>
    </div>
    <button v-if="!following" type="button" class="btn-sm jump" @click="toBottom">
      <Icon name="arrow-down" :size="14" />跳到最新
    </button>
  </div>
</template>

<style scoped>
.transcript-wrap {
  position: relative;
  min-height: 0;
}
.transcript {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  overflow-y: auto;
}
.think,
.process {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-muted);
}
.process-failed {
  border-color: color-mix(in srgb, var(--danger) 35%, transparent);
}
.think-head,
.process-head {
  width: 100%;
  height: 32px;
  justify-content: flex-start;
  gap: var(--space-2);
  padding: 0 10px;
  font-size: var(--text-sm);
  font-weight: 400;
  color: var(--text-secondary);
  background: transparent;
  border: 0;
}
.think-head:hover,
.process-head:hover {
  background: var(--surface-hover);
}
.think-chevron {
  color: var(--text-muted);
  transition: transform var(--dur-fast) var(--ease);
}
.think-head[aria-expanded='true'] .think-chevron,
.process-head[aria-expanded='true'] .think-chevron {
  transform: rotate(90deg);
}
.think-body {
  max-height: 280px;
  overflow-y: auto;
  padding: var(--space-2) 10px 10px;
  color: var(--text-secondary);
  border-top: 1px solid var(--border);
}
.process-body {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-2) 10px 10px;
  border-top: 1px solid var(--border);
}
.transcript-error {
  display: flex;
  align-items: center;
  gap: 6px;
}
.jump {
  position: absolute;
  bottom: var(--space-2);
  left: 50%;
  transform: translateX(-50%);
  box-shadow: var(--shadow-float);
}
</style>
