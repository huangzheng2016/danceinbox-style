import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import Combobox, { type ComboOption } from '@/base/Combobox.vue'
import Segmented from '@/base/Segmented.vue'
import SortTh from '@/base/SortTh.vue'
import { rangeBounds } from '@/base/time'
import { useSelection } from '@/base/useSelection'
import { useDirtyGuard } from '@/base/useDirtyGuard'
import { confirmState, resolveConfirm } from '@/base/useConfirm'

const OPTIONS: ComboOption[] = [
  { value: '1', label: 'alice' },
  { value: '2', label: 'bob' },
  { value: '3', label: 'carol' },
]

function mountCombo(props: Record<string, unknown>, initial: ComboOption | ComboOption[] | null) {
  const model = ref(initial)
  const w = mount(
    defineComponent({
      setup: () => () =>
        h(Combobox, { ...props, modelValue: model.value, 'onUpdate:modelValue': (v) => (model.value = v) }),
    }),
    { attachTo: document.body },
  )
  return { w, model, input: w.find('input') }
}

const listbox = () => document.querySelector('[role="listbox"]')
const options = () => [...document.querySelectorAll<HTMLElement>('[role="option"]')].map((o) => o.textContent?.trim())

afterEach(() => {
  document.body.innerHTML = ''
})

describe('Combobox', () => {
  it('filters static options and picks with the keyboard', async () => {
    const { input, model } = mountCombo({ options: OPTIONS }, null)
    await input.trigger('click')
    await input.setValue('o')
    await flushPromises()
    expect(options()).toEqual(['bob', 'carol'])
    await input.trigger('keydown', { key: 'ArrowDown' })
    await input.trigger('keydown', { key: 'Enter' })
    expect(model.value).toEqual({ value: '3', label: 'carol' })
    expect(listbox()).toBeNull()
  })

  it('closes on Escape without reopening when focus returns to the input', async () => {
    const { input } = mountCombo({ options: OPTIONS }, null)
    await input.trigger('click')
    expect(listbox()).not.toBeNull()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    expect(listbox()).toBeNull()
    expect(document.activeElement).toBe(input.element)
  })

  it('debounces server search and shows the saved label before any search', async () => {
    vi.useFakeTimers()
    const load = vi.fn(async (q: string) => OPTIONS.filter((o) => o.label.includes(q)))
    const { input } = mountCombo({ load }, { value: '9', label: 'zed' })
    expect(input.element.value).toBe('zed')
    await input.trigger('click')
    expect(load).toHaveBeenLastCalledWith('', expect.any(AbortSignal))
    await input.setValue('a')
    await input.setValue('al')
    vi.advanceTimersByTime(250)
    await flushPromises()
    expect(load).toHaveBeenCalledTimes(2)
    expect(load).toHaveBeenLastCalledWith('al', expect.any(AbortSignal))
    expect(options()).toEqual(['alice'])
    vi.useRealTimers()
  })

  it('multiple: toggles options and removes the last chip with Backspace', async () => {
    const { input, model } = mountCombo({ options: OPTIONS, multiple: true }, [])
    await input.trigger('click')
    document.querySelectorAll<HTMLElement>('[role="option"]')[0].click()
    await nextTick()
    document.querySelectorAll<HTMLElement>('[role="option"]')[1].click()
    await nextTick()
    expect((model.value as ComboOption[]).map((o) => o.label)).toEqual(['alice', 'bob'])
    await input.trigger('keydown', { key: 'Backspace' })
    expect((model.value as ComboOption[]).map((o) => o.label)).toEqual(['alice'])
  })
})

describe('Segmented', () => {
  it('moves the selection with arrow keys and wraps', async () => {
    const value = ref('a')
    const w = mount(() =>
      h(Segmented, {
        options: [
          { value: 'a', label: 'A' },
          { value: 'b', label: 'B' },
        ],
        modelValue: value.value,
        'onUpdate:modelValue': (v: string) => (value.value = v),
      }),
    )
    await w.find('[role="radiogroup"]').trigger('keydown', { key: 'ArrowLeft' })
    expect(value.value).toBe('b')
    await nextTick()
    expect(w.find('[aria-checked="true"]').text()).toBe('B')
  })
})

