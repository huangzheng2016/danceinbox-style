<script setup lang="ts">
// Reference settings page: one document edited as a page-level form, saved as a new
// version. Copy it for any "configure the system" page.
// - Grouped cards, one sticky .save-bar with the only primary button.
// - Edited from a version; if someone saved meanwhile, the save conflicts and a
//   DiffView shows their version against this draft before anything is overwritten.
// - Durations are edited in human units (UnitInput) and stored in seconds.
// - Every field here is read by a backend service on each use, so a save applies
//   at once; values that need a restart belong in config, not here.
// - Users who may read but not save get the same page inside a disabled <fieldset>.
// - Every saved version is listed under 历史版本 and can be compared or restored.
import { computed, reactive, ref } from 'vue'
import Alert from '@/base/Alert.vue'
import BaseDialog from '@/base/BaseDialog.vue'
import DiffView from '@/base/DiffView.vue'
import EmptyState from '@/base/EmptyState.vue'
import Loading from '@/base/Loading.vue'
import Switch from '@/base/Switch.vue'
import Tabs from '@/base/Tabs.vue'
import UnitInput from '@/base/UnitInput.vue'
import { AUDIT_FIELDS } from '@/base/labels'
import { fmtRelative, fmtTime } from '@/base/time'
import { toast } from '@/base/toast'
import { vTip } from '@/base/tooltip'
import { useConfirm } from '@/base/useConfirm'
import { useDirtyGuard } from '@/base/useDirtyGuard'
import { useFetch } from '@/base/useFetch'
import { useFormValidation } from '@/base/useFormValidation'
import { useRouteQuery } from '@/base/useRouteQuery'
import {
  formatSetting,
  getSettings,
  isConflict,
  updateSettings,
  type AppSettings,
  type SettingsSnapshot,
} from '@/api/settings'
import { useAuthStore } from '@/stores/auth'
import SettingsHistory from './SettingsHistory.vue'

const auth = useAuthStore()
const confirm = useConfirm()
const canEdit = computed(() => auth.hasRole('ADMIN'))

const TABS = [
  { key: 'form', label: '设置' },
  { key: 'history', label: '历史版本' },
] as const
type TabKey = (typeof TABS)[number]['key']
const { query, set } = useRouteQuery({ tab: 'form' })
const tab = computed<TabKey>({
  get: () => (query.value.tab === 'history' ? 'history' : 'form'),
  set: (t) => set({ tab: t }),
})

// ---------- form state ----------

const snapshot = ref<SettingsSnapshot | null>(null)
const form = reactive<AppSettings>({ announcement: '', run_timeout_seconds: 0, runs_paused: false, max_attachments: 0 })
const draft = (): AppSettings => ({ ...form, announcement: form.announcement.trim() })
const guard = useDirtyGuard(() => form, { route: true })

const v = useFormValidation(() => ({
  announcement: form.announcement.trim().length > 500 ? '公告不超过 500 个字符' : '',
  run_timeout:
    form.run_timeout_seconds < 10 || form.run_timeout_seconds > 86400 ? '运行超时需在 10 秒到 24 小时之间' : '',
  max_attachments:
    !Number.isInteger(form.max_attachments) || form.max_attachments < 0 || form.max_attachments > 1000
      ? '附件上限需为 0 到 1000 之间的整数'
      : '',
}))

function apply(s: SettingsSnapshot) {
  snapshot.value = s
  Object.assign(form, s.settings)
  v.reset()
  guard.reset()
}

const load = useFetch(async (signal) => apply(await getSettings(signal)), { immediate: true })

const versionText = computed(() => {
  const s = snapshot.value
  if (!s) return ''
  if (s.version === 0) return '尚未保存过，当前为默认值'
  return `版本 ${s.version} · ${s.updated_by_name || '系统'} 保存于 ${fmtRelative(s.updated_at)}`
})

// ---------- save, conflict ----------

const saving = ref(false)
const conflict = ref<SettingsSnapshot | null>(null)

async function save(base = snapshot.value?.version ?? 0) {
  if (!v.validate() || saving.value) return
  saving.value = true
  try {
    const before = snapshot.value?.version
    const saved = await updateSettings(base, draft())
    conflict.value = null
    apply(saved)
    toast.success(saved.version === before ? '没有需要保存的修改' : `已保存为版本 ${saved.version}`)
  } catch (e) {
    if (isConflict(e)) await showConflict()
    else toast.error('保存失败', e)
  } finally {
    saving.value = false
  }
}

async function showConflict() {
  try {
    conflict.value = await getSettings()
  } catch (e) {
    toast.error('加载最新设置失败', e)
  }
}

function discardMine() {
  if (!conflict.value) return
  apply(conflict.value)
  conflict.value = null
  toast.success('已载入最新设置')
}

async function revert() {
  if (snapshot.value && (await guard.confirmDiscard())) apply(snapshot.value)
}

// ---------- restore from history ----------

