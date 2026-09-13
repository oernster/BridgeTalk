// The shell: which pane is showing, what the menus do and how the window behaves when
// the backend speaks to it.
//
// Three things here are worth guarding above the rest. The window opens on the cast,
// because choosing a voice is the first thing anybody does. A band button PICKS a pane
// and never toggles it, since pressing Cast while reading the cast is a way of asking
// where you are. And the cross does not close the window: the backend cancels that and
// asks here, because in a resident application the cross means "put it away" at least
// as often as it means "stop it".

import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import type { State, Voice } from './api'

const state = vi.fn<() => Promise<State | null>>()
const setMuted = vi.fn<(muted: boolean) => Promise<void>>()
const setVolume = vi.fn<(level: number) => Promise<void>>()
const selectVoice = vi.fn<(name: string) => Promise<void>>()
const voices = vi.fn<() => Promise<Voice[]>>()
const takeKeyboard = vi.fn<() => Promise<void>>()
const quit = vi.fn<() => Promise<void>>()
const minimiseToTray = vi.fn<() => Promise<void>>()
const requestQuit = vi.fn<() => Promise<void>>()

const handlers = new Map<string, (...data: unknown[]) => void>()

vi.mock('./api', () => ({
  api: {
    state: () => state(),
    setMuted: (muted: boolean) => setMuted(muted),
    setVolume: (level: number) => setVolume(level),
    selectVoice: (name: string) => selectVoice(name),
    takeKeyboard: () => takeKeyboard(),
    quit: () => quit(),
    minimiseToTray: () => minimiseToTray(),
    requestQuit: () => requestQuit(),
    voices: () => voices(),
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

// hugo is one voice, so the cast pane draws a row a test can press. The fixture
// defaults to no voices at all, which is the state every other test wants.
const hugo: Voice = { name: 'Hugo', inUse: 1200 }

beforeEach(() => {
  handlers.clear()
  window.localStorage.clear()
  for (const spy of [
    state,
    setMuted,
    setVolume,
    selectVoice,
    voices,
    takeKeyboard,
    quit,
    minimiseToTray,
    requestQuit,
  ]) {
    spy.mockReset()
  }
  state.mockResolvedValue(watching)
  voices.mockResolvedValue([])
  for (const spy of [setMuted, setVolume, selectVoice, takeKeyboard, quit, minimiseToTray, requestQuit]) {
    spy.mockResolvedValue(undefined)
  }
})

afterEach(() => {
  vi.useRealTimers()
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

/** band presses one nav-band button by its label. */
function band(label: string) {
  fireEvent.click(inBand().getByRole('button', { name: label }))
}

/** menuItem opens a menu title and presses one of its items. */
function menuItem(title: string, item: string) {
  fireEvent.click(inMenuBar().getByRole('button', { name: title }))
  fireEvent.click(inMenuBar().getByRole('button', { name: item }))
}

describe('the shell', () => {
  // Choosing a voice is the first thing anybody does and the last thing they change;
  // the status pane is where you go once something is already speaking.
  it('opens on the cast rather than on the status', async () => {
    await show()

    expect(await screen.findByRole('heading', { name: 'Cast' })).toBeTruthy()
    expect(inBand().getByRole('button', { name: 'Cast' }).getAttribute('aria-current'))
      .toBe('page')
  })

  it('switches the pane under the band rather than opening a dialog over it', async () => {
    await show()

    band('Status')
    expect(await screen.findByRole('heading', { name: 'Monitoring' })).toBeTruthy()
    expect(screen.queryByRole('heading', { name: 'Cast' })).toBeNull()

    band('Audition')
    expect(await screen.findByRole('heading', { name: 'Audition' })).toBeTruthy()

    band('Settings')
    expect(await screen.findByRole('heading', { name: 'Settings' })).toBeTruthy()

    band('Guide')
    expect(await screen.findByRole('heading', { name: 'The buttons along the top' }))
      .toBeTruthy()
  })

  // Pressing Cast while reading the cast is a way of asking where you are, so
  // answering it by moving somewhere else is the one thing it must not do.
  it('leaves the window where it is when the pane already showing is pressed', async () => {
    await show()
    await screen.findByRole('heading', { name: 'Cast' })

    band('Cast')

    expect(screen.getByRole('heading', { name: 'Cast' })).toBeTruthy()
  })

  it('casts the voice a row asked for and re-reads the state', async () => {
    voices.mockResolvedValue([hugo])
    await show()

    fireEvent.click(await screen.findByRole('button', { name: /^Cast Hugo/ }))

    await waitFor(() => expect(selectVoice).toHaveBeenCalledWith('Hugo'))
    await waitFor(() => expect(state.mock.calls.length).toBeGreaterThan(1))
  })

  // A cast can succeed and still report a failure, when the choice was made but could
  // not be written down. Refusing to re-read on a rejection would leave the pane
  // naming the voice that has stopped speaking.
  it('re-reads the state even when the cast reports a failure', async () => {
    voices.mockResolvedValue([hugo])
    selectVoice.mockRejectedValue(new Error('the disk is full'))
    await show()
    const before = state.mock.calls.length

    fireEvent.click(await screen.findByRole('button', { name: /^Cast Hugo/ }))

    await waitFor(() => expect(state.mock.calls.length).toBeGreaterThan(before))
  })
})

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

  // Recording is done now and then rather than every session, so it is reached from
  // Audio and takes no room on the band. The pane opens on the voice already cast.
  it('reaches the record pane from Audio alone, on the cast voice', async () => {
    state.mockResolvedValue({ ...watching, muted: true })
    await show()
    // The band offers to unmute only once the state has landed, which is where the pane
    // reads its cast from; opening it sooner would test the order the promises settle in.
    await waitFor(() => expect(inBand().getByRole('button', { name: 'Unmute' })).toBeTruthy())

    menuItem('Audio', 'Record')

    expect(await screen.findByRole('heading', { name: 'Record' })).toBeTruthy()
    expect(await screen.findByText('Grace has recordings for 0 of 0 moments.')).toBeTruthy()
    expect(inBand().queryByRole('button', { name: 'Record' })).toBeNull()
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

describe('what the backend tells the window', () => {
  // The cross is cancelled by the backend and turned into a question here, because in
  // a resident application it means "put it away" as often as it means "stop it".
  it('asks rather than closing; both answers reach the backend', async () => {
    await show()

    act(() => handlers.get('close-request')?.())
    expect(await screen.findByRole('dialog', { name: 'Close the window' })).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: /Minimise/ }))
    await waitFor(() => expect(minimiseToTray).toHaveBeenCalled())
    expect(screen.queryByRole('dialog', { name: 'Close the window' })).toBeNull()

    act(() => handlers.get('close-request')?.())
    await screen.findByRole('dialog', { name: 'Close the window' })
    fireEvent.click(screen.getByRole('button', { name: 'Quit' }))
    await waitFor(() => expect(requestQuit).toHaveBeenCalled())
  })

  // An accidental press should cost nothing, so dismissing the question changes
  // neither the window nor the run.
  it('costs nothing when the close question is dismissed', async () => {
    await show()
    act(() => handlers.get('close-request')?.())
    await screen.findByRole('dialog', { name: 'Close the window' })

    fireEvent.keyDown(window, { key: 'Escape' })

    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: 'Close the window' })).toBeNull(),
    )
    expect(minimiseToTray).not.toHaveBeenCalled()
    expect(requestQuit).not.toHaveBeenCalled()
  })

  // Hiding the window never reloads the page, so a summoned window would otherwise
  // return to whichever pane was open when it was put away.
  it('opens a summoned window on the cast, whatever pane it was left on', async () => {
    await show()
    band('Status')
    await screen.findByRole('heading', { name: 'Monitoring' })

    act(() => handlers.get('window-shown')?.())

    expect(await screen.findByRole('heading', { name: 'Cast' })).toBeTruthy()
  })

  it('re-reads the state whenever the backend says it changed', async () => {
    await show()
    const before = state.mock.calls.length

    act(() => handlers.get('state')?.())

    await waitFor(() => expect(state.mock.calls.length).toBeGreaterThan(before))
  })
})

describe('the keyboard on the way in', () => {
  // Setting activeElement is not the same as the document HAVING focus. Where the host
  // window keeps the keyboard, every press goes to it and no control can paint its
  // ring, which reads as a dead keyboard rather than as focus that landed elsewhere.
  it('asks the window for the keyboard when the page finds it has none', async () => {
    vi.useFakeTimers()
    vi.spyOn(document, 'hasFocus').mockReturnValue(false)
    render(<App />)

    await act(async () => {
      vi.advanceTimersByTime(1000)
    })

    expect(takeKeyboard).toHaveBeenCalled()
  })

  it('asks for nothing where the page already has the keyboard', async () => {
    vi.useFakeTimers()
    vi.spyOn(document, 'hasFocus').mockReturnValue(true)
    render(<App />)

    await act(async () => {
      vi.advanceTimersByTime(1000)
    })

    expect(takeKeyboard).not.toHaveBeenCalled()
  })
})
