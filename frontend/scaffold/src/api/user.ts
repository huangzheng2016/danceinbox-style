import { http, type Envelope } from './http'

export interface User {
  id: string
  username: string
  email: string
  role: 'USER' | 'MODERATOR' | 'ADMIN'
}

export async function getMe() {
  const { data } = await http.get<Envelope & { user: User }>('/api/user/me', { skipAuthRedirect: true })
  return data.user
}

export async function login(identifier: string, password: string) {
  const { data } = await http.post<Envelope & { user: User }>(
    '/api/auth/login',
    { identifier, password },
    { skipAuthRedirect: true },
  )
  return data.user
}

export async function logout() {
  await http.post('/api/auth/logout')
}

export async function searchUsers(q: string, signal?: AbortSignal) {
  const { data } = await http.get<Envelope & { items: User[] }>('/api/user/search', { params: { q }, signal })
  return data.items ?? []
}
