<!-- copy verbatim; edit only the marked regions -->
<script setup lang="ts">
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Icon from './Icon.vue'
import type { IconName } from './icons'
import { errorBus } from './errorBus'
import { pushDialog, removeDialog, isTopDialog } from './dialogStack'
import ConfirmDialog from './ConfirmDialog.vue'

const route = useRoute()

watch(
  () => route.fullPath,
  async () => {
    closeNav()
    await nextTick()
    // at <= 760px the frame's scroll lock is released (`.page-stage` is
    // `overflow: visible`) and the document itself is the scroll container
    if (window.matchMedia('(max-width: 760px)').matches) window.scrollTo({ top: 0 })
    else document.querySelector('.page-stage')?.scrollTo({ top: 0 })
  },
)

const isDark = ref(false)
const collapsed = ref(false)
// EDIT: storage key prefix, rename per project
const sidebarCollapsedKey = 'yourapp.sidebar.collapsed'
const isNarrow = ref(false)
const navOpen = ref(false)
const railCollapsed = computed(() => collapsed.value && !isNarrow.value)

const navPanelId = Symbol('ShellNavPanel')
const navPanelEl = ref<HTMLElement | null>(null)
const navToggleEl = ref<HTMLElement | null>(null)
const navPanelStyle = ref<Record<string, string>>({})

const NAV_FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

function updateNavPanelPosition() {
  const rect = navToggleEl.value?.getBoundingClientRect()
  if (!rect) return
  const vw = window.innerWidth
  const width = Math.min(300, vw - 96)
  const top = rect.bottom + 8
  navPanelStyle.value = {
    top: `${top}px`,
    left: `${Math.max(14, Math.min(rect.left, vw - width - 14))}px`,
    maxHeight: `${window.innerHeight - top - 14}px`,
  }
}

function openNav() {
  updateNavPanelPosition()
  navOpen.value = true
}

function closeNav() {
  navOpen.value = false
}

function toggleNav() {
  if (navOpen.value) closeNav()
  else openNav()
}

function onNavKeydown(e: KeyboardEvent) {
  if (!navOpen.value || !isTopDialog(navPanelId)) return
  if (e.key === 'Escape') closeNav()
}

function onNavDocMousedown(e: MouseEvent) {
  const target = e.target as Node | null
  if (target && (navPanelEl.value?.contains(target) || navToggleEl.value?.contains(target))) return
  closeNav()
}

watch(navOpen, (open) => {
  if (open) {
    pushDialog(navPanelId)
    window.addEventListener('keydown', onNavKeydown)
    document.addEventListener('mousedown', onNavDocMousedown, true)
    window.addEventListener('scroll', updateNavPanelPosition, true)
    nextTick(() => {
      const root = navPanelEl.value
      if (!root) return
      const target = root.querySelector<HTMLElement>(NAV_FOCUSABLE)
      ;(target ?? root).focus()
    })
  } else {
    removeDialog(navPanelId)
    window.removeEventListener('keydown', onNavKeydown)
    document.removeEventListener('mousedown', onNavDocMousedown, true)
    window.removeEventListener('scroll', updateNavPanelPosition, true)
    const active = document.activeElement as HTMLElement | null
    if (active && navPanelEl.value?.contains(active)) navToggleEl.value?.focus()
  }
})

let narrowMq: MediaQueryList | null = null
function onNarrowChange(e: MediaQueryListEvent) {
  isNarrow.value = e.matches
  if (!e.matches) closeNav()
}

onMounted(() => {
  const saved = localStorage.getItem('theme')
  if (saved === 'dark') {
    document.documentElement.classList.add('dark')
    isDark.value = true
  }
  collapsed.value = localStorage.getItem(sidebarCollapsedKey) === 'true'
  expandedGroups.value = loadExpanded()
  if (typeof window.matchMedia === 'function') {
    narrowMq = window.matchMedia('(max-width: 960px)')
    isNarrow.value = narrowMq.matches
    narrowMq.addEventListener('change', onNarrowChange)
  }
})

onBeforeUnmount(() => {
  narrowMq?.removeEventListener('change', onNarrowChange)
  if (navOpen.value) {
    removeDialog(navPanelId)
    window.removeEventListener('keydown', onNavKeydown)
    document.removeEventListener('mousedown', onNavDocMousedown, true)
    window.removeEventListener('scroll', updateNavPanelPosition, true)
  }
})

