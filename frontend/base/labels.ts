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
