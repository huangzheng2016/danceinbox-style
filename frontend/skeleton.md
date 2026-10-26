# Frontend Skeleton Reference

Concrete templates for `frontend/` initial scaffolding: Vite config, dual-entry build, axios refresh interceptor, Pinia auth store, Vue Router lazy pattern, vue-i18n setup.

Pair this with Part 2 of this skill for the visual layer (flat opaque surfaces, single cyan accent, profile-driven radii, no ambient gradient, no glass). Part 2 ships `frontend/tokens.css`, `frontend/base.css`, and the shared component/composable contracts: copy them after the scaffolding here is in place. The frontend uses NO component library: `base.css` styles native elements globally, and shared Vue components (BaseDialog, StatusPill, Pagination, ...) cover the rest.

## package.json

```json
{
  "name": "<module_name>_frontend",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "rm -rf dist && vue-tsc -b && vite build --outDir dist && VITE_APP_ENTRY=admin vite build --outDir dist/admin",
    "preview": "vite preview",
    "test": "vitest run --passWithNoTests",
    "test:watch": "vitest",
    "lint": "eslint . --max-warnings 0",
    "lint:fix": "eslint . --fix",
    "format": "prettier --write .",
    "format:check": "prettier --check .",
    "check": "pnpm test && pnpm lint && pnpm format:check && pnpm build"
  },
  "dependencies": {
    "vue": "^3.5.0",
    "vue-router": "^4.4.0",
    "pinia": "^2.2.0",
    "@vueuse/core": "^11.2.0",
    "axios": "^1.7.0",
    "vue-i18n": "^10.0.0"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^6.0.0",
    "@vue/test-utils": "^2.4.0",
    "@vue/tsconfig": "^0.7.0",
    "typescript": "~5.6.0",
    "vite": "^7.2.0",
    "vitest": "^3.2.0",
    "jsdom": "^25.0.0",
    "vue-tsc": "^2.1.0",
    "@types/node": "^22.10.0",
    "eslint": "^9.15.0",
    "eslint-plugin-vue": "^9.30.0",
    "typescript-eslint": "^8.18.0",
    "eslint-config-prettier": "^10.1.8",
    "prettier": "^3.8.3"
  },
  "packageManager": "pnpm@10.0.0"
}
```

There is NO component library in this stack. The visual layer is `frontend/tokens.css` + `frontend/base.css` from this skill (Part 2, import in that order; `base.css` styles bare elements globally). Do NOT add Element Plus, Tailwind, UnoCSS, Ant Design Vue, Naive UI, or Vuetify - no exceptions. `pnpm check` is the standard full frontend gate; `vue-tsc -b` inside `build` covers typecheck. Add `dist/`, `coverage/`, and generated contract output to `.prettierignore`.

Version policy: all ranges are caret (`^`) on purpose — installs pull the latest minor/patch within the major. What matters is MAJOR pairing, not exact numbers: `vite ^7` requires `@vitejs/plugin-vue ^6` (a ^5 plugin does NOT work with Vite 7) and `vitest ^3.2` (Vite 7 support landed in Vitest 3.2; a ^2.x vitest does NOT pair with Vite 7), and `vue-tsc`/`typescript`/`@vue/tsconfig` track each other. When genuinely needed, bump with `pnpm update <pkg>` one at a time and rebuild — never hand-write versions you have not installed.

## eslint.config.js (REQUIRED — the lint script fails without it)

The `lint` script is `eslint . --max-warnings 0`, and CI plus the smoke gate run it. ESLint 9 ONLY reads flat config (`eslint.config.js`) — `.eslintrc.*` is ignored, and `pnpm create vite` does not scaffold one either. Copy this file to `frontend/eslint.config.js` at project birth, or the first `pnpm run lint` exits with "could not find config":

