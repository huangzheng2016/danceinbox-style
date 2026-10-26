<script setup lang="ts">
// Reference full-page detail: for objects with more than fits in the split pane.
// Route meta.parent makes the shell show "示例 / <name>"; tabs live in the URL.
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ActionMenu from '@/base/ActionMenu.vue'
import Alert from '@/base/Alert.vue'
import CodeBlock from '@/base/CodeBlock.vue'
import CopyText from '@/base/CopyText.vue'
import EmptyState from '@/base/EmptyState.vue'
import Icon from '@/base/Icon.vue'
import Loading from '@/base/Loading.vue'
import MarkdownText from '@/base/MarkdownText.vue'
import StatusPill from '@/base/StatusPill.vue'
import Tabs from '@/base/Tabs.vue'
import { EXAMPLE_STATUS } from '@/base/labels'
import { fmtRelative, fmtTime } from '@/base/time'
import { toast } from '@/base/toast'
import { useConfirm } from '@/base/useConfirm'
import { useFetch } from '@/base/useFetch'
import { usePageTitle } from '@/base/usePageTitle'
import { useRouteQuery } from '@/base/useRouteQuery'
import { deleteExample, getExample, setExampleEnabled } from '@/api/example'
import ExampleAttachments from './ExampleAttachments.vue'
import ExampleFormDialog from './ExampleFormDialog.vue'
import ExampleRuns from './ExampleRuns.vue'

const route = useRoute()
const router = useRouter()
const confirm = useConfirm()

const id = computed(() => String(route.params.id))
const detail = useFetch((signal) => getExample(id.value, signal))
watch(id, () => void detail.refresh(), { immediate: true })
const ex = computed(() => detail.data.value)
usePageTitle(() => ex.value?.name)

const TABS = [
  { key: 'overview', label: '概览' },
  { key: 'runs', label: '运行' },
  { key: 'files', label: '附件' },
  { key: 'raw', label: '原始数据' },
] as const
type TabKey = (typeof TABS)[number]['key']
const { query, set } = useRouteQuery({ tab: 'overview' })
const tab = computed<TabKey>({
  get: () => (TABS.some((t) => t.key === query.value.tab) ? (query.value.tab as TabKey) : 'overview'),
  set: (t) => set({ tab: t }),
})

const editing = ref(false)
const busy = ref(false)

async function enable() {
  busy.value = true
  try {
    await setExampleEnabled(id.value, true)
    toast.success('已启用')
    await detail.refresh()
  } catch (e) {
    toast.error('启用失败', e)
  } finally {
    busy.value = false
  }
}

async function remove() {
  if (!ex.value) return
  if (!(await confirm({ message: `删除「${ex.value.name}」后不可恢复。`, confirmText: '删除', danger: true }))) return
  try {
    await deleteExample(id.value)
    toast.success('已删除')
    await router.replace('/examples')
  } catch (e) {
    toast.error('删除失败', e)
  }
}

const menu = computed(() => [
  {
    label: '删除',
    icon: 'trash-2' as const,
    danger: true,
    disabled: ex.value?.status === 'enabled',
    title: ex.value?.status === 'enabled' ? '请先停用再删除' : undefined,
    onSelect: () => void remove(),
  },
])
</script>

<template>
  <div class="page">
    <Loading v-if="!ex && detail.loading.value" />
    <EmptyState v-else-if="detail.error.value" icon="alert-triangle" title="加载失败" :hint="detail.error.value">
      <button type="button" @click="detail.refresh">重试</button>
    </EmptyState>
    <template v-else-if="ex">
      <Alert v-if="ex.status === 'disabled'" tone="warning">
        该示例已停用，不会参与运行。
        <template #action>
          <button type="button" class="btn-sm" :disabled="busy" @click="enable">启用</button>
        </template>
      </Alert>

      <div class="toolbar">
        <StatusPill :value="ex.status" :map="EXAMPLE_STATUS" />
        <span class="text-muted text-sm">更新于 {{ fmtRelative(ex.updated_at) }}</span>
        <span class="spacer" />
        <button type="button" @click="editing = true"><Icon name="pencil" />编辑</button>
        <ActionMenu :items="menu" />
      </div>

      <Tabs v-model="tab" :items="TABS">
        <template #overview>
          <section class="card">
            <div class="card-head">
              <h3>基本信息</h3>
            </div>
            <dl class="kv">
              <dt>ID</dt>
              <dd><CopyText :value="ex.id" /></dd>
              <dt>负责人</dt>
              <dd>{{ ex.owner_name || '—' }}</dd>
              <dt>创建时间</dt>
              <dd class="tabular">{{ fmtTime(ex.created_at) }}</dd>
              <dt>更新时间</dt>
              <dd class="tabular">{{ fmtTime(ex.updated_at) }}</dd>
            </dl>
          </section>
          <section class="card">
            <div class="card-head"><h3>备注</h3></div>
            <MarkdownText v-if="ex.remark" :source="ex.remark" />
            <p v-else class="text-muted">暂无备注</p>
          </section>
          <section v-if="ex.config" class="card">
            <div class="card-head"><h3>配置</h3></div>
            <CodeBlock :value="JSON.parse(ex.config)" />
          </section>
        </template>
        <template #runs>
          <ExampleRuns
            :example-id="ex.id"
            :disabled-reason="ex.status === 'disabled' ? '示例已停用，启用后才能运行' : ''"
          />
        </template>
        <template #files>
          <ExampleAttachments :example-id="ex.id" />
        </template>
        <template #raw>
          <CodeBlock :value="ex" />
        </template>
      </Tabs>

      <ExampleFormDialog :open="editing" :editing="ex" @close="editing = false" @saved="detail.refresh" />
    </template>
  </div>
</template>