function toggleTheme() {
  document.documentElement.classList.toggle('dark')
  isDark.value = document.documentElement.classList.contains('dark')
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function toggleSidebar() {
  collapsed.value = !collapsed.value
  localStorage.setItem(sidebarCollapsedKey, String(collapsed.value))
}

interface NavItem {
  to: string
  label: string
  sub: string
  icon: IconName
}
interface NavGroup {
  key: string
  label: string
  icon: IconName
  children: NavItem[]
}
type NavEntry = NavItem | NavGroup
function isGroup(e: NavEntry): e is NavGroup {
  return (e as NavGroup).children !== undefined
}

// EDIT: replace with the project's routes. Every item and group needs an icons.ts icon;
// label is the Chinese menu text, sub is the one-line page description shown in the shell header.
const nav: NavEntry[] = [
  {
    key: 'workbench',
    label: '工作台',
    icon: 'home',
    children: [
      { to: '/', label: '概览', sub: '系统健康与近期汇总', icon: 'home' },
      { to: '/alerts', label: '告警', sub: '告警流与处置状态', icon: 'bell' },
      { to: '/findings', label: '调查发现', sub: '调查结论的受理与标记', icon: 'search' },
      { to: '/runs', label: '调查运行', sub: '执行轨迹与结果', icon: 'play' },
    ],
  },
  {
    key: 'config',
    label: '配置',
    icon: 'settings',
    children: [
      { to: '/sources', label: '数据接入', sub: '数据源上线与运行记录', icon: 'panel-list' },
      { to: '/rules', label: '规则', sub: '阈值、计数与去重规则', icon: 'list' },
      { to: '/graph', label: '图谱', sub: '实体关系检索与展开', icon: 'network' },
      { to: '/connectors', label: '连接器', sub: '外部数据源管理', icon: 'menu' },
    ],
  },
]

// EDIT: title-only groups never render in the sidebar; they only feed page-title
// computation for routes outside the visible nav. Keep or delete.
const titleOnlyGroups: NavGroup[] = [
  {
    key: 'proposals',
    label: '提案',
    icon: 'arrow-up',
    children: [
      { to: '/proposals', label: '提案', sub: '待审提案的列表与审批', icon: 'arrow-up' },
    ],
  },
]

const flatLeaves = computed<{ item: NavItem; group?: NavGroup }[]>(() => {
  const out: { item: NavItem; group?: NavGroup }[] = []
  for (const e of [...nav, ...titleOnlyGroups]) {
    if (isGroup(e)) for (const c of e.children) out.push({ item: c, group: e })
    else out.push({ item: e })
  }
  return out
})

const activeLeaf = computed(() => {
  const exact = flatLeaves.value.find((l) => l.item.to === route.path)
  if (exact) return exact
  return flatLeaves.value
    .filter((l) => l.item.to !== '/' && route.path.startsWith(l.item.to))
    .sort((a, b) => b.item.to.length - a.item.to.length)[0]
})
const pageTitle = computed(() => {
  if (route.meta.title) return route.meta.title as string
  const l = activeLeaf.value
  // EDIT: brand fallback title
  if (!l) return 'YourApp'
  if (!l.group) return l.item.label
  return l.group.label === l.item.label ? l.item.label : `${l.group.label} · ${l.item.label}`
})
const pageSub = computed(() => activeLeaf.value?.item.sub ?? '')
const activeGroupKey = computed(() => activeLeaf.value?.group?.key)

watch(
  pageTitle,
  (t) => {
    // EDIT: document.title brand suffix
    document.title = t === 'YourApp' ? t : `${t} - YourApp`
  },
  { immediate: true },
)

// EDIT: storage key prefix, rename per project
const navExpandedKey = 'yourapp.nav.expanded'
const expandedGroups = ref<Set<string>>(new Set())
function loadExpanded(): Set<string> {
  try {
    const raw = localStorage.getItem(navExpandedKey)
    if (raw) return new Set(JSON.parse(raw) as string[])
  } catch {}
  return new Set()
}
function isExpanded(key: string): boolean {
  return expandedGroups.value.has(key) || activeGroupKey.value === key
}
function toggleGroup(key: string) {
  const next = new Set(expandedGroups.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  expandedGroups.value = next
  localStorage.setItem(navExpandedKey, JSON.stringify([...next]))
}
</script>

<template>
  <div class="ambient-shell">
    <div class="shell-layout" :class="{ 'is-collapsed': railCollapsed, 'nav-open': navOpen }">
      <aside
        ref="navPanelEl"
        class="shell-sidebar"
        :style="isNarrow ? navPanelStyle : undefined"
        :aria-label="isNarrow ? '导航菜单' : undefined"
        :tabindex="isNarrow ? -1 : undefined"
      >
        <div class="brand">
          <div class="brand-mark"><Icon name="moon" :size="20" /></div>
          <div v-if="!railCollapsed" class="brand-text">
            <!-- EDIT: brand name and subtitle -->
            <span class="brand-name">YourApp</span>
            <span class="brand-sub">控制台</span>
          </div>
        </div>

        <nav class="nav-list">
          <template v-for="item in nav" :key="isGroup(item) ? item.key : item.to">
            <RouterLink
              v-if="!isGroup(item)"
              :to="item.to"
              class="nav-item"
              :class="{ 'router-link-active': activeLeaf?.item.to === item.to }"
              :aria-current="activeLeaf?.item.to === item.to ? 'page' : undefined"
              :title="railCollapsed ? item.label : undefined"
            >
              <Icon :name="item.icon" class="nav-icon" />
              <span v-if="!railCollapsed" class="nav-label">{{ item.label }}</span>
            </RouterLink>

            <div
              v-else
              class="nav-group"
              :class="{ 'is-open': isExpanded(item.key), 'has-active': activeGroupKey === item.key }"
            >
              <button
                type="button"
                class="nav-item nav-group-header"
                :title="railCollapsed ? item.label : undefined"
                :aria-expanded="isExpanded(item.key)"
                @click="toggleGroup(item.key)"
              >
                <Icon :name="item.icon" class="nav-icon" />
                <span v-if="!railCollapsed" class="nav-label">{{ item.label }}</span>
                <Icon
                  v-if="!railCollapsed"
                  name="chevron-right"
                  class="nav-chevron"
                  :size="14"
                />
              </button>

              <div class="nav-subitems">
                <RouterLink
                  v-for="child in item.children"
                  :key="child.to"
                  :to="child.to"
                  class="nav-item nav-subitem"
                  :class="{ 'router-link-active': activeLeaf?.item.to === child.to }"
                  :aria-current="activeLeaf?.item.to === child.to ? 'page' : undefined"
                  :title="railCollapsed ? child.label : undefined"
                >
                  <Icon :name="child.icon" class="nav-icon" />
                  <span v-if="!railCollapsed" class="nav-label">{{ child.label }}</span>
                </RouterLink>
              </div>
            </div>
          </template>
        </nav>

        <button
          class="rail-toggle"
          :title="collapsed ? '展开' : '收起'"
          :aria-label="collapsed ? '展开侧边栏' : '收起侧边栏'"
          :aria-expanded="!collapsed"
          @click="toggleSidebar"
        >
          <Icon :name="collapsed ? 'chevron-right' : 'chevron-left'" :size="16" />
        </button>
      </aside>

      <main class="shell-main">
        <header class="shell-header">
          <button
            ref="navToggleEl"
            class="toolbar-button shell-nav-toggle"
            aria-label="导航菜单"
            :aria-expanded="navOpen"
            @click="toggleNav"
          >
            <Icon name="menu" :size="16" />
          </button>
          <div class="header-title-block">
            <!-- EDIT: eyebrow text -->
            <span class="page-eyebrow">YourApp</span>
            <div class="header-title-row">
              <h1 class="page-title">{{ pageTitle }}</h1>
              <span v-if="pageSub" class="page-sub">{{ pageSub }}</span>
            </div>
          </div>
          <div class="header-actions">
            <button class="toolbar-button" :title="isDark ? '切换浅色' : '切换深色'" @click="toggleTheme">
              <Icon :name="isDark ? 'sun' : 'moon'" :size="16" />
            </button>
          </div>
        </header>

        <div class="toast-stack" aria-live="polite">
          <div
            v-for="toast in errorBus.toasts.value"
            :key="toast.id"
            class="toast"
            :class="toast.kind === 'ok' ? 'toast-ok' : 'toast-err'"
          >
            <Icon :name="toast.kind === 'ok' ? 'check-circle' : 'x-circle'" :size="16" class="toast-icon" />
            <span class="toast-text">{{ toast.text }}</span>
            <button
              v-if="toast.kind === 'err'"
              class="toast-close"
              aria-label="关闭"
              @click="errorBus.dismiss(toast.id)"
            ><Icon name="x" :size="14" /></button>
          </div>
        </div>

        <div class="page-stage">
          <RouterView v-slot="{ Component }">
            <transition name="page-fade">
              <component :is="Component" />
            </transition>
          </RouterView>
        </div>
      </main>
    </div>
    <ConfirmDialog />
  </div>
</template>

<style>
.ambient-shell {
  min-height: 100vh;
}

.shell-layout {
  display: grid;
  grid-template-columns: 232px minmax(0, 1fr);
  gap: 20px;
  padding: 20px;
  height: 100vh;
  overflow: hidden;
  transition: grid-template-columns var(--duration-base) var(--ease-spring);
}

.shell-layout.is-collapsed {
  grid-template-columns: 72px minmax(0, 1fr);
}

.shell-sidebar {
  position: sticky;
  top: 20px;
  align-self: start;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px 14px;
  height: calc(100vh - 40px);
  border-radius: var(--radius-md);
  background: var(--aside-bg);
  border: 1px solid var(--border-soft);
  overflow: hidden;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 6px 14px;
  border-bottom: 1px solid var(--border-soft);
}

.shell-layout.is-collapsed .brand {
  justify-content: center;
  padding: 4px 0 14px;
}

.brand-mark {
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-md);
  background: var(--brand-mark-bg);
  border: 1px solid var(--floating-border);
  color: var(--accent);
  flex-shrink: 0;
  font-size: 18px;
}

.brand-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow: hidden;
}

