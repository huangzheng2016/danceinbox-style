import { fmtLabel } from '@/base/format'
import { EXAMPLE_STATUS } from '@/base/labels'
import type { Page } from '@/base/useList'
import { formatSetting } from './settings'
import { rangeBounds } from '@/base/time'
import { http, type Envelope } from './http'

export interface AuditLog {
  id: string
  // '0' for changes made without a session
  actor_id: string
  actor_name: string
  action: string
  target_type: string
  // '0' when the change has no single target (batch, import)
  target_id: string
  target_name: string
  summary: string
  // JSON {"before": {...}, "after": {...}}, or '' when there is nothing to compare
  detail: string
  created_at: number
}

export interface AuditQuery {
  actor: string
  action: string
  target_type: string
  target_id: string
  range: string
  page: number
  size: number
}

export async function listAuditLogs(p: AuditQuery, signal?: AbortSignal): Promise<Page<AuditLog>> {
  const bounds = rangeBounds(p.range)
  const { data } = await http.get<Envelope & { items: AuditLog[]; total: number }>('/papi/audit/list', {
    params: {
      actor_id: p.actor || undefined,
      action: p.action,
      target_type: p.target_type,
      target_id: p.target_id || undefined,
      created_from: bounds?.from,
      created_to: bounds?.to,
      page: p.page,
      page_size: p.size,
    },
    signal,
  })
  return { items: data.items ?? [], total: data.total }
}

export function auditDiff(e: AuditLog): { before: unknown; after: unknown } | null {
  if (!e.detail) return null
  const d = JSON.parse(e.detail) as { before: unknown; after: unknown }
  return { before: d.before ?? {}, after: d.after ?? {} }
}

// DiffView's `format` for an entry's target type.
export function auditFormatter(targetType: string): (value: unknown, field: string) => string {
  if (targetType === 'settings') return formatSetting
  return (value, field) => {
    if (value === null || value === undefined || value === '') return '—'
    if (targetType === 'example' && field === 'status') return fmtLabel(EXAMPLE_STATUS, value as string)
    return typeof value === 'object' ? JSON.stringify(value) : String(value)
  }
}
