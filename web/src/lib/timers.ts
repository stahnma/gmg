const SECONDS_IN_DAY = 86400
const SECONDS_IN_HOUR = 3600
const SECONDS_IN_MIN = 60

/**
 * Parses a countdown: either a number of seconds ("90") or dd:hh:mm:ss
 * ("00:01:30:00"). Throws on anything else or on zero.
 */
export function parseCountdown(input: string): number {
  const text = String(input).trim()
  let result = 0
  if (!text.includes(':') && /^\d+$/.test(text)) {
    result = parseInt(text, 10)
  } else {
    const m = /^(\d{1,2}):(\d{1,2}):(\d{1,2}):(\d{1,2})$/.exec(text)
    if (m) {
      const [d, h, min, s] = m.slice(1).map(Number)
      result = d * SECONDS_IN_DAY + h * SECONDS_IN_HOUR + min * SECONDS_IN_MIN + s
    }
  }
  if (!result) throw new Error('Input could not be parsed!')
  return result
}

/** Formats seconds as dd:hh:mm:ss. */
export function formatSeconds(total: number): string {
  let s = Math.max(0, Math.trunc(total))
  const days = Math.trunc(s / SECONDS_IN_DAY)
  s -= days * SECONDS_IN_DAY
  const hours = Math.trunc(s / SECONDS_IN_HOUR)
  s -= hours * SECONDS_IN_HOUR
  const mins = Math.trunc(s / SECONDS_IN_MIN)
  s -= mins * SECONDS_IN_MIN
  return [days, hours, mins, s].map((n) => String(n).padStart(2, '0')).join(':')
}
