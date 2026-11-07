import { render, screen } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import Toolbar from './Toolbar.svelte'

const power = () => screen.getByRole('button', { name: 'Power' })

describe('Toolbar', () => {
  test('power is disabled when the grill is not connected', () => {
    render(Toolbar, { grillConnected: false, loading: false })
    expect(power()).toBeDisabled()
  })

  test('power is disabled in fan mode', () => {
    render(Toolbar, { grillConnected: true, fanModeActive: true, loading: false })
    expect(power()).toBeDisabled()
    expect(screen.getByRole('img', { name: 'Fan mode is active!' })).toBeInTheDocument()
  })

  test('power calls onpower when connected', async () => {
    const onpower = vi.fn()
    render(Toolbar, { grillConnected: true, loading: false, onpower })
    await userEvent.setup().click(power())
    expect(onpower).toHaveBeenCalledTimes(1)
  })

  test('shows a progress bar while loading', () => {
    render(Toolbar, { grillConnected: true, loading: true })
    expect(screen.getByRole('progressbar')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Timers' })).toBeDisabled()
  })

  test('no progress bar when idle', () => {
    render(Toolbar, { grillConnected: true, loading: false })
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument()
  })

  test('reports connection and low pellets', () => {
    render(Toolbar, { grillConnected: true, lowPelletAlarmActive: true, loading: false })
    expect(screen.getByRole('img', { name: 'Grill connected' })).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Low Pellet alarm is active!' })).toBeInTheDocument()
  })
})
