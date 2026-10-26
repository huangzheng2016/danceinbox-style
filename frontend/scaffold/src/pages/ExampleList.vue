<script setup lang="ts">
// Reference list + detail page. New management pages start as a copy of this file
// (and ExampleFormDialog.vue): rename, swap the API calls and columns, keep the structure.
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import ActionMenu from '@/base/ActionMenu.vue'
import CopyText from '@/base/CopyText.vue'
import DateRange from '@/base/DateRange.vue'
import Icon from '@/base/Icon.vue'
import Loading from '@/base/Loading.vue'
import Pagination from '@/base/Pagination.vue'
import Segmented from '@/base/Segmented.vue'
import SortTh from '@/base/SortTh.vue'
import SplitResizer from '@/base/SplitResizer.vue'
import StatusPill from '@/base/StatusPill.vue'
import Switch from '@/base/Switch.vue'
import TableState from '@/base/TableState.vue'
import { EXAMPLE_STATUS } from '@/base/labels'
import { copyText } from '@/base/clipboard'
import { fmtRelative, fmtTime } from '@/base/time'
import { toast } from '@/base/toast'
import { useConfirm } from '@/base/useConfirm'
import { useFetch } from '@/base/useFetch'
import { useList } from '@/base/useList'
import { useRouteQuery } from '@/base/useRouteQuery'
import { useSelection } from '@/base/useSelection'
import {
  batchExamples,
  deleteExample,
  getExample,
  listExamples,
  setExampleEnabled,
  type BatchAction,
  type Example,
} from '@/api/example'
import ExampleFormDialog from './ExampleFormDialog.vue'

const list = useList(listExamples, { q: '', status: '', range: '' }, { search: 'q' })
const { query, set } = useRouteQuery({ id: '' })
const selection = useSelection(list.items)
const confirm = useConfirm()

const STATUS_OPTIONS = [
  { value: '', label: '全部' },
  ...Object.entries(EXAMPLE_STATUS).map(([value, l]) => ({ value, label: l.text })),
]
const filtered = computed(() => !!(list.filters.q || list.filters.status || list.filters.range))

// ---------- selection and detail ----------

const selectedId = computed(() => query.value.id)
const detail = useFetch((signal) => getExample(selectedId.value, signal))
watch(selectedId, (id) => (id ? void detail.refresh() : detail.cancel()), { immediate: true })

function select(row: Example) {
  set({ id: row.id })
}

// ---------- create / edit ----------

const dialogOpen = ref(false)
const editing = ref<Example | null>(null)

function openCreate() {
  editing.value = null
  dialogOpen.value = true
}

function openEdit(row: Example) {
  editing.value = row
  dialogOpen.value = true
}

function onSaved(row: Example) {
  void list.refresh()
  if (row.id === selectedId.value) void detail.refresh()
  else if (!editing.value) select(row)
}

// ---------- row actions ----------

const toggling = ref(new Set<string>())

async function toggle(row: Example, enabled: boolean) {
  toggling.value.add(row.id)
  try {
    await setExampleEnabled(row.id, enabled)
    toast.success(enabled ? '已启用' : '已停用')
    await list.refresh()
    if (row.id === selectedId.value) void detail.refresh()
  } catch (e) {
    toast.error(enabled ? '启用失败' : '停用失败', e)
  } finally {
    toggling.value.delete(row.id)
  }
}

async function remove(row: Example) {
  const ok = await confirm({ message: `删除「${row.name}」后不可恢复。`, confirmText: '删除', danger: true })
  if (!ok) return
  try {
    await deleteExample(row.id)
    toast.success('已删除')
    if (row.id === selectedId.value) set({ id: '' })
    void list.refresh()
  } catch (e) {
    toast.error('删除失败', e)
  }
}

const menuFor = (row: Example) => [
  { label: '复制 ID', icon: 'copy' as const, onSelect: () => void copyText(row.id) },
  {
    label: '删除',
    icon: 'trash-2' as const,
    danger: true,
    disabled: row.status === 'enabled',
    title: row.status === 'enabled' ? '请先停用再删除' : undefined,
    onSelect: () => void remove(row),
  },
]

// ---------- batch ----------

const BATCH_VERB: Record<BatchAction, string> = { enable: '启用', disable: '停用', delete: '删除' }
const batching = ref(false)

async function batch(action: BatchAction) {
  const n = selection.count.value
  if (
    action === 'delete' &&
    !(await confirm({ message: `删除选中的 ${n} 项？启用中的示例会被跳过。`, confirmText: '删除', danger: true }))
  )
    return
  batching.value = true
  try {
    const { done, skipped } = await batchExamples(selection.ids.value, action)
    const verb = BATCH_VERB[action]
    if (skipped) toast.success(`已${verb} ${done} 项，跳过 ${skipped} 项`)
    else toast.success(`已${verb} ${done} 项`)
    selection.clear()
    await list.refresh()
    if (selectedId.value) void detail.refresh()
  } catch (e) {
    toast.error(`批量${BATCH_VERB[action]}失败`, e)
  } finally {
    batching.value = false
  }
}
</script>

