import { chromium } from 'playwright'
import { writeFileSync } from 'node:fs'
const base = process.env.E2E_BASE ?? 'http://127.0.0.1:8080'
const shots = process.env.E2E_SHOTS ?? 'shots'
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, acceptDownloads: true })
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(e.message))
let failed = 0
const step = async (name, fn) => {
  try { await fn(); console.log('ok  ', name) }
  catch (e) { failed++; console.log('FAIL', name, e.message.split('\n')[0]); await page.screenshot({ path: `${shots}/fail3-${name.slice(0, 16)}.png` }) }
}
const api = (path, body) => page.evaluate(([p, b]) => fetch(p, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(b) }).then((r) => r.json()), [path, body])

await page.goto(base + '/#/login')
await page.fill('input[autocomplete=username]', 'admin')
await page.fill('input[type=password]', 'password1')
await page.click('button[type=submit]')
await page.waitForURL(/#\/$/)
const ok = (await api('/api/example/create', { name: '订单同步', remark: '## 说明\n每日 **02:00** 执行,详见 https://example.com', enabled: true, owner_id: '0', config: '{"retry":3}' })).item
const bad = (await api('/api/example/create', { name: '库存校准', remark: 'this one will fail', enabled: true, owner_id: '0', config: '' })).item

await step('run streams text and tool calls, then renders the final Markdown', async () => {
  await page.goto(`${base}/#/examples/${ok.id}?tab=runs`)
  await page.click('button.primary:has-text("运行")')
  await page.locator('.md.streaming').waitFor()
  await page.locator('.tool .tool-name', { hasText: 'count_examples' }).waitFor()
  await page.screenshot({ path: `${shots}/30-run-streaming.png` })
  await page.locator('.run-output .pill', { hasText: '成功' }).waitFor({ timeout: 15000 })
  await page.locator('.run-output .md table').waitFor()
  if (await page.locator('.md.streaming').count()) throw new Error('cursor still shown after done')
  await page.locator('.tool').first().click()
  await page.locator('.tool-body').first().waitFor()
  await page.screenshot({ path: `${shots}/31-run-done.png` })
})
await step('reload mid-run resumes the transcript from the server', async () => {
  await page.click('button.primary:has-text("运行")')
  await page.locator('.md.streaming').waitFor()
  await page.waitForTimeout(700)
  await page.reload()
  await page.locator('.run-output .pill', { hasText: '成功' }).waitFor({ timeout: 15000 })
  const text = await page.locator('.run-output .transcript').innerText()
  if (!text.includes('正在检查示例') || !text.includes('检查完成')) throw new Error('replay incomplete')
})
await step('stop ends a run as 已停止', async () => {
  await page.click('button.primary:has-text("运行")')
  await page.locator('.md.streaming').waitFor()
  await page.click('button:has-text("停止")')
  await page.locator('.run-output .pill', { hasText: '已停止' }).waitFor({ timeout: 10000 })
  await page.locator('.transcript-error', { hasText: '已停止' }).waitFor()
})
await step('failing tool opens itself and the run is 失败', async () => {
  await page.goto(`${base}/#/examples/${bad.id}?tab=runs`)
  await page.click('button.primary:has-text("运行")')
  await page.locator('.run-output .pill', { hasText: '失败' }).waitFor({ timeout: 15000 })
  await page.locator('.tool.failed .tool-body', { hasText: '缺少必填项 target' }).waitFor()
  await page.screenshot({ path: `${shots}/32-run-failed.png` })
})
await step('attachments: reject, upload with progress, download, delete', async () => {
  await page.goto(`${base}/#/examples/${ok.id}?tab=files`)
  await page.locator('.upload input[type=file]').setInputFiles({ name: 'x.exe', mimeType: 'application/octet-stream', buffer: Buffer.from('MZ') })
  await page.locator('.toast', { hasText: '无法添加「x.exe」' }).waitFor()
  await page.locator('.upload input[type=file]').setInputFiles([
    { name: '报告.csv', mimeType: 'text/csv', buffer: Buffer.from('a,b\n1,2\n') },
    { name: 'big.txt', mimeType: 'text/plain', buffer: Buffer.alloc(3 << 20, 'x') },
  ])
  await page.locator('tbody tr', { hasText: '报告.csv' }).waitFor()
  await page.locator('tbody tr', { hasText: 'big.txt' }).locator('td', { hasText: '3.0 MB' }).waitFor()
  await page.screenshot({ path: `${shots}/33-attachments.png` })
  const [dl] = await Promise.all([page.waitForEvent('download'), page.locator('tbody tr', { hasText: '报告.csv' }).locator('button:has-text("下载")').click()])
  if (dl.suggestedFilename() !== '报告.csv') throw new Error('filename ' + dl.suggestedFilename())
  await page.locator('tbody tr', { hasText: 'big.txt' }).locator('button:has-text("删除")').click()
  await page.locator('.dialog', { hasText: '确认删除' }).locator('button.danger').click()
  await page.locator('tbody tr', { hasText: 'big.txt' }).waitFor({ state: 'detached' })
})
await step('import wizard: check rows, confirm, report failures by file line', async () => {
  await page.goto(`${base}/#/examples`)
  await page.click('a:has-text("导入")')
  await page.waitForURL(/examples\/import/)
  await page.locator('header .crumb', { hasText: '示例' }).waitFor()
  const csv = '﻿名称,备注,启用,配置\n日志归档,,是,\n订单同步,重名,是,\n报表导出,,否,"{""k"": 1}"\n,空名称,是,\n坏配置,,是,[1]\n日志归档,文件内重复,是,\n'
  await page.locator('.upload input[type=file]').setInputFiles({ name: 'import.csv', mimeType: 'text/csv', buffer: Buffer.from(csv) })
  await page.locator('.alert', { hasText: '共 6 行' }).waitFor()
  await page.click('button.primary:has-text("下一步")')
  await page.locator('.alert', { hasText: '3 行有问题' }).waitFor()
  await page.screenshot({ path: `${shots}/34-import-check.png` })
  await page.click('button.primary:has-text("下一步")')
  await page.click('button.primary:has-text("导入 3 行")')
  await page.locator('.alert', { hasText: '成功 2 行，失败 1 行' }).waitFor()
  await page.locator('tbody tr', { hasText: '名称已存在' }).locator('td.num', { hasText: '3' }).waitFor()
  await page.screenshot({ path: `${shots}/35-import-result.png` })
  if (await page.locator('.step-btn:not([disabled])').count()) throw new Error('steps should lock after import')
})
await step('form: Markdown preview, JSON error after submit, format', async () => {
  await page.goto(`${base}/#/examples?id=${ok.id}`)
  await page.click('.split-detail button:has-text("编辑")')
  await page.fill('.dialog textarea[name=remark]', '# 标题\n- 一\n- 二')
  await page.click('.dialog [role=radio]:has-text("预览")')
  await page.locator('.dialog .md h1', { hasText: '标题' }).waitFor()
  const cfg = page.locator('.dialog textarea[name=config]')
  await cfg.fill('{"a": 1,}')
  await page.click('.dialog-foot button.primary')
  await page.locator('.dialog .field-error', { hasText: 'JSON 格式错误' }).waitFor()
  await page.screenshot({ path: `${shots}/36-form.png` })
  await cfg.fill('{"a":1,"b":[1,2]}')
  await page.click('.dialog button:has-text("格式化")')
  if (!(await cfg.inputValue()).includes('\n  "a": 1')) throw new Error('not formatted')
  await page.click('.dialog-foot button.primary')
  await page.locator('.toast', { hasText: '已保存' }).waitFor()
})
await step('home: charts draw, hover card explains the metric', async () => {
  await page.goto(`${base}/#/`)
  await page.locator('.chart canvas').first().waitFor()
  if ((await page.locator('.chart canvas').count()) < 2) throw new Error('expected two charts')
  await page.hover('button[aria-label=统计口径]')
  await page.locator('[role=tooltip]', { hasText: '今天 0 点以来' }).waitFor()
  await page.screenshot({ path: `${shots}/37-home.png` })
})
const dark = await browser.newContext({ viewport: { width: 1440, height: 900 }, colorScheme: 'dark' })
await dark.addCookies(await ctx.cookies())
const dp = await dark.newPage()
await dp.goto(`${base}/#/examples/${bad.id}?tab=runs`)
await dp.waitForTimeout(1200)
await dp.screenshot({ path: `${shots}/38-dark-run.png` })
await dp.goto(`${base}/#/`)
await dp.waitForTimeout(1500)
await dp.screenshot({ path: `${shots}/39-dark-home.png` })
console.log('page errors:', errors.length ? errors : 'none', '| failed:', failed)
await browser.close()
if (errors.length || failed) process.exit(1)
