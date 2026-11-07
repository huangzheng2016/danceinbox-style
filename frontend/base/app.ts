// PROJECT FILE: the only template file a project edits. Everything else in this
// directory is copied verbatim and updated by re-copying from the skill.
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import type { IconName } from './icons'
import { listExamples } from '@/api/example'
import { useAuthStore, type Role } from '@/stores/auth'

export const app = {
  name: 'YourApp',
  // prefix for every localStorage key the shell and templates write
  storagePrefix: 'yourapp',
}

export interface NavItem {
  to: string
  label: string
  icon: IconName
  // hidden below this role; the route needs the same meta.role, hiding is not access control
  role?: Role
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
    children: [
      { to: '/examples', label: '示例', icon: 'list' },
      { to: '/audit', label: '审计日志', icon: 'history', role: 'MODERATOR' },
      { to: '/settings', label: '系统设置', icon: 'sliders', role: 'MODERATOR' },
    ],
  },
]

export interface NavBadge {
  count: number
  // what the number means, shown on hover ("3 个示例已停用")
  tip: string
}

// Counts beside nav entries, keyed by NavItem.to, for things that wait on the user
// (pending reviews, failed jobs). The shell polls this every minute while the tab is
// visible and after each navigation; a count of 0 shows nothing. Keep it to one cheap
// request per badge (a list with page_size 1 reads its total).
export async function loadNavBadges(signal: AbortSignal): Promise<Record<string, NavBadge>> {
  const disabled = await listExamples({ q: '', status: 'disabled', range: '', sort: '', page: 1, size: 1 }, signal)
  return { '/examples': { count: disabled.total, tip: `${disabled.total} 个示例已停用` } }
}

// Called once inside the shell's setup.
export function useShellUser() {
  const auth = useAuthStore()
  const router = useRouter()
  return {
    name: computed(() => auth.user?.username ?? ''),
    canSee: (item: NavItem) => !item.role || auth.hasRole(item.role),
    logout: async () => {
      await auth.logout()
      await router.replace('/login')
    },
  }
}
