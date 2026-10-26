<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import ActionMenu from './ActionMenu.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import Icon from './Icon.vue'
import { app, nav, useShellUser, type NavEntry, type NavGroup, type NavItem } from './app'
import { isTopLayer, pushLayer, removeLayer } from './dialogStack'
import { dismissToast, toasts } from './toast'
import { pageTitle } from './usePageTitle'

const route = useRoute()
const user = useShellUser()

const COLLAPSED_KEY = `${app.storagePrefix}.sidebar.collapsed`
const EXPANDED_KEY = `${app.storagePrefix}.nav.expanded`

function load<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(key)
    return raw ? (JSON.parse(raw) as T) : fallback
  } catch {
    return fallback
  }
}
function save(key: string, value: unknown) {
  try {
    localStorage.setItem(key, JSON.stringify(value))
  } catch {}
}

const isGroup = (e: NavEntry): e is NavGroup => 'children' in e

// ---------- active item and title ----------

interface Leaf {
  item: NavItem
  group?: NavGroup
}
const leaves = computed<Leaf[]>(() =>
  nav.flatMap((e) => (isGroup(e) ? e.children.map((item) => ({ item, group: e })) : [{ item: e }])),
)

const active = computed<Leaf | undefined>(() => {
  const exact = leaves.value.find((l) => l.item.to === route.path)
  if (exact) return exact
  return leaves.value
    .filter((l) => l.item.to !== '/' && route.path.startsWith(`${l.item.to}/`))
    .sort((a, b) => b.item.to.length - a.item.to.length)[0]
})

const title = computed(() => pageTitle.value || route.meta.title || active.value?.item.label || app.name)

// Detail routes set meta.parent to their list's path; the header then reads "示例 / 订单同步".
const parent = computed(() => {
  const to = route.meta.parent
  return to ? leaves.value.find((l) => l.item.to === to)?.item : undefined
})

watch(title, (t) => (document.title = t === app.name ? t : `${t} - ${app.name}`), { immediate: true })

const isActive = (item: NavItem) => active.value?.item === item

// ---------- sidebar: desktop collapse, narrow floating panel ----------

const collapsed = ref(load(COLLAPSED_KEY, false))
const narrow = ref(false)
const navOpen = ref(false)
const rail = computed(() => collapsed.value && !narrow.value)
const sidebar = ref<HTMLElement | null>(null)
const navToggle = ref<HTMLButtonElement | null>(null)
const panelLayer = Symbol('nav-panel')

function toggleCollapsed() {
  collapsed.value = !collapsed.value
  save(COLLAPSED_KEY, collapsed.value)
}

const expanded = ref(new Set<string>(load<string[]>(EXPANDED_KEY, [])))
const isExpanded = (g: NavGroup) => expanded.value.has(g.key) || active.value?.group === g
function toggleGroup(g: NavGroup) {
  const next = new Set(expanded.value)
  if (next.has(g.key)) next.delete(g.key)
  else next.add(g.key)
  expanded.value = next
  save(EXPANDED_KEY, [...next])
}

function onPanelKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && isTopLayer(panelLayer)) {
    navOpen.value = false
    navToggle.value?.focus()
  }
}
function onPanelOutside(e: PointerEvent) {
  const t = e.target as Node
  if (!sidebar.value?.contains(t) && !navToggle.value?.contains(t)) navOpen.value = false
}

watch(navOpen, (open) => {
  if (open) {
    pushLayer(panelLayer)
    window.addEventListener('keydown', onPanelKey)
    document.addEventListener('pointerdown', onPanelOutside, true)
    void nextTick(() => sidebar.value?.querySelector<HTMLElement>('a, button')?.focus())
  } else {
    removeLayer(panelLayer)
    window.removeEventListener('keydown', onPanelKey)
    document.removeEventListener('pointerdown', onPanelOutside, true)
  }
})

const stage = ref<HTMLElement | null>(null)
watch(
  () => route.path,
  () => {
    navOpen.value = false
    stage.value?.scrollTo({ top: 0 })
  },
)

let mq: MediaQueryList | undefined
const onMq = (e: MediaQueryListEvent | MediaQueryList) => {
  narrow.value = e.matches
  if (!e.matches) navOpen.value = false
}
onMounted(() => {
  mq = window.matchMedia('(max-width: 960px)')
  onMq(mq)
  mq.addEventListener('change', onMq)
})
onBeforeUnmount(() => {
  mq?.removeEventListener('change', onMq)
  navOpen.value = false
})

