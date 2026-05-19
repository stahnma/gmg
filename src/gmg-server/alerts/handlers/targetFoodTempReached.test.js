const handler = require('./targetFoodTempReached')

afterEach(() => {
  handler.reset()
})

describe('targetFoodTempReached handler', () => {
  test('triggers when food temp reaches target', () => {
    const result = handler.handle({
      isOn: true,
      currentFoodTemp: 165,
      desiredFoodTemp: 165,
    })
    expect(result.triggered).toBeTruthy()
  })

  test('triggers when food temp exceeds target', () => {
    const result = handler.handle({
      isOn: true,
      currentFoodTemp: 170,
      desiredFoodTemp: 165,
    })
    expect(result.triggered).toBeTruthy()
  })

  test('does not trigger when food temp below target', () => {
    const result = handler.handle({
      isOn: true,
      currentFoodTemp: 100,
      desiredFoodTemp: 165,
    })
    expect(result.triggered).toBeFalsy()
  })

  test('does not trigger when grill is off', () => {
    const result = handler.handle({
      isOn: false,
      currentFoodTemp: 200,
      desiredFoodTemp: 165,
    })
    expect(result.triggered).toBeFalsy()
  })

  test('does not trigger when no desired food temp set', () => {
    const result = handler.handle({
      isOn: true,
      currentFoodTemp: 200,
      desiredFoodTemp: 0,
    })
    expect(result.triggered).toBeFalsy()
  })

  test('does not trigger when no current food temp', () => {
    const result = handler.handle({
      isOn: true,
      currentFoodTemp: 0,
      desiredFoodTemp: 165,
    })
    expect(result.triggered).toBeFalsy()
  })

  test('createAlert includes target temperature in reason', () => {
    const result = handler.handle({
      isOn: true,
      currentFoodTemp: 165,
      desiredFoodTemp: 165,
    })
    const alert = result.createAlert()

    expect(alert.type).toBe('targetFoodTempReached')
    expect(alert.reason).toContain('165')
    expect(alert.level).toBe('info')
  })

  test('only fires once for same target (resend interval is MAX_SAFE_INTEGER)', () => {
    handler.handle({
      isOn: true,
      currentFoodTemp: 165,
      desiredFoodTemp: 165,
    })

    const second = handler.handle({
      isOn: true,
      currentFoodTemp: 170,
      desiredFoodTemp: 165,
    })
    expect(second.triggered).toBe(false)
  })
})
