import { render, screen } from '@testing-library/svelte'
import BottomNav from './BottomNav.svelte'

test('links to the API documentation', () => {
  render(BottomNav)
  expect(screen.getByRole('link', { name: 'API' })).toHaveAttribute('href', '/api')
})
