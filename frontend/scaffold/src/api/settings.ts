import { fmtLimit } from '@/base/format'
import { fmtDuration } from '@/base/time'
import type { Page } from '@/base/useList'
import { http, type Envelope } from './http'

// Each field is read by a backend service (see biz/service/settings.Settings).
export interface AppSettings {
  announcement: string
  run_timeout_seconds: number
  // new runs are refused while true
  runs_paused: boolean
  // per example; 0 means no limit
  max_attachments: number
}

export interface SettingsSnapshot {
  // 0: never saved, the settings are the defaults and updated_* are empty
  version: number
  settings: AppSettings
  // the version this one restored, 0 for an ordinary save
  restored_from: number
  updated_by_name: string
  updated_at: number
}

export async function getSettings(signal?: AbortSignal) {
  const { data } = await http.get<Envelope & { item: SettingsSnapshot }>('/papi/settings/get', { signal })
  return data.item
}

// 409 when someone saved after baseVersion; reload and compare before retrying.
export async function updateSettings(baseVersion: number, settings: AppSettings, restoredFrom = 0) {
  const { data } = await http.post<Envelope & { item: SettingsSnapshot }>('/papi/settings/update', {
    base_version: baseVersion,
    settings,
    restored_from: restoredFrom,
  })
  return data.item
}

export async function listSettingsVersions(
  p: { page: number; size: number },
  signal?: AbortSignal,
): Promise<Page<SettingsSnapshot>> {
  const { data } = await http.get<Envelope & { items: SettingsSnapshot[]; total: number }>('/papi/settings/versions', {
    params: { page: p.page, page_size: p.size },
    signal,
  })
  return { items: data.items ?? [], total: data.total }
}

// The version and the one before it (the defaults for version 1), for the diff.
export async function getSettingsVersion(version: number, signal?: AbortSignal) {
  const { data } = await http.get<Envelope & { item: SettingsSnapshot; previous: AppSettings }>(
    '/papi/settings/version',
    { params: { version }, signal },
  )
  return { item: data.item, previous: data.previous }
}

export async function getPublicSettings(signal?: AbortSignal) {
  const { data } = await http.get<Envelope & { announcement: string }>('/api/settings/public', { signal })
  return data
}

export function isConflict(e: unknown) {
  return (e as { response?: { status?: number } })?.response?.status === 409
}

// How a setting reads in diffs and history. DiffView's `format`.
export function formatSetting(value: unknown, field: string): string {
  if (value === null || value === undefined || value === '') return '—'
  switch (field) {
    case 'run_timeout_seconds':
      return fmtDuration(value as number)
    case 'max_attachments':
      return fmtLimit(value as number, (n) => `${n} 个`)
    case 'runs_paused':
      return value ? '已暂停' : '正常'
    default:
      return String(value)
  }
}
