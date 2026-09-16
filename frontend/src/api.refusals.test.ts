// What the api answers when a call does not happen: refused by the facade, else made with no
// bridge behind the page at all.
//
// Both are the same answer on purpose; neither is ever a rejection. A promise that rejects with
// nobody to catch it leaves the pane showing the words it says while it waits, so every call that
// can be refused takes a handler and answers with nothing.

import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, on } from './api'
import { nothingMade } from './making'
import { installBridge, removeBridge } from './testBridge'

afterEach(() => {
  removeBridge()
  vi.restoreAllMocks()
})

/** said stands where a test is not reading the reason; nothing here refuses through it. */
const said = (): void => undefined

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
    expect(await api.openDonation(said)).toBeUndefined()
    expect(await api.pluginChecklist()).toEqual({
      voice: '',
      recorded: 0,
      total: 0,
      missing: [],
      folder: '',
    })
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
