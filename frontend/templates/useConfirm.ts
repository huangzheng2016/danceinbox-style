// copy verbatim; edit only the marked regions
import { reactive } from 'vue'

export type ConfirmOptions = {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

type State = ConfirmOptions & {
  open: boolean
  resolver: ((ok: boolean) => void) | null
}

export const confirmState = reactive<State>({
  open: false,
  message: '',
  resolver: null,
})

export function useConfirm() {
  return (opts: ConfirmOptions): Promise<boolean> => {
    if (confirmState.resolver) {
      confirmState.resolver(false)
    }
    Object.assign(confirmState, {
      open: true,
      title: opts.title ?? '确认',
      message: opts.message,
      confirmText: opts.confirmText ?? '确定',
      cancelText: opts.cancelText ?? '取消',
      danger: !!opts.danger,
    })
    return new Promise<boolean>((resolve) => {
      confirmState.resolver = resolve
    })
  }
}

export function resolveConfirm(ok: boolean) {
  const r = confirmState.resolver
  confirmState.resolver = null
  confirmState.open = false
  if (r) r(ok)
}
