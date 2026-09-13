// What a press of Browse tells the reader.
//
// The fault these exist to prevent was silence: choosing a directory with no voices in
// it was refused correctly and said so in the body colour, between two paragraphs of
// the same colour, so the press read as a button that did nothing. Every outcome is
// asserted here, including the one that is deliberately quiet.

import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'

const chooseLibraryRoot = vi.fn<() => Promise<string>>()
const chooseJournalDir = vi.fn<() => Promise<string>>()
const setLaunchOnBoot = vi.fn<(enabled: boolean) => Promise<void>>()

vi.mock('./api', () => ({
  api: {
    chooseLibraryRoot: () => chooseLibraryRoot(),
    chooseJournalDir: () => chooseJournalDir(),
    setLaunchOnBoot: (enabled: boolean) => setLaunchOnBoot(enabled),
  },
}))

const { SettingsPane } = await import('./panes')

/**
 * browse presses the button on the given row, the library root first.
 *
 * fireEvent rather than a bare click, so the state the press clears is settled inside
 * act and the run stays free of the warning that hides real ordering faults.
 */
function browse(row: number) {
  fireEvent.click(screen.getAllByRole('button', { name: 'Browse' })[row])
}

describe('the settings pane', () => {
  it('says why a directory was refused, as a refusal', async () => {
    chooseLibraryRoot.mockRejectedValue(
      'no voices in C:\\Recordings: choose the folder that holds one directory per person',
    )
    render(<SettingsPane state={null} />)
    browse(0)

    const said = await screen.findByRole('alert')
    expect(said.textContent).toContain('no voices in')
    expect(said.textContent).toContain('one directory per person')
    // Drawn as a refusal rather than as another paragraph of explanation.
    expect(said.className).toContain('refused')
  })

  it('answers under the row that was pressed, not at the foot of the pane', async () => {
    // A refusal about the recordings once sat below the journal row, where it read as
    // though the journal were the thing that had gone wrong.
    chooseLibraryRoot.mockRejectedValue('no voices in C:\\Recordings')
    render(<SettingsPane state={{ libraryRoot: 'C:\\Sounds' } as never} />)
    browse(0)

    const said = await screen.findByRole('alert')
    expect(said.previousElementSibling?.textContent).toContain('Recordings')
  })

  it('answers under the journal row when that is the one pressed', async () => {
    chooseJournalDir.mockRejectedValue('reading C:\\Nowhere: no such directory')
    render(<SettingsPane state={null} />)
    browse(1)

    const said = await screen.findByRole('alert')
    expect(said.previousElementSibling?.textContent).toContain('Journal directory')
  })

  it('confirms the directory it took', async () => {
    chooseLibraryRoot.mockResolvedValue('C:\\Sounds')
    render(<SettingsPane state={null} />)
    browse(0)

    const said = await screen.findByRole('status')
    expect(said.textContent).toContain('The recordings directory is now C:\\Sounds')
    expect(said.className).toContain('taken')
  })

  it('names the journal directory it took, not the library root', async () => {
    chooseJournalDir.mockResolvedValue('C:\\Journal')
    render(<SettingsPane state={null} />)
    browse(1)

    const said = await screen.findByRole('status')
    expect(said.textContent).toContain('The journal directory is now C:\\Journal')
  })

  it('says nothing at all when the dialog is cancelled', async () => {
    chooseLibraryRoot.mockResolvedValue('')
    render(<SettingsPane state={null} />)
    browse(0)

    // Waiting for the promise to settle first, so this cannot pass by being early.
    await waitFor(() => expect(chooseLibraryRoot).toHaveBeenCalled())
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

    await waitFor(() => expect(setLaunchOnBoot).toHaveBeenCalledWith(true))
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('answers Enter as well as Space, which a box does not do on its own', async () => {
    setLaunchOnBoot.mockResolvedValue()
    render(<SettingsPane state={{ launchOnBoot: false } as never} />)
    fireEvent.keyDown(screen.getByRole('checkbox'), { key: 'Enter' })

    await waitFor(() => expect(setLaunchOnBoot).toHaveBeenCalledWith(true))
  })

  it('says why the entry could not be written; the box stays as it was', async () => {
    // The box follows the state, so a refusal leaves it unticked on its own. Without a
    // reason beside it that reads as a dead control, which is the fault Browse had.
    setLaunchOnBoot.mockRejectedValue('this copy is running from a temporary directory')
    render(<SettingsPane state={{ launchOnBoot: false } as never} />)
    fireEvent.click(screen.getByRole('checkbox'))

    const said = await screen.findByRole('alert')
    expect(said.textContent).toContain('temporary directory')
    expect((screen.getByRole('checkbox') as HTMLInputElement).checked).toBe(false)
  })

  it('clears what the last press said before the next one answers', async () => {
    chooseLibraryRoot.mockRejectedValue('no voices in C:\\Recordings')
    render(<SettingsPane state={null} />)
    browse(0)
    await screen.findByRole('alert')

    chooseLibraryRoot.mockResolvedValue('C:\\Sounds')
    browse(0)
    await waitFor(() => expect(screen.queryByRole('alert')).toBeNull())
  })
})
