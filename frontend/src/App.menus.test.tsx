// The menu bar: what each menu reaches and how a menu opens and closes.
//
// The menus hold what is done now and then rather than every session, so what matters
// here is that each item reaches the pane or the act it names. Missing takes lives only
// in Audio; the band has no room for it and no need of it.

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import type { State } from './api'

const state = vi.fn<() => Promise<State | null>>()
const setMuted = vi.fn<(muted: boolean) => Promise<void>>()
const quit = vi.fn<() => Promise<void>>()

const handlers = new Map<string, (...data: unknown[]) => void>()

vi.mock('./api', () => ({
  api: {
    state: () => state(),
    setMuted: (muted: boolean) => setMuted(muted),
    quit: () => quit(),
    setVolume: () => Promise.resolve(),
    selectVoice: () => Promise.resolve(),
    takeKeyboard: () => Promise.resolve(),
    minimiseToTray: () => Promise.resolve(),
    requestQuit: () => Promise.resolve(),
    voices: () => Promise.resolve([]),
    cueBreakdown: (name: string) =>
      Promise.resolve({ voice: name, served: [], unserved: [] }),
    auditionGroups: () => Promise.resolve([]),
    audition: () => Promise.resolve(null),
    stopAudition: () => Promise.resolve(),
    reactions: () => Promise.resolve([]),
    about: () => Promise.resolve(null),
    licence: () => Promise.resolve('the full terms'),
    chooseLibraryRoot: () => Promise.resolve(''),
    chooseJournalDir: () => Promise.resolve(''),
    setLaunchOnBoot: () => Promise.resolve(),
    // Two folders with the cast voice second, so a pane that ignored the cast and fell
    // back to the first folder would name the wrong voice.
    voiceDirectories: () => Promise.resolve(['Hugo', 'Grace']),
    checklist: (voice: string) => Promise.resolve({ voice, recorded: 0, total: 0, missing: [] }),
    rescan: () => Promise.resolve(0),
  },
  on: (name: string, handler: (...data: unknown[]) => void) => {
    handlers.set(name, handler)
    return () => handlers.delete(name)
  },
}))

const { App } = await import('./App')

const watching: State = {
  voice: 'Grace',
  bound: 40,
  total: 60,
  muted: false,
  silent: false,
  journalDir: 'D:/Journals',
  statusPath: 'D:/Journals/Status.json',
  libraryRoot: 'D:/Recordings',
  version: '9.9.9',
  launchOnBoot: false,
  stalls: 0,
  worstStall: 0,
}

beforeEach(() => {
  handlers.clear()
  window.localStorage.clear()
  for (const spy of [state, setMuted, quit]) spy.mockReset()
  state.mockResolvedValue(watching)
  setMuted.mockResolvedValue(undefined)
  quit.mockResolvedValue(undefined)
})

afterEach(() => {
  document.documentElement.removeAttribute('data-theme')
})

/** show renders the shell and waits for its first state to land. */
async function show() {
  render(<App />)
  await waitFor(() => expect(state).toHaveBeenCalled())
}

// The menu bar and the nav band deliberately share names: the band's Settings button
// and the Settings menu open the same pane. Every query below is scoped to one of the
// two, so a test says which surface it is pressing rather than relying on there being
// only one control by that name.

/** inBand queries within the nav band alone. */
function inBand() {
  return within(document.querySelector('.navband') as HTMLElement)
}

/** inMenuBar queries within the menu bar alone. */
function inMenuBar() {
  return within(document.querySelector('.menubar') as HTMLElement)
}

/** menuItem opens a menu title and presses one of its items. */
function menuItem(title: string, item: string) {
  fireEvent.click(inMenuBar().getByRole('button', { name: title }))
  fireEvent.click(inMenuBar().getByRole('button', { name: item }))
}

