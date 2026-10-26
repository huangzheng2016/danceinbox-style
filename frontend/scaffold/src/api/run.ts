import type { Page } from '@/base/useList'
import { HttpError, readEventStream, type SseEvent } from '@/base/sse'
import type { RunStatus } from '@/base/agent'
import { http, refreshSession, sessionExpired, type Envelope } from './http'

export interface Run {
  id: string
  example_id: string
  status: RunStatus
  error: string
  created_at: number
  ended_at: number
}

export async function startRun(exampleId: string) {
  const { data } = await http.post<Envelope & { item: Run }>('/api/example/run/start', { example_id: exampleId })
  return data.item
}

export async function cancelRun(runId: string) {
  await http.post('/api/example/run/cancel', { run_id: runId })
}

export async function listRuns(
  exampleId: string,
  page: number,
  size: number,
  signal?: AbortSignal,
): Promise<Page<Run>> {
  const { data } = await http.get<Envelope & { items: Run[]; total: number }>('/api/example/run/list', {
    params: { example_id: exampleId, page, page_size: size },
    signal,
  })
  return { items: data.items ?? [], total: data.total }
}

// Follows a run's events until it is done. Expired access tokens are refreshed
// and the stream resumes where it stopped; a lost session goes to login.
export function streamRun(runId: string, onEvent: (e: SseEvent) => void, signal: AbortSignal) {
  return readEventStream(`/api/example/run/stream?run_id=${encodeURIComponent(runId)}`, {
    onEvent,
    isFinal: (e) => e.event === 'done',
    signal,
    onError: async (err) => {
      if (!(err instanceof HttpError)) return true
      if (err.status === 401 && err.refreshNeeded) {
        await refreshSession()
        return true
      }
      if (err.status === 401) sessionExpired()
      return false
    },
  })
}
