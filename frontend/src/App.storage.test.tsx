// What the window remembers between runs: the theme and the playback level.
//
// Both are the front end's own preference rather than the backend's, so both live in
// browser storage and both have to survive a browser that refuses it. The volume is
// the one with a trap in it: nothing stored converts to zero rather than to a failure,
// so a first run would start silent and read as an application that is broken.

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { State } from './api'
import { watching } from './testState'

const state = vi.fn<() => Promise<State | null>>()
const setVolume = vi.fn<(level: number) => Promise<void>>()

vi.mock('./api', () => ({
  api: {
    state: () => state(),
    setVolume: (level: number) => setVolume(level),
    setMuted: () => Promise.resolve(),
    selectVoice: () => Promise.resolve(),
    takeKeyboard: () => Promise.resolve(),
    quit: () => Promise.resolve(),
    minimiseToTray: () => Promise.resolve(),
    requestQuit: () => Promise.resolve(),
    voices: () => Promise.resolve([]),
    cueBreakdown: (name: string) =>
      Promise.resolve({ voice: name, served: [], unserved: [] }),
    auditionGroups: () => Promise.resolve([]),
    audition: () => Promise.resolve(null),
    stopAudition: () => Promise.resolve(),
    playing: () => Promise.resolve(false),
    reactions: () => Promise.resolve([]),
    about: () => Promise.resolve(null),
    chooseLibraryRoot: () => Promise.resolve(''),
    chooseJournalDir: () => Promise.resolve(''),
    setLaunchOnBoot: () => Promise.resolve(),
    machineVoices: () => Promise.resolve([]),
    pluginVoices: () => Promise.resolve([]),
    making: async () => (await import('./making')).nothingMade,
    castMachineVoice: () => Promise.resolve(),
  },
  on: () => () => undefined,
}))

const { App } = await import('./App')


beforeEach(() => {
  window.localStorage.clear()
  state.mockReset()
  state.mockResolvedValue(watching)
  setVolume.mockReset()
  setVolume.mockResolvedValue(undefined)
})

afterEach(() => {
  document.documentElement.removeAttribute('data-theme')
  vi.restoreAllMocks()
})

/** show renders the shell and waits for its first state to land. */
async function show() {
  render(<App />)
  await waitFor(() => expect(state).toHaveBeenCalled())
}

describe('the theme and the volume, between runs', () => {
  it('restores the theme that was chosen last time', async () => {
    window.localStorage.setItem('bridge-talk.theme', 'light')

    await show()

    await waitFor(() =>
      expect(document.documentElement.getAttribute('data-theme')).toBe('light'),
    )
  })

  // Dark is the palette this application is designed around, so anything unreadable
  // falls back to it rather than to the browser's own default.
  it('falls back to dark for a stored theme that means nothing', async () => {
    window.localStorage.setItem('bridge-talk.theme', 'chartreuse')

    await show()

    await waitFor(() =>
      expect(document.documentElement.getAttribute('data-theme')).toBe('dark'),
    )
  })

  // Nothing stored has to be tested before converting: Number(null) is zero rather
  // than a failure, so a first run would otherwise start silent and read as broken.
  it('starts at full volume on a first run rather than at silence', async () => {
    await show()

    await waitFor(() => expect(setVolume).toHaveBeenCalledWith(1))
    expect(screen.getByText('100%')).toBeTruthy()
  })

  it('restores the level that was chosen last time and pushes it to the player', async () => {
    window.localStorage.setItem('bridge-talk.volume', '0.25')

    await show()

    await waitFor(() => expect(setVolume).toHaveBeenCalledWith(0.25))
    expect(screen.getByText('25%')).toBeTruthy()
  })

  it('ignores a stored level that is not a level at all', async () => {
    window.localStorage.setItem('bridge-talk.volume', 'loud')

    await show()

    await waitFor(() => expect(setVolume).toHaveBeenCalledWith(1))
  })

  it('ignores a stored level outside the range the slider offers', async () => {
    window.localStorage.setItem('bridge-talk.volume', '4')

    await show()

    await waitFor(() => expect(setVolume).toHaveBeenCalledWith(1))
  })

  it('remembers a level the slider was moved to', async () => {
    await show()
    await waitFor(() => expect(setVolume).toHaveBeenCalled())

    fireEvent.change(screen.getByRole('slider', { name: 'Playback volume' }), {
      target: { value: '0.5' },
    })

    await waitFor(() => expect(setVolume).toHaveBeenLastCalledWith(0.5))
    expect(window.localStorage.getItem('bridge-talk.volume')).toBe('0.5')
  })

  // A viewer with storage disabled keeps the choice for this run rather than failing
  // to draw the window at all.
  it('still runs where the browser refuses storage', async () => {
    const refuse = () => {
      throw new Error('storage is disabled')
    }
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(refuse)
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(refuse)

    await show()

    await waitFor(() => expect(setVolume).toHaveBeenCalledWith(1))
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')

    fireEvent.change(screen.getByRole('slider', { name: 'Playback volume' }), {
      target: { value: '0.5' },
    })
    await waitFor(() => expect(setVolume).toHaveBeenLastCalledWith(0.5))

    vi.restoreAllMocks()
  })
})