```js
// eslint.config.js — flat config, ESLint 9 only.
import pluginVue from 'eslint-plugin-vue'
import eslintConfigPrettier from 'eslint-config-prettier'
import tseslint from 'typescript-eslint'

export default tseslint.config(
  {
    ignores: ['dist/**', 'node_modules/**', 'coverage/**'],
  },
  ...pluginVue.configs['flat/recommended'],
  ...tseslint.configs.recommended,
  {
    // .vue <script lang="ts"> blocks are parsed by the TS parser; the vue
    // parser stays on top for the template/SFC structure.
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
      },
    },
  },
  eslintConfigPrettier,
)
```

This is the minimal official vue + ts set, aligned with the devDependencies above: `eslint-plugin-vue` ships the `flat/recommended` preset, `typescript-eslint` (the meta-package) ships `tseslint.config()` / `configs.recommended` / `tseslint.parser`, and `eslint-config-prettier` disables stylistic rules that conflict with Prettier. Do not add `@vue/eslint-config-typescript` or airbnb-style presets on top.

## vite.config.ts

```ts
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const isAdmin = env.VITE_APP_ENTRY === 'admin'

  return {
    base: isAdmin ? '/admin/' : '/', // admin bundle is served under /admin/*; without this its asset URLs 404 in prod
    plugins: [vue()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    build: {
      target: 'es2022',
      sourcemap: false,
      cssCodeSplit: true,
      rollupOptions: {
        input: isAdmin
          ? fileURLToPath(new URL('./admin.html', import.meta.url))
          : fileURLToPath(new URL('./index.html', import.meta.url)),
      },
    },
    server: {
      port: 5173,
      proxy: {
        '/api':  { target: 'http://127.0.0.1:8080', changeOrigin: true },
        '/papi': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      },
    },
  }
})
```

`pnpm create vite` creates only `index.html`. Create `admin.html` inside `frontend/`, next to `index.html` (the `rollupOptions.input` paths above resolve relative to `vite.config.ts`, so both HTML entries live in the same directory) by copying `index.html` and changing the script tag to `/src/main.admin.ts`. Public builds use `index.html`, admin builds use `admin.html`, and Vite emits each under its input filename (`dist/index.html`, `dist/admin/admin.html`). The admin build runs with `VITE_APP_ENTRY=admin vite build --outDir dist/admin`.

## src/main.ts (public entry)

There is NO `App.vue` root component. The app root is `MainLayout.vue` — the app shell template shipped in this skill's `frontend/templates/` and copied verbatim per Part 2 (`frontend/layouts/app-shell.md`). The auth init call is NOT optional: the route guard waits on `auth.loading`, and only `init()` ever flips it. Skip this and every `requiresAuth` navigation hangs on a blank page.

```ts
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import MainLayout from './base/MainLayout.vue'
import { router } from './routes'
import { i18n } from './i18n'
import { useAuthStore } from '@/stores/auth'
import './styles/index.css'

const app = createApp(MainLayout)
const pinia = createPinia()
app.use(pinia)
app.use(router)
app.use(i18n)

useAuthStore(pinia).init() // ONCE, here, before mount. Not on route change, not in guards.
app.mount('#app')
```

`src/main.admin.ts` mirrors this file and mounts the SAME `MainLayout.vue` — this skill ships no `AdminLayout`, and forking the template to invent one is forbidden (SKILL.md: customize ONLY the `EDIT:` markers, never fork a template). The admin entry differs only in importing `./routes.admin` (its own route table + role guard) and in the navigation configuration: the `EDIT:`-marked nav arrays inside `MainLayout.vue` can branch on `import.meta.env.VITE_APP_ENTRY` to render the admin menu in the admin bundle. It keeps the same `useAuthStore(pinia).init()` call before `app.mount('#app')`. There is no `App.admin.vue` either — both entries mount the shell layout directly.

## src/api.ts (axios + refresh interceptor)

