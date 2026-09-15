// Shared by the home pane's tests: the api they stand in for, the decisions its log shows and the
// About that names the product. Each test file mocks './api' with mockedApi and resets before each test.

import { vi } from 'vitest'
import type { About, Reaction } from './api'

export const reactions = vi.fn<() => Promise<Reaction[]>>()
export const about = vi.fn<() => Promise<About | null>>()
export const handlers = new Map<string, (...data: unknown[]) => void>()

/** mockedApi stands in for the api module: the calls the home pane makes and the events it hears. */
export const mockedApi = {
  api: {
    reactions: () => reactions(),
    about: () => about(),
    chooseLibraryRoot: () => Promise.resolve(''),
    chooseJournalDir: () => Promise.resolve(''),
    setLaunchOnBoot: () => Promise.resolve(),
  },
  on: (name: string, handler: (...data: unknown[]) => void) => {
    handlers.set(name, handler)
    return () => handlers.delete(name)
  },
}

export const played: Reaction = {
  at: '09:30:00',
  cue: 'StartJump',
  title: 'Start jump',
  clip: 'a.mp3',
  outcome: 'played',
}
export const dropped: Reaction = {
  at: '09:30:01',
  cue: 'ShieldState.ShieldsUp.false',
  title: 'Shield state: shields up false',
  clip: '',
  outcome: 'dropped',
}

/** named is About naming the product, so a tagline naming it has a name to show. */
export const named: About = {
  name: 'The Product',
  tagline: '',
  version: '9.9.9',
  author: '',
  copyright: '',
  authorship: '',
  attribution: '',
  licence: '',
  credits: [],
}

/** resetHome forgets what the last test heard, answering with an empty log and a named product. */
export function resetHome(): void {
  handlers.clear()
  reactions.mockReset()
  reactions.mockResolvedValue([])
  about.mockReset()
  about.mockResolvedValue(named)
}
