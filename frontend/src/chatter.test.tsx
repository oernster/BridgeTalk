// The Chatter pane: which moments are spoken for.
//
// The switches live in the application, so the fake below plays its part: it holds the switches and
// answers every press with the pane as it then stands. What these guard is that the pane lists and
// counts what it is given, that each press asks for exactly the change it names and that a press
// changing more than one moment changes nothing until the question is answered.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { refuses } from './testRefusal'
import type { Chatter, CueEntry, Refused } from './api'

const chatter = vi.fn<() => Promise<Chatter>>()
const setMoment = vi.fn<(id: string, on: boolean, refused: Refused) => Promise<Chatter | null>>()
const setCategory =
  vi.fn<(name: string, on: boolean, refused: Refused) => Promise<Chatter | null>>()
const setAllMoments = vi.fn<(on: boolean, refused: Refused) => Promise<Chatter | null>>()

vi.mock('./api', () => ({
  api: {
    chatter: () => chatter(),
    setMoment: (id: string, on: boolean, refused: Refused) => setMoment(id, on, refused),
    setCategory: (name: string, on: boolean, refused: Refused) => setCategory(name, on, refused),
    setAllMoments: (on: boolean, refused: Refused) => setAllMoments(on, refused),
  },
}))

const { ChatterPane } = await import('./chatter')

/** entry names one moment as the pane is given it. */
const entry = (id: string, title: string, purpose: string): CueEntry => ({ id, title, folder: id, purpose })

const docked = entry('Docked', 'Docked', 'When the ship finishes docking, as the journal records it.')
const undocked = entry('Undocked', 'Undocked', 'When the ship leaves its pad.')
const loadGame = entry('LoadGame', 'Load game', 'When the game finishes loading.')
const shutdown = entry('Shutdown', 'Shutdown', 'When the game closes.')

// categories are the table's categories in order, each moment's switch on unless named in `off`.
const categories = [
  { name: 'Docking and stations', moments: [docked, undocked] },
  { name: 'Session', moments: [loadGame, shutdown] },
]

// off holds the ids switched off; problem is what the next answer says about keeping it.
let off = new Set<string>()
let problem = ''

/** answer is the pane as the fake application holds it now. */
const answer = (): Chatter => ({
  categories: categories.map((category) => ({
    name: category.name,
    moments: category.moments.map((cue) => ({ cue, on: !off.has(cue.id) })),
  })),
  problem,
})

/** switchTo switches the ids given to one state. */
const switchTo = (ids: string[], on: boolean) => {
  for (const id of ids) {
    if (on) off.delete(id)
    else off.add(id)
  }
  return Promise.resolve(answer())
}

const every = categories.flatMap((category) => category.moments.map((cue) => cue.id))

beforeEach(() => {
  off = new Set()
  problem = ''
  for (const spy of [chatter, setMoment, setCategory, setAllMoments]) spy.mockReset()
  chatter.mockImplementation(() => Promise.resolve(answer()))
  setMoment.mockImplementation((id, on) => switchTo([id], on))
  setCategory.mockImplementation((name, on) =>
    switchTo(categories.find((category) => category.name === name)?.moments.map((cue) => cue.id) ?? [], on),
  )
  setAllMoments.mockImplementation((on) => switchTo(every, on))
})

/** shown opens the pane and waits for its list to land. */
async function shown() {
  render(<ChatterPane />)
  await screen.findByRole('switch', { name: 'Docked' })
}

/** switchNamed finds one switch by the moment or category it is named for. */
const switchNamed = (name: string) => screen.getByRole('switch', { name })

/** checked reads a switch's state as it is announced. */
const checked = (name: string) => switchNamed(name).getAttribute('aria-checked')

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

  // FR-728.
  it('counts the moments switched on under each heading', async () => {
    off = new Set(['Docked'])
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
    off = new Set(['LoadGame'])
    const { unmount } = render(<ChatterPane />)
    await screen.findByRole('switch', { name: 'Session' })
    expect(checked('Session')).toBe('true')
    unmount()

    off = new Set(['LoadGame', 'Shutdown'])
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
    off = new Set(['LoadGame'])
    await shown()

    fireEvent.click(switchNamed('Session'))

    await waitFor(() => expect(checked('Session')).toBe('false'))
    expect(setCategory).toHaveBeenCalledWith('Session', false, expect.any(Function))
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  // FR-732.
  it('switches every moment on or off from the two buttons', async () => {
    off = new Set(['Docked', 'LoadGame'])
    await shown()

    fireEvent.click(screen.getByRole('button', { name: 'Switch all on' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Switch 2 on' }))
    await waitFor(() => expect(checked('Docked')).toBe('true'))
    expect(every.every((id) => !off.has(id))).toBe(true)

    fireEvent.click(screen.getByRole('button', { name: 'Switch all off' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Switch 4 off' }))
    await waitFor(() => expect(checked('Docked')).toBe('false'))
    expect(setAllMoments).toHaveBeenLastCalledWith(false, expect.any(Function))
  })

  // FR-733.
  it('asks before changing many moments and changes nothing when declined', async () => {
    off = new Set(['Docked', 'LoadGame'])
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
    off = new Set(['Undocked'])
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
    off = new Set(['Docked'])
    await shown()

    expect(checked('Docked')).toBe('false')
    expect(checked('Undocked')).toBe('true')
    expect(checked('Docking and stations')).toBe('true')
  })

  // FR-633: the switch applied, so the pane shows it off and says why it was not kept.
  it('says why a switch could not be kept', async () => {
    await shown()
    problem = 'the settings file could not be written'

    fireEvent.click(switchNamed('Docked'))

    expect((await screen.findByRole('alert')).textContent).toBe(problem)
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