```ts
import axios, { type AxiosInstance, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios'

export const api: AxiosInstance = axios.create({
  baseURL: '/',
  withCredentials: true,
  timeout: 30_000,
})

let refreshing: Promise<void> | null = null

async function refresh(): Promise<void> {
  if (!refreshing) {
    refreshing = axios.get('/api/user/refresh', { withCredentials: true })
      .then(() => undefined)
      .finally(() => { refreshing = null })
  }
  return refreshing
}

api.interceptors.response.use(
  (resp: AxiosResponse) => resp,
  async (error) => {
    const original = error.config as InternalAxiosRequestConfig & { _retried?: boolean }
    const status = error.response?.status
    const needsRefresh = error.response?.headers?.['x-token-refresh-needed'] === 'true'

    if (status === 403 && needsRefresh && !original._retried) {
      try {
        await refresh()
        original._retried = true
        return api.request(original)
      } catch {
        // refresh failed - let the caller see the original 403.
      }
    }
    return Promise.reject(error)
  },
)

export type ApiEnvelope<T = unknown> = {
  code: number
  msg?: string
} & T

export async function getUserInfo() {
  const { data } = await api.get<ApiEnvelope<{ user?: User }>>('/api/user/info')
  return data
}

export async function userLogin(identifier: string, password: string) {
  const { data } = await api.post<ApiEnvelope<{ user?: User }>>('/api/user/login', { identifier, password })
  return data
}

export async function userLogout() {
  const { data } = await api.get<ApiEnvelope<{}>>('/api/user/logout')
  return data
}

export type User = {
  id: number
  username: string
  email: string
  role: 'USER' | 'MODERATOR' | 'ADMIN'
  avatar?: string
}
```

**Critical**: under the default HttpOnly-cookie scheme there is NO `Authorization` header. Cookies carry the JWT. The refresh interceptor fires ONLY when the backend explicitly signals `X-Token-Refresh-Needed: true` AND status is 403. Never refresh on first-paint just because `getUserInfo` returned 401.

## src/stores/auth.ts (Pinia)

```ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'
import { getUserInfo, userLogin, userLogout, type ApiEnvelope, type User } from '@/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loading = ref<boolean>(true)
  const isAuthed = computed(() => user.value !== null)

  async function init() {
    loading.value = true
    try {
      const resp = await getUserInfo()
      user.value = resp.user ?? null
    } catch {
      user.value = null
    } finally {
      loading.value = false
    }
  }

  async function login(identifier: string, password: string) {
    try {
      const resp = await userLogin(identifier, password)
      if (resp.code !== 0) throw new Error(resp.msg ?? '') // i18n placeholder: map to t('login.failed') once the error-key set exists
      user.value = resp.user ?? null
    } catch (err) {
      // Bad credentials come back as HTTP 401, and axios REJECTS on non-2xx —
      // the resp.code branch above never sees them. Read the backend msg out
      // of the error body instead, or the UI shows a raw axios error string.
      if (axios.isAxiosError(err)) {
        const data = err.response?.data as ApiEnvelope | undefined
        throw new Error(data?.msg ?? err.message) // i18n placeholder: map to t('login.failed') once the error-key set exists
      }
      throw err
    }
  }

  async function logout() {
    await userLogout()
    user.value = null
  }

  return { user, loading, isAuthed, init, login, logout }
})
```

The entry calls `authStore.init()` ONCE in `main.ts` before `app.mount()`. Do NOT call refresh on init failure — guests get a plain 401, and refreshing for them spams 401 logs.

## src/routes.ts (lazy + auth guard)