async function restore(target: SettingsSnapshot) {
  if (!snapshot.value) return
  if (guard.dirty.value && !(await guard.confirmDiscard())) return
  const ok = await confirm({
    title: '恢复版本',
    message: `用版本 ${target.version} 的设置替换当前设置？当前设置会保留在历史版本中。`,
    confirmText: '恢复',
  })
  if (!ok) return
  try {
    const saved = await updateSettings(snapshot.value.version, target.settings, target.version)
    apply(saved)
    toast.success(`已恢复版本 ${target.version}，保存为版本 ${saved.version}`)
  } catch (e) {
    if (isConflict(e)) {
      toast.error('恢复失败：设置刚被其他人修改，请重新查看后再恢复')
      await load.refresh()
    } else toast.error('恢复失败', e)
  }
}

const saveBlocked = computed(() => (!guard.dirty.value ? '没有修改' : ''))
</script>

<template>
  <div class="page">
    <Loading v-if="!snapshot && load.loading.value" />
    <EmptyState
      v-else-if="!snapshot && load.error.value"
      icon="alert-triangle"
      title="加载失败"
      :hint="load.error.value"
    >
      <button type="button" @click="load.refresh">重试</button>
    </EmptyState>
    <template v-else-if="snapshot">
      <Alert v-if="!canEdit">只有管理员可以修改设置。</Alert>

      <Tabs v-model="tab" :items="TABS">
        <template #form>
          <form class="page" novalidate v-on="v.form" @submit.prevent="save()">
            <p class="text-muted text-sm">{{ versionText }}</p>
            <fieldset :disabled="!canEdit" class="page">
              <section class="card">
                <div class="card-head"><h3>公告</h3></div>
                <label class="field">
                  <span class="field-label">公告内容</span>
                  <textarea
                    v-model="form.announcement"
                    name="announcement"
                    rows="3"
                    :aria-invalid="!!v.error('announcement')"
                  />
                  <span v-if="v.error('announcement')" class="field-error">{{ v.error('announcement') }}</span>
                  <span v-else class="field-hint">显示在概览页顶部，支持 Markdown，留空则不显示</span>
                </label>
              </section>

              <section class="card">
                <div class="card-head"><h3>运行</h3></div>
                <div class="form-grid">
                  <div class="field">
                    <span class="field-label">运行超时</span>
                    <UnitInput
                      v-model="form.run_timeout_seconds"
                      kind="duration"
                      name="run_timeout"
                      label="运行超时"
                      :invalid="!!v.error('run_timeout')"
                    />
                    <span v-if="v.error('run_timeout')" class="field-error">{{ v.error('run_timeout') }}</span>
                  </div>
                  <div class="field">
                    <span class="field-label">暂停新的运行</span>
                    <Switch v-model="form.runs_paused" label="暂停新的运行" />
                    <span class="field-hint">进行中的运行不受影响</span>
                  </div>
                </div>
              </section>

              <section class="card">
                <div class="card-head"><h3>附件</h3></div>
                <div class="form-grid">
                  <label class="field">
                    <span class="field-label">每个示例的附件上限</span>
                    <input
                      v-model.number="form.max_attachments"
                      type="number"
                      min="0"
                      max="1000"
                      name="max_attachments"
                      :aria-invalid="!!v.error('max_attachments')"
                    />
                    <span v-if="v.error('max_attachments')" class="field-error">{{ v.error('max_attachments') }}</span>
                    <span v-else class="field-hint">0 表示不限</span>
                  </label>
                </div>
              </section>
            </fieldset>

            <div v-if="canEdit" class="save-bar">
              <span v-if="guard.dirty.value" class="text-sm">有未保存的修改</span>
              <span class="spacer" />
              <button type="button" :disabled="!guard.dirty.value || saving" @click="revert">撤销修改</button>
              <button v-tip="saveBlocked" type="submit" class="primary" :disabled="saving || !!saveBlocked">
                {{ saving ? '保存中' : '保存' }}
              </button>
            </div>
          </form>
        </template>
        <template #history>
          <SettingsHistory :current-version="snapshot.version" :can-restore="canEdit" @restore="restore" />
        </template>
      </Tabs>

      <BaseDialog :open="!!conflict" title="设置已被其他人修改" width="720px" @close="conflict = null">
        <template v-if="conflict">
          <p>
            {{ conflict.updated_by_name || '其他人' }} 在 {{ fmtTime(conflict.updated_at) }} 保存了版本
            {{ conflict.version }}。下表是它与你的修改的不同之处。
          </p>
          <DiffView
            :before="conflict.settings"
            :after="draft()"
            :labels="AUDIT_FIELDS.settings"
            :format="formatSetting"
            before-label="已保存"
            after-label="你的修改"
          />
        </template>
        <template #footer>
          <button type="button" @click="conflict = null">继续编辑</button>
          <button type="button" @click="discardMine">放弃我的修改</button>
          <button type="button" class="primary" :disabled="saving" @click="save(conflict!.version)">
            {{ saving ? '保存中' : '用我的修改覆盖' }}
          </button>
        </template>
      </BaseDialog>
    </template>
  </div>
</template>
