import { createApp } from 'vue'
import { createPinia } from 'pinia'
import MainLayout from '@/base/MainLayout.vue'
import { setSessionExpiredHandler } from '@/api/http'
import { router } from '@/router'
import { useAuthStore } from '@/stores/auth'
import '@/styles/tokens.css'
import '@/styles/base.css'

const app = createApp(MainLayout).use(createPinia()).use(router)

setSessionExpiredHandler(() => {
  useAuthStore().clear()
  const current = router.currentRoute.value
  if (!current.meta.public) void router.replace({ path: '/login', query: { next: current.fullPath } })
})

// Wait for the first navigation so the shell never flashes before a bare route.
void router.isReady().then(() => app.mount('#app'))
