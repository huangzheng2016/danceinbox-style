import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore, type Role } from '@/stores/auth'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    // render without the shell (login, error pages)
    bare?: boolean
    public?: boolean
    role?: Role
    // detail pages: path of the list they belong to, shown as a link before the title
    parent?: string
  }
}

const routes: RouteRecordRaw[] = [
  { path: '/login', component: () => import('@/pages/Login.vue'), meta: { bare: true, public: true, title: '登录' } },
  { path: '/', component: () => import('@/pages/Home.vue') },
  { path: '/examples', component: () => import('@/pages/ExampleList.vue') },
  {
    path: '/examples/import',
    component: () => import('@/pages/ExampleImport.vue'),
    meta: { parent: '/examples', title: '批量导入' },
  },
  {
    path: '/examples/:id',
    component: () => import('@/pages/ExampleDetail.vue'),
    meta: { parent: '/examples', title: '示例详情' },
  },
  { path: '/:rest(.*)*', component: () => import('@/pages/NotFound.vue'), meta: { title: '页面不存在' } },
]

export const router = createRouter({ history: createWebHashHistory(), routes })

// Every route requires login unless it opts out with meta.public.
router.beforeEach(async (to) => {
  const auth = useAuthStore()
  await auth.init()
  if (to.meta.public) return true
  if (!auth.isAuthed) return { path: '/login', query: { next: to.fullPath } }
  if (to.meta.role && !auth.hasRole(to.meta.role)) return { path: '/' }
  return true
})