// ---------- theme (initial class is set by the inline script in index.html) ----------

const dark = ref(document.documentElement.classList.contains('dark'))
function toggleTheme() {
  dark.value = !dark.value
  document.documentElement.classList.toggle('dark', dark.value)
  try {
    localStorage.setItem('theme', dark.value ? 'dark' : 'light')
  } catch {}
}

const userMenu = computed(() => [{ label: '退出登录', icon: 'log-out' as const, onSelect: () => void user.logout() }])
</script>

<template>
  <RouterView v-if="route.meta.bare" />
  <div v-else class="shell" :class="{ rail, 'nav-open': navOpen }">
    <aside ref="sidebar" class="sidebar" :aria-label="narrow ? '导航' : undefined">
      <RouterLink to="/" class="brand">
        <span class="brand-mark">{{ app.name.slice(0, 1) }}</span>
        <span v-if="!rail" class="brand-name">{{ app.name }}</span>
      </RouterLink>

      <nav class="nav">
        <template v-for="entry in nav" :key="isGroup(entry) ? entry.key : entry.to">
          <RouterLink
            v-if="!isGroup(entry)"
            :to="entry.to"
            class="nav-item"
            :class="{ active: isActive(entry) }"
            :aria-current="isActive(entry) ? 'page' : undefined"
            :title="rail ? entry.label : undefined"
          >
            <Icon :name="entry.icon" />
            <span v-if="!rail">{{ entry.label }}</span>
          </RouterLink>

          <template v-else>
            <button
              v-if="!rail"
              type="button"
              class="nav-item nav-group"
              :aria-expanded="isExpanded(entry)"
              @click="toggleGroup(entry)"
            >
              <Icon :name="entry.icon" />
              <span>{{ entry.label }}</span>
              <Icon name="chevron-right" :size="14" class="nav-chevron" />
            </button>
            <template v-if="rail || isExpanded(entry)">
              <RouterLink
                v-for="child in entry.children"
                :key="child.to"
                :to="child.to"
                class="nav-item"
                :class="{ active: isActive(child), 'nav-child': !rail }"
                :aria-current="isActive(child) ? 'page' : undefined"
                :title="rail ? child.label : undefined"
              >
                <Icon :name="child.icon" />
                <span v-if="!rail">{{ child.label }}</span>
              </RouterLink>
            </template>
          </template>
        </template>
      </nav>

      <button
        type="button"
        class="ghost nav-item collapse-toggle"
        :aria-label="collapsed ? '展开侧边栏' : '收起侧边栏'"
        :title="collapsed ? '展开侧边栏' : '收起侧边栏'"
        @click="toggleCollapsed"
      >
        <Icon name="panel-left" />
      </button>
    </aside>

    <div class="main">
      <header class="header">
        <button
          ref="navToggle"
          type="button"
          class="ghost btn-icon nav-toggle"
          aria-label="导航"
          :aria-expanded="navOpen"
          @click="navOpen = !navOpen"
        >
          <Icon name="menu" />
        </button>
        <div class="title">
          <template v-if="parent">
            <RouterLink :to="parent.to" class="crumb">{{ parent.label }}</RouterLink>
            <span class="crumb-sep" aria-hidden="true">/</span>
          </template>
          <h1>{{ title }}</h1>
        </div>
        <button
          type="button"
          class="ghost btn-icon"
          :aria-label="dark ? '切换到浅色' : '切换到深色'"
          :title="dark ? '切换到浅色' : '切换到深色'"
          @click="toggleTheme"
        >
          <Icon :name="dark ? 'sun' : 'moon'" />
        </button>
        <ActionMenu v-if="user.name.value" :items="userMenu" :label="user.name.value" icon="user" />
      </header>

      <main ref="stage" class="stage">
        <RouterView />
      </main>
    </div>
  </div>

  <div class="toasts" aria-live="polite">
    <div v-for="t in toasts" :key="t.id" class="toast" :role="t.kind === 'error' ? 'alert' : 'status'">
      <Icon :name="t.kind === 'success' ? 'check-circle' : 'x-circle'" :class="t.kind" />
      <span class="toast-text">{{ t.text }}</span>
      <button type="button" class="ghost btn-icon btn-sm" aria-label="关闭" @click="dismissToast(t.id)">
        <Icon name="x" :size="14" />
      </button>
    </div>
  </div>
  <ConfirmDialog />
</template>

