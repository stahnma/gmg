import { render, screen, act } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import App from './App.svelte'
import { toasts } from './lib/toasts.svelte'

// Minimal EventSource stand-in that tests can drive.
class FakeEventSource {
  static last: FakeEventSource
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  listeners: Record<string, (e: MessageEvent) => void> = {}
  url: string
  constructor(url: string) {
    this.url = url
    FakeEventSource.last = this
  }
  addEventListener(type: string, fn: (e: MessageEvent) => void) {
    this.listeners[type] = fn
  }
  close() {}
  emit(type: string, data: unknown) {
    this.listeners[type]({ data: JSON.stringify(data) } as MessageEvent)
  }
}

const status = (over = {}) => ({
  state: 'on', isOn: true, currentGrillTemp: 180, desiredGrillTemp: 225, currentFoodTemp: 120,
  desiredFoodTemp: 165, fanModeActive: false, lowPelletAlarmActive: false, ...over,
})

describe('App', () => {
  let play: ReturnType<typeof vi.fn>
  beforeEach(() => {
    toasts.clear()
    vi.stubGlobal('EventSource', FakeEventSource)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => [], text: async () => '' }))
    play = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('Audio', vi.fn().mockImplementation(function (this: { play: typeof play }) { this.play = play }))
  })
  afterEach(() => vi.unstubAllGlobals())

  test('connects to the event stream and shows the connecting overlay until open', async () => {
    render(App)
    expect(FakeEventSource.last.url).toBe('/api/events')
    expect(screen.getByRole('status')).toHaveTextContent('Connecting')
    await act(() => FakeEventSource.last.onopen!())
    expect(screen.queryByText(/Connecting/)).not.toBeInTheDocument()
  })

  test('shows readings from status events and enables controls', async () => {
    render(App)
    await act(() => {
      FakeEventSource.last.onopen!()
      FakeEventSource.last.emit('status', status())
    })
    expect(screen.getByText('Current: 180 ℉')).toBeInTheDocument()
    expect(screen.getByText('Desired: 165 ℉')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Set Grill Temperature' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Power' })).toBeEnabled()
  })

  test('shows "No probe" when the food probe is unplugged', async () => {
    render(App)
    await act(() => {
      FakeEventSource.last.onopen!()
      FakeEventSource.last.emit('status', status({ foodProbeConnected: false, currentFoodTemp: 0 }))
    })
    expect(screen.getByText('Current: No probe')).toBeInTheDocument()
    expect(screen.getByText('Current: 180 ℉')).toBeInTheDocument()
  })

  test('marks readings stale and disables controls when the grill stops responding', async () => {
    render(App)
    const lastSeen = new Date(Date.now() - 3 * 60_000).toISOString()
    await act(() => {
      FakeEventSource.last.onopen!()
      FakeEventSource.last.emit('status', status())
      FakeEventSource.last.emit('connection', { reachable: false, lastSeen, consecutiveFailures: 4, error: 'timeout' })
    })
    expect(screen.getByRole('alert')).toHaveTextContent('Grill not responding.')
    expect(screen.getByRole('alert')).toHaveTextContent('3 min ago')
    expect(screen.getByText('Current: 180 ℉')).toBeInTheDocument() // last known reading stays visible
    expect(screen.getByRole('button', { name: 'Set Grill Temperature' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Power' })).toBeDisabled()
    expect(screen.getByRole('img', { name: 'Grill not connected' })).toBeInTheDocument()

    await act(() => FakeEventSource.last.emit('connection', { reachable: true, lastSeen: new Date().toISOString(), consecutiveFailures: 0 }))
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Set Grill Temperature' })).toBeEnabled()
  })

  test.each([
    ['info', 'Target Food Temperature Reached', 'Food temperature has reached target temperature of 165'],
    ['warning', 'Grill Pellet Alarm', 'The grill is low on pellets.'],
    ['error', 'Grill Error', 'Something went wrong'],
  ])('shows a %s alert as a notification', async (level, name, reason) => {
    render(App)
    await act(() => FakeEventSource.last.emit('alert', { type: 'x', name, reason, level }))
    expect(await screen.findByText(`${name}: ${reason}`)).toBeInTheDocument()
    expect(toasts.visible[0].level).toBe(level)
  })

  test('plays the alert sound when one is sent, and not otherwise', async () => {
    render(App)
    await act(() => FakeEventSource.last.emit('alert', { type: 'x', name: 'n', reason: 'r', level: 'info', beep: 'alerts/x.mp3' }))
    expect(Audio).toHaveBeenCalledWith(`${window.location.origin}/alerts/x.mp3`)
    expect(play).toHaveBeenCalledTimes(1)
    await act(() => FakeEventSource.last.emit('alert', { type: 'x', name: 'n', reason: 'r', level: 'info' }))
    expect(play).toHaveBeenCalledTimes(1)
  })

  test('a failed command shows the server error', async () => {
    render(App)
    await act(() => {
      FakeEventSource.last.onopen!()
      FakeEventSource.last.emit('status', status({ isOn: false }))
    })
    vi.mocked(fetch).mockResolvedValue({ ok: false, text: async () => 'No response from Grill' } as Response)
    await userEvent.setup().click(screen.getByRole('button', { name: 'Power' }))
    expect(await screen.findByText('No response from Grill')).toBeInTheDocument()
  })

  test('toggles the timers card', async () => {
    render(App)
    await act(() => {
      FakeEventSource.last.onopen!()
      FakeEventSource.last.emit('status', status())
    })
    expect(screen.queryByText('Set a grilling stopwatch or countdown timer.')).not.toBeInTheDocument()
    await userEvent.setup().click(screen.getByRole('button', { name: 'Timers' }))
    expect(screen.getByText('Set a grilling stopwatch or countdown timer.')).toBeInTheDocument()
  })
})
