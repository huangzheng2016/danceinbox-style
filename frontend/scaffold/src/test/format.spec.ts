import { describe, expect, it } from 'vitest'
import { readFrom } from '@/base/backLink'
import { errorMessage } from '@/base/errors'
import { fmtLabel, fmtLimit, fmtMs, fmtRatio, labelOptions } from '@/base/format'
import { RUN_STATUS } from '@/base/labels'

describe('formatters own their unit', () => {
  it('renders a missing value as a bare dash', () => {
    expect(fmtMs(null)).toBe('—')
    expect(fmtRatio(undefined)).toBe('—')
    expect(fmtLimit(null)).toBe('—')
  })

  it('formats ratios', () => {
    expect(fmtRatio(0.124)).toBe('12%')
    expect(fmtRatio(0.034)).toBe('3.4%')
    expect(fmtRatio(1)).toBe('100%')
  })

  it('renders unlimited sentinels as 不限', () => {
    expect(fmtLimit(0)).toBe('不限')
    expect(fmtLimit(-1)).toBe('不限')
    expect(fmtLimit(1200)).toBe('1,200')
  })
})

describe('labels', () => {
  it('maps enum codes to text and keeps unknown codes visible', () => {
    expect(fmtLabel(RUN_STATUS, 'failed')).toBe('失败')
    expect(fmtLabel(RUN_STATUS, 'paused')).toBe('paused')
    expect(fmtLabel(RUN_STATUS, '')).toBe('—')
  })

  it('builds filter options from the registry', () => {
    expect(labelOptions(RUN_STATUS)[0]).toEqual({ value: 'queued', label: '排队中' })
  })
})

describe('readFrom', () => {
  it('accepts in-app paths only', () => {
    expect(readFrom({ from: '/examples/1?tab=runs' })).toBe('/examples/1?tab=runs')
    expect(readFrom({ from: '//evil.example' })).toBeUndefined()
    expect(readFrom({ from: 'https://evil.example' })).toBeUndefined()
    expect(readFrom({})).toBeUndefined()
  })
})

describe('errorMessage', () => {
  it('never shows transport text', () => {
    expect(errorMessage(new TypeError('Failed to fetch'))).toBe('网络连接失败')
    expect(errorMessage(new Error('rpc error: code = InvalidArgument desc = 端口无效'))).toBe('端口无效')
    expect(errorMessage({ response: { data: { msg: '名称已存在' } } })).toBe('名称已存在')
  })
})
