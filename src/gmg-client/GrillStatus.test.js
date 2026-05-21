const GrillStatus = require('./GrillStatus')

// Helper: build a hex string that represents a grill status response.
// The GrillStatus constructor expects a Buffer whose hex encoding is parsed
// at specific character positions.
function buildStatusBuffer(opts = {}) {
  const {
    currentGrillTemp = 225,
    currentFoodTemp = 165,
    desiredGrillTemp = 250,
    desiredFoodTemp = 185,
    state = 1, // 0=off, 1=on, 2=fan mode
    lowPelletAlarm = false,
  } = opts

  // We need a hex string long enough (at least 62 chars to cover position 61).
  // Positions used:
  //   4-7:   current grill temp (little-endian, 2 hex bytes)
  //   8-11:  current food temp
  //  12-15:  desired grill temp
  //  48-51:  low pellet alarm (128 = active)
  //  56-59:  desired food temp
  //  61:     state character
  const hex = new Array(64).fill('0')

  const setLE16 = (pos, value) => {
    const lo = value & 0xff
    const hi = (value >> 8) & 0xff
    hex[pos] = lo.toString(16).padStart(2, '0')[0]
    hex[pos + 1] = lo.toString(16).padStart(2, '0')[1]
    hex[pos + 2] = hi.toString(16).padStart(2, '0')[0]
    hex[pos + 3] = hi.toString(16).padStart(2, '0')[1]
  }

  setLE16(4, currentGrillTemp)
  setLE16(8, currentFoodTemp)
  setLE16(12, desiredGrillTemp)
  setLE16(56, desiredFoodTemp)

  // Low pellet alarm at position 48-51
  setLE16(48, lowPelletAlarm ? 128 : 0)

  // State at position 61
  hex[61] = state.toString()

  const hexStr = hex.join('')
  return Buffer.from(hexStr, 'hex')
}

describe('GrillStatus', () => {
  test('parses grill-on status correctly', () => {
    const buf = buildStatusBuffer({
      currentGrillTemp: 225,
      currentFoodTemp: 165,
      desiredGrillTemp: 250,
      desiredFoodTemp: 185,
      state: 1,
    })
    const status = new GrillStatus(buf)

    expect(status.state).toBe('on')
    expect(status.isOn).toBe(true)
    expect(status.fanModeActive).toBe(false)
    expect(status.currentGrillTemp).toBe(225)
    expect(status.currentFoodTemp).toBe(165)
    expect(status.desiredGrillTemp).toBe(250)
    expect(status.desiredFoodTemp).toBe(185)
    expect(status.lowPelletAlarmActive).toBe(false)
  })

  test('parses grill-off status correctly', () => {
    const buf = buildStatusBuffer({
      currentGrillTemp: 72,
      currentFoodTemp: 0,
      desiredGrillTemp: 200,
      desiredFoodTemp: 150,
      state: 0,
    })
    const status = new GrillStatus(buf)

    expect(status.state).toBe('off')
    expect(status.isOn).toBe(false)
    // When off, desired temps should be 0
    expect(status.desiredGrillTemp).toBe(0)
    expect(status.desiredFoodTemp).toBe(0)
    expect(status.currentGrillTemp).toBe(72)
  })

  test('parses fan mode', () => {
    const buf = buildStatusBuffer({ state: 2 })
    const status = new GrillStatus(buf)

    expect(status.state).toBe('fan mode')
    expect(status.isOn).toBe(false)
    expect(status.fanModeActive).toBe(true)
  })

  test('unknown state for unexpected value', () => {
    const buf = buildStatusBuffer({ state: 9 })
    const status = new GrillStatus(buf)

    expect(status.state).toBe('unknown')
    expect(status.isOn).toBe(false)
  })

  test('low pellet alarm detection', () => {
    const buf = buildStatusBuffer({ lowPelletAlarm: true, state: 1 })
    const status = new GrillStatus(buf)

    expect(status.lowPelletAlarmActive).toBe(true)
  })

  test('food temp >= 557 returns 0', () => {
    const buf = buildStatusBuffer({ currentFoodTemp: 557, state: 1 })
    const status = new GrillStatus(buf)

    expect(status.currentFoodTemp).toBe(0)
  })

  test('stores raw hex representation', () => {
    const buf = buildStatusBuffer({ state: 1 })
    const status = new GrillStatus(buf)

    expect(typeof status._hex).toBe('string')
    expect(status._hex.length).toBeGreaterThan(0)
  })
})