describe('the menu bar', () => {
  it('ends the application from File', async () => {
    await show()

    menuItem('File', 'Quit')

    expect(quit).toHaveBeenCalled()
  })

  it('reaches the cast and the audition from Audio', async () => {
    await show()

    menuItem('Audio', 'Audition')
    expect(await screen.findByRole('heading', { name: 'Audition' })).toBeTruthy()

    menuItem('Audio', 'Cast')
    expect(await screen.findByRole('heading', { name: 'Cast' })).toBeTruthy()
  })

  // Filling in a voice is done now and then rather than every session, so Missing takes
  // is reached from Audio and takes no room on the band. It opens on the voice cast.
  it('reaches missing takes from Audio alone, on the cast voice', async () => {
    state.mockResolvedValue({ ...watching, muted: true })
    await show()
    // The band offers to unmute only once the state has landed, which is where the pane
    // reads its cast from; opening it sooner would test the order the promises settle in.
    await waitFor(() => expect(inBand().getByRole('button', { name: 'Unmute' })).toBeTruthy())

    menuItem('Audio', 'Missing takes')

    expect(await screen.findByRole('heading', { name: 'Missing takes' })).toBeTruthy()
    expect(await screen.findByText('Grace has recordings for 0 of 0 moments.')).toBeTruthy()
    expect(inBand().queryByRole('button', { name: 'Missing takes' })).toBeNull()
  })

  // The icon is the state and the name is the action: a muted application shows a
  // silenced speaker and offers to unmute.
  it('offers the mute as the act rather than as the state', async () => {
    await show()

    menuItem('Audio', 'Mute')
    expect(setMuted).toHaveBeenCalledWith(true)

    state.mockResolvedValue({ ...watching, muted: true })
    act(() => handlers.get('state')?.())
    await waitFor(() =>
      expect(inBand().getByRole('button', { name: 'Unmute' })).toBeTruthy(),
    )

    fireEvent.click(inBand().getByRole('button', { name: 'Unmute' }))
    await waitFor(() => expect(setMuted).toHaveBeenLastCalledWith(false))
  })

  it('opens the settings pane from Settings', async () => {
    await show()

    menuItem('Settings', 'Open settings')

    expect(await screen.findByRole('heading', { name: 'Settings' })).toBeTruthy()
  })

  // One item, not two: it names the theme it would switch to, so there is never a
  // choice between the mode you are in and the one you are not.
  it('names the theme it would switch to rather than the one in use', async () => {
    await show()

    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    menuItem('Settings', 'Light mode')

    await waitFor(() =>
      expect(document.documentElement.getAttribute('data-theme')).toBe('light'),
    )
    fireEvent.click(inMenuBar().getByRole('button', { name: 'Settings' }))
    expect(inMenuBar().getByRole('button', { name: 'Dark mode' })).toBeTruthy()

    // The same item takes the window back, so it is a switch in both directions rather
    // than a way into light mode with no way out from the menu.
    fireEvent.click(inMenuBar().getByRole('button', { name: 'Dark mode' }))
    await waitFor(() =>
      expect(document.documentElement.getAttribute('data-theme')).toBe('dark'),
    )
  })

  it('reaches the guide, the licence and the About dialog from Help', async () => {
    await show()

    menuItem('Help', 'Guide')
    expect(await screen.findByRole('heading', { name: 'The buttons along the top' }))
      .toBeTruthy()

    menuItem('Help', 'Licence')
    expect(await screen.findByText('the full terms')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))

    menuItem('Help', 'About')
    expect(await screen.findByRole('dialog', { name: 'About' })).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'About' })).toBeNull())
  })

  it('closes a menu that is open when its own title is pressed again', async () => {
    await show()
    const title = inMenuBar().getByRole('button', { name: 'File' })

    fireEvent.click(title)
    expect(title.getAttribute('aria-expanded')).toBe('true')
    fireEvent.click(title)
    expect(title.getAttribute('aria-expanded')).toBe('false')
  })
})