<template>
  <div class="split" :class="{ 'has-detail': !!selectedId }">
    <section class="split-master">
      <div v-if="selection.count.value" class="toolbar batch">
        <span>已选 {{ selection.count.value }} 项</span>
        <span class="spacer" />
        <button type="button" class="btn-sm" :disabled="batching" @click="batch('enable')">启用</button>
        <button type="button" class="btn-sm" :disabled="batching" @click="batch('disable')">停用</button>
        <button type="button" class="btn-sm" :disabled="batching" @click="batch('delete')">删除</button>
        <button type="button" class="ghost btn-sm" @click="selection.clear">取消选择</button>
      </div>
      <div v-else class="toolbar">
        <input v-model="list.filters.q" type="search" class="search" placeholder="搜索名称" aria-label="搜索名称" />
        <Segmented v-model="list.filters.status" :options="STATUS_OPTIONS" label="状态" />
        <DateRange v-model="list.filters.range" />
        <span class="spacer" />
        <button
          type="button"
          class="btn-icon"
          aria-label="刷新"
          title="刷新"
          :disabled="list.loading.value"
          @click="list.refresh"
        >
          <Icon name="refresh-cw" />
        </button>
        <RouterLink to="/examples/import" class="btn"><Icon name="upload" />导入</RouterLink>
        <button type="button" class="primary" @click="openCreate"><Icon name="plus" />新建</button>
      </div>

      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th class="col-check">
                <input
                  type="checkbox"
                  aria-label="全选本页"
                  :checked="selection.all.value"
                  :indeterminate="selection.some.value"
                  :disabled="list.items.value.length === 0"
                  @change="selection.toggleAll"
                />
              </th>
              <SortTh field="name" :sort="list.sort.value" @sort="list.setSort">名称</SortTh>
              <th>负责人</th>
              <th>启用</th>
              <SortTh field="updated_at" :sort="list.sort.value" num @sort="list.setSort">更新时间</SortTh>
              <th class="col-actions">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in list.items.value"
              :key="row.id"
              class="clickable"
              :class="{ selected: row.id === selectedId }"
              tabindex="0"
              @click="select(row)"
              @keydown.enter.self.prevent="select(row)"
            >
              <td class="col-check" @click.stop>
                <input
                  type="checkbox"
                  :aria-label="`选择 ${row.name}`"
                  :checked="selection.has(row.id)"
                  @change="selection.toggle(row.id)"
                />
              </td>
              <td>
                <span class="truncate">{{ row.name }}</span>
              </td>
              <td>{{ row.owner_name || '—' }}</td>
              <td>
                <Switch
                  :model-value="row.status === 'enabled'"
                  :loading="toggling.has(row.id)"
                  :label="`启用 ${row.name}`"
                  @update:model-value="toggle(row, $event)"
                />
              </td>
              <td class="num" :title="fmtTime(row.updated_at)">{{ fmtRelative(row.updated_at) }}</td>
              <td class="row-actions">
                <span>
                  <button type="button" class="ghost btn-sm" @click.stop="openEdit(row)">编辑</button>
                  <ActionMenu :items="menuFor(row)" />
                </span>
              </td>
            </tr>
            <TableState
              :colspan="6"
              :loading="list.loading.value"
              :error="list.error.value"
              :empty="list.items.value.length === 0"
              :empty-text="filtered ? '没有符合条件的示例' : '暂无示例'"
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

    <SplitResizer storage-key="examples" />

    <section class="split-detail">
      <div v-if="!selectedId" class="split-empty">选择左侧一条记录查看详情</div>
      <Loading v-else-if="!detail.data.value && detail.loading.value" />
      <div v-else-if="detail.error.value" class="split-empty text-danger">加载失败:{{ detail.error.value }}</div>
      <template v-else-if="detail.data.value">
        <div class="detail-head">
          <button type="button" class="ghost btn-icon split-back" aria-label="返回列表" @click="set({ id: '' })">
            <Icon name="arrow-left" />
          </button>
          <h2>{{ detail.data.value.name }}</h2>
          <RouterLink :to="`/examples/${detail.data.value.id}`" class="btn ghost">
            <Icon name="external-link" />完整详情
          </RouterLink>
          <button type="button" @click="openEdit(detail.data.value)"><Icon name="pencil" />编辑</button>
        </div>
        <dl class="kv">
          <dt>ID</dt>
          <dd><CopyText :value="detail.data.value.id" /></dd>
          <dt>状态</dt>
          <dd>
            <StatusPill :value="detail.data.value.status" :map="EXAMPLE_STATUS" />
          </dd>
          <dt>负责人</dt>
          <dd>{{ detail.data.value.owner_name || '—' }}</dd>
          <dt>备注</dt>
          <dd>{{ detail.data.value.remark || '—' }}</dd>
          <dt>更新时间</dt>
          <dd class="tabular">{{ fmtTime(detail.data.value.updated_at) }}</dd>
        </dl>
      </template>
    </section>

    <ExampleFormDialog :open="dialogOpen" :editing="editing" @close="dialogOpen = false" @saved="onSaved" />
  </div>
</template>
