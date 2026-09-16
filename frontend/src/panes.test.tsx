// What the settings pane holds and what a press of Browse tells the reader.
//
// The fault these exist to prevent was silence: choosing a directory that could not be
// used was refused correctly and said so in the body colour, between two paragraphs of
// the same colour, so the press read as a button that did nothing. Every outcome is
// asserted here, including the one that is deliberately quiet.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { refuses } from './testRefusal'
import type { Refused } from './api'

const chooseLibraryRoot = vi.fn<(refused: Refused) => Promise<string | null>>()
const chooseJournalDir = vi.fn<(refused: Refused) => Promise<string | null>>()
const setLaunchOnBoot = vi.fn<(enabled: boolean, refused: Refused) => Promise<void>>()

vi.mock('./api', () => ({
  api: {
    chooseLibraryRoot: (refused: Refused) => chooseLibraryRoot(refused),
    chooseJournalDir: (refused: Refused) => chooseJournalDir(refused),
    setLaunchOnBoot: (enabled: boolean, refused: Refused) => setLaunchOnBoot(enabled, refused),
  },
}))

const { SettingsPane } = await import('./panes')

// Every spy starts empty, so a count asserted in one test is that test's own presses
// rather than the running total of every test before it.
beforeEach(() => {
  for (const spy of [chooseLibraryRoot, chooseJournalDir, setLaunchOnBoot]) spy.mockReset()
})

/**
 * browse presses the journal directory's Browse button, the only one the pane holds.
 *
 * fireEvent rather than a bare click, so the state the press clears is settled inside
 * act and the run stays free of the warning that hides real ordering faults.
 */
function browse() {
  fireEvent.click(screen.getByRole('button', { name: 'Browse' }))
}

describe('the settings pane', () => {
  // The recordings directory is chosen on the Missing takes pane, beside the voices it
  // holds; offering it here as well would give one setting two homes.
  it('holds the journal directory and no recordings directory', () => {
    render(<SettingsPane state={{ libraryRoot: 'C:\\Sounds', journalDir: 'C:\\Journal' } as never} />)

    expect(screen.getByText('Journal directory')).toBeTruthy()
    expect(screen.queryByText('Recordings')).toBeNull()
    expect(screen.queryByText('C:\\Sounds')).toBeNull()
    expect(screen.getAllByRole('button', { name: 'Browse' })).toHaveLength(1)
  })

  it('says why a directory was refused, as a refusal under its own row', async () => {
    chooseJournalDir.mockImplementation(
      refuses('reading C:\\Nowhere: no such directory', null),
    )
    render(<SettingsPane state={null} />)
    browse()

    const said = await screen.findByRole('alert')
    expect(said.textContent).toContain('no such directory')
    // Drawn as a refusal rather than as another paragraph of explanation.
    expect(said.className).toContain('refused')
    expect(said.previousElementSibling?.textContent).toContain('Journal directory')
  })

  it('names the journal directory it took', async () => {
    chooseJournalDir.mockResolvedValue('C:\\Journal')
    render(<SettingsPane state={null} />)
    browse()

    const said = await screen.findByRole('status')
    // FR-234: the row shows the path, so the message does not repeat it.
    expect(said.textContent).toBe('The journal directory was changed.')
    expect(said.className).toContain('taken')
  })

  it('says nothing at all when the dialog is cancelled', async () => {
    chooseJournalDir.mockResolvedValue('')
    render(<SettingsPane state={null} />)
    browse()

    // Waiting for the promise to settle first, so this cannot pass by being early.
    await waitFor(() => expect(chooseJournalDir).toHaveBeenCalled())
    expect(screen.queryByRole('alert')).toBeNull()
    expect(screen.queryByRole('status')).toBeNull()
  })

  it('draws the sign-in box from the state, never from a local copy', () => {
    const { unmount } = render(<SettingsPane state={{ launchOnBoot: true } as never} />)
    expect(screen.getByRole('checkbox')).toBeTruthy()
    expect((screen.getByRole('checkbox') as HTMLInputElement).checked).toBe(true)
    unmount()

    render(<SettingsPane state={{ launchOnBoot: false } as never} />)
    expect((screen.getByRole('checkbox') as HTMLInputElement).checked).toBe(false)
  })

  it('asks for the entry to be written when the box is ticked', async () => {
    setLaunchOnBoot.mockResolvedValue()
    render(<SettingsPane state={{ launchOnBoot: false } as never} />)
    fireEvent.click(screen.getByRole('checkbox'))

    await waitFor(() => expect(setLaunchOnBoot).toHaveBeenCalledWith(true, expect.any(Function)))
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('answers Enter as well as Space, which a box does not do on its own', async () => {
    setLaunchOnBoot.mockResolvedValue()
    render(<SettingsPane state={{ launchOnBoot: false } as never} />)
    fireEvent.keyDown(screen.getByRole('checkbox'), { key: 'Enter' })

    await waitFor(() => expect(setLaunchOnBoot).toHaveBeenCalledWith(true, expect.any(Function)))
  })

  it('says why the entry could not be written; the box stays as it was', async () => {
    // The box follows the state, so a refusal leaves it unticked on its own. Without a
    // reason beside it that reads as a dead control, which is the fault Browse had.
    setLaunchOnBoot.mockImplementation(
      refuses('this copy is running from a temporary directory', undefined),
    )
    render(<SettingsPane state={{ launchOnBoot: false } as never} />)
    fireEvent.click(screen.getByRole('checkbox'))

    const said = await screen.findByRole('alert')
    expect(said.textContent).toContain('temporary directory')
    expect((screen.getByRole('checkbox') as HTMLInputElement).checked).toBe(false)
  })

  // FR-238: startup could not watch the directory the row names, so the row says why
  // before anything is pressed; a press that takes a directory answers in its place.
  it('carries why startup could not watch the directory until a press answers', async () => {
    const problem = 'reading the journal directory C:\\Nowhere: cannot be found'
    chooseJournalDir.mockResolvedValue('C:\\Journal')
    render(<SettingsPane state={{ journalDir: 'C:\\Nowhere', journalProblem: problem } as never} />)

    const said = screen.getByRole('alert')
    expect(said.textContent).toBe(problem)
    expect(said.previousElementSibling?.textContent).toContain('Journal directory')

    browse()
    expect((await screen.findByRole('status')).textContent).toBe('The journal directory was changed.')
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('clears what the last press said before the next one answers', async () => {
    chooseJournalDir.mockImplementation(
      refuses('reading C:\\Nowhere: no such directory', null),
    )
    render(<SettingsPane state={null} />)
    browse()
    await screen.findByRole('alert')

    // The second press is cancelled, so it says nothing of its own; only clearing the
    // first answer can take the refusal down. A press that succeeded would replace the
    // refusal with its own answer and pass whether anything was cleared or not.
    chooseJournalDir.mockResolvedValue('')
    browse()
    await waitFor(() => expect(chooseJournalDir).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(screen.queryByRole('alert')).toBeNull())
  })
})
