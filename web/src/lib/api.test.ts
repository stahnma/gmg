import { GrillClient } from './api'

const ok = (json?: unknown) => ({ ok: true, json: vi.fn().mockResolvedValue(json), text: vi.fn() })
const fail = (body: string) => ({ ok: false, text: vi.fn().mockResolvedValue(body) })

describe('GrillClient', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  test.each([
    ['http://grill.local/', 'http://grill.local'],
    ['http://grill.local\\', 'http://grill.local'],
    ['http://grill.local', 'http://grill.local'],
  ])('normalizes base address %s', (input, expected) => {
    expect(new GrillClient(input).base).toBe(expected)
  })

  test.each([
    ['powerToggle', [], '/api/powertoggle'],
    ['setDesiredGrillTemp', [225], '/api/temperature/grill/225'],
    ['setDesiredFoodTemp', [165], '/api/temperature/food/165'],
  ] as const)('%s sends PUT to %s', async (method, args, path) => {
    vi.mocked(fetch).mockResolvedValue(ok() as unknown as Response)
    const client = new GrillClient('http://grill.local/')
    await (client[method] as (...a: unknown[]) => Promise<void>)(...args)
    expect(fetch).toHaveBeenCalledWith(`http://grill.local${path}`, { method: 'PUT' })
  })

  test.each([
    ['powerToggle', []],
    ['setDesiredGrillTemp', [225]],
    ['setDesiredFoodTemp', [165]],
    ['getTemperatureHistory', [0]],
    ['getLimits', []],
  ] as const)('%s throws the response body when the request fails', async (method, args) => {
    vi.mocked(fetch).mockResolvedValue(fail('Cannot set grill temperature when the grill is off!') as unknown as Response)
    const client = new GrillClient('http://grill.local')
    await expect((client[method] as (...a: unknown[]) => Promise<unknown>)(...args)).rejects.toThrow(
      'Cannot set grill temperature when the grill is off!',
    )
  })

  test('getTemperatureHistory requests rows since the given epoch', async () => {
    const rows = [{ temperature_log_id: 1, timestamp: 1700000000, grill_temperature: 225, food_temperature: 150 }]
    vi.mocked(fetch).mockResolvedValue(ok(rows) as unknown as Response)
    const result = await new GrillClient('http://grill.local').getTemperatureHistory(1700000000)
    expect(fetch).toHaveBeenCalledWith('http://grill.local/api/temperature/history?since=1700000000')
    expect(result).toEqual(rows)
  })
})
