// The Missing takes pane with a voice cast from a plugin: the moments it has no take for, listed
// with no folder to open and no path to show, beside the recorded voices that keep both (FR-571).
//
// It sits apart from missingTakes.test.tsx because a plugin voice is cast apart from a recorded
// one and reaches the pane by a call of its own; the two suites share their fixtures.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { Checklist, CueEntry, Refused } from './api'
import { chooser, docked, fromPlugin, hyperspace, progress } from './missingTakesFixtures'

const voiceDirectories = vi.fn<(refused: Refused) => Promise<string[] | null>>()
const checklist = vi.fn<(voice: string, refused: Refused) => Promise<Checklist | null>>()
const pluginChecklist = vi.fn<() => Promise<Checklist>>()
const openMomentFolder = vi.fn<(voice: string, id: string, refused: Refused) => Promise<void>>()
const rescan = vi.fn<(refused: Refused) => Promise<number | null>>()
const chooseLibraryRoot = vi.fn<(refused: Refused) => Promise<string | null>>()

vi.mock('./api', () => ({
  api: {
    voiceDirectories: (refused: Refused) => voiceDirectories(refused),
    checklist: (voice: string, refused: Refused) => checklist(voice, refused),
    pluginChecklist: () => pluginChecklist(),
    openMomentFolder: (voice: string, id: string, refused: Refused) =>
      openMomentFolder(voice, id, refused),
    rescan: (refused: Refused) => rescan(refused),
    chooseLibraryRoot: (refused: Refused) => chooseLibraryRoot(refused),
  },
}))

const { MissingTakesPane } = await import('./missingTakes')

/** What each voice folder misses. Oliver holds one take of three; the rest are complete. */
let fixture: Record<string, CueEntry[]> = {}

beforeEach(() => {
  fixture = { Grace: [], Oliver: [docked, hyperspace] }
  for (const spy of [
    voiceDirectories,
    checklist,
    pluginChecklist,
    openMomentFolder,
    rescan,
    chooseLibraryRoot,
  ]) {
    spy.mockReset()
  }
  voiceDirectories.mockResolvedValue(['Grace', 'Oliver'])
  checklist.mockImplementation((voice) => Promise.resolve(progress(voice, fixture[voice] ?? [])))
  pluginChecklist.mockResolvedValue({ voice: '', recorded: 0, total: 0, missing: [], folder: '' })
  openMomentFolder.mockResolvedValue(undefined)
  rescan.mockResolvedValue(1)
  chooseLibraryRoot.mockResolvedValue('')
})

// A plugin voice has no folder this application owns, so its list names the moments it has no
// take for and offers no way to open or create anything (FR-571).
describe('a cast plugin voice', () => {
  /** castFromAPlugin opens the pane with a plugin voice cast and its list chosen. */
  async function castFromAPlugin(missing: CueEntry[] = [docked, hyperspace]) {
    pluginChecklist.mockResolvedValue(fromPlugin('The First Officer', missing))
    render(<MissingTakesPane cast="one" plugin="Bridge Crew" libraryRoot="D:/Recordings" />)
    const picker = await chooser()
    fireEvent.change(picker, { target: { value: 'The First Officer' } })
    await screen.findByText(/The First Officer has takes for/)
  }

  it('lists the moments it has no take for', async () => {
    await castFromAPlugin()

    expect(screen.getByText(docked.title)).toBeTruthy()
    expect(screen.getByText(hyperspace.title)).toBeTruthy()
  })

  it('offers no folder for any of them', async () => {
    await castFromAPlugin()

    expect(screen.queryByRole('button', { name: /Open the folder/ })).toBeNull()
    expect(screen.queryByText(/D:..Recordings..The First Officer/)).toBeNull()
  })

  // The folder voices are still listed beside it, each keeping its own folder and its button.
  it('leaves the recorded voices their folders', async () => {
    await castFromAPlugin()
    fireEvent.change(await chooser(), { target: { value: 'Oliver' } })
    await screen.findByText(/Oliver has recordings for/)

    expect(screen.getAllByRole('button', { name: /Open the folder/ }).length).toBe(2)
  })

  // Nothing is asked of a plugin where none is cast, which is almost every run (FR-562).
  it('is not asked for where no plugin voice is cast', async () => {
    render(<MissingTakesPane cast="Oliver" plugin="" libraryRoot="D:/Recordings" />)
    await chooser()

    expect(pluginChecklist).not.toHaveBeenCalled()
  })
})
