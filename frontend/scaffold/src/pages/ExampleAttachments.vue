<script setup lang="ts">
// Reference attachments panel: drop or pick files, watch each upload, cancel or
// retry it, then download or delete stored files.
import ProgressBar from '@/base/ProgressBar.vue'
import FileUpload from '@/base/FileUpload.vue'
import Icon from '@/base/Icon.vue'
import TableState from '@/base/TableState.vue'
import { fmtBytes } from '@/base/format'
import { fmtTime } from '@/base/time'
import { toast } from '@/base/toast'
import { useConfirm } from '@/base/useConfirm'
import { useFetch } from '@/base/useFetch'
import { useUpload } from '@/base/useUpload'
import {
  deleteAttachment,
  downloadAttachment,
  listAttachments,
  uploadAttachment,
  type Attachment,
} from '@/api/attachment'

const props = defineProps<{ exampleId: string }>()

// Mirrors the server's rules (attachment service and config upload.max_mb); the
// server still decides, these only spare the user a failed upload.
const ACCEPT = [
  '.png',
  '.jpg',
  '.jpeg',
  '.gif',
  '.webp',
  '.pdf',
  '.txt',
  '.csv',
  '.json',
  '.md',
  '.zip',
  '.xlsx',
  '.docx',
]
const MAX_BYTES = 20 << 20

const list = useFetch((signal) => listAttachments(props.exampleId, signal), { immediate: true })
const upload = useUpload((file, opts) => uploadAttachment(props.exampleId, file, opts), {
  onDone: () => void list.refresh(),
})
const confirm = useConfirm()

async function remove(a: Attachment) {
  if (!(await confirm({ message: `删除附件「${a.name}」后不可恢复。`, confirmText: '删除', danger: true }))) return
  try {
    await deleteAttachment(a.id)
    toast.success('已删除')
    void list.refresh()
  } catch (e) {
    toast.error('删除失败', e)
  }
}

async function download(a: Attachment) {
  try {
    await downloadAttachment(a.id)
  } catch (e) {
    toast.error('下载失败', e)
  }
}

const STATUS_TEXT = { waiting: '等待中', uploading: '上传中', done: '已完成', failed: '失败', canceled: '已取消' }
</script>

<template>
  <div class="attachments">
    <FileUpload :accept="ACCEPT" :max-bytes="MAX_BYTES" @select="upload.add" />

    <ul v-if="upload.items.value.length" class="queue" aria-label="上传队列">
      <li v-for="it in upload.items.value" :key="it.id" class="queue-item">
        <span class="queue-name truncate" :title="it.file.name">{{ it.file.name }}</span>
        <ProgressBar
          v-if="it.status === 'uploading' || it.status === 'done'"
          :value="it.progress"
          :max="1"
          :tone="it.status === 'done' ? 'success' : 'accent'"
          class="queue-bar"
        />
        <span v-else class="text-sm" :class="it.status === 'failed' ? 'text-danger' : 'text-muted'">
          {{ STATUS_TEXT[it.status] }}{{ it.error ? `:${it.error}` : '' }}
        </span>
        <span class="queue-actions">
          <button
            v-if="it.status === 'uploading' || it.status === 'waiting'"
            type="button"
            class="ghost btn-sm"
            @click="upload.cancel(it.id)"
          >
            取消
          </button>
          <template v-else-if="it.status === 'failed' || it.status === 'canceled'">
            <button type="button" class="ghost btn-sm" @click="upload.retry(it.id)">重试</button>
            <button type="button" class="ghost btn-icon btn-sm" aria-label="移除" @click="upload.remove(it.id)">
              <Icon name="x" :size="14" />
            </button>
          </template>
        </span>
      </li>
    </ul>

    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>名称</th>
            <th class="num">大小</th>
            <th class="num">上传时间</th>
            <th class="col-actions">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in list.data.value ?? []" :key="a.id">
            <td>
              <span class="truncate" :title="a.name">{{ a.name }}</span>
            </td>
            <td class="num">{{ fmtBytes(a.size) }}</td>
            <td class="num">{{ fmtTime(a.created_at) }}</td>
            <td class="row-actions">
              <span>
                <button type="button" class="ghost btn-sm" @click="download(a)">
                  <Icon name="download" :size="14" />下载
                </button>
                <button type="button" class="ghost danger btn-sm" @click="remove(a)">删除</button>
              </span>
            </td>
          </tr>
          <TableState
            :colspan="4"
            :loading="list.loading.value"
            :error="list.error.value"
            :empty="(list.data.value ?? []).length === 0"
            empty-text="暂无附件"
            @retry="list.refresh"
          />
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.attachments {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.queue {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.queue-item {
  display: grid;
  grid-template-columns: minmax(0, 220px) minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-3);
  min-height: 32px;
  padding: 0 var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--surface-muted);
}
.queue-actions {
  display: inline-flex;
  gap: 2px;
}
</style>
