import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import ExampleList from '@/pages/ExampleList.vue'
import ConfirmDialog from '@/base/ConfirmDialog.vue'
import { toasts } from '@/base/toast'
import * as api from '@/api/example'

const row = (id: string, name: string, status: 'enabled' | 'disabled') => ({
  id,
  name,
  status,
  remark: '',
  owner_id: '0',
  owner_name: '',
  created_at: 1761456000,
  updated_at: 1761456000,
})

vi.mock('@/api/example', () => ({
  listExamples: vi.fn(async () => ({
    items: [row('9007199254740993', '甲', 'disabled'), row('9007199254740994', '乙', 'enabled')],
    total: 2,
  })),
  getExample: vi.fn(async (id: string) => row(id, '甲', 'disabled')),
  deleteExample: vi.fn(async () => {}),
  setExampleEnabled: vi.fn(async () => {}),
  batchExamples: vi.fn(async () => ({ code: 0, done: 1, skipped: 1 })),
  createExample: vi.fn(),
  updateExample: vi.fn(),
}))

async function render(path = '/') {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: ExampleList }] })
  await router.push(path)
  const w = mount(
    { template: '<ExampleList /><ConfirmDialog />', components: { ExampleList, ConfirmDialog } },
    { global: { plugins: [router] }, attachTo: document.body },
  )
  await flushPromises()
  return { w, router }
}

const dialogButton = (text: string) =>
  [...document.querySelectorAll<HTMLButtonElement>('.dialog-foot button')].find((b) => b.textContent?.trim() === text)!

describe('ExampleList', () => {
  it('selects a row into the URL and loads its detail', async () => {
    const { w, router } = await render()
    await w.find('tbody tr.clickable').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.id).toBe('9007199254740993')
    expect(api.getExample).toHaveBeenCalledWith('9007199254740993', expect.any(AbortSignal))
    expect(w.find('.split').classes()).toContain('has-detail')
    expect(w.find('.split-detail h2').text()).toBe('甲')
  })

  it('deletes only after confirmation, which focuses 取消', async () => {
    const { w } = await render()
    await w.find('td.row-actions [aria-haspopup="menu"]').trigger('click')
    const del = [...document.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')].find((b) =>
      b.textContent?.includes('删除'),
    )!
    del.click()
    await flushPromises()
    expect(api.deleteExample).not.toHaveBeenCalled()
    expect(document.activeElement?.textContent?.trim()).toBe('取消')
    dialogButton('删除').click()
    await flushPromises()
    expect(api.deleteExample).toHaveBeenCalledWith('9007199254740993')
  })

  it('sorting goes to the URL and the request', async () => {
    const { w, router } = await render()
    await w.find('th[aria-sort] button').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.sort).toBe('name')
    expect(api.listExamples).toHaveBeenLastCalledWith(
      expect.objectContaining({ sort: 'name', page: 1 }),
      expect.any(AbortSignal),
    )
  })

  it('batch delete: bar replaces the toolbar, confirms, reports skipped rows', async () => {
    const { w } = await render()
    const checks = w.findAll('tbody td.col-check input')
    await checks[0].setValue(true)
    await checks[1].setValue(true)
    expect(w.find('.toolbar.batch').text()).toContain('已选 2 项')
    const del = w.findAll('.toolbar.batch button').find((b) => b.text() === '删除')!
    await del.trigger('click')
    await flushPromises()
    dialogButton('删除').click()
    await flushPromises()
    expect(api.batchExamples).toHaveBeenCalledWith(['9007199254740993', '9007199254740994'], 'delete')
    expect(toasts.value.at(-1)?.text).toBe('已删除 1 项，跳过 1 项')
    expect(w.find('.toolbar.batch').exists()).toBe(false)
  })
})
