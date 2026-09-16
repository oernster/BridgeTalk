// The audition pane over machine voices (FR-545 to FR-548): they follow the recorded voices, are
// auditioned through calls of their own and hold the buttons while a line is made.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { Audition } from './api'
import {
  audition,
  auditionGroups,
  auditionMachineVoice,
  emma,
  grace,
  groupButton,
  machineAuditionGroups,
  machineVoices,
  michael,
  resetAudition,
  shown,
  voices,
} from './testAudition'

vi.mock('./api', async () => (await import('./testAudition')).mockedApi)

const { AuditionPane } = await import('./audition')

beforeEach(resetAudition)

describe('the audition pane with machine voices', () => {
  // FR-545: the machine voices follow the recorded voices, each under the name the screen shows.
  it('offers the machine voices after the recorded voices', async () => {
    machineVoices.mockResolvedValue([emma, michael])
    await shown(<AuditionPane cast="Grace" />)

    const chooser = screen.getByRole('combobox') as HTMLSelectElement
    await waitFor(() => expect(chooser.options).toHaveLength(4))
    expect(Array.from(chooser.options).map((option) => option.text)).toEqual([
      'Grace (cast)',
      'Kate',
      'Emma (British, female)',
      'Michael (American, male)',
    ])
  })

  // FR-546: a machine voice is auditioned on the script's groups through its own call, never through
  // a recorded voice's, since a recordings folder may carry a machine voice's id as its name.
  it('auditions a machine voice through the machine voice calls', async () => {
    machineVoices.mockResolvedValue([emma])
    await shown(<AuditionPane cast="Grace" />)
    const chooser = screen.getByRole('combobox') as HTMLSelectElement
    await waitFor(() => expect(chooser.options).toHaveLength(3))
    auditionGroups.mockClear()

    fireEvent.change(chooser, { target: { value: chooser.options[2].value } })
    await waitFor(() => expect(machineAuditionGroups).toHaveBeenCalled())
    fireEvent.click(await screen.findByRole('button', { name: /Shields/ }))

    await waitFor(() => expect(auditionMachineVoice).toHaveBeenCalledWith('bf_emma', 'shields', expect.any(Function)))
    expect(auditionGroups).not.toHaveBeenCalled()
    expect(audition).not.toHaveBeenCalled()
  })

  // A recordings folder may carry a machine voice's id as its name, so the two stay apart in the
  // chooser and the one cast is the one marked.
  it('opens on a cast machine voice and marks it rather than a folder of the same name', async () => {
    voices.mockResolvedValue([{ ...grace, name: 'bf_emma', display: 'bf_emma' }])
    machineVoices.mockResolvedValue([emma])
    render(<AuditionPane cast="bf_emma" machine />)

    expect(await screen.findByRole('option', { name: 'Emma (British, female) (cast)' })).toBeTruthy()
    expect(screen.getByRole('option', { name: 'bf_emma' })).toBeTruthy()
    const chooser = screen.getByRole('combobox') as HTMLSelectElement
    expect(chooser.selectedOptions[0].text).toBe('Emma (British, female) (cast)')
    await waitFor(() => expect(machineAuditionGroups).toHaveBeenCalled())
    expect(auditionGroups).not.toHaveBeenCalled()
  })

  // FR-545: with no recordings the machine voices are still there to audition, so there is no None.
  it('offers the machine voices with no recorded voice at all', async () => {
    voices.mockResolvedValue([])
    machineVoices.mockResolvedValue([emma])
    render(<AuditionPane cast="" />)

    expect(await screen.findByRole('option', { name: 'Emma (British, female)' })).toBeTruthy()
    expect(screen.queryByRole('option', { name: 'None' })).toBeNull()
    expect((screen.getByRole('combobox') as HTMLSelectElement).disabled).toBe(false)
    expect(screen.queryByText(/No voices found/)).toBeNull()
  })

  // FR-547 and FR-548: a line being made holds every button as a clip playing does; a line that
  // cannot be made says why and frees them.
  it('holds the buttons while a line is made and says why one cannot be', async () => {
    voices.mockResolvedValue([])
    machineVoices.mockResolvedValue([emma])
    render(<AuditionPane cast="bf_emma" machine />)
    const button = (await screen.findByRole('button', { name: /Shields/ })) as HTMLButtonElement
    // The call is held open while the line is made, then refused the way the api refuses: the
    // handler is told why and the call answers with nothing, rather than rejecting.
    let refuse: (reason: string) => void = () => undefined
    auditionMachineVoice.mockImplementation(
      (_id, _group, refused) =>
        new Promise<Audition | null>((resolve) => {
          refuse = (reason) => {
            refused(reason)
            resolve(null)
          }
        }),
    )

    fireEvent.click(button)
    await waitFor(() => expect(groupButton(/Combat/).disabled).toBe(true))
    await act(async () => refuse('reading bf_emma.bin: missing'))

    expect(await screen.findByText(/reading bf_emma.bin: missing/)).toBeTruthy()
    expect(groupButton(/Combat/).disabled).toBe(false)
  })
})
