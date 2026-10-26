<script setup lang="ts">
// Reference agent-run panel: start, follow the streamed transcript, stop, and
// browse earlier runs. The run list polls only while a run is active; the
// transcript streams over SSE and resumes after a dropped connection.
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import AgentTranscript from '@/base/AgentTranscript.vue'
import EmptyState from '@/base/EmptyState.vue'
import Icon from '@/base/Icon.vue'
import StatusPill from '@/base/StatusPill.vue'
import { applyEvent, emptyTranscript, isActive } from '@/base/agent'
import { RUN_STATUS } from '@/base/labels'
import { fmtMs } from '@/base/format'
import { fmtRelative, fmtTime } from '@/base/time'
import { toast } from '@/base/toast'
import { useFetch } from '@/base/useFetch'
import { usePolling } from '@/base/usePolling'
import { useRouteQuery } from '@/base/useRouteQuery'
import { cancelRun, listRuns, startRun, streamRun } from '@/api/run'

// disabledReason: why runs cannot start right now ('' when they can)
const props = defineProps<{ exampleId: string; disabledReason: string }>()

const { query, set } = useRouteQuery({ run: '' })
const runs = useFetch((signal) => listRuns(props.exampleId, 1, 10, signal), { immediate: true })
const items = computed(() => runs.data.value?.items ?? [])
const anyActive = computed(() => items.value.some((r) => isActive(r.status)))

usePolling(
  async () => {
    await runs.refresh()
    if (runs.error.value) throw new Error(runs.error.value)
  },
  { interval: 3000, enabled: anyActive },
)

const selectedId = computed(() => query.value.run || items.value[0]?.id || '')
const selected = computed(() => items.value.find((r) => r.id === selectedId.value))

// ---------- transcript ----------

const transcript = ref(emptyTranscript())
let follower: AbortController | null = null

function follow(id: string) {
  follower?.abort()
  transcript.value = emptyTranscript()
  if (!id) return
  const ctrl = new AbortController()
  follower = ctrl
  streamRun(
    id,
    (e) => (transcript.value = applyEvent(transcript.value, e.event, JSON.parse(e.data), Number(e.id))),
    ctrl.signal,
  )
    .catch((err) => toast.error('读取运行输出失败', err))
    .finally(() => {
      // the list shows the final status as soon as the stream ends
      if (follower === ctrl) void runs.refresh()
    })
}

watch(selectedId, follow, { immediate: true })
onBeforeUnmount(() => follower?.abort())

// ---------- actions ----------

const starting = ref(false)
const stopping = ref(false)
const startBlocked = computed(() => props.disabledReason || (anyActive.value ? '已有运行中的任务' : ''))

async function start() {
  starting.value = true
  try {
    const run = await startRun(props.exampleId)
    await runs.refresh()
    set({ run: run.id })
  } catch (e) {
    toast.error('启动失败', e)
  } finally {
    starting.value = false
  }
}

async function stop() {
  stopping.value = true
  try {
    await cancelRun(selectedId.value)
  } catch (e) {
    toast.error('停止失败', e)
  } finally {
    stopping.value = false
  }
}

const duration = (r: { created_at: number; ended_at: number }) =>
  r.ended_at ? fmtMs((r.ended_at - r.created_at) * 1000) : '—'
</script>

<template>
  <div class="runs">
    <div class="toolbar">
      <button
        type="button"
        class="primary"
        :disabled="starting || !!startBlocked"
        :title="startBlocked || undefined"
        @click="start"
      >
        <Icon name="play" />运行
      </button>
      <button
        v-if="selected && isActive(selected.status) && !transcript.done"
        type="button"
        :disabled="stopping"
        @click="stop"
      >
        <Icon name="stop" />停止
      </button>
    </div>

    <EmptyState
      v-if="runs.loaded.value && items.length === 0"
      icon="play"
      title="暂无运行记录"
      hint="点击「运行」开始一次运行"
    />
    <div v-else class="runs-body">
      <nav class="run-list" aria-label="运行记录">
        <button
          v-for="r in items"
          :key="r.id"
          type="button"
          class="run-item"
          :class="{ selected: r.id === selectedId }"
          :aria-current="r.id === selectedId || undefined"
          @click="set({ run: r.id })"
        >
          <span class="run-line">
            <span class="mono">#{{ r.id }}</span>
            <StatusPill :value="r.status" :map="RUN_STATUS" />
          </span>
          <span class="text-muted text-sm" :title="fmtTime(r.created_at)">
            {{ fmtRelative(r.created_at) }} · {{ duration(r) }}
          </span>
        </button>
      </nav>

      <section v-if="selected" class="card run-output">
        <div class="card-head">
          <h3>运行 #{{ selected.id }}</h3>
          <StatusPill :value="transcript.done ? transcript.status : selected.status" :map="RUN_STATUS" />
        </div>
        <AgentTranscript :transcript="transcript" max-height="60vh" />
      </section>
    </div>
  </div>
</template>

<style scoped>
.runs {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.runs-body {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  gap: var(--space-3);
  align-items: start;
}
.run-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.run-item {
  flex-direction: column;
  align-items: stretch;
  gap: 2px;
  height: auto;
  padding: 8px 10px;
  text-align: left;
  font-weight: 400;
  background: transparent;
  border: 1px solid transparent;
}
.run-item:hover {
  background: var(--surface-hover);
}
.run-item.selected {
  background: var(--accent-soft);
  border-color: color-mix(in srgb, var(--accent) 30%, transparent);
}
.run-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
@media (max-width: 760px) {
  .runs-body {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
