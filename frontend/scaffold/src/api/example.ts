import type { Page } from '@/base/useList'
import { rangeBounds } from '@/base/time'
import { http, type Envelope } from './http'

export interface Example {
  id: string
  name: string
  status: 'enabled' | 'disabled'
  remark: string
  owner_id: string
  owner_name: string
  created_at: number
  updated_at: number
  // JSON object as text, '' for none
  config: string
}

export interface ExampleInput {
  name: string
  remark: string
  enabled: boolean
  // '0' for no owner
  owner_id: string
  config: string
}

export interface ExampleQuery {
  q: string
  status: string
  // DateRange value, see time.ts
  range: string
  sort: string
  page: number
  size: number
}

export async function listExamples(p: ExampleQuery, signal?: AbortSignal): Promise<Page<Example>> {
  const bounds = rangeBounds(p.range)
  const { data } = await http.get<Envelope & { items: Example[]; total: number }>('/api/example/list', {
    params: {
      q: p.q,
      status: p.status,
      sort: p.sort,
      updated_from: bounds?.from,
      updated_to: bounds?.to,
      page: p.page,
      page_size: p.size,
    },
    signal,
  })
  return { items: data.items ?? [], total: data.total }
}

export async function getExample(id: string, signal?: AbortSignal) {
  const { data } = await http.get<Envelope & { item: Example }>('/api/example/get', { params: { id }, signal })
  return data.item
}

export async function createExample(body: ExampleInput) {
  const { data } = await http.post<Envelope & { item: Example }>('/api/example/create', body)
  return data.item
}

export async function updateExample(id: string, body: ExampleInput) {
  const { data } = await http.post<Envelope & { item: Example }>('/api/example/update', { id, ...body })
  return data.item
}

export async function setExampleEnabled(id: string, enabled: boolean) {
  await http.post('/api/example/set_enabled', { id, enabled })
}

export async function deleteExample(id: string) {
  await http.post('/api/example/delete', { id })
}

export type BatchAction = 'enable' | 'disable' | 'delete'

export async function batchExamples(ids: string[], action: BatchAction) {
  const { data } = await http.post<Envelope & { done: number; skipped: number }>('/api/example/batch', { ids, action })
  return data
}

export interface ImportRow {
  name: string
  remark: string
  enabled: boolean
  config: string
}

export async function importExamples(rows: ImportRow[]) {
  const { data } = await http.post<Envelope & { created: number; failed: { row: number; reason: string }[] }>(
    '/api/example/import',
    { rows },
  )
  return data
}

export async function exampleStats(days: number, signal?: AbortSignal) {
  const { data } = await http.get<Envelope & { days: { date: string; created: number }[] }>('/api/example/stats', {
    params: { days },
    signal,
  })
  return data.days ?? []
}
