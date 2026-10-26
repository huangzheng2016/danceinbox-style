<script setup lang="ts">
// Reference form dialog: one component for create and edit.
// Copy, rename, replace the fields; keep the structure.
import { computed, reactive, ref, watch } from 'vue'
import BaseDialog from '@/base/BaseDialog.vue'
import Combobox, { type ComboOption } from '@/base/Combobox.vue'
import JsonField from '@/base/JsonField.vue'
import MarkdownEditor from '@/base/MarkdownEditor.vue'
import Switch from '@/base/Switch.vue'
import { jsonObjectError } from '@/base/highlight'
import { toast } from '@/base/toast'
import { useDirtyGuard } from '@/base/useDirtyGuard'
import { useFormValidation } from '@/base/useFormValidation'
import { createExample, updateExample, type Example } from '@/api/example'
import { searchUsers } from '@/api/user'

const props = defineProps<{ open: boolean; editing: Example | null }>()
const emit = defineEmits<{ close: []; saved: [Example] }>()

const empty = () => ({ name: '', remark: '', enabled: true, owner: null as ComboOption | null, config: '' })
const form = reactive(empty())
const saving = ref(false)

const v = useFormValidation(() => ({
  name: !form.name.trim() ? '请输入名称' : form.name.length > 64 ? '名称不超过 64 个字符' : '',
  config: jsonObjectError(form.config),
}))
const guard = useDirtyGuard(() => form)

async function loadOwners(q: string, signal: AbortSignal) {
  const users = await searchUsers(q, signal)
  return users.map((u) => ({ value: u.id, label: u.username, hint: u.email }))
}

watch(
  () => props.open,
  (open) => {
    if (!open) return
    const e = props.editing
    Object.assign(
      form,
      e
        ? {
            name: e.name,
            remark: e.remark,
            enabled: e.status === 'enabled',
            owner: e.owner_id !== '0' ? { value: e.owner_id, label: e.owner_name } : null,
            config: e.config ? JSON.stringify(JSON.parse(e.config), null, 2) : '',
          }
        : empty(),
    )
    v.reset()
    guard.reset()
  },
)

const title = computed(() => (props.editing ? '编辑示例' : '新建示例'))

async function close() {
  if (await guard.confirmDiscard()) emit('close')
}

async function submit() {
  if (!v.validate() || saving.value) return
  saving.value = true
  try {
    const body = {
      name: form.name.trim(),
      remark: form.remark,
      enabled: form.enabled,
      owner_id: form.owner?.value ?? '0',
      // compact on the wire and in storage; the editor shows it indented
      config: form.config.trim() ? JSON.stringify(JSON.parse(form.config)) : '',
    }
    const saved = props.editing ? await updateExample(props.editing.id, body) : await createExample(body)
    toast.success(props.editing ? '已保存' : '已创建')
    guard.reset()
    emit('saved', saved)
    emit('close')
  } catch (e) {
    toast.error(props.editing ? '保存失败' : '创建失败', e)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <BaseDialog :open="open" :title="title" width="560px" @close="close">
    <form id="example-form" class="form" novalidate v-on="v.form" @submit.prevent="submit">
      <label class="field">
        <span class="field-label required">名称</span>
        <input v-model="form.name" name="name" maxlength="64" :aria-invalid="!!v.error('name')" />
        <span v-if="v.error('name')" class="field-error">{{ v.error('name') }}</span>
      </label>
      <div class="field">
        <span class="field-label">负责人</span>
        <Combobox v-model="form.owner" :load="loadOwners" placeholder="搜索用户名" />
      </div>
      <div class="field">
        <span class="field-label">备注</span>
        <MarkdownEditor v-model="form.remark" name="remark" />
      </div>
      <div class="field">
        <span class="field-label">配置</span>
        <JsonField v-model="form.config" name="config" :invalid="!!v.error('config')" />
        <span v-if="v.error('config')" class="field-error">{{ v.error('config') }}</span>
        <span v-else class="field-hint">JSON 对象，运行时读取</span>
      </div>
      <div class="field">
        <span class="field-label">启用</span>
        <Switch v-model="form.enabled" label="启用" />
      </div>
    </form>
    <template #footer>
      <button type="button" @click="close">取消</button>
      <button type="submit" form="example-form" class="primary" :disabled="saving">
        {{ saving ? '保存中' : '保存' }}
      </button>
    </template>
  </BaseDialog>
</template>
