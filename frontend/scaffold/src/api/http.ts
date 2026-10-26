import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'

export interface Envelope {
  code: number
  msg?: string
}

export const http = axios.create({ withCredentials: true, timeout: 30_000 })

let refreshing: Promise<void> | null = null
let onSessionExpired: () => void = () => {}

export function setSessionExpiredHandler(fn: () => void) {
  onSessionExpired = fn
}

// One refresh at a time, shared by every request (and event streams) that needs it.
export function refreshSession(): Promise<void> {
  refreshing ??= axios
    .post('/api/auth/refresh', null, { withCredentials: true })
    .then(() => undefined)
    .finally(() => (refreshing = null))
  return refreshing
}

export function sessionExpired() {
  onSessionExpired()
}

// File downloads are plain navigations, which cannot refresh a session. Touch an
// API first so an expired access token is refreshed (or the user sent to login)
// instead of the browser showing a JSON error.
export async function download(url: string) {
  await http.get('/api/user/me')
  const a = document.createElement('a')
  a.href = url
  a.click()
}

// 401 + X-Token-Refresh-Needed: access expired, refresh still valid -> refresh once, replay.
// 401 without the header: no session. Requests marked skipAuthRedirect (the startup
// user-info probe) just fail; anything else means the session ended mid-use.
http.interceptors.response.use(undefined, async (error: AxiosError) => {
  const config = error.config as
    (InternalAxiosRequestConfig & { _retried?: boolean; skipAuthRedirect?: boolean }) | undefined
  if (error.response?.status !== 401 || !config) throw error

  if (error.response.headers['x-token-refresh-needed'] === 'true' && !config._retried) {
    config._retried = true
    try {
      await refreshSession()
      return http.request(config)
    } catch {
      // fall through: refresh token is gone too
    }
  }
  if (!config.skipAuthRedirect) sessionExpired()
  throw error
})

declare module 'axios' {
  interface AxiosRequestConfig {
    skipAuthRedirect?: boolean
  }
}
