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
    PluginVoices: record('PluginVoices', [{ plugin: 'Bridge Crew', id: 'one' }]),
    CastPluginVoice: record('CastPluginVoice', undefined),
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
    Chatter: record('Chatter', { categories: [], problem: '' }),
    SetMoment: record('SetMoment', { categories: [], problem: '' }),
    SetCategory: record('SetCategory', { categories: [], problem: '' }),
    SetAllMoments: record('SetAllMoments', { categories: [], problem: '' }),
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

/** said stands where a test only checks a call reaches the bridge; nothing is refused there. */
const said = (): void => undefined

describe('with the window bridge present', () => {
  it('sends every call through to the method it names', async () => {
    const calls = installBridge()

    await api.state()
    await api.voices()
    await api.selectVoice('Alpha', said)
    await api.machineVoices()
    await api.castMachineVoice('bf_emma', said)
    await api.pluginVoices()
    await api.castPluginVoice('Bridge Crew', 'one', said)
    await api.making()
    await api.setMuted(true)
    await api.volume()
    await api.setVolume(0.25)
    await api.auditionGroups('Alpha')
    await api.audition('Alpha', 'ShieldState', said)
    await api.stopAudition()
    await api.machineAuditionGroups()
    await api.auditionMachineVoice('bf_emma', 'Docked', said)
    await api.takeKeyboard()
    await api.quit()
    await api.reactions()
    await api.cueBreakdown('Alpha')
    await api.about()
    await api.licence()
    await api.chooseLibraryRoot(said)
    await api.chooseJournalDir(said)
    await api.makeVoiceFolders('Alpha', said)
    await api.rescan(said)
    await api.voiceDirectories(said)
    await api.checklist('Alpha', said)
    await api.openMomentFolder('Alpha', 'Docked', said)
    await api.setLaunchOnBoot(true, said)
    await api.minimiseToTray()
    await api.requestQuit()
    await api.chatter()
    await api.setMoment('Docked', false, said)
    await api.setCategory('Session', true, said)
    await api.setAllMoments(false, said)

    expect(calls.map((call) => call.name)).toEqual([
      'State',
      'Voices',
      'SelectVoice',
      'MachineVoices',
      'CastMachineVoice',
      'PluginVoices',
      'CastPluginVoice',
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
      'Chatter',
      'SetMoment',
      'SetCategory',
      'SetAllMoments',
    ])
  })

  it('carries the arguments each call was given', async () => {
    const calls = installBridge()

    await api.selectVoice('Beta', said)
    await api.castMachineVoice('am_michael', said)
    await api.setMuted(false)
    await api.setVolume(0.75)
    await api.auditionGroups('Beta')
    await api.audition('Beta', 'combat', said)
    await api.auditionMachineVoice('am_michael', 'combat', said)
    await api.cueBreakdown('Beta')
    await api.makeVoiceFolders('Beta', said)
    await api.checklist('Beta', said)
    await api.openMomentFolder('Beta', 'Docked', said)
    await api.setLaunchOnBoot(false, said)

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
    expect(await api.pluginVoices()).toEqual([{ plugin: 'Bridge Crew', id: 'one' }])
    expect(await api.making()).toEqual({ voice: 'bf_emma' })
    expect(await api.volume()).toBe(0.5)
    expect(await api.auditionGroups('Alpha')).toEqual([{ key: 'ShieldState' }])
    expect(await api.audition('Alpha', 'ShieldState', said)).toEqual({
      group: 'ShieldState',
      clip: 'a.mp3',
    })
    expect(await api.machineAuditionGroups()).toEqual([{ key: 'Docked' }])
    expect(await api.auditionMachineVoice('bf_emma', 'Docked', said)).toEqual({
      group: 'Docked',
      clip: 'k.flac',
    })
    expect(await api.reactions()).toEqual([{ cue: 'ShieldState.ShieldsUp.false' }])
    expect(await api.about()).toEqual({ name: 'the application' })
    expect(await api.chooseLibraryRoot(said)).toBe('D:/Recordings')
    expect(await api.chooseJournalDir(said)).toBe('D:/Journals')
    expect(await api.makeVoiceFolders('Alpha', said)).toEqual({ path: 'D:/Recordings/Alpha', made: 3 })
    expect(await api.rescan(said)).toBe(2)
    expect(await api.voiceDirectories(said)).toEqual(['Alpha', 'Beta'])
    expect(await api.checklist('Alpha', said)).toEqual({
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

// The whole point of the handler: a call that can be refused cannot be written without one; what
// comes back is never a rejection. A pane that forgets to catch is not a thing that can be
// written any more, so the reader is never left watching the words a pane says while it waits.
describe('a call the facade refuses', () => {
  it('tells the handler why and answers with nothing', async () => {
    installBridge({ Checklist: () => Promise.reject(new Error('reading Alpha: cannot be found')) })
    const said: string[] = []

    const answer = await api.checklist('Alpha', (reason) => said.push(reason))

    expect(answer).toBeNull()
    expect(said).toEqual(['Error: reading Alpha: cannot be found'])
  })

  it('leaves nothing to reject, so a caller that adds no catch is safe', async () => {
    installBridge({ SelectVoice: () => Promise.reject(new Error('no voice named Zeta')) })
    let reached = false

    await api.selectVoice('Zeta', () => undefined).then(() => {
      reached = true
    })

    expect(reached).toBe(true)
  })
})

describe('with no window bridge at all', () => {
  // A missing bridge means the page is being viewed outside the window. Every call has to answer
  // with something the shell can render; the first await otherwise throws and the reader is left
  // with a blank screen rather than an empty one. A call that can be refused answers null, which
  // is the same answer it gives for a refusal: the page cannot tell them apart and has no reason
  // to, since neither happened.
  it('answers every call with an empty value rather than throwing', async () => {
    removeBridge()

    expect(await api.state()).toBeNull()
    expect(await api.voices()).toEqual([])
    expect(await api.selectVoice('Alpha', said)).toBeUndefined()
    expect(await api.machineVoices()).toEqual([])
    expect(await api.castMachineVoice('bf_emma', said)).toBeUndefined()
    expect(await api.pluginVoices()).toEqual([])
    expect(await api.castPluginVoice('Bridge Crew', 'one', said)).toBeUndefined()
    expect(await api.making()).toEqual(nothingMade)
    expect(await api.setMuted(true)).toBeUndefined()
    expect(await api.setVolume(0.5)).toBeUndefined()
    expect(await api.auditionGroups('Alpha')).toEqual([])
    expect(await api.audition('Alpha', 'ShieldState', said)).toBeNull()
    expect(await api.stopAudition()).toBeUndefined()
    expect(await api.machineAuditionGroups()).toEqual([])
    expect(await api.auditionMachineVoice('bf_emma', 'Docked', said)).toBeNull()
    expect(await api.takeKeyboard()).toBeUndefined()
    expect(await api.quit()).toBeUndefined()
    expect(await api.reactions()).toEqual([])
    expect(await api.about()).toBeNull()
    expect(await api.licence()).toBeNull()
    expect(await api.rescan(said)).toBeNull()
    expect(await api.voiceDirectories(said)).toBeNull()
    expect(await api.openMomentFolder('Alpha', 'Docked', said)).toBeUndefined()
    expect(await api.setLaunchOnBoot(true, said)).toBeUndefined()
    expect(await api.minimiseToTray()).toBeUndefined()
    expect(await api.requestQuit()).toBeUndefined()
    expect(await api.chatter()).toEqual({ categories: [], problem: '' })
    expect(await api.setMoment('Docked', false, said)).toBeNull()
    expect(await api.setCategory('Session', true, said)).toBeNull()
    expect(await api.setAllMoments(false, said)).toBeNull()
  })

  // Making folders with no bridge makes none and answers with nothing, so the Cast pane reads it
  // as nothing done rather than as a folder at an empty path.
  it('answers making folders with nothing made', async () => {
    removeBridge()
    expect(await api.makeVoiceFolders('Alpha', said)).toBeNull()
  })

  // The checklist answers with nothing rather than an empty list for a voice nobody read. An
  // empty list is a claim, that the folder holds a recording for no moment at all; the pane would
  // then say a voice is complete on the strength of a call that never happened.
  it('answers with no checklist at all rather than an empty one', async () => {
    removeBridge()
    expect(await api.checklist('Alpha', said)).toBeNull()
  })

  // The volume falls back to full rather than to zero. A slider that opened at
  // silence outside the window would read as the application being muted.
  it('falls back to full volume rather than to silence', async () => {
    removeBridge()
    expect(await api.volume()).toBe(1)
  })

  // A chooser that could not be opened answers with nothing, which the pane reads as it reads a
  // refusal: nothing changed. A cancelled dialog is the other quiet case and answers an empty
  // string, which only a bridge can give.
  it('answers the directory choosers with nothing taken', async () => {
    removeBridge()
    expect(await api.chooseLibraryRoot(said)).toBeNull()
    expect(await api.chooseJournalDir(said)).toBeNull()
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