.brand-name {
  font-weight: 800;
  font-size: var(--text-lg);
  letter-spacing: -0.01em;
  color: var(--text-primary);
}

.brand-sub {
  font-size: var(--text-xs);
  color: var(--text-muted);
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.nav-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  overflow-y: auto;
  margin: 0 -4px;
  padding: 0 4px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--radius-md);
  font-size: var(--text-md);
  font-weight: 500;
  color: var(--text-secondary);
  text-decoration: none;
  border: 1px solid transparent;
  transition: background var(--duration-fast) var(--ease-spring),
              color var(--duration-fast) var(--ease-spring),
              border-color var(--duration-fast) var(--ease-spring);
}

.shell-layout.is-collapsed .nav-item {
  justify-content: center;
  padding: 8px;
}

.nav-item:hover {
  background: var(--accent-soft);
  color: var(--text-primary);
  box-shadow: none;
}

.nav-item.router-link-active {
  background: var(--accent-soft);
  color: var(--text-primary);
  border-color: var(--floating-border);
  font-weight: 600;
}

:root.dark .nav-item:hover,
:root.dark .nav-item.router-link-active {
  color: var(--accent);
}

.nav-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  opacity: 0.85;
}

.nav-item.router-link-active .nav-icon {
  opacity: 1;
}

