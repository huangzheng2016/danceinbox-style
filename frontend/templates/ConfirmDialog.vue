<!-- copy verbatim; edit only the marked regions -->
<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { confirmState, resolveConfirm } from './useConfirm'
import { pushDialog, removeDialog, isTopDialog, lockBodyScroll, unlockBodyScroll } from './dialogStack'

const dialogId = Symbol('ConfirmDialog')
const dialogEl = ref<HTMLElement | null>(null)

function focusInside(): boolean {
  const root = dialogEl.value
  return !!root && root.contains(document.activeElement)
}

function onKeyDown(e: KeyboardEvent) {
  if (!confirmState.open || !isTopDialog(dialogId) || !focusInside()) return
  if (e.key === 'Escape') {
    e.preventDefault()
    resolveConfirm(false)
  } else if (e.key === 'Enter') {
    if ((e.target as HTMLElement | null)?.tagName === 'BUTTON') return
    e.preventDefault()
    resolveConfirm(true)
  }
}

onMounted(() => window.addEventListener('keydown', onKeyDown))
onUnmounted(() => window.removeEventListener('keydown', onKeyDown))

watch(
  () => confirmState.open,
  async (open) => {
    if (open) {
      lockBodyScroll()
      pushDialog(dialogId)
      await nextTick()
      const root = dialogEl.value
      const btn = root?.querySelector<HTMLElement>('.cd-actions button:last-child')
      ;(btn ?? root)?.focus()
    } else {
      unlockBodyScroll()
      removeDialog(dialogId)
    }
  },
)
</script>

<template>
  <Teleport to="body">
    <Transition name="cd-fade">
      <div v-if="confirmState.open" class="cd-mask" @click.self="resolveConfirm(false)">
        <div
          ref="dialogEl"
          class="cd-dialog"
          role="dialog"
          aria-modal="true"
          tabindex="-1"
        >
          <div class="cd-title">{{ confirmState.title }}</div>
          <div class="cd-message">{{ confirmState.message }}</div>
          <div class="cd-actions">
            <button class="muted" @click="resolveConfirm(false)">{{ confirmState.cancelText }}</button>
            <button :class="confirmState.danger ? 'danger' : ''" @click="resolveConfirm(true)">
              {{ confirmState.confirmText }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.cd-mask {
  position: fixed;
  inset: 0;
  background: var(--overlay-mask);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: var(--z-confirm);
}
.cd-dialog {
  background: var(--surface-strong);
  color: var(--text-primary);
  border-radius: var(--radius-md);
  padding: 20px 24px;
  max-width: 520px;
  min-width: 0;
  width: min(520px, calc(100vw - 24px));
  box-shadow: var(--shadow-dialog);
  border: 1px solid var(--floating-border);
  outline: none;
}
.cd-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin-bottom: 10px;
}
.cd-message {
  font-size: var(--text-md);
  line-height: 1.6;
  margin-bottom: 18px;
  white-space: pre-wrap;
  word-break: break-word;
}
.cd-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
.cd-actions button.danger {
  background: var(--danger);
  color: var(--text-on-danger);
  border-color: var(--danger);
}
.cd-actions button.danger:hover {
  filter: brightness(0.92);
}

.cd-fade-enter-active, .cd-fade-leave-active {
  transition: opacity var(--duration-fast) var(--ease-spring);
}
.cd-fade-enter-active .cd-dialog, .cd-fade-leave-active .cd-dialog {
  transition: transform var(--duration-fast) var(--ease-spring);
}
.cd-fade-enter-from, .cd-fade-leave-to { opacity: 0; }
.cd-fade-enter-from .cd-dialog, .cd-fade-leave-to .cd-dialog {
  transform: scale(0.96);
}
</style>
