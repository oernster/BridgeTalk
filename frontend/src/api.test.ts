// The bridge to the Go facade, in both the states it has to work in.
//
// Inside the window Wails injects window.go.main.App and window.runtime; every
// call has to reach the method it names: a rename on the Go side that is not
// mirrored here fails silently, because the optional chain simply finds nothing.
//
// Outside the window there is no bridge at all, which is what a browser and a test
// see. Every call degrades to an empty value rather than throwing, so the shell still
// renders and a fault shows as empty state rather than as a blank screen. Both halves
// are asserted, because the fallback is the half nothing else exercises.

import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, on } from './api'
import { nothingMade } from './making'

/** installBridge puts a recording double where Wails would put the real binding. */
function installBridge(overrides: Record<string, unknown> = {}) {
  const calls: Array<{ name: string; args: unknown[] }> = []
  const record =
    (name: string, answer: unknown) =>
    (...args: unknown[]) => {
      calls.push({ name, args })
      return Promise.resolve(answer)
    }

  const App: Record<string, unknown> = {
    State: record('State', { voice: 'Alpha' }),
    Voices: record('Voices', [{ name: 'Alpha' }]),
    SelectVoice: record('SelectVoice', undefined),
    MachineVoices: record('MachineVoices', [{ id: 'bf_emma', name: 'Emma (British, female)' }]),
    CastMachineVoice: record('CastMachineVoice', undefined),
    Making: record('Making', { voice: 'bf_emma' }),
    Muted: record('Muted', true),
    SetMuted: record('SetMuted', undefined),
    Volume: record('Volume', 0.5),
    SetVolume: record('SetVolume', undefined),
    AuditionGroups: record('AuditionGroups', [{ key: 'ShieldState' }]),
    Audition: record('Audition', { group: 'ShieldState', clip: 'a.mp3' }),
    StopAudition: record('StopAudition', undefined),
    MachineAuditionGroups: record('MachineAuditionGroups', [{ key: 'Docked' }]),
    AuditionMachineVoice: record('AuditionMachineVoice', { group: 'Docked', clip: 'k.flac' }),
    TakeKeyboard: record('TakeKeyboard', undefined),
    Quit: record('Quit', undefined),
    Reactions: record('Reactions', [{ cue: 'ShieldState.ShieldsUp.false' }]),
    CueBreakdown: record('CueBreakdown', { voice: 'Alpha', served: [], unserved: [] }),
    About: record('About', { name: 'the application' }),
    Licence: record('Licence', 'the terms'),
    ChooseLibraryRoot: record('ChooseLibraryRoot', 'D:/Recordings'),
    ChooseJournalDir: record('ChooseJournalDir', 'D:/Journals'),
    MakeVoiceFolders: record('MakeVoiceFolders', { path: 'D:/Recordings/Alpha', made: 3 }),
    Rescan: record('Rescan', 2),
    VoiceDirectories: record('VoiceDirectories', ['Alpha', 'Beta']),
    Checklist: record('Checklist', {
      voice: 'Alpha',
      recorded: 1,
      total: 3,
      missing: [],
      folder: 'D:/Recordings/Alpha/',
    }),
    OpenMomentFolder: record('OpenMomentFolder', undefined),
    SetLaunchOnBoot: record('SetLaunchOnBoot', undefined),
    MinimiseToTray: record('MinimiseToTray', undefined),
    RequestQuit: record('RequestQuit', undefined),
    ...overrides,
  }
  ;(window as unknown as { go: unknown }).go = { main: { App } }
  return calls
}

/** removeBridge leaves the page as a browser or a test sees it. */
function removeBridge() {
  delete (window as unknown as { go?: unknown }).go
  delete (window as unknown as { runtime?: unknown }).runtime
}

afterEach(() => {
  removeBridge()
  vi.restoreAllMocks()
})

