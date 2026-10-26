<script setup lang="ts">
// Reference multi-step flow (useWizard + Steps): choose a CSV, check every row in
// the browser, confirm, then see what the server created and which rows failed
// and why. Route: /examples/import with meta.parent '/examples'.
import { computed, reactive, ref } from 'vue'
import { RouterLink } from 'vue-router'
import Alert from '@/base/Alert.vue'
import FileUpload from '@/base/FileUpload.vue'
import Icon from '@/base/Icon.vue'
import Steps from '@/base/Steps.vue'
import { parseCsv } from '@/base/csv'
import { jsonObjectError } from '@/base/highlight'
import { toast } from '@/base/toast'
import { useDirtyGuard } from '@/base/useDirtyGuard'
import { useWizard } from '@/base/useWizard'
import { importExamples, type ImportRow } from '@/api/example'

const MAX_ROWS = 500
const COLUMNS = {
  name: ['名称', 'name'],
  remark: ['备注', 'remark'],
  enabled: ['启用', 'enabled'],
  config: ['配置', 'config'],
}
const TEMPLATE = '\uFEFF名称,备注,启用,配置\n订单同步,每日 02:00 执行,是,"{""retry"": 3}"\n'

interface Parsed extends ImportRow {
  line: number
  errors: string[]
}

const state = reactive({
  fileName: '',
  fileError: '',
  rows: [] as Parsed[],
  result: null as null | { created: number; failed: { row: number; reason: string }[] },
})
const valid = computed(() => state.rows.filter((r) => r.errors.length === 0))
const invalid = computed(() => state.rows.filter((r) => r.errors.length > 0))

const wizard = useWizard([
  { key: 'file', label: '选择文件', validate: () => state.rows.length > 0 && !state.fileError },
  {
    key: 'check',
    label: '检查数据',
    validate: () => {
      if (valid.value.length > 0) return true
      toast.error('没有可导入的行，请修正文件后重新选择')
      return false
    },
  },
  { key: 'confirm', label: '确认导入' },
  { key: 'result', label: '完成' },
] as const)

const guard = useDirtyGuard(() => (state.result ? '' : state.fileName), { route: true })

function parseEnabled(v: string): boolean | null {
  const t = v.trim().toLowerCase()
  if (['', '是', 'true', '1', '启用', 'yes'].includes(t)) return true
  if (['否', 'false', '0', '停用', 'no'].includes(t)) return false
  return null
}

async function onFile(files: File[]) {
  const file = files[0]
  state.fileName = file.name
  state.fileError = ''
  state.rows = []
  const table = parseCsv(await file.text())
  if (table.length < 2) {
    state.fileError = '文件中没有数据行'
    return
  }
  const header = table[0].map((h) => h.trim().toLowerCase())
  const col = (names: string[]) => header.findIndex((h) => names.includes(h))
  const idx = {
    name: col(COLUMNS.name),
    remark: col(COLUMNS.remark),
    enabled: col(COLUMNS.enabled),
    config: col(COLUMNS.config),
  }
  if (idx.name < 0) {
    state.fileError = '缺少「名称」列，请使用模板'
    return
  }
  if (table.length - 1 > MAX_ROWS) {
    state.fileError = `一次最多导入 ${MAX_ROWS} 行，当前 ${table.length - 1} 行`
    return
  }
  const seen = new Map<string, number>()
  state.rows = table.slice(1).map((cells, i) => {
    const get = (k: keyof typeof idx) => (idx[k] >= 0 ? (cells[idx[k]] ?? '').trim() : '')
    const enabled = parseEnabled(get('enabled'))
    const row: Parsed = {
      line: i + 2,
      name: get('name'),
      remark: get('remark'),
      enabled: enabled ?? true,
      config: get('config'),
      errors: [],
    }
    if (!row.name) row.errors.push('名称为空')
    else if (row.name.length > 64) row.errors.push('名称超过 64 个字符')
    else if (seen.has(row.name)) row.errors.push(`与第 ${seen.get(row.name)} 行名称重复`)
    else seen.set(row.name, row.line)
    if (enabled === null) row.errors.push('启用只能填 是 或 否')
    const cfg = jsonObjectError(row.config)
    if (cfg) row.errors.push(cfg)
    return row
  })
}

function downloadTemplate() {
  const a = document.createElement('a')
  a.href = URL.createObjectURL(new Blob([TEMPLATE], { type: 'text/csv' }))
  a.download = '示例导入模板.csv'
  a.click()
  URL.revokeObjectURL(a.href)
}

const submitting = ref(false)

async function submit() {
  submitting.value = true
  try {
    state.result = await importExamples(
      valid.value.map(({ name, remark, enabled, config }) => ({ name, remark, enabled, config })),
    )
    wizard.finish()
    guard.reset()
  } catch (e) {
    toast.error('导入失败', e)
  } finally {
    submitting.value = false
  }
}

