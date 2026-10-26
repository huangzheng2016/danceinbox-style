import { download, http, type Envelope } from './http'

export interface Attachment {
  id: string
  example_id: string
  name: string
  size: number
  content_type: string
  sha256: string
  created_at: number
}

export async function listAttachments(exampleId: string, signal?: AbortSignal) {
  const { data } = await http.get<Envelope & { items: Attachment[] }>('/api/example/attachment/list', {
    params: { example_id: exampleId },
    signal,
  })
  return data.items ?? []
}

export async function uploadAttachment(
  exampleId: string,
  file: File,
  opts: { signal: AbortSignal; onProgress: (ratio: number) => void },
) {
  const form = new FormData()
  form.append('example_id', exampleId)
  form.append('file', file)
  const { data } = await http.post<Envelope & { item: Attachment }>('/api/example/attachment/upload', form, {
    signal: opts.signal,
    // uploads take as long as they take; the size limit bounds them, not a timeout
    timeout: 0,
    onUploadProgress: (e) => e.total && opts.onProgress(e.loaded / e.total),
  })
  return data.item
}

export async function deleteAttachment(id: string) {
  await http.post('/api/example/attachment/delete', { id })
}

export function downloadAttachment(id: string) {
  return download(`/api/example/attachment/download?id=${encodeURIComponent(id)}`)
}
