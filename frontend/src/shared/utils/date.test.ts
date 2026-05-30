import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { formatLastSeen } from './date'

// Fix "now" to a mid-afternoon instant so day-boundary math never lands near
// midnight. Local time — the formatter renders in the machine's timezone.
const NOW = new Date(2026, 4, 29, 14, 0, 0)

function minutesAgo(n: number): string {
  return new Date(NOW.getTime() - n * 60_000).toISOString()
}

describe('formatLastSeen', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(NOW)
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns empty string for missing input', () => {
    expect(formatLastSeen('')).toBe('')
    expect(formatLastSeen(null)).toBe('')
    expect(formatLastSeen(undefined)).toBe('')
  })

  it('returns empty string for the Go zero time', () => {
    expect(formatLastSeen('0001-01-01T00:00:00Z')).toBe('')
  })

  it('returns empty string for an unparseable value', () => {
    expect(formatLastSeen('not-a-date')).toBe('')
  })

  it('shows "только что" under a minute', () => {
    expect(formatLastSeen(minutesAgo(0))).toBe('был(а) в сети только что')
  })

  it('pluralises minutes the Russian way', () => {
    expect(formatLastSeen(minutesAgo(1))).toBe('был(а) в сети 1 минуту назад')
    expect(formatLastSeen(minutesAgo(2))).toBe('был(а) в сети 2 минуты назад')
    expect(formatLastSeen(minutesAgo(5))).toBe('был(а) в сети 5 минут назад')
    expect(formatLastSeen(minutesAgo(11))).toBe('был(а) в сети 11 минут назад')
    expect(formatLastSeen(minutesAgo(21))).toBe('был(а) в сети 21 минуту назад')
    expect(formatLastSeen(minutesAgo(22))).toBe('был(а) в сети 22 минуты назад')
  })

  it('shows "сегодня в HH:MM" earlier the same day', () => {
    const earlierToday = new Date(2026, 4, 29, 9, 30, 0).toISOString()
    expect(formatLastSeen(earlierToday)).toMatch(/^был\(а\) в сети сегодня в \d{2}:\d{2}$/)
  })

  it('shows "вчера в HH:MM" the previous day', () => {
    const yesterday = new Date(2026, 4, 28, 23, 0, 0).toISOString()
    expect(formatLastSeen(yesterday)).toMatch(/^был\(а\) в сети вчера в \d{2}:\d{2}$/)
  })

  it('shows a DD.MM.YYYY date for older timestamps', () => {
    const lastWeek = new Date(2026, 4, 20, 12, 0, 0).toISOString()
    expect(formatLastSeen(lastWeek)).toMatch(/^был\(а\) в сети \d{2}\.\d{2}\.\d{4}$/)
  })
})
