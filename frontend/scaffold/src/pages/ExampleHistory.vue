<script setup lang="ts">
// The 记录 tab of an example: its audit trail as a Timeline, newest first, each
// change as a DiffView. Moderators get a link to the full audit log filtered to it.
import { computed, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import DiffView from '@/base/DiffView.vue'
import Loading from '@/base/Loading.vue'
import Timeline, { type TimelineItem } from '@/base/Timeline.vue'
import { linkFrom } from '@/base/backLink'
import { fmtLabel } from '@/base/format'
import { AUDIT_ACTION, AUDIT_FIELDS } from '@/base/labels'
import { useFetch } from '@/base/useFetch'
import { auditDiff, auditFormatter, type AuditLog } from '@/api/audit'
import { exampleHistory } from '@/api/example'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{ exampleId: string; exampleName: string }>()
const route = useRoute()
const auth = useAuthStore()

const LIMIT = 50
const history = useFetch((signal) => exampleHistory(props.exampleId, { page: 1, size: LIMIT }, signal))
watch(
  () => props.exampleId,
  () => void history.refresh(),
  { immediate: true },
)

const items = computed<TimelineItem<AuditLog>[]>(() =>
  (history.data.value?.items ?? []).map((e) => ({
    key: e.id,
    time: e.created_at,
    title: fmtLabel(AUDIT_ACTION, e.action),
    actor: e.actor_name || '系统',
    tone: AUDIT_ACTION[e.action as keyof typeof AUDIT_ACTION]?.tone,
    data: e,
  })),
)
const format = auditFormatter('example')
const fullLog = computed(() =>
  linkFrom('/audit', route, { target_type: 'example', target_id: props.exampleId, target_name: props.exampleName }),
)
</script>

<template>
  <section class="card">
    <div class="card-head">
      <h3>操作记录</h3>
      <RouterLink v-if="auth.hasRole('MODERATOR')" :to="fullLog" class="btn ghost btn-sm">在审计日志中查看</RouterLink>
    </div>
    <Loading v-if="!history.data.value && history.loading.value" />
    <p v-else-if="history.error.value" class="text-danger">加载失败：{{ history.error.value }}</p>
    <template v-else>
      <Timeline :items="items" empty-text="暂无操作记录">
        <template #default="{ item }">
          <p v-if="item.data!.summary" class="text-muted text-sm">{{ item.data!.summary }}</p>
          <DiffView
            v-if="item.data!.action === 'example.update' && auditDiff(item.data!)"
            :before="auditDiff(item.data!)!.before"
            :after="auditDiff(item.data!)!.after"
            :labels="AUDIT_FIELDS.example"
            :format="format"
          />
        </template>
      </Timeline>
      <p v-if="(history.data.value?.total ?? 0) > LIMIT" class="text-muted text-sm">
        只显示最近 {{ LIMIT }} 条，共 {{ history.data.value!.total }} 条
      </p>
    </template>
  </section>
</template>