describe('SortTh', () => {
  it('cycles ascending, descending, default', async () => {
    const sort = ref('')
    const w = mount(() =>
      h('table', [
        h('tr', [
          h(SortTh, { field: 'name', sort: sort.value, onSort: (s: string) => (sort.value = s) }, () => '名称'),
        ]),
      ]),
    )
    const seen = []
    for (let i = 0; i < 3; i++) {
      await w.find('button').trigger('click')
      seen.push(sort.value)
    }
    expect(seen).toEqual(['name', '-name', ''])
  })
})

describe('rangeBounds', () => {
  const now = new Date(2026, 9, 26, 15, 30)
  const day = (y: number, m: number, d: number) => new Date(y, m - 1, d).getTime() / 1000

  it.each([
    ['', null],
    ['today', { from: day(2026, 10, 26), to: day(2026, 10, 27) }],
    ['7d', { from: day(2026, 10, 20), to: day(2026, 10, 27) }],
    ['2026-10-01~2026-10-03', { from: day(2026, 10, 1), to: day(2026, 10, 4) }],
    ['2026-10-03~2026-10-01', null],
    ['garbage', null],
  ])('%s', (range, want) => {
    expect(rangeBounds(range, now)).toEqual(want)
  })
})

describe('useSelection', () => {
  it('drops ids whose rows leave the page', async () => {
    const rows = ref([{ id: 'a' }, { id: 'b' }])
    const s = useSelection(rows)
    s.toggleAll()
    expect(s.all.value).toBe(true)
    rows.value = [{ id: 'b' }, { id: 'c' }]
    await nextTick()
    expect(s.ids.value).toEqual(['b'])
    expect(s.some.value).toBe(true)
  })
})

describe('useDirtyGuard', () => {
  it('asks only when the form changed since reset', async () => {
    const form = ref({ name: 'a' })
    const guard = useDirtyGuard(() => form.value)
    expect(await guard.confirmDiscard()).toBe(true)
    form.value.name = 'b'
    const answer = guard.confirmDiscard()
    expect(confirmState.open).toBe(true)
    resolveConfirm(false)
    expect(await answer).toBe(false)
    guard.reset()
    expect(guard.dirty.value).toBe(false)
  })
})

describe('stacked modals', () => {
  it('a confirm opened from inside a dialog renders above it', async () => {
    const { default: BaseDialog } = await import('@/base/BaseDialog.vue')
    const { default: ConfirmDialog } = await import('@/base/ConfirmDialog.vue')
    const { useConfirm } = await import('@/base/useConfirm')
    // ConfirmDialog mounts first, like the shell does, so it comes first in the DOM
    mount(ConfirmDialog, { attachTo: document.body })
    mount(BaseDialog, { props: { open: true, title: '编辑' }, attachTo: document.body })
    await nextTick()
    void useConfirm()({ message: '放弃修改?' })
    await flushPromises()
    const z = [...document.querySelectorAll<HTMLElement>('.dialog-mask')].map((m) => ({
      title: m.querySelector('h2')?.textContent,
      z: m.style.zIndex,
    }))
    const depth = (t: string) => Number(/\+ (\d+)/.exec(z.find((d) => d.title === t)!.z)![1])
    expect(depth('确认')).toBeGreaterThan(depth('编辑'))
    resolveConfirm(false)
  })
})

describe('useFormValidation', () => {
  it('stays quiet when a field is only passed through, speaks after typing or submit', async () => {
    const { useFormValidation } = await import('@/base/useFormValidation')
    const form = ref({ name: '', note: '' })
    const v = useFormValidation(() => ({ name: form.value.name ? '' : '请输入名称', note: '' }))
    const w = mount(() =>
      h('form', { ...{ onInput: v.form.input, onFocusout: v.form.focusout } }, [
        h('input', {
          name: 'name',
          value: form.value.name,
          onInput: (e: Event) => (form.value.name = (e.target as HTMLInputElement).value),
        }),
        h('input', { name: 'note' }),
      ]),
    )
    const name = w.find('input[name="name"]')
    await name.trigger('focusout')
    expect(v.error('name')).toBe('')
    await name.setValue('a')
    await name.setValue('')
    expect(v.error('name')).toBe('')
    await name.trigger('focusout')
    expect(v.error('name')).toBe('请输入名称')
    v.reset()
    expect(v.error('name')).toBe('')
    expect(v.validate()).toBe(false)
    expect(v.error('name')).toBe('请输入名称')
  })
})
