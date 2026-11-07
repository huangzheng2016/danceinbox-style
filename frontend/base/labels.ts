// The one registry for state labels. A new status is added here, never as a map in a page.
export type Tone = 'neutral' | 'accent' | 'success' | 'warning' | 'danger' | 'info'

export interface Label {
  text: string
  tone: Tone
}

export type LabelMap = Record<string, Label>

// PROJECT: replace the example below with the project's enums.
export const EXAMPLE_STATUS = {
  enabled: { text: '已启用', tone: 'success' },
  disabled: { text: '已停用', tone: 'neutral' },
} satisfies LabelMap

export const RUN_STATUS = {
  queued: { text: '排队中', tone: 'neutral' },
  running: { text: '运行中', tone: 'accent' },
  succeeded: { text: '成功', tone: 'success' },
  failed: { text: '失败', tone: 'danger' },
  canceled: { text: '已停止', tone: 'warning' },
} satisfies LabelMap

// Audit actions and target types: model.Audit* / model.Target* in the backend.
export const AUDIT_ACTION = {
  'example.create': { text: '新建示例', tone: 'success' },
  'example.update': { text: '修改示例', tone: 'accent' },
  'example.enable': { text: '启用示例', tone: 'success' },
  'example.disable': { text: '停用示例', tone: 'warning' },
  'example.delete': { text: '删除示例', tone: 'danger' },
  'example.batch': { text: '批量操作示例', tone: 'accent' },
  'example.import': { text: '导入示例', tone: 'success' },
  'settings.update': { text: '修改系统设置', tone: 'accent' },
} satisfies LabelMap

export const AUDIT_TARGET = {
  example: { text: '示例', tone: 'neutral' },
  settings: { text: '系统设置', tone: 'neutral' },
} satisfies LabelMap

// Field names for DiffView, per target type: the keys of each service's audit snapshot.
// A field missing here shows its raw key in the diff, which is the cue to add it.
export const AUDIT_FIELDS: Record<string, Record<string, string>> = {
  example: { name: '名称', status: '状态', owner: '负责人', remark: '备注', config: '配置' },
  settings: {
    announcement: '公告',
    run_timeout_seconds: '运行超时',
    runs_paused: '暂停新的运行',
    max_attachments: '每个示例的附件上限',
  },
}
