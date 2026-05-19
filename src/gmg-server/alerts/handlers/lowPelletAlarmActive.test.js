const handler = require('./lowPelletAlarmActive')

afterEach(() => {
  handler.reset()
})

describe('lowPelletAlarmActive handler', () => {
  test('triggers when low pellet alarm is active', () => {
    const result = handler.handle({ lowPelletAlarmActive: true })
    expect(result.triggered).toBe(true)
  })

  test('does not trigger when alarm is inactive', () => {
    const result = handler.handle({ lowPelletAlarmActive: false })
    expect(result.triggered).toBe(false)
  })

  test('createAlert returns correct alert object', () => {
    const result = handler.handle({ lowPelletAlarmActive: true })
    const alert = result.createAlert()

    expect(alert.type).toBe('lowPelletAlarmActive')
    expect(alert.name).toBe('Grill Pellet Alarm')
    expect(alert.level).toBe('warning')
    expect(alert.reason).toContain('low on pellets')
    expect(alert.beep).toContain('.mp3')
  })

  test('suppresses duplicate alerts within resend interval', () => {
    // First trigger
    const first = handler.handle({ lowPelletAlarmActive: true })
    expect(first.triggered).toBe(true)

    // Immediate second call with same state — should be suppressed
    const second = handler.handle({ lowPelletAlarmActive: true })
    expect(second.triggered).toBe(false)
  })

  test('reset clears state so it can trigger again', () => {
    handler.handle({ lowPelletAlarmActive: true })
    handler.reset()

    const result = handler.handle({ lowPelletAlarmActive: true })
    expect(result.triggered).toBe(true)
  })

  test('has a name derived from filename', () => {
    expect(handler.name).toBe('lowPelletAlarmActive.js')
  })
})
