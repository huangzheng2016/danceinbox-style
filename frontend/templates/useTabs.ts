// copy verbatim; edit only the marked regions
import { nextTick, type Ref } from 'vue'

export function useTabs<T>(tabs: readonly T[], tab: Ref<T>, selectTab: (next: T) => Promise<void>) {
  async function onTabKeydown(event: KeyboardEvent) {
    const current = tabs.indexOf(tab.value)
    let next: number
    if (event.key === 'ArrowRight') next = (current + 1) % tabs.length
    else if (event.key === 'ArrowLeft') next = (current - 1 + tabs.length) % tabs.length
    else if (event.key === 'Home') next = 0
    else if (event.key === 'End') next = tabs.length - 1
    else return
    event.preventDefault()
    const tablist = (event.currentTarget as HTMLElement | null)?.parentElement
    await selectTab(tabs[next])
    await nextTick()
    tablist?.querySelector<HTMLElement>('[role="tab"][aria-selected="true"]')?.focus()
  }
  return { onTabKeydown }
}
