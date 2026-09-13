// The Missing takes pane: what a voice is missing and the way to each moment's folder.
//
// What these guard is that every voice folder is offered, recorded or not; that the list
// says what is missing under the moment's own heading with the id a file would carry;
// that the button opens the folder for the moment it sits beside.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { Checklist, CueEntry } from './api'

const voiceDirectories = vi.fn<() => Promise<string[]>>()
const checklist = vi.fn<(voice: string) => Promise<Checklist>>()
const openMomentFolder = vi.fn<(voice: string, id: string) => Promise<void>>()
const rescan = vi.fn<() => Promise<number>>()
const chooseLibraryRoot = vi.fn<() => Promise<string>>()

vi.mock('./api', () => ({
  api: {
    voiceDirectories: () => voiceDirectories(),
    checklist: (voice: string) => checklist(voice),
    openMomentFolder: (voice: string, id: string) => openMomentFolder(voice, id),
    rescan: () => rescan(),
    chooseLibraryRoot: () => chooseLibraryRoot(),
  },
}))

const { MissingTakesPane } = await import('./missingTakes')

const docked: CueEntry = { id: 'Docked', title: 'Docked', group: 'Docked', heading: 'Docked at a station' }
const hyperspace: CueEntry = {
  id: 'StartJump.JumpType.Hyperspace',
  title: 'Start jump: jump type hyperspace',
  group: 'StartJump',
  heading: 'Start jump',
}

/** progress builds a checklist for a voice with one moment of three recorded. */
const progress = (voice: string): Checklist => ({
  voice,
  recorded: 1,
  total: 3,
  missing: [docked, hyperspace],
})

beforeEach(() => {
  for (const spy of [voiceDirectories, checklist, openMomentFolder, rescan, chooseLibraryRoot]) {
    spy.mockReset()
  }
  voiceDirectories.mockResolvedValue(['Grace', 'Oliver'])
  checklist.mockImplementation((voice) => Promise.resolve(progress(voice)))
  openMomentFolder.mockResolvedValue(undefined)
  rescan.mockResolvedValue(1)
  chooseLibraryRoot.mockResolvedValue('')
})

/** browse presses the recordings row's Browse button. */
function browse() {
  fireEvent.click(screen.getByRole('button', { name: 'Browse' }))
}

// The recordings directory lives here rather than in Settings, beside the voices it holds.
describe('the recordings directory on the missing takes pane', () => {
  it('shows the directory it reads from', () => {
    render(<MissingTakesPane cast="Oliver" libraryRoot="D:/Recordings" />)

    expect(screen.getByText('D:/Recordings')).toBeTruthy()
  })

  it('says when no directory is chosen and waits while the state is on its way', () => {
    const { unmount } = render(<MissingTakesPane cast="Oliver" libraryRoot="" />)
    expect(screen.getByText('None chosen yet')).toBeTruthy()
    unmount()

    render(<MissingTakesPane cast="Oliver" />)
    expect(screen.getByText('...')).toBeTruthy()
  })

  it('confirms the directory it took, then reads its folders at once', async () => {
    chooseLibraryRoot.mockResolvedValue('D:/Takes')
    render(<MissingTakesPane cast="Oliver" libraryRoot="D:/Recordings" />)
    await waitFor(() => expect(voiceDirectories).toHaveBeenCalledTimes(1))

    browse()

    const said = await screen.findByRole('status')
    expect(said.textContent).toContain('The recordings directory is now D:/Takes')
    expect(said.className).toContain('taken')
    await waitFor(() => expect(voiceDirectories).toHaveBeenCalledTimes(2))
  })

  // The fault this guards was silence: a directory with no voices refused in the body
  // colour read as a button that did nothing.
  it('says why a directory was refused, as a refusal under its own row', async () => {
    chooseLibraryRoot.mockRejectedValue('no voices in D:/Empty')
    render(<MissingTakesPane cast="Oliver" libraryRoot="D:/Recordings" />)

    browse()

    const said = await screen.findByRole('alert')
    expect(said.textContent).toContain('no voices in D:/Empty')
    expect(said.className).toContain('refused')
    expect(said.previousElementSibling?.textContent).toContain('Recordings')
  })

  it('says nothing at all when the dialog is cancelled', async () => {
    render(<MissingTakesPane cast="Oliver" libraryRoot="D:/Recordings" />)
    await screen.findByText('Oliver has recordings for 1 of 3 moments.')

    browse()

    // Waiting for the press to settle first, so this cannot pass by being early.
    await waitFor(() => expect(chooseLibraryRoot).toHaveBeenCalled())
    expect(screen.queryByRole('alert')).toBeNull()
    expect(screen.queryByRole('status')).toBeNull()
    expect(voiceDirectories).toHaveBeenCalledTimes(1)
  })

  it('clears what the last press said before the next one answers', async () => {
    chooseLibraryRoot.mockRejectedValue('no voices in D:/Empty')
    render(<MissingTakesPane cast="Oliver" libraryRoot="D:/Recordings" />)
    browse()
    await screen.findByRole('alert')

    chooseLibraryRoot.mockResolvedValue('')
    browse()

    await waitFor(() => expect(screen.queryByRole('alert')).toBeNull())
  })
})

