import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick, ref, withDirectives } from 'vue'
import { mount } from '@vue/test-utils'
import Select from '@/base/Select.vue'
import UnitInput from '@/base/UnitInput.vue'
import { diffValues } from '@/base/diff'
import { installTooltips, vTip } from '@/base/tooltip'

const OPTIONS = [
  { value: '', label: '全部' },
  { value: 'mail', label: '邮件' },
  { value: 'sms', label: '短信', disabledReason: '未开通短信服务' },
  { value: 'hook', label: 'Webhook' },
]

function mountSelect(initial = '') {
  const model = ref(initial)
  const w = mount(
    defineComponent({
      setup: () => () =>
        h(Select, {
          options: OPTIONS,
          label: '渠道',
          modelValue: model.value,
          'onUpdate:modelValue': (v: string | number) => (model.value = String(v)),
        }),
    }),
    { attachTo: document.body },
  )
  return { w, model, button: w.find('button[role="combobox"]') }
}

const shown = () => [...document.querySelectorAll<HTMLElement>('[role="option"]')]

afterEach(() => {
  document.body.innerHTML = ''
})

describe('Select', () => {
  it('shows the current label and opens on the current option', async () => {
    const { button } = mountSelect('mail')
    expect(button.text()).toBe('邮件')
    await button.trigger('keydown', { key: 'ArrowDown' })
    expect(button.attributes('aria-expanded')).toBe('true')
    const active = shown().find((o) => o.classList.contains('active'))
    expect(active?.textContent).toContain('邮件')
  })

  it('skips disabled options with the keyboard and chooses with Enter', async () => {
    const { button, model } = mountSelect('mail')
    await button.trigger('keydown', { key: 'ArrowDown' })
    await button.trigger('keydown', { key: 'ArrowDown' })
    await button.trigger('keydown', { key: 'Enter' })
    expect(model.value).toBe('hook')
    expect(button.attributes('aria-expanded')).toBe('false')
  })

  it('ignores clicks on a disabled option', async () => {
    const { button, model } = mountSelect('')
    await button.trigger('click')
    shown()[2].click()
    await nextTick()
    expect(model.value).toBe('')
  })

  it('type-ahead picks by label on the closed button', async () => {
    const { button, model } = mountSelect('')
    await button.trigger('keydown', { key: 'w' })
    expect(model.value).toBe('hook')
  })
})

describe('tooltip', () => {
  it('shows a v-tip after hover and follows value changes', async () => {
    vi.useFakeTimers()
    installTooltips()
    const reason = ref<string | undefined>('运行中不能删除')
    mount(
      defineComponent({
        setup: () => () => withDirectives(h('button', { disabled: !!reason.value }, '删除'), [[vTip, reason.value]]),
      }),
      { attachTo: document.body },
    )
    const btn = document.querySelector('button')!
    btn.dispatchEvent(new MouseEvent('pointerover', { bubbles: true }))
    vi.advanceTimersByTime(500)
    const tip = document.querySelector<HTMLElement>('.tooltip')!
    expect(tip.hidden).toBe(false)
    expect(tip.textContent).toBe('运行中不能删除')
    expect(btn.getAttribute('aria-describedby')).toBe(tip.id)

    reason.value = undefined
    await nextTick()
    expect(tip.hidden).toBe(true)
    vi.useRealTimers()
  })

  it('labels icon-only buttons from aria-label', () => {
    vi.useFakeTimers()
    installTooltips()
    document.body.innerHTML = '<button class="btn-icon" aria-label="刷新"></button>'
    document.querySelector('button')!.dispatchEvent(new MouseEvent('pointerover', { bubbles: true }))
    vi.advanceTimersByTime(500)
    expect(document.querySelector('.tooltip')?.textContent).toBe('刷新')
    vi.useRealTimers()
  })
})

describe('diffValues', () => {
  it('reports field-level changes', () => {
    const changes = diffValues(
      { name: 'a', retry: { max: 3, backoff: 1 }, tags: ['x'] },
      { name: 'a', retry: { max: 5 }, tags: ['x', 'y'], owner: 'bob' },
    )
    expect(changes.map((c) => [c.path.join('.'), c.kind])).toEqual([
      ['retry.max', 'changed'],
      ['retry.backoff', 'removed'],
      ['tags', 'changed'],
      ['owner', 'added'],
    ])
  })
})

describe('UnitInput', () => {
  function mountUnit(kind: 'duration' | 'bytes', initial: number) {
    const model = ref(initial)
    const w = mount(
      defineComponent({
        setup: () => () =>
          h(UnitInput, {
            kind,
            label: '超时',
            modelValue: model.value,
            'onUpdate:modelValue': (v: number) => (model.value = v),
          }),
      }),
      { attachTo: document.body },
    )
    return { w, model, input: w.find('input'), unit: w.find('button[role="combobox"]') }
  }

  it('shows the largest unit that divides the value', () => {
    const { input, unit } = mountUnit('duration', 900)
    expect((input.element as HTMLInputElement).value).toBe('15')
    expect(unit.text()).toBe('分钟')
    const big = mountUnit('bytes', 64 * 1024 ** 2)
    expect((big.input.element as HTMLInputElement).value).toBe('64')
    expect(big.unit.text()).toBe('MB')
  })

  it('writes base units and keeps the number when the unit changes', async () => {
    const { input, unit, model } = mountUnit('duration', 900)
    await input.setValue('2')
    expect(model.value).toBe(120)
    await unit.trigger('click')
    shown()
      .find((o) => o.textContent?.includes('小时'))!
      .click()
    await nextTick()
    expect(model.value).toBe(7200)
    await input.setValue('')
    expect(model.value).toBe(0)
  })
})
