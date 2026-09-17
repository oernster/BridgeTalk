// The Chatter pane: finding a category in a list too long to scroll through. A name in the header
// moves the list to its category, a heading collapses its group and each carries a mark saying it
// can be pressed. None of it changes a moment. The fake application is chatterFixtures.tsx.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { categories, docked, resetFake, setCategory, setMoment, shown, undocked } from
  './chatterFixtures'

vi.mock('./api', async () => ({ api: (await import('./chatterFixtures')).fakeApi }))

beforeEach(resetFake)

describe('finding a category on the chatter pane', () => {
  // FR-744: a heading collapses its group and opens it again, still counting, changing no moment.
  it('collapses a category from its heading and opens it again', async () => {
    await shown()
    const heading = screen.getByRole('button', { name: 'Docking and stations (2 of 2 on)' })
    expect(heading.getAttribute('aria-expanded')).toBe('true')

    fireEvent.click(heading)
    const group = within(screen.getByRole('region', { name: 'Docking and stations' }))
    expect(group.queryAllByRole('switch')).toEqual([])
    expect(heading.getAttribute('aria-expanded')).toBe('false')
    expect(screen.getByRole('button', { name: 'Docking and stations (2 of 2 on)' })).toBe(heading)

    fireEvent.click(heading)
    expect(group.getAllByRole('switch').map((each) => each.getAttribute('aria-label'))).toEqual([
      docked.title,
      undocked.title,
    ])
    expect(setMoment).not.toHaveBeenCalled()
    expect(setCategory).not.toHaveBeenCalled()
  })

  // FR-755: a heading's mark says whether its group is open; a name in the header carries an arrow.
  it('marks every heading open or shut and every name in the header with an arrow', async () => {
    await shown()

    for (const category of categories) {
      const move = screen.getByRole('button', { name: `Move to ${category.name}` })
      expect(move.querySelectorAll('svg.mark[aria-hidden="true"]')).toHaveLength(1)
    }
    const heading = screen.getByRole('button', { name: 'Session (2 of 2 on)' })
    const mark = () => heading.querySelector('svg.mark[aria-hidden="true"]')
    expect(mark()?.classList.contains('shut')).toBe(false)

    fireEvent.click(heading)
    expect(mark()?.classList.contains('shut')).toBe(true)

    fireEvent.click(heading)
    expect(mark()?.classList.contains('shut')).toBe(false)
  })

  // FR-743: a category's name in the header moves the list to it, opening it, changing no moment.
  it('moves the list to a category named in the header, opening it', async () => {
    await shown()
    const list = document.querySelector('.chatter-list') as HTMLElement
    const session = screen.getByRole('region', { name: 'Session' })
    list.getBoundingClientRect = () => ({ top: 100 }) as DOMRect
    session.getBoundingClientRect = () => ({ top: 700 }) as DOMRect
    fireEvent.click(screen.getByRole('button', { name: 'Session (2 of 2 on)' }))
    expect(within(session).queryAllByRole('switch')).toEqual([])

    fireEvent.click(screen.getByRole('button', { name: 'Move to Session' }))

    await waitFor(() => expect(list.scrollTop).toBe(600))
    expect(within(session).getAllByRole('switch')).toHaveLength(2)
    expect(setMoment).not.toHaveBeenCalled()
    expect(setCategory).not.toHaveBeenCalled()
  })
})
