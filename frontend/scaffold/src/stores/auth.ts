import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as userApi from '@/api/user'

const RANK = { USER: 1, MODERATOR: 2, ADMIN: 3 } as const
export type Role = keyof typeof RANK

export const useAuthStore = defineStore('auth', () => {
  const user = ref<userApi.User | null>(null)
  let ready: Promise<void> | null = null

  // Called by the router guard; the first call probes the session, later calls reuse it.
  function init() {
    ready ??= userApi
      .getMe()
      .then((u) => void (user.value = u))
      .catch(() => void (user.value = null))
    return ready
  }

  async function login(identifier: string, password: string) {
    user.value = await userApi.login(identifier, password)
  }

  async function logout() {
    try {
      await userApi.logout()
    } finally {
      clear()
    }
  }

  function clear() {
    user.value = null
  }

  const hasRole = (role: Role) => !!user.value && RANK[user.value.role] >= RANK[role]

  return { user, isAuthed: computed(() => !!user.value), init, login, logout, clear, hasRole }
})
