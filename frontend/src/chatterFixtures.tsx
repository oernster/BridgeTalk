// The fixtures the Chatter suites share: the fake application holding the switches, the four moments
// it lists under two categories and the helpers that open the pane and read a switch.
//
// They sit apart from the suites because both read them: a moment or the fake's answer written out
// twice is two statements that can disagree; the suite that was not edited is then the one that passes
// while it should not. The pane is imported when it is opened rather than at the top, because the
// suites point their api mock at this module and the pane imports the api.

import { vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import type { Chatter, CueEntry, Refused } from './api'

export const chatter = vi.fn<() => Promise<Chatter>>()
export const setMoment = vi.fn<(id: string, on: boolean, refused: Refused) => Promise<Chatter | null>>()
export const setCategory =
  vi.fn<(name: string, on: boolean, refused: Refused) => Promise<Chatter | null>>()
export const setAllMoments = vi.fn<(on: boolean, refused: Refused) => Promise<Chatter | null>>()

/** fakeApi is the api as the suites mock it, each call handed to its spy. */
export const fakeApi = {
  chatter: () => chatter(),
  setMoment: (id: string, on: boolean, refused: Refused) => setMoment(id, on, refused),
  setCategory: (name: string, on: boolean, refused: Refused) => setCategory(name, on, refused),
  setAllMoments: (on: boolean, refused: Refused) => setAllMoments(on, refused),
}

/** entry names one moment as the pane is given it. */
const entry = (id: string, title: string, purpose: string): CueEntry => ({ id, title, folder: id, purpose })

export const docked = entry('Docked', 'Docked', 'When the ship finishes docking, as the journal records it.')
export const undocked = entry('Undocked', 'Undocked', 'When the ship leaves its pad.')
const loadGame = entry('LoadGame', 'Load game', 'When the game finishes loading.')
const shutdown = entry('Shutdown', 'Shutdown', 'When the game closes.')

// categories are the table's categories in order, each moment's switch on unless named in `held.off`.
export const categories = [
  { name: 'Docking and stations', moments: [docked, undocked] },
  { name: 'Session', moments: [loadGame, shutdown] },
]

// held.off holds the ids switched off; held.problem is what the next answer says about keeping it.
export const held = { off: new Set<string>(), problem: '' }

/** answer is the pane as the fake application holds it now. */
const answer = (): Chatter => ({
  categories: categories.map((category) => ({
    name: category.name,
    moments: category.moments.map((cue) => ({ cue, on: !held.off.has(cue.id) })),
  })),
  problem: held.problem,
})

/** switchTo switches the ids given to one state. */
const switchTo = (ids: string[], on: boolean) => {
  for (const id of ids) {
    if (on) held.off.delete(id)
    else held.off.add(id)
  }
  return Promise.resolve(answer())
}

export const every = categories.flatMap((category) => category.moments.map((cue) => cue.id))

/** resetFake switches every moment on and gives each spy the fake application's answers. */
export function resetFake() {
  held.off = new Set()
  held.problem = ''
  for (const spy of [chatter, setMoment, setCategory, setAllMoments]) spy.mockReset()
  chatter.mockImplementation(() => Promise.resolve(answer()))
  setMoment.mockImplementation((id, on) => switchTo([id], on))
  setCategory.mockImplementation((name, on) =>
    switchTo(categories.find((category) => category.name === name)?.moments.map((cue) => cue.id) ?? [], on),
  )
  setAllMoments.mockImplementation((on) => switchTo(every, on))
}

/** shown opens the pane and waits for its list to land. */
export async function shown() {
  const { ChatterPane } = await import('./chatter')
  render(<ChatterPane />)
  await screen.findByRole('switch', { name: 'Docked' })
}

/** switchNamed finds one switch by the moment or category it is named for. */
export const switchNamed = (name: string) => screen.getByRole('switch', { name })

/** checked reads a switch's state as it is announced. */
export const checked = (name: string) => switchNamed(name).getAttribute('aria-checked')
