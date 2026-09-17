// The Chatter pane: which moments are spoken for.
//
// The switches live in the application, so the fake in chatterFixtures.tsx plays its part: it holds the
// switches and answers every press with the pane as it then stands. What these guard is that the pane
// lists and counts what it is given, that each press asks for exactly the change it names and that a
// press changing more than one moment changes nothing until the question is answered. Finding a
// category, by moving to it, collapsing it and the marks saying so, is chatter.find.test.tsx.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { refuses } from './testRefusal'
import {
  categories,
  checked,
  docked,
  every,
  held,
  resetFake,
  setAllMoments,
  setCategory,
  setMoment,
  shown,
  switchNamed,
} from './chatterFixtures'

vi.mock('./api', async () => ({ api: (await import('./chatterFixtures')).fakeApi }))

const { ChatterPane } = await import('./chatter')

beforeEach(resetFake)

describe('the chatter pane', () => {
  // FR-727.
  it('lists each moment under its category with its purpose and a switch', async () => {
    await shown()

    const heading = screen.getByRole('heading', { name: 'Docking and stations (2 of 2 on)' })
    const title = screen.getByText(docked.title)
    const purpose = screen.getByText(docked.purpose)
    expect(purpose.className).toBe('purpose')
    expect(heading.compareDocumentPosition(title) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    const row = title.closest('.row') as HTMLElement
    expect(within(row).getByRole('switch', { name: 'Docked' })).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Session (2 of 2 on)' })).toBeTruthy()
  })

  // FR-740: the header holds the two buttons and every category's switch in order; the list holds none.
  it('holds a switch for each category in the header and none in the list', async () => {
    await shown()

    const head = within(document.querySelector('.chatter-head') as HTMLElement)
    expect(head.getByRole('button', { name: 'Switch all on' })).toBeTruthy()
    expect(head.getByRole('button', { name: 'Switch all off' })).toBeTruthy()
    expect(head.getAllByRole('switch').map((each) => each.getAttribute('aria-label'))).toEqual([
      'Docking and stations',
      'Session',
    ])
    const list = within(document.querySelector('.chatter-list') as HTMLElement)
    for (const category of categories) {
      expect(list.queryByRole('switch', { name: category.name })).toBeNull()
    }
  })

  // FR-741: every moment is drawn inside its own category's group, headed by that category.
  it("keeps each moment inside its category's group", async () => {
    await shown()

    for (const category of categories) {
      const group = within(screen.getByRole('region', { name: category.name }))
      expect(group.getByRole('heading', { name: new RegExp(`^${category.name} \\(`) })).toBeTruthy()
      expect(group.getAllByRole('switch').map((each) => each.getAttribute('aria-label'))).toEqual(
        category.moments.map((cue) => cue.title),
      )
    }
  })

  // FR-752 in the markup: the moments stand together in one grid after the heading, in order. The three
  // columns are the style sheet's, which jsdom does not compute.
  it("holds each category's moments together beneath its heading", async () => {
    await shown()

    for (const category of categories) {
      const group = screen.getByRole('region', { name: category.name })
      const grid = group.querySelector(':scope > .chatter-moments') as HTMLElement
      expect(grid).not.toBeNull()
      expect(group.querySelector(':scope > h3')?.nextElementSibling).toBe(grid)
      const named = Array.from(grid.children).map((row) =>
        within(row as HTMLElement).getByRole('switch').getAttribute('aria-label'),
      )
      expect(named).toEqual(category.moments.map((cue) => cue.title))
    }
  })

  // FR-754 in the markup: each heading's button is its pill. How the pill is drawn is the style
  // sheet's, which jsdom does not compute.
  it('draws each category heading as a pill', async () => {
    await shown()

    for (const category of categories) {
      const group = within(screen.getByRole('region', { name: category.name }))
      const heading = group.getByRole('heading', { name: new RegExp(`^${category.name} \\(`) })
      expect(within(heading).getByRole('button').classList.contains('heading-pill')).toBe(true)
    }
  })

  // FR-728.
  it('counts the moments switched on under each heading', async () => {
    held.off = new Set(['Docked'])
    await shown()

    expect(screen.getByRole('heading', { name: 'Docking and stations (1 of 2 on)' })).toBeTruthy()
  })

  // FR-729.
  it('turns a moment off then on again from its switch', async () => {
    await shown()

    fireEvent.click(switchNamed('Docked'))
    await waitFor(() => expect(checked('Docked')).toBe('false'))
    expect(setMoment).toHaveBeenLastCalledWith('Docked', false, expect.any(Function))

    fireEvent.click(switchNamed('Docked'))
    await waitFor(() => expect(checked('Docked')).toBe('true'))
    expect(setMoment).toHaveBeenLastCalledWith('Docked', true, expect.any(Function))
  })

  // FR-730.
  it('reads a category on while any moment in it is on', async () => {
    held.off = new Set(['LoadGame'])
    const { unmount } = render(<ChatterPane />)
    await screen.findByRole('switch', { name: 'Session' })
    expect(checked('Session')).toBe('true')
    unmount()

    held.off = new Set(['LoadGame', 'Shutdown'])
    await shown()
    expect(checked('Session')).toBe('false')
  })

  // FR-731 and FR-733: two moments change, so the question comes first.
  it('turns every moment in a category off from its heading', async () => {
    await shown()

    fireEvent.click(switchNamed('Session'))
    const question = await screen.findByRole('dialog', { name: 'Switch 2 moments off' })
    fireEvent.click(within(question).getByRole('button', { name: 'Switch 2 off' }))

    await waitFor(() => expect(checked('Session')).toBe('false'))
    expect(setCategory).toHaveBeenCalledWith('Session', false, expect.any(Function))
    expect(checked('Load game')).toBe('false')
    expect(checked('Shutdown')).toBe('false')
    expect(checked('Docked')).toBe('true')
  })

  // FR-733 asks only where more than one moment would change.
  it('changes a category with one moment on without asking', async () => {
    held.off = new Set(['LoadGame'])
    await shown()

    fireEvent.click(switchNamed('Session'))

    await waitFor(() => expect(checked('Session')).toBe('false'))
    expect(setCategory).toHaveBeenCalledWith('Session', false, expect.any(Function))
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  // FR-732.
  it('switches every moment on or off from the two buttons', async () => {
    held.off = new Set(['Docked', 'LoadGame'])
    await shown()

    fireEvent.click(screen.getByRole('button', { name: 'Switch all on' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Switch 2 on' }))
    await waitFor(() => expect(checked('Docked')).toBe('true'))
    expect(every.every((id) => !held.off.has(id))).toBe(true)

    fireEvent.click(screen.getByRole('button', { name: 'Switch all off' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Switch 4 off' }))
    await waitFor(() => expect(checked('Docked')).toBe('false'))
    expect(setAllMoments).toHaveBeenLastCalledWith(false, expect.any(Function))
  })

  // FR-733.
  it('asks before changing many moments and changes nothing when declined', async () => {
    held.off = new Set(['Docked', 'LoadGame'])
    await shown()

    fireEvent.click(screen.getByRole('button', { name: 'Switch all on' }))
    const question = await screen.findByRole('dialog', { name: 'Switch 2 moments on' })
    expect(question.textContent).toContain('2 moments will be switched on')
    fireEvent.click(within(question).getByRole('button', { name: 'Cancel' }))

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(setAllMoments).not.toHaveBeenCalled()
    expect(checked('Docked')).toBe('false')
    expect(checked('Load game')).toBe('false')
  })

  it('names the category a question about it changes', async () => {
    await shown()

    fireEvent.click(switchNamed('Docking and stations'))

    const question = await screen.findByRole('dialog', { name: 'Switch 2 moments off' })
    expect(question.textContent).toContain('2 moments in Docking and stations will be switched off')
  })

  // FR-734.
  it('disables the button that would change nothing', async () => {
    await shown()

    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Switch all on' }).disabled).toBe(true)
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Switch all off' }).disabled).toBe(false)
  })

  // FR-735 in the markup: the state the style sheet places the thumb by. Where the thumb is drawn
  // is the style sheet's, which jsdom does not compute.
  it('draws a switch on at its end and off at its start', async () => {
    held.off = new Set(['Undocked'])
    await shown()

    expect(checked('Docked')).toBe('true')
    expect(checked('Undocked')).toBe('false')
    expect(switchNamed('Docked').querySelector('.thumb')).not.toBeNull()
  })

  // FR-737: a button is pressed by Space and Enter in the webview itself, which jsdom does not do,
  // so what is held here is that each switch is a button on the ring.
  it('makes every switch a button on the ring', async () => {
    await shown()

    for (const each of screen.getAllByRole('switch')) {
      expect(each.tagName).toBe('BUTTON')
      expect(each.getAttribute('type')).toBe('button')
      expect(each.hasAttribute('data-stop')).toBe(true)
    }
  })

  // FR-738.
  it('names each switch and says whether it is on', async () => {
    held.off = new Set(['Docked'])
    await shown()

    expect(checked('Docked')).toBe('false')
    expect(checked('Undocked')).toBe('true')
    expect(checked('Docking and stations')).toBe('true')
  })

  // FR-633: the switch applied, so the pane shows it off and says why it was not kept.
  it('says why a switch could not be kept', async () => {
    await shown()
    held.problem = 'the settings file could not be written'

    fireEvent.click(switchNamed('Docked'))

    expect((await screen.findByRole('alert')).textContent).toBe(held.problem)
    expect(checked('Docked')).toBe('false')
  })

  it('draws a refused switch as a refusal', async () => {
    setMoment.mockImplementation(refuses('no such moment: Docked', null))
    await shown()

    fireEvent.click(switchNamed('Docked'))

    expect((await screen.findByRole('alert')).textContent).toMatch(/no such moment/)
    expect(checked('Docked')).toBe('true')
  })
})
