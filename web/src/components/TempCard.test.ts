import { render, screen, within } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import TempCard from './TempCard.svelte'

describe.each(['Grill', 'Food'] as const)('%s card', (what) => {
  const button = `Set ${what} Temperature`
  const renderCard = (props: Record<string, unknown> = {}) =>
    render(TempCard, {
      title: `${what} Temp ℉`, subtitle: 'x', image: '', what,
      current: 180, desired: 225, enabled: true, min: 150, max: 550, onsubmit: vi.fn(), ...props,
    })

  test('shows current and desired temperatures', () => {
    renderCard()
    expect(screen.getByText('Current: 180 ℉')).toBeInTheDocument()
    expect(screen.getByText('Desired: 225 ℉')).toBeInTheDocument()
  })

  test('shows "Not set" when there is no desired temperature', () => {
    renderCard({ desired: 0 })
    expect(screen.getByText('Desired: Not set')).toBeInTheDocument()
  })

  test('shows "No probe" without a reading', () => {
    renderCard({ current: null })
    expect(screen.getByText('Current: No probe')).toBeInTheDocument()
  })

  test('disables the set button when not enabled', () => {
    renderCard({ enabled: false })
    expect(screen.getByRole('button', { name: button })).toBeDisabled()
  })

  test('submits the entered temperature', async () => {
    const user = userEvent.setup()
    const onsubmit = vi.fn()
    renderCard({ onsubmit })
    await user.click(screen.getByRole('button', { name: button }))
    const dialog = screen.getByRole('dialog', { hidden: true })
    await user.type(within(dialog).getByRole('textbox', { hidden: true }), '250')
    await user.click(within(dialog).getByRole('button', { name: 'Set', hidden: true }))
    expect(onsubmit).toHaveBeenCalledWith('250')
  })

  test.each([
    ['abc', 'Desired temperature must be a number!'],
    ['551', 'Desired temperature must be at most 550 ℉!'],
    ['149', 'Desired temperature must be at least 150 ℉!'],
    ['-5', 'Desired temperature must be at least 150 ℉!'],
    ['225.5', 'Desired temperature must be a whole number!'],
  ])('rejects %s with a validation message and blocks Set', async (input, message) => {
    const user = userEvent.setup()
    const onsubmit = vi.fn()
    renderCard({ onsubmit })
    await user.click(screen.getByRole('button', { name: button }))
    const dialog = screen.getByRole('dialog', { hidden: true })
    await user.type(within(dialog).getByRole('textbox', { hidden: true }), input)
    expect(within(dialog).getByText(message)).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: 'Set', hidden: true })).toBeDisabled()
    await user.keyboard('{Enter}')
    expect(onsubmit).not.toHaveBeenCalled()
  })

  test('uses the limits it is given', async () => {
    const user = userEvent.setup()
    renderCard({ min: 0, max: 300 })
    await user.click(screen.getByRole('button', { name: button }))
    const dialog = screen.getByRole('dialog', { hidden: true })
    await user.type(within(dialog).getByRole('textbox', { hidden: true }), '301')
    expect(within(dialog).getByText('Desired temperature must be at most 300 ℉!')).toBeInTheDocument()
    await user.clear(within(dialog).getByRole('textbox', { hidden: true }))
    await user.type(within(dialog).getByRole('textbox', { hidden: true }), '0')
    expect(within(dialog).getByRole('button', { name: 'Set', hidden: true })).toBeEnabled()
  })

  test('cancel closes the dialog without submitting', async () => {
    const user = userEvent.setup()
    const onsubmit = vi.fn()
    renderCard({ onsubmit })
    await user.click(screen.getByRole('button', { name: button }))
    const dialog = screen.getByRole('dialog', { hidden: true }) as HTMLDialogElement
    expect(dialog.open).toBe(true)
    await user.click(within(dialog).getByRole('button', { name: 'Cancel', hidden: true }))
    expect(dialog.open).toBe(false)
    expect(onsubmit).not.toHaveBeenCalled()
  })
})