```ts
import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
//                        ^^^^^^^^^^^^^^^^^^^^^
// 使用 hash 模式 (#/)，避免刷新 404，无需后端 SPA fallback。
// 仅当用户确认整站 SEO 需求时才改用 createWebHistory，并同时在 Go 端加
// SPA fallback（非 /api、/papi 的 GET 路径一律回退到 index.html）。
import { useAuthStore } from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'home', component: () => import('@/pages/Home.vue') },
  { path: '/login', name: 'login', component: () => import('@/pages/Login.vue') },
  {
    path: '/profile',
    name: 'profile',
    component: () => import('@/pages/Profile.vue'),
    meta: { requiresAuth: true },
  },
  { path: '/:catchAll(.*)', component: () => import('@/pages/NotFound.vue') },
]

export const router = createRouter({ history: createWebHashHistory(), routes })

router.beforeEach(async (to) => {
  if (!to.meta.requiresAuth) return true
  const auth = useAuthStore()
  if (auth.loading) {
    await new Promise<void>((resolve) => {
      const stop = auth.$subscribe(() => {
        if (!auth.loading) { stop(); resolve() }
      })
    })
  }
  if (!auth.isAuthed) return { name: 'login', query: { next: to.fullPath } }
  return true
})
```

`src/routes.admin.ts` is similar but guards every route with `requiresRole: 'ADMIN' | 'MODERATOR'`. Admin guard checks `auth.user?.role`.

## src/i18n.ts

```ts
import { createI18n } from 'vue-i18n'
import zh from './locales/zh.json'
import en from './locales/en.json'

function readCookie(name: string): string {
  const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`))
  return match ? decodeURIComponent(match[1]) : ''
}

const cookieLocale = readCookie('LOCALE')
const locale = cookieLocale === 'en' ? 'en' : 'zh'

export const i18n = createI18n({
  legacy: false,
  locale,
  fallbackLocale: 'en',
  messages: { zh, en },
})

// The LOCALE cookie read above is one-shot at boot. Switching language writes
// the cookie AND flips the live locale; no reload needed.
export function setLocale(l: 'zh' | 'en') {
  document.cookie = `LOCALE=${l}; path=/; max-age=31536000`
  i18n.global.locale.value = l
}
```

`src/locales/zh.json` and `src/locales/en.json` MUST have identical key shape. CI lint should fail on missing keys.

```json
{
  "common": {
    "confirm": "确认",
    "cancel": "取消",
    "save": "保存",
    "delete": "删除",
    "loading": "加载中"
  },
  "auth": {
    "login": "登录",
    "logout": "退出",
    "username": "用户名"
  }
}
```

## src/composables/

Custom composables live here. Prefer VueUse first:
- Persistent state -> `useLocalStorage`, `useSessionStorage`
- Debounce -> `useDebounceFn`
- Breakpoints -> `useBreakpoints`
- Async resource -> `useAsyncState` or write a thin wrapper around the SWR-style pattern

Custom composables (e.g. `useUnreadInbox.ts`) MUST gate on auth:
```ts
import { computed, watchEffect, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/api'

export function useUnreadInbox() {
  const auth = useAuthStore()
  const count = ref(0)

  watchEffect(async () => {
    if (auth.loading || !auth.isAuthed) return
    const { data } = await api.get('/api/inbox/unread_count')
    if (data.code === 0) count.value = data.count
  })

  return { count: computed(() => count.value) }
}
```

## Styles Setup

Choose `data-shape` and `data-surface` once per `frontend/design-profiles.md`, set them on both `index.html` and `admin.html`, and record the pair in project guidance. Copy `frontend/assets/fonts/` to `public/fonts/`, including licenses, and preload both WOFF2 files in each HTML entry. No external font CDN is needed.

`src/styles/index.css` — import order matters: tokens first, the global base stylesheet second, the editor styles only if the project has code/JSON inputs or viewers:

```css
@import "./tokens.css";       /* verbatim from this skill */
@import "./base.css";         /* verbatim from this skill: styles bare button/input/select/table globally + all shared classes */
@import "./code-editor.css";  /* verbatim; only when CodeEditor / JsonField is used */
```

Import `styles/index.css` once in each entry (`main.ts`, `main.admin.ts`). No Tailwind, no other CSS framework resets, no component library CSS. `base.css` is the component library replacement: it owns every element style and shared class (`.card`, `.pill`, `.list-toolbar`, `.split-*`, `.tabs`, `.toggle-btn`, ...), so those are NEVER redefined in a page's `<style scoped>`. Richer controls (BaseDialog, StatusPill, Pagination, CodeEditor/CodeViewer, JsonViewer, RowActionsMenu, ...) are shared Vue components + composables set up per Part 2 of this skill, not per-page inventions. Component-scoped styles reference `var(--*)` tokens from `tokens.css` only.

API methods used with `useFetch` or `usePaginatedFetch` accept an optional Axios `signal` and pass it to the request config. New refreshes abort the previous request, `cancel()` invalidates pending work, optional `timeoutMs` aborts overdue requests, and component scope disposal cancels automatically. Keep the existing `seq` guard: an aborted response must never overwrite newer data.

## Vue Test Setup (Vitest)

`vitest.config.ts` — STANDALONE, do not `mergeConfig(viteConfig, ...)`: our `vite.config.ts` is the factory form `defineConfig(({ mode }) => ...)`, and `mergeConfig` spreads a function into `{}`, silently dropping the vue plugin and the `@` alias. Restate what tests need:

```ts
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
  },
})
```

`src/test/setup.ts`:
```ts
import { config } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