.nav-label {
  white-space: nowrap;
  flex: 1;
}

.nav-group {
  display: flex;
  flex-direction: column;
}
.nav-group-header {
  width: 100%;
  background: transparent;
  cursor: pointer;
  text-align: left;
}
.nav-chevron {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  opacity: 0.55;
  transition: transform var(--duration-fast) var(--ease-spring);
}
.nav-group.is-open .nav-chevron {
  transform: rotate(90deg);
}
.nav-group.has-active > .nav-group-header {
  color: var(--text-primary);
}
.nav-group.has-active > .nav-group-header .nav-icon {
  opacity: 1;
}

.nav-subitems {
  display: none;
  flex-direction: column;
  gap: 2px;
  margin-top: 2px;
}
.nav-group.is-open > .nav-subitems {
  display: flex;
}
.nav-subitem {
  padding-left: 28px;
  font-size: var(--text-sm);
  color: var(--text-muted);
}
.nav-subitem .nav-icon {
  width: 15px;
  height: 15px;
}
.shell-layout.is-collapsed .nav-group-header {
  display: none;
}
.shell-layout.is-collapsed .nav-subitems {
  display: flex;
}

.rail-toggle {
  align-self: stretch;
  display: grid;
  place-items: center;
  height: 32px;
  min-height: 32px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-soft);
  background: var(--input-bg);
  color: var(--text-secondary);
  cursor: pointer;
  font-size: var(--text-lg);
  font-family: var(--font-display);
  transition:
    background var(--duration-fast) var(--ease-spring),
    color var(--duration-fast) var(--ease-spring),
    border-color var(--duration-fast) var(--ease-spring);
}

.rail-toggle:hover {
  background: var(--input-bg-hover);
  color: var(--accent);
  border-color: var(--floating-border);
  box-shadow: none;
}

.shell-main {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
  height: calc(100vh - 40px);
  overflow: hidden;
}

.shell-header {
  flex-shrink: 0;
  z-index: var(--z-shell-header);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 22px;
  border-radius: var(--radius-md);
  background: var(--header-bg);
  border: 1px solid var(--border-soft);
}

