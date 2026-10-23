import { afterEach, describe, expect, it, vi } from 'vitest'
import { errorMessage, formatTimestamp, timeAgo, timestampToDate } from '@/lib/utils'

afterEach(() => vi.useRealTimers())

describe('formatting utilities', () => {
  it('formats seconds and milliseconds as the same instant', () => {
    const options: Intl.DateTimeFormatOptions = { dateStyle: 'medium', timeZone: 'UTC' }
    expect(formatTimestamp(1_700_000_000, options)).toBe(
      formatTimestamp(1_700_000_000_000, options),
    )
  })

  it('normalizes ISO strings, numeric strings, and Unix epoch zero', () => {
    expect(timestampToDate('2026-08-07T07:56:17Z')?.toISOString()).toBe('2026-08-07T07:56:17.000Z')
    expect(timestampToDate('1700000000')?.getTime()).toBe(1_700_000_000_000)
    expect(formatTimestamp(0, { timeZone: 'UTC', year: 'numeric' })).toBe('1970')
  })

  it('handles invalid and future relative times', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-08T00:00:00Z'))
    expect(timeAgo('not-a-date')).toBe('—')
    expect(timeAgo('2026-08-08T00:01:00Z')).toBe('just now')
    expect(timeAgo('2026-08-07T23:58:00Z')).toBe('2m ago')
  })

  it('uses useful error messages and a safe fallback', () => {
    expect(errorMessage(new Error('Network unavailable'))).toBe('Network unavailable')
    expect(errorMessage({ message: 'not trusted' }, 'Try again later.')).toBe('Try again later.')
  })
})
