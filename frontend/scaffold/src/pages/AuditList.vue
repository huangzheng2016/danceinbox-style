<script setup lang="ts">
// Reference read-only log page: list + detail split with no create or edit. Copy it
// for any history people search (audit, deliveries, job attempts).
// - Every filter is a control except the target, which only arrives from a link
//   ("在审计日志中查看" on an object) and shows as a .filter-chip.
// - Names travel in the URL next to ids (actor_name, target_name), so a reloaded or
//   shared link shows who and what without an extra lookup.
// - The detail is the selected row itself; changes render as a DiffView with
//   Chinese field names, never as raw JSON.
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import Combobox, { type ComboOption } from '@/base/Combobox.vue'
import DateRange from '@/base/DateRange.vue'
import DiffView from '@/base/DiffView.vue'
import Icon from '@/base/Icon.vue'
import Pagination from '@/base/Pagination.vue'
import Select from '@/base/Select.vue'
import SplitResizer from '@/base/SplitResizer.vue'
import StatusPill from '@/base/StatusPill.vue'
import TableState from '@/base/TableState.vue'
import { linkFrom } from '@/base/backLink'
import { fmtLabel, labelOptions } from '@/base/format'
import { AUDIT_ACTION, AUDIT_FIELDS, AUDIT_TARGET } from '@/base/labels'
import { fmtRelative, fmtTime } from '@/base/time'
import { vTip } from '@/base/tooltip'
import { useList } from '@/base/useList'
import { useRouteQuery } from '@/base/useRouteQuery'
import { auditDiff, auditFormatter, listAuditLogs, type AuditLog } from '@/api/audit'
import { searchUsers } from '@/api/user'

const route = useRoute()
const list = useList(listAuditLogs, {
  actor: '',
  actor_name: '',
  action: '',
  target_type: '',
  target_id: '',
  target_name: '',
  range: '',
})
const { query, set } = useRouteQuery({ id: '' })

const ACTION_OPTIONS = [{ value: '', label: '全部操作' }, ...labelOptions(AUDIT_ACTION)]
const TARGET_OPTIONS = [{ value: '', label: '全部对象' }, ...labelOptions(AUDIT_TARGET)]
const filtered = computed(() =>
  Object.entries(list.filters).some(([k, v]) => k !== 'actor_name' && k !== 'target_name' && v !== ''),
)

const actor = computed<ComboOption | null>({
  get: () => (list.filters.actor ? { value: list.filters.actor, label: list.filters.actor_name } : null),
  set: (o) => Object.assign(list.filters, { actor: o?.value ?? '', actor_name: o?.label ?? '' }),
})
async function loadUsers(q: string, signal: AbortSignal) {
  return (await searchUsers(q, signal)).map((u) => ({ value: u.id, label: u.username }))
}

function clearTarget() {
  Object.assign(list.filters, { target_id: '', target_name: '' })
}

// ---------- selection ----------

const selected = computed(() => list.items.value.find((r) => r.id === query.value.id))
const diff = computed(() => (selected.value ? auditDiff(selected.value) : null))

// 示例「订单同步」, 系统设置 · 版本 4, or just the type for batch entries
function targetText(r: AuditLog) {
  const type = fmtLabel(AUDIT_TARGET, r.target_type)
  if (!r.target_name) return type
  return r.target_type === 'settings' ? `${type} · ${r.target_name}` : `${type}「${r.target_name}」`
}

// Only targets that still have a page get a link; settings live on one page.
function targetLink(r: AuditLog) {
  if (r.target_type === 'example' && r.target_id !== '0' && r.action !== 'example.delete')
    return linkFrom(`/examples/${r.target_id}`, route)
  if (r.target_type === 'settings') return linkFrom('/settings', route, { tab: 'history', v: r.target_id })
  return null
}
</script>

<template>
  <div class="split" :class="{ 'has-detail': !!query.id }">
    <section class="split-master">
      <div class="toolbar">
        <div class="actor-filter">
          <Combobox v-model="actor" :load="loadUsers" placeholder="操作人" />
        </div>
        <Select v-model="list.filters.action" :options="ACTION_OPTIONS" label="操作" />
        <Select v-model="list.filters.target_type" :options="TARGET_OPTIONS" label="对象类型" />
        <DateRange v-model="list.filters.range" />
        <span v-if="list.filters.target_id" class="filter-chip">
          {{ fmtLabel(AUDIT_TARGET, list.filters.target_type) }}：{{ list.filters.target_name || '指定对象' }}
          <button type="button" aria-label="清除对象筛选" @click="clearTarget"><Icon name="x" :size="14" /></button>
        </span>
        <span class="spacer" />
        <button type="button" class="btn-icon" aria-label="刷新" :disabled="list.loading.value" @click="list.refresh">
          <Icon name="refresh-cw" />
        </button>
      </div>

      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th class="num">时间</th>
              <th>操作人</th>
              <th>操作</th>
              <th>对象</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in list.items.value"
              :key="row.id"
              class="clickable"
              :class="{ selected: row.id === query.id }"
              tabindex="0"
              @click="set({ id: row.id })"
              @keydown.enter.self.prevent="set({ id: row.id })"
            >
              <td v-tip="fmtTime(row.created_at)" class="num">{{ fmtRelative(row.created_at) }}</td>
              <td>{{ row.actor_name || '系统' }}</td>
              <td><StatusPill :value="row.action" :map="AUDIT_ACTION" /></td>
              <td>
                <span class="truncate">{{ targetText(row) }}</span>
              </td>
            </tr>
            <TableState
              :colspan="4"
              :loading="list.loading.value"
              :error="list.error.value"
              :empty="list.items.value.length === 0"
              :empty-text="filtered ? '没有符合条件的记录' : '暂无记录'"
              @retry="list.refresh"
            >
              <button v-if="filtered" type="button" class="btn-sm" @click="list.reset">清除筛选</button>
            </TableState>
          </tbody>
        </table>
      </div>

      <Pagination
        :total="list.total.value"
        :page="list.page.value"
        :size="list.size.value"
        @update:page="list.setPage"
        @update:size="list.setSize"
      />
    </section>

    <SplitResizer storage-key="audit" />

    <section class="split-detail">
      <div v-if="!query.id" class="split-empty">选择左侧一条记录查看改动</div>
      <div v-else-if="!selected" class="split-empty">这条记录不在当前页</div>
      <template v-else>
        <div class="detail-head">
          <button type="button" class="ghost btn-icon split-back" aria-label="返回列表" @click="set({ id: '' })">
            <Icon name="arrow-left" />
          </button>
          <h2>{{ fmtLabel(AUDIT_ACTION, selected.action) }}</h2>
          <RouterLink v-if="targetLink(selected)" :to="targetLink(selected)!" class="btn ghost">
            <Icon name="external-link" />查看对象
          </RouterLink>
        </div>
        <dl class="kv">
          <dt>时间</dt>
          <dd class="tabular">{{ fmtTime(selected.created_at) }}</dd>
          <dt>操作人</dt>
          <dd>{{ selected.actor_name || '系统' }}</dd>
          <dt>对象</dt>
          <dd>{{ targetText(selected) }}</dd>
          <template v-if="selected.summary">
            <dt>说明</dt>
            <dd>{{ selected.summary }}</dd>
          </template>
        </dl>
        <DiffView
          v-if="diff"
          :before="diff.before"
          :after="diff.after"
          :labels="AUDIT_FIELDS[selected.target_type]"
          :format="auditFormatter(selected.target_type)"
        />
      </template>
    </section>
  </div>
</template>

<style scoped>
.actor-filter {
  width: 160px;
}
</style>