.header-title-block {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.page-eyebrow {
  font-size: var(--text-xs);
  font-weight: 600;
  letter-spacing: 0.04em;
  color: var(--text-muted);
  font-family: var(--font-mono);
}

.header-title-row {
  display: flex;
  align-items: baseline;
  gap: 14px;
  min-width: 0;
}

.page-title {
  margin: 0;
  font-size: var(--text-xl);
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--text-primary);
  line-height: 1.2;
  white-space: nowrap;
  flex-shrink: 0;
}

.page-sub {
  font-size: var(--text-sm);
  color: var(--text-muted);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.toolbar-button {
  height: 36px;
  min-width: 36px;
  padding: 0 10px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-soft);
  background: var(--input-bg);
  color: var(--text-secondary);
  cursor: pointer;
  font-family: var(--font-display);
  font-size: var(--text-base);
  transition:
    background var(--duration-fast) var(--ease-spring),
    color var(--duration-fast) var(--ease-spring),
    border-color var(--duration-fast) var(--ease-spring),
    transform var(--duration-fast) var(--ease-spring);
}

.toolbar-button:hover {
  background: var(--input-bg-hover);
  color: var(--accent);
  border-color: var(--floating-border);
  transform: translateY(-1px);
  box-shadow: none;
}

.toast-stack {
  position: fixed;
  top: 16px;
  right: 16px;
  z-index: var(--z-toast);
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: min(380px, calc(100vw - 32px));
}

.toast {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-floating);
  background: var(--surface-elevated);
  color: var(--text-primary);
  font-size: var(--text-md);
}

.toast-icon {
  flex-shrink: 0;
  margin-top: 1px;
}

.toast-ok .toast-icon {
  color: var(--success);
}

.toast-err .toast-icon {
  color: var(--danger);
}

.toast-text {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
}

.toast-close {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  min-height: 24px;
  border: none;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  border-radius: var(--radius-sm);
  transition: background var(--duration-fast) var(--ease-spring);
  padding: 0;
  flex-shrink: 0;
}

.toast-close:hover {
  background: var(--surface-soft);
  box-shadow: none;
}

.page-stage {
  position: relative;
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}
.page-stage:has(.split-layout) {
  overflow: hidden;
}
.page-stage *:has(.split-layout) {
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-height: 0;
}
.page-stage *:has(.split-layout) > *:not(:has(.split-layout)):not(.split-layout) {
  flex-shrink: 0;
}
.page-fade-enter-active,
.page-fade-leave-active {
  transition: opacity var(--duration-fast) var(--ease-spring), transform var(--duration-fast) var(--ease-spring);
}
.page-fade-leave-active {
  position: absolute;
  inset: 0;
}
.page-fade-enter-from { opacity: 0; transform: translateY(4px); }
.page-fade-leave-to   { opacity: 0; transform: translateY(-4px); }

.shell-nav-toggle {
  display: none;
}

@media (max-width: 960px) {
  .shell-layout {
    grid-template-columns: minmax(0, 1fr);
    padding: 14px;
    gap: 14px;
  }
  .shell-main { height: calc(100vh - 28px); }
  .shell-header { padding: 12px 16px; }
  .shell-nav-toggle { display: grid; }
  .rail-toggle { display: none; }
  .shell-sidebar {
    position: fixed;
    top: 0;
    left: 0;
    width: min(300px, calc(100vw - 96px));
    height: auto;
    max-height: calc(100vh - 28px);
    z-index: var(--z-nav-panel);
    outline: none;
    box-shadow: var(--shadow-floating);
    visibility: hidden;
    opacity: 0;
    transform: translateY(-6px);
    pointer-events: none;
    transition:
      opacity var(--duration-fast) var(--ease-spring),
      transform var(--duration-fast) var(--ease-spring),
      visibility 0s linear var(--duration-fast);
  }
  .shell-layout.nav-open .shell-sidebar {
    visibility: visible;
    opacity: 1;
    transform: none;
    pointer-events: auto;
    transition:
      opacity var(--duration-fast) var(--ease-spring),
      transform var(--duration-fast) var(--ease-spring);
  }
}

@media (max-width: 760px) {
  .shell-layout {
    height: auto;
    min-height: 100vh;
    overflow: visible;
  }
  .shell-main {
    height: auto;
    min-height: calc(100vh - 28px);
    overflow: visible;
  }
  .page-stage,
  .page-stage:has(.split-layout) {
    overflow: visible;
  }
}

@media (max-width: 640px) {
  .shell-layout {
    padding: 10px;
    gap: 10px;
  }
  .shell-main {
    min-height: calc(100vh - 20px);
  }
}

</style>