config.global.plugins = []
beforeEach(() => { setActivePinia(createPinia()) })
```

Test stores, interceptors, route guards, shared async composables, and non-trivial shared interactions before leaf components. Don't snapshot every `.vue` file. Required behavior coverage and the `pnpm check` gate: `backend/testing-conventions.md`.

## Things to NOT do on the frontend

- Do NOT call `axios.get('/api/...')` directly in `<script setup>` of a route component. Wrap in `composables/` so the auth gate stays consistent.
- Do NOT store JWT in `localStorage` / `sessionStorage` under the default HttpOnly-cookie scheme. The server sets HttpOnly cookies. The frontend never reads them. (The bearer-header deployment exception — sessionStorage-only `<app>.token` — is documented in SKILL.md State Persistence.)
- Do NOT install `pinia-plugin-persistedstate` for auth. The single source of truth is the server.
- Do NOT call `auth.init()` on every route change. ONCE at startup in `main.ts`, before `app.mount()`.
- Do NOT hardcode Chinese or English strings in `.vue` templates of pages and business (project-specific) components. Use `t('...')`. This rule does NOT apply to the shared templates copied verbatim from this skill (`MainLayout.vue`, `Pagination.vue`, `BaseDialog.vue`, ...): their built-in Chinese copy is locked, by-design behavior — do not "fix" it by editing the templates.
- Do NOT add a key to `zh.json` without the same key in `en.json`.
- Do NOT install Tailwind, UnoCSS, Ant Design Vue, Naive UI, Vuetify, or Element Plus. The locked UI stack is this skill's design system without a component library (`tokens.css` + `base.css` + shared components). A bare native `<select>` / `<input>` / `<button>` is already fully styled by `base.css`; restyling one in a page's `<style scoped>` is the bug.
- Do NOT use `<style>` (non-scoped) in components. `<style scoped>` only, referencing `var(--*)` tokens — no hardcoded colors.

## Build Output Layout

After `pnpm run build` (the script above chains the public build, then the admin build into `dist/admin`):
```
frontend/dist/
├── index.html
├── assets/
│   ├── index-<hash>.js
│   ├── index-<hash>.css
│   └── ...
└── admin/
    ├── admin.html        # Vite keeps the input filename - NOT index.html
    ├── assets/
    │   └── ...
```

The Go embed reads `frontend/dist/` recursively. The admin SPA fallback in `router.go` serves `dist/admin/admin.html` for `/admin/*` — do not write `index.html` here, the emitted file keeps the input name `admin.html`.