describe('with the window bridge present', () => {
  it('sends every call through to the method it names', async () => {
    const calls = installBridge()

    await api.state()
    await api.voices()
    await api.selectVoice('Alpha')
    await api.machineVoices()
    await api.castMachineVoice('bf_emma')
    await api.making()
    await api.setMuted(true)
    await api.volume()
    await api.setVolume(0.25)
    await api.auditionGroups('Alpha')
    await api.audition('Alpha', 'ShieldState')
    await api.stopAudition()
    await api.machineAuditionGroups()
    await api.auditionMachineVoice('bf_emma', 'Docked')
    await api.takeKeyboard()
    await api.quit()
    await api.reactions()
    await api.cueBreakdown('Alpha')
    await api.about()
    await api.licence()
    await api.chooseLibraryRoot()
    await api.chooseJournalDir()
    await api.makeVoiceFolders('Alpha')
    await api.rescan()
    await api.voiceDirectories()
    await api.checklist('Alpha')
    await api.openMomentFolder('Alpha', 'Docked')
    await api.setLaunchOnBoot(true)
    await api.minimiseToTray()
    await api.requestQuit()

    expect(calls.map((call) => call.name)).toEqual([
      'State',
      'Voices',
      'SelectVoice',
      'MachineVoices',
      'CastMachineVoice',
      'Making',
      'SetMuted',
      'Volume',
      'SetVolume',
      'AuditionGroups',
      'Audition',
      'StopAudition',
      'MachineAuditionGroups',
      'AuditionMachineVoice',
      'TakeKeyboard',
      'Quit',
      'Reactions',
      'CueBreakdown',
      'About',
      'Licence',
      'ChooseLibraryRoot',
      'ChooseJournalDir',
      'MakeVoiceFolders',
      'Rescan',
      'VoiceDirectories',
      'Checklist',
      'OpenMomentFolder',
      'SetLaunchOnBoot',
      'MinimiseToTray',
      'RequestQuit',
    ])
  })

  it('carries the arguments each call was given', async () => {
    const calls = installBridge()

    await api.selectVoice('Beta')
    await api.castMachineVoice('am_michael')
    await api.setMuted(false)
    await api.setVolume(0.75)
    await api.auditionGroups('Beta')
    await api.audition('Beta', 'combat')
    await api.auditionMachineVoice('am_michael', 'combat')
    await api.cueBreakdown('Beta')
    await api.makeVoiceFolders('Beta')
    await api.checklist('Beta')
    await api.openMomentFolder('Beta', 'Docked')
    await api.setLaunchOnBoot(false)

    expect(calls.map((call) => call.args)).toEqual([
      ['Beta'],
      ['am_michael'],
      [false],
      [0.75],
      ['Beta'],
      ['Beta', 'combat'],
      ['am_michael', 'combat'],
      ['Beta'],
      ['Beta'],
      ['Beta'],
      ['Beta', 'Docked'],
      [false],
    ])
  })

  it('answers with what the facade returned', async () => {
    installBridge()

    expect(await api.state()).toEqual({ voice: 'Alpha' })
    expect(await api.voices()).toEqual([{ name: 'Alpha' }])
    expect(await api.machineVoices()).toEqual([{ id: 'bf_emma', name: 'Emma (British, female)' }])
    expect(await api.making()).toEqual({ voice: 'bf_emma' })
    expect(await api.volume()).toBe(0.5)
    expect(await api.auditionGroups('Alpha')).toEqual([{ key: 'ShieldState' }])
    expect(await api.audition('Alpha', 'ShieldState')).toEqual({
      group: 'ShieldState',
      clip: 'a.mp3',
    })
    expect(await api.machineAuditionGroups()).toEqual([{ key: 'Docked' }])
    expect(await api.auditionMachineVoice('bf_emma', 'Docked')).toEqual({
      group: 'Docked',
      clip: 'k.flac',
    })
    expect(await api.reactions()).toEqual([{ cue: 'ShieldState.ShieldsUp.false' }])
    expect(await api.about()).toEqual({ name: 'the application' })
    expect(await api.chooseLibraryRoot()).toBe('D:/Recordings')
    expect(await api.chooseJournalDir()).toBe('D:/Journals')
    expect(await api.makeVoiceFolders('Alpha')).toEqual({ path: 'D:/Recordings/Alpha', made: 3 })
    expect(await api.rescan()).toBe(2)
    expect(await api.voiceDirectories()).toEqual(['Alpha', 'Beta'])
    expect(await api.checklist('Alpha')).toEqual({
      voice: 'Alpha',
      recorded: 1,
      total: 3,
      missing: [],
      folder: 'D:/Recordings/Alpha/',
    })
  })

  it('subscribes to an event and hands back the way to stop', () => {
    const stop = vi.fn()
    // Typed with its real signature, so the assertion below can read the name
    // it was called with rather than indexing into an argument list of none.
    const EventsOn = vi.fn(
      (_name: string, _handler: (...data: unknown[]) => void) => stop,
    )
    ;(window as unknown as { runtime: unknown }).runtime = { EventsOn }

    const unsubscribe = on('state', () => undefined)

    expect(EventsOn).toHaveBeenCalledOnce()
    expect(EventsOn.mock.calls[0][0]).toBe('state')
    expect(unsubscribe).toBe(stop)
  })
})