<style scoped>
.shell {
  --header-h: 52px;
  display: grid;
  grid-template-columns: var(--sidebar-w) minmax(0, 1fr);
  height: 100vh;
  height: 100dvh;
  transition: grid-template-columns var(--dur) var(--ease);
}
.shell.rail {
  grid-template-columns: var(--sidebar-w-collapsed) minmax(0, 1fr);
}

.sidebar {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  min-height: 0;
  padding: var(--space-2);
  overflow: hidden;
  background: var(--surface);
  border-right: 1px solid var(--border);
}

.brand {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  height: calc(var(--header-h) - var(--space-2) * 2);
  padding: 0 6px;
  color: var(--text);
  text-decoration: none;
}
.brand:hover {
  text-decoration: none;
}
.brand-mark {
  display: grid;
  place-items: center;
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  font-weight: 700;
  color: var(--on-accent);
  background: var(--accent);
  border-radius: var(--radius-sm);
}
.brand-name {
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 10px;
  width: 100%;
  height: 34px;
  padding: 0 10px;
  font-size: var(--text-md);
  font-weight: 400;
  color: var(--text-secondary);
  background: transparent;
  border: 0;
  border-radius: var(--radius-sm);
  text-decoration: none;
  white-space: nowrap;
}
.nav-item:hover {
  color: var(--text);
  background: var(--surface-hover);
  text-decoration: none;
}
.nav-item.active {
  color: var(--accent-text);
  font-weight: 600;
  background: var(--accent-soft);
}
.shell.rail .nav-item {
  justify-content: center;
  padding: 0;
}
.nav-child {
  padding-left: 36px;
}
.nav-chevron {
  margin-left: auto;
  transition: transform var(--dur-fast) var(--ease);
}
.nav-group[aria-expanded='true'] .nav-chevron {
  transform: rotate(90deg);
}
.collapse-toggle {
  flex-shrink: 0;
}

.main {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}

.header {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  flex-shrink: 0;
  height: var(--header-h);
  padding: 0 var(--space-3) 0 var(--space-5);
  background: var(--surface);
  border-bottom: 1px solid var(--border);
}
.title {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  flex: 1;
  min-width: 0;
}
.title h1 {
  min-width: 0;
  font-size: var(--text-lg);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.crumb {
  flex-shrink: 0;
  font-size: var(--text-lg);
  color: var(--text-muted);
}
.crumb:hover {
  color: var(--text);
  text-decoration: none;
}
.crumb-sep {
  color: var(--text-muted);
}
.nav-toggle {
  display: none;
}

.stage {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  padding: var(--space-5);
  overflow-y: auto;
}
.stage:has(> .split) {
  overflow: hidden;
}

.toasts {
  position: fixed;
  right: var(--space-4);
  bottom: var(--space-4);
  z-index: var(--z-toast);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  width: min(360px, calc(100vw - 24px));
}
.toast {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  padding: 10px 8px 10px 12px;
  font-size: var(--text-md);
  background: var(--surface-raised);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-float);
}
.toast > .icon {
  margin-top: 2px;
}
.toast .success {
  color: var(--success);
}
.toast .error {
  color: var(--danger);
}
.toast-text {
  flex: 1;
  min-width: 0;
  padding-top: 1px;
  overflow-wrap: anywhere;
}
.toast button {
  margin: -3px 0;
}

@media (max-width: 960px) {
  .shell,
  .shell.rail {
    grid-template-columns: minmax(0, 1fr);
  }
  .nav-toggle {
    display: inline-flex;
  }
  .header {
    padding-left: var(--space-2);
  }
  .collapse-toggle {
    display: none;
  }
  .sidebar {
    position: fixed;
    top: calc(var(--header-h) + 4px);
    left: var(--space-2);
    z-index: var(--z-popover);
    width: min(280px, calc(100vw - 16px));
    max-height: calc(100dvh - var(--header-h) - 12px);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-float);
    visibility: hidden;
    opacity: 0;
    transform: translateY(-4px);
    transition:
      opacity var(--dur-fast) var(--ease),
      transform var(--dur-fast) var(--ease),
      visibility 0s linear var(--dur-fast);
  }
  .shell.nav-open .sidebar {
    visibility: visible;
    opacity: 1;
    transform: none;
    transition:
      opacity var(--dur-fast) var(--ease),
      transform var(--dur-fast) var(--ease);
  }
  .stage {
    padding: var(--space-3);
  }
  .toasts {
    right: 50%;
    transform: translateX(50%);
  }
}
</style>
