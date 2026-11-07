import { toasts } from './toasts.svelte'

describe('toasts', () => {
  beforeEach(() => toasts.clear())

  test('shows at most three, queueing the rest', () => {
    for (const m of ['a', 'b', 'c', 'd']) toasts.info(m)
    expect(toasts.visible.map((t) => t.message)).toEqual(['a', 'b', 'c'])
    toasts.dismiss(toasts.visible[0].id)
    expect(toasts.visible.map((t) => t.message)).toEqual(['b', 'c', 'd'])
  })

  test('records the level', () => {
    toasts.warning('w')
    toasts.error('e')
    expect(toasts.visible.map((t) => t.level)).toEqual(['warning', 'error'])
  })
})
