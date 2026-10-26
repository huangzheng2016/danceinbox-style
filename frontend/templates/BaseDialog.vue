<!-- copy verbatim; edit only the marked regions -->
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import Icon from './Icon.vue'
import { pushDialog, removeDialog, isTopDialog, lockBodyScroll, unlockBodyScroll } from './dialogStack'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    width?: string
  }>(),
  { width: '520px' },
)

const emit = defineEmits<{
  (e: 'close'): void
}>()

const dialogStyle = computed(() => ({
  '--dialog-width': props.width,
}))

const dialogId = Symbol('BaseDialog')
const dialogEl = ref<HTMLElement | null>(null)
let previouslyFocused: HTMLElement | null = null

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

function close() {
  emit('close')
}

function trapTab(e: KeyboardEvent) {
  const root = dialogEl.value
  if (!root) return
  const items = Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
    (el) => el.offsetParent !== null || el === document.activeElement,
  )
  if (items.length === 0) {
    e.preventDefault()
    root.focus()
    return
  }
  const first = items[0]
  const last = items[items.length - 1]
  const active = document.activeElement as HTMLElement | null
  if (e.shiftKey) {
    if (!active || active === first || !root.contains(active)) {
      e.preventDefault()
      last.focus()
    }
  } else if (!active || active === last || !root.contains(active)) {
    e.preventDefault()
    first.focus()
  }
}

function onKeydown(e: KeyboardEvent) {
  if (!props.open || !isTopDialog(dialogId)) return
  if (e.key === 'Escape') close()
  else if (e.key === 'Tab') trapTab(e)
}

watch(
  () => props.open,
  (open, wasOpen) => {
    if (typeof document === 'undefined') return
    if (open === wasOpen) return
    if (open) {
      previouslyFocused = document.activeElement as HTMLElement | null
      lockBodyScroll()
      pushDialog(dialogId)
      window.addEventListener('keydown', onKeydown)
      nextTick(() => {
        const root = dialogEl.value
        if (!root) return
        const target = root.querySelector<HTMLElement>(FOCUSABLE)
        ;(target ?? root).focus()
      })
    } else {
      unlockBodyScroll()
      removeDialog(dialogId)
      window.removeEventListener('keydown', onKeydown)
      previouslyFocused?.focus()
      previouslyFocused = null
    }
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  if (typeof document === 'undefined') return
  if (props.open) {
    unlockBodyScroll()
    removeDialog(dialogId)
    window.removeEventListener('keydown', onKeydown)
  }
})
</script>

<template>
  <Teleport to="body">
    <Transition name="dialog-fade">
      <div
        v-if="open"
        class="base-dialog-overlay"
        @click.self="close"
      >
        <div
          ref="dialogEl"
          class="base-dialog"
          :style="dialogStyle"
          role="dialog"
          aria-modal="true"
          :aria-label="title"
          tabindex="-1"
        >
          <header class="base-dialog-head">
            <h3>{{ title }}</h3>
            <button
              type="button"
              class="muted base-dialog-close"
              aria-label="关闭"
              @click="close"
            >
              <Icon
                name="x"
                :size="14"
              />
            </button>
          </header>
          <section class="base-dialog-body">
            <slot />
          </section>
          <footer
            v-if="$slots.footer"
            class="base-dialog-footer"
          >
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.base-dialog-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-dialog);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: var(--overlay-mask);
}

.base-dialog {
  width: min(var(--dialog-width), 100%);
  max-width: 100%;
  min-width: 0;
  max-height: calc(100vh - 48px);
  max-height: calc(100dvh - 48px);
  display: flex;
  flex-direction: column;
  background: var(--surface-strong);
  color: var(--text-primary);
  border: 1px solid var(--floating-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-dialog);
  outline: none;
}

.base-dialog-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-soft);
}

.base-dialog-head h3 {
  margin: 0;
  font-size: var(--text-lg);
}

.base-dialog-close {
  width: 32px;
  height: 32px;
  min-height: 32px;
  padding: 0;
  border-radius: var(--radius-sm);
}

.base-dialog-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.base-dialog-footer {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  padding: 12px 20px;
  border-top: 1px solid var(--border-soft);
}

.dialog-fade-enter-active,
.dialog-fade-leave-active {
  transition: opacity var(--duration-fast) var(--ease-spring);
}

.dialog-fade-enter-active .base-dialog,
.dialog-fade-leave-active .base-dialog {
  transition:
    transform var(--duration-base) var(--ease-spring),
    opacity var(--duration-fast) var(--ease-spring);
}

.dialog-fade-enter-from,
.dialog-fade-leave-to {
  opacity: 0;
}

.dialog-fade-enter-from .base-dialog,
.dialog-fade-leave-to .base-dialog {
  transform: translateY(8px) scale(0.97);
}
</style>
