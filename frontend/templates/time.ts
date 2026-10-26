// copy verbatim; edit only the marked regions
export function fmtTs(sec?: number | null): string {
  if (!sec) return '—'
  const d = new Date(sec * 1000)
  if (isNaN(d.getTime())) return '—'
  return d.toLocaleString()
}

export function fmtIso(iso?: string | null): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (isNaN(d.getTime())) return '—'
  return d.toLocaleString()
}

export function fmtRel(sec: number, now?: number): string {
  if (!sec) return ''
  const nowSec = now ?? Math.floor(Date.now() / 1000)
  const delta = sec - nowSec
  const abs = Math.abs(delta)
  const future = delta > 0
  if (abs < 10) return future ? '即将' : '刚才'
  let s: string
  if (abs < 60) s = `${abs} 秒`
  else if (abs < 3600) s = `${Math.floor(abs / 60)} 分钟`
  else if (abs < 86400) s = `${Math.floor(abs / 3600)} 小时`
  else s = `${Math.floor(abs / 86400)} 天`
  return future ? `${s}后` : `${s}前`
}

export function fmtAge(sec?: number | null): string {
  if (!sec || sec <= 0) return '—'
  const abs = Math.floor(sec)
  if (abs < 60) return `${abs} 秒前`
  if (abs < 3600) return `${Math.floor(abs / 60)} 分钟前`
  if (abs < 86400) return `${Math.floor(abs / 3600)} 小时前`
  return `${Math.floor(abs / 86400)} 天前`
}

export function fmtDur(sec: number): string {
  const abs = Math.max(0, Math.floor(sec))
  if (abs < 60) return `${abs} 秒`
  if (abs < 3600) return `${Math.floor(abs / 60)} 分钟`
  if (abs < 86400) return `${Math.floor(abs / 3600)} 小时`
  return `${Math.floor(abs / 86400)} 天`
}

export function fmtInputTs(sec: number): string {
  if (!sec) return ''
  const d = new Date(sec * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function parseInputTs(raw: string): number {
  if (!raw) return 0
  const t = new Date(raw).getTime()
  return Number.isFinite(t) ? Math.floor(t / 1000) : 0
}
