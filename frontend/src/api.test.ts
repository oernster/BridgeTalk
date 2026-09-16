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

import { installBridge, removeBridge } from './testBridge'

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
    await api.pluginChecklist()
    await api.openMomentFolder('Alpha', 'Docked', said)
    await api.openDonation(said)
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
      'PluginChecklist',
      'OpenMomentFolder',
      'OpenDonation',
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
