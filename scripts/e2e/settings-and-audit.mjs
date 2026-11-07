import { chromium } from 'playwright'
const base = process.env.E2E_BASE ?? 'http://127.0.0.1:8080'
const shots = process.env.E2E_SHOTS ?? 'shots'
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(e.message))
let failed = 0
const step = async (name, fn) => {
  try { await fn(); console.log('ok  ', name) }
  catch (e) { failed++; console.log('FAIL', name, e.message.split('\n')[0]); await page.screenshot({ path: `${shots}/fail4-${name.slice(0, 16)}.png` }) }
}
const api = (path, body) => page.evaluate(([p, b]) => fetch(p, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(b) }).then((r) => r.json()), [path, body])
const toast = (text) => page.locator('.toast', { hasText: text }).waitFor()

async function login(p, user) {
  await p.goto(base + '/#/login')
  await p.fill('input[autocomplete=username]', user)
  await p.fill('input[type=password]', 'password1')
  await p.click('button[type=submit]')
  await p.waitForURL(/#\/$/)
}

await login(page, 'admin')
const ex = (await api('/api/example/create', { name: '订单同步', remark: '', enabled: true, owner_id: '0', config: '' })).item
await api('/api/example/update', { id: ex.id, name: '订单同步 v2', remark: '每日执行', enabled: true, owner_id: '0', config: '' })

await step('settings start from defaults and save as version 1', async () => {
  await page.goto(base + '/#/settings')
  await page.getByText('尚未保存过，当前为默认值').waitFor()
  await page.fill('textarea[name=announcement]', '今晚 **22:00** 维护')
  await page.fill('input[name=run_timeout]', '2')
  await page.click('[role=combobox][aria-label=运行超时单位]')
  await page.click('[role=option]:has-text("小时")')
  await page.click('[role=switch][aria-label=暂停新的运行]')
  await page.locator('.save-bar', { hasText: '有未保存的修改' }).waitFor()
  await page.screenshot({ path: `${shots}/40-settings.png` })
  await page.click('.save-bar button.primary')
  await toast('已保存为版本 1')
  await page.getByText(/^版本 1 · admin 保存于/).waitFor()
})

await step('a stale save shows the diff and can overwrite', async () => {
  // someone else saves version 2 while this form still edits version 1
  await api('/papi/settings/update', {
    base_version: 1,
    settings: { announcement: '由他人修改', run_timeout_seconds: 7200, runs_paused: true, max_attachments: 50 },
    restored_from: 0,
  })
  await page.fill('input[name=max_attachments]', '3')
  await page.click('.save-bar button.primary')
  const dialog = page.locator('.dialog', { hasText: '设置已被其他人修改' })
  await dialog.waitFor()
  await dialog.getByText('由他人修改').waitFor()
  await dialog.getByText('每个示例的附件上限').waitFor()
  await page.waitForTimeout(300)
  await page.screenshot({ path: `${shots}/41-settings-conflict.png` })
  await dialog.locator('button.primary', { hasText: '用我的修改覆盖' }).click()
  await toast('已保存为版本 3')
})

await step('the overview shows the announcement as Markdown', async () => {
  await page.goto(base + '/#/')
  await page.locator('.alert strong', { hasText: '22:00' }).waitFor()
})

await step('paused runs are refused with the reason', async () => {
  await page.goto(`${base}/#/examples/${ex.id}?tab=runs`)
  await page.click('button.primary:has-text("运行")')
  await toast('管理员已暂停新的运行')
})

await step('history compares a version with the one before and restores it', async () => {
  await page.goto(base + '/#/settings?tab=history')
  await page.locator('tbody tr.clickable').nth(2).waitFor()
  await page.click('tbody tr.clickable >> text=版本 1')
  await page.locator('.card', { hasText: '版本 1 的改动' }).getByText('2 小时').waitFor()
  await page.screenshot({ path: `${shots}/42-settings-history.png` })
  await page.click('button:has-text("恢复此版本")')
  await page.click('.dialog button:has-text("恢复")')
  await toast('已恢复版本 1，保存为版本 4')
  await page.getByText('恢复自版本 1').waitFor()
})

await step('audit log filters by action and shows Chinese field names', async () => {
  await page.goto(base + '/#/audit')
  await page.click('[role=combobox][aria-label=操作]')
  await page.click('[role=option]:has-text("修改系统设置")')
  await page.waitForURL(/action=settings\.update/)
  await page.waitForFunction(() => document.querySelectorAll('tbody tr.clickable').length === 4)
  // the oldest entry is the first save, which changed the run timeout
  await page.click('tbody tr.clickable >> nth=3')
  const detail = page.locator('.split-detail')
  await detail.getByText('系统设置 · 版本 1').waitFor()
  await detail.getByText('运行超时').waitFor()
  const text = await detail.innerText()
  if (/run_timeout|runs_paused|max_attachments/.test(text)) throw new Error('raw field name on screen')
  await page.screenshot({ path: `${shots}/43-audit.png` })
})

await step('example 记录 tab links to the audit log with a clearable target chip', async () => {
  await page.goto(`${base}/#/examples/${ex.id}?tab=history`)
  await page.locator('.timeline-title', { hasText: '修改示例' }).waitFor()
  await page.locator('.timeline').getByText('订单同步 v2').waitFor()
  await page.screenshot({ path: `${shots}/44-example-history.png` })
  await page.click('a:has-text("在审计日志中查看")')
  await page.waitForURL(/target_id=/)
  await page.locator('.filter-chip', { hasText: '示例：订单同步 v2' }).waitFor()
  await page.waitForFunction(() => document.querySelectorAll('tbody tr.clickable').length === 2)
  // the header's back link returns to the example's tab, not to the nav list
  const back = await page.locator('.crumb').getAttribute('href')
  if (!back?.includes(`/examples/${ex.id}`)) throw new Error(`back link is ${back}`)
  await page.click('.filter-chip button')
  await page.locator('.filter-chip').waitFor({ state: 'detached' })
})

await step('nav badge counts disabled examples', async () => {
  await api('/api/example/set_enabled', { id: ex.id, enabled: false })
  await page.goto(base + '/#/examples')
  await page.locator('.nav-item', { hasText: '示例' }).locator('.nav-badge', { hasText: '1' }).waitFor()
  // away from the group it collapses, and the group carries the count
  await page.goto(base + '/#/')
  await page.locator('.nav-group .nav-badge', { hasText: '1' }).waitFor()
})

await step('users without the role see neither the entries nor the pages', async () => {
  const other = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  const p = await other.newPage()
  await login(p, 'alice')
  if (await p.locator('.nav-item', { hasText: '审计日志' }).count()) throw new Error('audit entry visible to USER')
  await p.goto(base + '/#/settings')
  await p.waitForURL(/#\/$/)
  await other.close()
})

await step('new pages fit a phone screen', async () => {
  const phone = await browser.newContext({ viewport: { width: 390, height: 844 } })
  await phone.addCookies(await ctx.cookies())
  const p = await phone.newPage()
  for (const [path, shot] of [['/#/settings', '45-settings-narrow'], ['/#/audit', '46-audit-narrow'], [`/#/examples/${ex.id}?tab=history`, '47-history-narrow']]) {
    await p.goto(base + path)
    await p.waitForTimeout(800)
    await p.screenshot({ path: `${shots}/${shot}.png` })
    const overflow = await p.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
    if (overflow > 0) throw new Error(`${path} scrolls sideways by ${overflow}px`)
  }
  await phone.close()
})

console.log('page errors:', errors.length ? errors : 'none', '| failed:', failed)
await browser.close()
if (errors.length || failed) process.exit(1)