describe('with no window bridge at all', () => {
  // A missing bridge means the page is being viewed outside the window. Every call
  // has to answer with something the shell can render; the first await otherwise throws and
  // the reader is left with a blank screen rather than an empty one.
  it('answers every call with an empty value rather than throwing', async () => {
    removeBridge()

    expect(await api.state()).toBeNull()
    expect(await api.voices()).toEqual([])
    expect(await api.selectVoice('Alpha')).toBeUndefined()
    expect(await api.machineVoices()).toEqual([])
    expect(await api.castMachineVoice('bf_emma')).toBeUndefined()
    expect(await api.making()).toEqual(nothingMade)
    expect(await api.setMuted(true)).toBeUndefined()
    expect(await api.setVolume(0.5)).toBeUndefined()
    expect(await api.auditionGroups('Alpha')).toEqual([])
    expect(await api.audition('Alpha', 'ShieldState')).toBeNull()
    expect(await api.stopAudition()).toBeUndefined()
    expect(await api.machineAuditionGroups()).toEqual([])
    expect(await api.auditionMachineVoice('bf_emma', 'Docked')).toBeNull()
    expect(await api.takeKeyboard()).toBeUndefined()
    expect(await api.quit()).toBeUndefined()
    expect(await api.reactions()).toEqual([])
    expect(await api.about()).toBeNull()
    expect(await api.licence()).toBeNull()
    expect(await api.rescan()).toBe(0)
    expect(await api.voiceDirectories()).toEqual([])
    expect(await api.openMomentFolder('Alpha', 'Docked')).toBeUndefined()
    expect(await api.setLaunchOnBoot(true)).toBeUndefined()
    expect(await api.minimiseToTray()).toBeUndefined()
    expect(await api.requestQuit()).toBeUndefined()
  })

  // Making folders with no bridge makes none and answers with an empty path, so the Cast
  // pane reads it as nothing done rather than as a folder at an empty path.
  it('answers making folders with nothing made', async () => {
    removeBridge()
    expect(await api.makeVoiceFolders('Alpha')).toEqual({ path: '', made: 0 })
  })

  // The checklist answers for the voice that was asked about, as the breakdown does, so
  // the Missing takes pane's count still names a voice.
  it('answers the checklist for the voice that was asked about', async () => {
    removeBridge()
    expect(await api.checklist('Alpha')).toEqual({
      voice: 'Alpha',
      recorded: 0,
      total: 0,
      missing: [],
      folder: '',
    })
  })

  // The volume falls back to full rather than to zero. A slider that opened at
  // silence outside the window would read as the application being muted.
  it('falls back to full volume rather than to silence', async () => {
    removeBridge()
    expect(await api.volume()).toBe(1)
  })

  // A cancelled chooser and a missing bridge are the same answer on purpose: an
  // empty string, which the pane already reads as "nothing changed".
  it('answers the directory choosers with nothing taken', async () => {
    removeBridge()
    expect(await api.chooseLibraryRoot()).toBe('')
    expect(await api.chooseJournalDir()).toBe('')
  })

  // The breakdown answers for the voice that was asked about, so the dialog behind
  // the row still has a heading rather than rendering an undefined name.
  it('answers the breakdown for the voice that was asked about', async () => {
    removeBridge()
    expect(await api.cueBreakdown('Alpha')).toEqual({
      voice: 'Alpha',
      served: [],
      unserved: [],
    })
  })

  it('subscribes to nothing and still hands back a way to stop', () => {
    removeBridge()

    const unsubscribe = on('state', () => undefined)

    expect(typeof unsubscribe).toBe('function')
    expect(unsubscribe()).toBeUndefined()
  })
})
