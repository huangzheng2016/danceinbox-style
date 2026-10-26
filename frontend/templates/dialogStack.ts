// copy verbatim; edit only the marked regions
const stack: symbol[] = []

export function pushDialog(id: symbol) {
  removeDialog(id)
  stack.push(id)
}

export function removeDialog(id: symbol) {
  const i = stack.indexOf(id)
  if (i >= 0) stack.splice(i, 1)
}

export function isTopDialog(id: symbol): boolean {
  return stack[stack.length - 1] === id
}

let scrollLockCount = 0

export function lockBodyScroll() {
  scrollLockCount += 1
  document.body.classList.add('dialog-scroll-locked')
}

export function unlockBodyScroll() {
  scrollLockCount = Math.max(0, scrollLockCount - 1)
  if (scrollLockCount === 0) {
    document.body.classList.remove('dialog-scroll-locked')
  }
}
