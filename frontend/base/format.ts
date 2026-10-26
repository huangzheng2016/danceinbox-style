// Sizes and counts on screen go through these, like times go through time.ts.

export function fmtBytes(n?: number | null): string {
  if (n == null || !Number.isFinite(n)) return '—'
  if (n < 1024) return `${n} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let v = n / 1024
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v < 10 ? v.toFixed(1) : Math.round(v)} ${units[i]}`
}

// 850 ms / 1.2 s / 3 分 4 秒
export function fmtMs(ms?: number | null): string {
  if (ms == null || !Number.isFinite(ms)) return '—'
  if (ms < 1000) return `${Math.round(ms)} ms`
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)} s`
  const s = Math.round(ms / 1000)
  return `${Math.floor(s / 60)} 分 ${s % 60} 秒`
}

export function fmtNumber(n?: number | null): string {
  return n == null || !Number.isFinite(n) ? '—' : n.toLocaleString('zh-CN')
}
