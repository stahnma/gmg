const handler = require('./targetGrillTempReached')

afterEach(() => {
  handler.reset()
})

describe('targetGrillTempReached handler', () => {
  test('triggers when grill temp reaches target', () => {
    const result = handler.handle({
      isOn: true,
      currentGrillTemp: 250,
      desiredGrillTemp: 250,
    })
    expect(result.triggered).toBe(true)
  })

  test('triggers when grill temp exceeds target', () => {
    const result = handler.handle({
      isOn: true,
      currentGrillTemp: 260,
      desiredGrillTemp: 250,
    })
    expect(result.triggered).toBe(true)
  })

  test('does not trigger when grill temp below target', () => {
    const result = handler.handle({
      isOn: true,
      currentGrillTemp: 200,
      desiredGrillTemp: 250,
    })
    expect(result.triggered).toBe(false)
  })

  test('does not trigger when grill is off', () => {
    const result = handler.handle({
      isOn: false,
      currentGrillTemp: 300,
      desiredGrillTemp: 250,
    })
    expect(result.triggered).toBe(false)
  })

  test('createAlert includes target temperature', () => {
    const result = handler.handle({
      isOn: true,
      currentGrillTemp: 250,
      desiredGrillTemp: 250,
    })
    const alert = result.createAlert()

    expect(alert.type).toBe('targetGrillTempReached')
    expect(alert.reason).toContain('250')
    expect(alert.level).toBe('info')
  })

  test('only fires once per target (effectively no resend)', () => {
    handler.handle({
      isOn: true,
      currentGrillTemp: 250,
      desiredGrillTemp: 250,
    })

    const second = handler.handle({
      isOn: true,
      currentGrillTemp: 255,
      desiredGrillTemp: 250,
    })
    expect(second.triggered).toBe(false)
  })

  test('fires again if target changes', () => {
    handler.handle({
      isOn: true,
      currentGrillTemp: 250,
      desiredGrillTemp: 250,
    })

    const second = handler.handle({
      isOn: true,
      currentGrillTemp: 300,
      desiredGrillTemp: 300,
    })
    expect(second.triggered).toBe(true)
  })
})
