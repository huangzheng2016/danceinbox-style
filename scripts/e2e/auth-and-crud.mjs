import { chromium } from 'playwright'
const base = process.env.E2E_BASE ?? 'http://127.0.0.1:8080'
const shots = process.env.E2E_SHOTS ?? 'shots'
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(e.message))
let failed = 0
const step = async (name, fn) => { try { await fn(); console.log('ok  ', name) } catch (e) { failed++; console.log('FAIL', name, e.message.split('\n')[0]); await page.screenshot({ path: `${shots}/fail-${name.slice(0, 16)}.png` }) } }

await step('redirects to login with next', async () => {
  await page.goto(base + '/#/examples')
  await page.waitForURL(/#\/login\?next=/)
})
await step('logs in and returns to next', async () => {
  await page.fill('input[autocomplete=username]', 'admin')
  await page.fill('input[type=password]', 'password1')
  await page.click('button[type=submit]')
  await page.waitForURL(/#\/examples/)
  await page.getByText('暂无示例').waitFor()
})
await step('creates via dialog, shows toast, row appears', async () => {
  await page.click('button.primary:has-text("新建")')
  await page.fill('.dialog input', '订单同步')
  await page.fill('.dialog textarea', '每日 02:00')
  await page.click('.dialog-foot button.primary')
  await page.getByText('已创建').waitFor()
  await page.locator('tbody tr.clickable', { hasText: '订单同步' }).waitFor()
})
await step('duplicate name surfaces backend message', async () => {
  await page.click('button.primary:has-text("新建")')
  await page.fill('.dialog input', '订单同步')
  await page.click('.dialog-foot button.primary')
  await page.getByText('创建失败:名称已存在').waitFor()
  await page.keyboard.press('Escape')
  await page.locator('.dialog', { hasText: '放弃修改' }).locator('button.danger').click()
  await page.waitForFunction(() => document.querySelectorAll('.dialog').length === 0)
})
await step('row click opens detail and puts id in URL', async () => {
  await page.click('tbody tr.clickable >> text=订单同步')
  await page.waitForURL(/id=\d+/)
  await page.locator('.split-detail h2', { hasText: '订单同步' }).waitFor()
})
await step('delete is disabled while enabled', async () => {
  await page.click('td.row-actions [aria-haspopup=menu]')
  const del = page.locator('[role=menuitem]', { hasText: '删除' })
  if (!(await del.isDisabled())) throw new Error('delete should be disabled')
  if ((await del.getAttribute('title')) !== '请先停用再删除') throw new Error('missing reason')
  await page.keyboard.press('Escape')
})
await step('switch disables the row', async () => {
  await page.click('tbody tr.clickable [role=switch]')
  await page.locator('.toast', { hasText: '已停用' }).waitFor()
  await page.waitForFunction(() => document.querySelector('tbody [role=switch]')?.getAttribute('aria-checked') === 'false')
})
await step('confirmed delete removes row and clears selection', async () => {
  await page.click('td.row-actions [aria-haspopup=menu]')
  await page.click('[role=menuitem]:has-text("删除")')
  await page.click('.dialog-foot button.danger')
  await page.getByText('已删除').waitFor()
  await page.getByText('暂无示例').waitFor()
  if (page.url().includes('id=')) throw new Error('selection not cleared')
})
await step('filters survive reload', async () => {
  await page.click('[role=radio]:has-text("已停用")')
  await page.waitForURL(/status=disabled/)
  await page.reload()
  await page.locator('[role=radio][aria-checked=true]', { hasText: '已停用' }).waitFor()
})
await step('expired session redirects to login', async () => {
  await ctx.clearCookies()
  await page.click('button[aria-label="刷新"]')
  await page.waitForURL(/#\/login\?next=/)
})
await step('logout via user menu', async () => {
  await page.fill('input[autocomplete=username]', 'admin')
  await page.fill('input[type=password]', 'password1')
  await page.click('button[type=submit]')
  await page.waitForURL(/#\/examples/)
  await page.click('header [aria-haspopup=menu]')
  await page.click('[role=menuitem]:has-text("退出登录")')
  await page.waitForURL(/#\/login/)
  await page.goto(base + '/#/')
  await page.waitForURL(/#\/login/)
})
console.log('page errors:', errors.length ? errors : 'none')
await browser.close()
if (errors.length || failed) process.exit(1)
