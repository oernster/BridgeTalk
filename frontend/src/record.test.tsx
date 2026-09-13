// The record pane: what a voice is missing and the way to each moment's folder.
//
// What these guard is that every voice folder is offered, recorded or not; that the list
// says what is missing under the moment's own heading with the id a file would carry;
// that the button opens the folder for the moment it sits beside.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { Checklist, CueEntry } from './api'

const voiceDirectories = vi.fn<() => Promise<string[]>>()
const checklist = vi.fn<(voice: string) => Promise<Checklist>>()
const openMomentFolder = vi.fn<(voice: string, id: string) => Promise<void>>()
const rescan = vi.fn<() => Promise<number>>()

vi.mock('./api', () => ({
  api: {
    voiceDirectories: () => voiceDirectories(),
    checklist: (voice: string) => checklist(voice),
    openMomentFolder: (voice: string, id: string) => openMomentFolder(voice, id),
    rescan: () => rescan(),
  },
}))

const { RecordPane } = await import('./record')

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
  for (const spy of [voiceDirectories, checklist, openMomentFolder, rescan]) spy.mockReset()
  voiceDirectories.mockResolvedValue(['Grace', 'Oliver'])
  checklist.mockImplementation((voice) => Promise.resolve(progress(voice)))
  openMomentFolder.mockResolvedValue(undefined)
  rescan.mockResolvedValue(1)
})

describe('the record pane', () => {
  // FR-312: every folder is offered and the pane opens on the cast voice.
  it('offers every voice folder and opens on the cast voice', async () => {
    render(<RecordPane cast="Oliver" />)

    const chooser = (await screen.findByRole('option', { name: 'Oliver' })).closest(
      'select',
    ) as HTMLSelectElement
    expect(Array.from(chooser.options).map((option) => option.value)).toEqual(['Grace', 'Oliver'])
    await waitFor(() => expect(chooser.value).toBe('Oliver'))
    await waitFor(() => expect(checklist).toHaveBeenLastCalledWith('Oliver'))
  })

  it('opens on the first folder when the cast voice has none', async () => {
    render(<RecordPane cast="" />)

    await waitFor(() => expect(checklist).toHaveBeenLastCalledWith('Grace'))
  })

  // FR-311 and FR-313.
  it('lists what is missing under its heading and counts what is recorded', async () => {
    render(<RecordPane cast="Oliver" />)

    expect(await screen.findByText('Oliver has recordings for 1 of 3 moments.')).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Start jump' })).toBeTruthy()
    expect(screen.getByText('StartJump.JumpType.Hyperspace')).toBeTruthy()
  })

  // FR-314.
  it('opens the folder for the moment whose button was pressed', async () => {
    render(<RecordPane cast="Oliver" />)

    fireEvent.click(await screen.findByRole('button', { name: 'Open the folder for Docked' }))

    expect(openMomentFolder).toHaveBeenCalledWith('Oliver', 'Docked')
  })

  // FR-315.
  it('draws a folder that could not be opened as a refusal', async () => {
    openMomentFolder.mockRejectedValue('that folder could not be made')
    render(<RecordPane cast="Oliver" />)

    fireEvent.click(await screen.findByRole('button', { name: 'Open the folder for Docked' }))

    expect((await screen.findByRole('alert')).textContent).toMatch(/could not be made/)
  })

  it('switches the list to the voice chosen', async () => {
    render(<RecordPane cast="Oliver" />)
    const chooser = (await screen.findByRole('option', { name: 'Grace' })).closest(
      'select',
    ) as HTMLSelectElement

    fireEvent.change(chooser, { target: { value: 'Grace' } })

    expect(await screen.findByText('Grace has recordings for 1 of 3 moments.')).toBeTruthy()
  })

  it('says so when every moment has a recording', async () => {
    checklist.mockResolvedValue({ voice: 'Oliver', recorded: 3, total: 3, missing: [] })
    render(<RecordPane cast="Oliver" />)

    expect(await screen.findByText('Every moment has a recording.')).toBeTruthy()
  })

  it('points at Make folders when there is no voice folder yet', async () => {
    voiceDirectories.mockResolvedValue([])
    render(<RecordPane cast="" />)

    expect(await screen.findByText('No voice folders yet.')).toBeTruthy()
    expect(checklist).not.toHaveBeenCalled()
  })

  // FR-214 from here: a look rescans for the whole window, then fetches the list again.
  it('looks again and fetches the list again', async () => {
    render(<RecordPane cast="Oliver" />)
    await screen.findByText('Oliver has recordings for 1 of 3 moments.')
    checklist.mockResolvedValue({ voice: 'Oliver', recorded: 2, total: 3, missing: [hyperspace] })

    fireEvent.click(screen.getByRole('button', { name: 'Look again' }))

    expect(await screen.findByText('Oliver has recordings for 2 of 3 moments.')).toBeTruthy()
    expect(rescan).toHaveBeenCalled()
  })

  // A rescan refused for want of a voice is no reason to leave a stale list on screen.
  it('fetches the list again even when the rescan is refused', async () => {
    rescan.mockRejectedValue('no voice was found')
    render(<RecordPane cast="Oliver" />)
    await screen.findByText('Oliver has recordings for 1 of 3 moments.')
    checklist.mockResolvedValue({ voice: 'Oliver', recorded: 2, total: 3, missing: [hyperspace] })

    fireEvent.click(screen.getByRole('button', { name: 'Look again' }))

    expect(await screen.findByText('Oliver has recordings for 2 of 3 moments.')).toBeTruthy()
  })
})
