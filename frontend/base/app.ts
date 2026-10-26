// PROJECT FILE: the only template file a project edits. Everything else in this
// directory is copied verbatim and updated by re-copying from the skill.
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import type { IconName } from './icons'
import { useAuthStore } from '@/stores/auth'

export const app = {
  name: 'YourApp',
  // prefix for every localStorage key the shell and templates write
  storagePrefix: 'yourapp',
}

export interface NavItem {
  to: string
  label: string
  icon: IconName
}

export interface NavGroup {
  key: string
  label: string
  icon: IconName
  children: NavItem[]
}

export type NavEntry = NavItem | NavGroup

// Page titles come from here (or route meta.title). Keep labels short; they are also the page title.
export const nav: NavEntry[] = [
  { to: '/', label: '概览', icon: 'home' },
  {
    key: 'system',
    label: '系统',
    icon: 'settings',
    children: [{ to: '/examples', label: '示例', icon: 'list' }],
  },
]

// Called once inside the shell's setup.
export function useShellUser() {
  const auth = useAuthStore()
  const router = useRouter()
  return {
    name: computed(() => auth.user?.username ?? ''),
    logout: async () => {
      await auth.logout()
      await router.replace('/login')
    },
  }
}
