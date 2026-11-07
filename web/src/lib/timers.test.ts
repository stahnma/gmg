import { formatSeconds, parseCountdown } from './timers'

describe('parseCountdown', () => {
  test.each([
    ['90', 90],
    ['00:00:01:30', 90],
    ['00:01:30:00', 5400],
    ['1:2:3:4', 86400 + 7200 + 180 + 4],
  ])('%s -> %i seconds', (input, want) => expect(parseCountdown(input)).toBe(want))

  test.each(['', 'abc', '00:00:00:00', '0', '1:30', '00:00:00:00:00'])('rejects %j', (input) => {
    expect(() => parseCountdown(input)).toThrow()
  })
})

describe('formatSeconds', () => {
  test.each([
    [0, '00:00:00:00'],
    [59, '00:00:00:59'],
    [5400, '00:01:30:00'],
    [90061, '01:01:01:01'],
  ])('%i -> %s', (input, want) => expect(formatSeconds(input)).toBe(want))
})