function restart() {
  Object.assign(state, { fileName: '', fileError: '', rows: [], result: null })
  wizard.reset()
  guard.reset()
}

// Server failure rows are 1-based positions in what was sent; map them back to file lines.
const failedLines = computed(() =>
  (state.result?.failed ?? []).map((f) => ({ line: valid.value[f.row - 1]?.line ?? f.row, reason: f.reason })),
)
</script>

<template>
  <div class="page">
    <section class="card">
      <Steps
        :steps="wizard.steps"
        :index="wizard.index.value"
        :reached="wizard.reached.value"
        :locked="!!state.result"
        @go="wizard.goTo"
      />
    </section>

    <section class="card">
      <template v-if="wizard.current.value.key === 'file'">
        <div class="card-head">
          <div>
            <h3>选择 CSV 文件</h3>
            <p class="card-desc">
              第一行为表头:名称(必填)、备注、启用(是/否)、配置(JSON)。一次最多 {{ MAX_ROWS }} 行。
            </p>
          </div>
          <button type="button" class="ghost btn-sm" @click="downloadTemplate">
            <Icon name="download" :size="14" />下载模板
          </button>
        </div>
        <FileUpload :accept="['.csv']" :max-bytes="2 << 20" :multiple="false" @select="onFile" />
        <Alert v-if="state.fileError" tone="danger">{{ state.fileName }}:{{ state.fileError }}</Alert>
        <Alert v-else-if="state.rows.length" tone="success">
          已读取「{{ state.fileName }}」，共 {{ state.rows.length }} 行。
        </Alert>
      </template>

      <template v-else-if="wizard.current.value.key === 'check'">
        <Alert v-if="invalid.length" tone="warning">
          {{ invalid.length }} 行有问题，导入时会跳过。可以修正文件后返回上一步重新选择。
        </Alert>
        <Alert v-else tone="success">全部 {{ state.rows.length }} 行检查通过。</Alert>
        <div class="table-wrap preview">
          <table>
            <thead>
              <tr>
                <th class="num">行</th>
                <th>名称</th>
                <th>启用</th>
                <th>配置</th>
                <th>问题</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in state.rows" :key="r.line" :class="{ 'row-bad': r.errors.length }">
                <td class="num">{{ r.line }}</td>
                <td>
                  <span class="truncate">{{ r.name || '—' }}</span>
                </td>
                <td>{{ r.enabled ? '是' : '否' }}</td>
                <td>
                  <span class="truncate mono">{{ r.config || '—' }}</span>
                </td>
                <td class="wrap text-danger">{{ r.errors.join(';') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>

      <template v-else-if="wizard.current.value.key === 'confirm'">
        <div class="card-head"><h3>确认导入</h3></div>
        <dl class="kv">
          <dt>文件</dt>
          <dd>{{ state.fileName }}</dd>
          <dt>将导入</dt>
          <dd>{{ valid.length }} 行</dd>
          <dt>将跳过</dt>
          <dd>{{ invalid.length }} 行(检查未通过)</dd>
        </dl>
        <p class="text-muted text-sm">与已有示例重名的行会在导入时失败，并在结果中列出。</p>
      </template>

      <template v-else-if="state.result">
        <Alert :tone="failedLines.length ? 'warning' : 'success'" title="导入完成">
          成功 {{ state.result.created }} 行{{ failedLines.length ? `，失败 ${failedLines.length} 行` : ''
          }}{{ invalid.length ? `，检查未通过跳过 ${invalid.length} 行` : '' }}。
        </Alert>
        <div v-if="failedLines.length" class="table-wrap">
          <table>
            <thead>
              <tr>
                <th class="num">文件行</th>
                <th>原因</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="f in failedLines" :key="f.line">
                <td class="num">{{ f.line }}</td>
                <td class="wrap">{{ f.reason }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </section>

    <div class="form-actions wizard-foot">
      <template v-if="!state.result">
        <RouterLink to="/examples" class="btn ghost">取消</RouterLink>
        <span class="spacer" />
        <button v-if="!wizard.isFirst.value" type="button" @click="wizard.back">上一步</button>
        <button
          v-if="wizard.current.value.key === 'confirm'"
          type="button"
          class="primary"
          :disabled="submitting"
          @click="submit"
        >
          {{ submitting ? '导入中' : `导入 ${valid.length} 行` }}
        </button>
        <button
          v-else
          type="button"
          class="primary"
          :disabled="wizard.busy.value || !state.rows.length"
          @click="wizard.next"
        >
          下一步
        </button>
      </template>
      <template v-else>
        <span class="spacer" />
        <button type="button" @click="restart">继续导入</button>
        <RouterLink to="/examples" class="btn primary">返回列表</RouterLink>
      </template>
    </div>
  </div>
</template>

<style scoped>
.preview {
  max-height: 420px;
}
.row-bad td {
  background: var(--danger-soft);
}
.wizard-foot .spacer {
  flex: 1;
}
</style>
