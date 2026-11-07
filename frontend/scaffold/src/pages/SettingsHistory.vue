<script setup lang="ts">
// Version history of the settings: every save, what it changed, and a way back.
// Selecting a version shows its diff against the version before it.
import { computed, watch } from 'vue'
import DiffView from '@/base/DiffView.vue'
import Loading from '@/base/Loading.vue'
import Pagination from '@/base/Pagination.vue'
import TableState from '@/base/TableState.vue'
import { AUDIT_FIELDS } from '@/base/labels'
import { fmtRelative, fmtTime } from '@/base/time'
import { vTip } from '@/base/tooltip'
import { useFetch } from '@/base/useFetch'
import { useList } from '@/base/useList'
import { useRouteQuery } from '@/base/useRouteQuery'
import { formatSetting, getSettingsVersion, listSettingsVersions, type SettingsSnapshot } from '@/api/settings'
import Icon from '@/base/Icon.vue'

const props = defineProps<{ currentVersion: number; canRestore: boolean }>()
const emit = defineEmits<{ restore: [SettingsSnapshot] }>()

const list = useList(listSettingsVersions, {})
const { query, set } = useRouteQuery({ v: 0 })

// a new save on the form tab shows up here without a manual refresh
watch(
  () => props.currentVersion,
  () => void list.refresh(),
)

const selected = computed(() => query.value.v)
const detail = useFetch((signal) => getSettingsVersion(selected.value, signal))
watch(selected, (v) => (v ? void detail.refresh() : detail.cancel()), { immediate: true })

const note = (s: SettingsSnapshot) => (s.restored_from ? `恢复自版本 ${s.restored_from}` : '')
const restoreBlocked = computed(() =>
  !props.canRestore ? '只有管理员可以恢复' : selected.value === props.currentVersion ? '这是当前版本' : '',
)
</script>

<template>
  <div class="page">
    <section class="card">
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>版本</th>
              <th>保存人</th>
              <th>说明</th>
              <th class="num">保存时间</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in list.items.value"
              :key="row.version"
              class="clickable"
              :class="{ selected: row.version === selected }"
              tabindex="0"
              @click="set({ v: row.version })"
              @keydown.enter.self.prevent="set({ v: row.version })"
            >
              <td class="tabular">
                版本 {{ row.version }}
                <span v-if="row.version === currentVersion" class="pill tone-accent">当前</span>
              </td>
              <td>{{ row.updated_by_name || '系统' }}</td>
              <td>{{ note(row) || '—' }}</td>
              <td v-tip="fmtTime(row.updated_at)" class="num">{{ fmtRelative(row.updated_at) }}</td>
            </tr>
            <TableState
              :colspan="4"
              :loading="list.loading.value"
              :error="list.error.value"
              :empty="list.items.value.length === 0"
              empty-text="尚未保存过设置"
              @retry="list.refresh"
            />
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

    <section v-if="selected" class="card">
      <div class="card-head">
        <h3>版本 {{ selected }} 的改动</h3>
        <button
          v-if="detail.data.value"
          v-tip="restoreBlocked"
          type="button"
          :disabled="!!restoreBlocked"
          @click="emit('restore', detail.data.value.item)"
        >
          <Icon name="rotate-ccw" />恢复此版本
        </button>
      </div>
      <Loading v-if="!detail.data.value && detail.loading.value" />
      <p v-else-if="detail.error.value" class="text-danger">加载失败：{{ detail.error.value }}</p>
      <DiffView
        v-else-if="detail.data.value"
        :before="detail.data.value.previous"
        :after="detail.data.value.item.settings"
        :labels="AUDIT_FIELDS.settings"
        :format="formatSetting"
        :before-label="selected === 1 ? '默认值' : `版本 ${selected - 1}`"
        :after-label="`版本 ${selected}`"
      />
    </section>
  </div>
</template>
