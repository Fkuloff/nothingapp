/**
 * Format a message timestamp for display
 */
export function formatMessageTime(dateString: string): string {
  return new Date(dateString).toLocaleTimeString('ru-RU', {
    hour: '2-digit',
    minute: '2-digit',
  })
}

const LAST_SEEN_PREFIX = 'был(а) в сети'

// Russian plural for "минута": 1 минуту, 2 минуты, 5 минут (with the 11-14 exception).
function pluralMinutes(n: number): string {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return 'минуту'
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return 'минуты'
  return 'минут'
}

/**
 * Format a peer's "last seen" timestamp for the chat header, e.g.
 * "был(а) в сети только что / 5 минут назад / сегодня в 14:03 / вчера в 09:12 / 03.05.2026".
 *
 * Returns '' for missing/unparseable input or the Go zero time
 * ("0001-01-01T00:00:00Z" → negative epoch), so callers can fall back to a
 * plain "Не в сети".
 */
export function formatLastSeen(iso: string | null | undefined): string {
  if (!iso) return ''
  const then = new Date(iso)
  const ts = then.getTime()
  if (Number.isNaN(ts) || ts <= 0) return ''

  const diffMin = Math.floor((Date.now() - ts) / 60000)
  if (diffMin < 1) return `${LAST_SEEN_PREFIX} только что`
  if (diffMin < 60) return `${LAST_SEEN_PREFIX} ${diffMin} ${pluralMinutes(diffMin)} назад`

  const time = then.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })

  const startOfToday = new Date()
  startOfToday.setHours(0, 0, 0, 0)
  if (ts >= startOfToday.getTime()) return `${LAST_SEEN_PREFIX} сегодня в ${time}`
  if (ts >= startOfToday.getTime() - 86400000) return `${LAST_SEEN_PREFIX} вчера в ${time}`

  const date = then.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
  return `${LAST_SEEN_PREFIX} ${date}`
}