describe('the missing takes pane', () => {
  // FR-312: every folder is offered and the pane opens on the cast voice.
  it('offers every voice folder and opens on the cast voice', async () => {
    render(<MissingTakesPane cast="Oliver" />)

    const chooser = (await screen.findByRole('option', { name: 'Oliver' })).closest(
      'select',
    ) as HTMLSelectElement
    expect(Array.from(chooser.options).map((option) => option.value)).toEqual(['Grace', 'Oliver'])
    await waitFor(() => expect(chooser.value).toBe('Oliver'))
    await waitFor(() => expect(checklist).toHaveBeenLastCalledWith('Oliver'))
  })

  it('opens on the first folder when the cast voice has none', async () => {
    render(<MissingTakesPane cast="" />)

    await waitFor(() => expect(checklist).toHaveBeenLastCalledWith('Grace'))
  })

  // FR-311 and FR-313.
  it('lists what is missing under its heading and counts what is recorded', async () => {
    render(<MissingTakesPane cast="Oliver" />)

    expect(await screen.findByText('Oliver has recordings for 1 of 3 moments.')).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Start jump' })).toBeTruthy()
    expect(screen.getByText('StartJump.JumpType.Hyperspace')).toBeTruthy()
  })

  // FR-314. The button pressed is the second moment's, so one wired to whichever moment
  // happens to be missing first is caught rather than passing by coincidence.
  it('opens the folder for the moment whose button was pressed', async () => {
    render(<MissingTakesPane cast="Oliver" />)

    fireEvent.click(
      await screen.findByRole('button', {
        name: 'Open the folder for Start jump: jump type hyperspace',
      }),
    )

    expect(openMomentFolder).toHaveBeenCalledWith('Oliver', 'StartJump.JumpType.Hyperspace')
  })

  // FR-315.
  it('draws a folder that could not be opened as a refusal', async () => {
    openMomentFolder.mockRejectedValue('that folder could not be made')
    render(<MissingTakesPane cast="Oliver" />)

    fireEvent.click(await screen.findByRole('button', { name: 'Open the folder for Docked' }))

    expect((await screen.findByRole('alert')).textContent).toMatch(/could not be made/)
  })

  it('switches the list to the voice chosen', async () => {
    render(<MissingTakesPane cast="Oliver" />)
    const chooser = (await screen.findByRole('option', { name: 'Grace' })).closest(
      'select',
    ) as HTMLSelectElement

    fireEvent.change(chooser, { target: { value: 'Grace' } })

    expect(await screen.findByText('Grace has recordings for 1 of 3 moments.')).toBeTruthy()
  })

  // A look reads the folders again; it must not move the reader off the voice they chose
  // back to the cast voice. The folders answer is held until the test releases it, so the
  // assertion runs after the look has landed rather than before.
  it('keeps the voice chosen across a look', async () => {
    render(<MissingTakesPane cast="Oliver" />)
    const chooser = (await screen.findByRole('option', { name: 'Grace' })).closest(
      'select',
    ) as HTMLSelectElement
    fireEvent.change(chooser, { target: { value: 'Grace' } })
    await screen.findByText('Grace has recordings for 1 of 3 moments.')
    let release: (found: string[]) => void = () => undefined
    voiceDirectories.mockReturnValueOnce(
      new Promise((resolve) => {
        release = resolve
      }),
    )

    fireEvent.click(screen.getByRole('button', { name: 'Look again' }))
    await waitFor(() => expect(voiceDirectories).toHaveBeenCalledTimes(2))
    await act(async () => release(['Grace', 'Oliver']))

    expect(chooser.value).toBe('Grace')
    expect(checklist).toHaveBeenLastCalledWith('Grace')
  })

  it('draws voice folders that could not be read as a refusal', async () => {
    voiceDirectories.mockRejectedValue('the recordings directory could not be read')
    render(<MissingTakesPane cast="Oliver" />)

    expect((await screen.findByRole('alert')).textContent).toMatch(/directory could not be read/)
  })

  it('draws a checklist that could not be read as a refusal', async () => {
    checklist.mockRejectedValue('that voice folder could not be read')
    render(<MissingTakesPane cast="Oliver" />)

    expect((await screen.findByRole('alert')).textContent).toMatch(/folder could not be read/)
  })

  // A refusal belongs to the attempt that met it, so the next attempt takes it down
  // rather than leaving an old reason on screen beside a list that has since loaded.
  it('takes a refusal down when the next look begins', async () => {
    checklist.mockRejectedValueOnce('that voice folder could not be read')
    render(<MissingTakesPane cast="Oliver" />)
    await screen.findByRole('alert')

    fireEvent.click(screen.getByRole('button', { name: 'Look again' }))

    expect(await screen.findByText('Oliver has recordings for 1 of 3 moments.')).toBeTruthy()
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('takes a refusal down when the next open begins', async () => {
    openMomentFolder.mockRejectedValueOnce('that folder could not be made')
    render(<MissingTakesPane cast="Oliver" />)
    const button = await screen.findByRole('button', { name: 'Open the folder for Docked' })
    fireEvent.click(button)
    await screen.findByRole('alert')

    fireEvent.click(button)

    await waitFor(() => expect(openMomentFolder).toHaveBeenCalledTimes(2))
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('says so when every moment has a recording', async () => {
    checklist.mockResolvedValue({ voice: 'Oliver', recorded: 3, total: 3, missing: [] })
    render(<MissingTakesPane cast="Oliver" />)

    expect(await screen.findByText('Every moment has a recording.')).toBeTruthy()
  })

  it('points at Make folders when there is no voice folder yet', async () => {
    voiceDirectories.mockResolvedValue([])
    render(<MissingTakesPane cast="" />)

    expect(await screen.findByText('No voice folders yet.')).toBeTruthy()
    expect(checklist).not.toHaveBeenCalled()
  })

  // FR-214 from here: a look rescans for the whole window, then fetches the list again.
  it('looks again and fetches the list again', async () => {
    render(<MissingTakesPane cast="Oliver" />)
    await screen.findByText('Oliver has recordings for 1 of 3 moments.')
    checklist.mockResolvedValue({ voice: 'Oliver', recorded: 2, total: 3, missing: [hyperspace] })

    fireEvent.click(screen.getByRole('button', { name: 'Look again' }))

    expect(await screen.findByText('Oliver has recordings for 2 of 3 moments.')).toBeTruthy()
    expect(rescan).toHaveBeenCalled()
  })

  // A rescan refused for want of a voice is no reason to leave a stale list on screen.
  it('fetches the list again even when the rescan is refused', async () => {
    rescan.mockRejectedValue('no voice was found')
    render(<MissingTakesPane cast="Oliver" />)
    await screen.findByText('Oliver has recordings for 1 of 3 moments.')
    checklist.mockResolvedValue({ voice: 'Oliver', recorded: 2, total: 3, missing: [hyperspace] })

    fireEvent.click(screen.getByRole('button', { name: 'Look again' }))

    expect(await screen.findByText('Oliver has recordings for 2 of 3 moments.')).toBeTruthy()
  })
})
