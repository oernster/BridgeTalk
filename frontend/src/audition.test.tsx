// The audition pane: hearing a voice deliberately.
//
// The behaviour worth guarding here is the pulse and the held buttons. An audition
// call returns the moment the clip STARTS, so nothing on this side knows when the
// sound stopped: both are released by a playback event from the backend. Releasing
// them on the call's own resolution would free the buttons while the clip was still
// audible, so a second press could cut it short (FR-236); never releasing them would
// leave the pane dead for the rest of the session.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { refuses } from './testRefusal'
import type { Voice } from './api'
import {
  audition,
  auditionGroups,
  docked,
  grace,
  groupButton,
  handlers,
  kate,
  playing,
  resetAudition,
  shown,
  stopAudition,
  voices,
} from './testAudition'

vi.mock('./api', async () => (await import('./testAudition')).mockedApi)

const { AuditionPane } = await import('./audition')

beforeEach(resetAudition)

/** show renders the pane over whoever is cast and waits for its groups. */
async function show(cast = 'Grace') {
  await shown(<AuditionPane cast={cast} />)
}

describe('the audition pane', () => {
  it('offers every voice the scan found', async () => {
    await show()

    const chooser = screen.getByRole('combobox') as HTMLSelectElement
    const offered = Array.from(chooser.options).map((option) => option.value)
    expect(offered).toEqual(['Grace', 'Kate'])
  })

  // The pane opens on whoever is cast and marks them, so the reader can tell the
  // voice they are hearing from the voice their ship speaks with.
  it('opens on the cast voice and marks it as cast', async () => {
    await show()

    expect((screen.getByRole('combobox') as HTMLSelectElement).value).toBe('Grace')
    expect(screen.getByRole('option', { name: 'Grace (cast)' })).toBeTruthy()
    expect(screen.getByRole('option', { name: 'Kate' })).toBeTruthy()
  })

  // FR-210: each voice under the name it is shown by, still chosen by the name that identifies it.
  it('offers each voice under the name it is shown by', async () => {
    voices.mockResolvedValue([{ ...grace, display: 'Grace Hart' }, kate])
    await show()

    const chooser = screen.getByRole('combobox') as HTMLSelectElement
    expect(Array.from(chooser.options).map((option) => option.value)).toEqual(['Grace', 'Kate'])
    expect(await screen.findByRole('option', { name: 'Grace Hart (cast)' })).toBeTruthy()
  })

  it('counts the groups and the samples between them', async () => {
    await show()

    expect(screen.getByText('2 groups, 5 samples between them.')).toBeTruthy()
  })

  // One sample is a sample, not samples. A count that reads "1 samples" is the sort
  // of thing a reader notices and nothing else does.
  it('names a single sample in the singular', async () => {
    await show()

    expect(screen.getByText(/^1 sample$/)).toBeTruthy()
    expect(screen.getByText(/^4 samples$/)).toBeTruthy()
  })

  // Hearing a voice before committing to it is the entire point of an audition, so
  // choosing one here is not casting it.
  it('auditions a voice that is not cast without casting it', async () => {
    await show()

    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'Kate' } })

    await waitFor(() => expect(auditionGroups).toHaveBeenLastCalledWith('Kate'))
    expect((screen.getByRole('combobox') as HTMLSelectElement).value).toBe('Kate')
  })

  it('plays the group whose button was pressed', async () => {
    await show()

    fireEvent.click(groupButton(/Shields/))

    await waitFor(() => expect(audition).toHaveBeenCalledWith('Grace', 'shields', expect.any(Function)))
  })

  // The pulse stays on until the backend says the sound has stopped. Clearing it when
  // the call resolves would put the light out while the clip was still playing.
  it('keeps the pulse on until the backend says the sound stopped', async () => {
    await show()
    const button = groupButton(/Shields/)

    fireEvent.click(button)
    await waitFor(() => expect(button.getAttribute('aria-current')).toBe('true'))

    handlers.get('playback')?.({ playing: true })
    expect(button.getAttribute('aria-current')).toBe('true')

    handlers.get('playback')?.({ playing: false })
    await waitFor(() => expect(button.getAttribute('aria-current')).toBe('false'))
  })

  // A playback event carrying nothing at all is the same as silence: a pulse left on
  // for the rest of the session is worse than one cleared a moment early.
  it('clears the pulse on a playback event that says nothing', async () => {
    await show()
    const button = groupButton(/Shields/)
    fireEvent.click(button)
    await waitFor(() => expect(button.getAttribute('aria-current')).toBe('true'))

    handlers.get('playback')?.(undefined)

    await waitFor(() => expect(button.getAttribute('aria-current')).toBe('false'))
  })

  // FR-236: a press must never cut a clip short, so every button is held from the
  // press until the backend says the sound has stopped, the one pressed included.
  it('holds every button while a clip plays, then frees them', async () => {
    await show()

    fireEvent.click(groupButton(/Shields/))
    await waitFor(() => expect(groupButton(/Combat/).disabled).toBe(true))
    expect(groupButton(/Shields/).disabled).toBe(true)

    act(() => handlers.get('playback')?.({ playing: false }))
    await waitFor(() => expect(groupButton(/Combat/).disabled).toBe(false))
    expect(groupButton(/Shields/).disabled).toBe(false)
  })

  // The ship speaking is a clip playing too, so a reaction to the game holds the
  // buttons exactly as an audition does.
  it('holds every button while the ship is speaking', async () => {
    await show()

    act(() => handlers.get('playback')?.({ playing: true }))

    await waitFor(() => expect(groupButton(/Combat/).disabled).toBe(true))
    expect(groupButton(/Shields/).disabled).toBe(true)
  })

  // A pane opened part way through a clip holds its buttons from the start rather
  // than waiting for an event that will only say when the clip ends.
  it('opens with its buttons held while something is already playing', async () => {
    playing.mockResolvedValue(true)
    await show()

    await waitFor(() => expect(groupButton(/Combat/).disabled).toBe(true))
  })

  // An event is newer than the answer to the opening question, so a late answer that
  // something is playing does not hold buttons an event has already freed.
  it('lets a playback event outrank a late opening answer', async () => {
    let answer: (sounding: boolean) => void = () => undefined
    playing.mockReturnValue(
      new Promise<boolean>((resolve) => {
        answer = resolve
      }),
    )
    await show()

    act(() => handlers.get('playback')?.({ playing: false }))
    await act(async () => answer(true))

    expect(groupButton(/Combat/).disabled).toBe(false)
  })

  // A refused clip is the one case this side knows means no sound at all, so it
  // clears the pulse, frees the buttons and says what went wrong.
  it('reports a refused clip and puts the pulse out', async () => {
    await show()
    audition.mockImplementation(refuses('Error: the device is busy', null))
    const button = groupButton(/Shields/)

    fireEvent.click(button)

    expect(await screen.findByText(/the device is busy/)).toBeTruthy()
    expect(button.getAttribute('aria-current')).toBe('false')
    expect(button.disabled).toBe(false)
  })

  it('stops whatever is playing when asked', async () => {
    await show()

    fireEvent.click(screen.getByRole('button', { name: 'Stop' }))

    expect(stopAudition).toHaveBeenCalled()
  })

  // A voice with nothing to audition says so rather than showing an empty row of
  // buttons, which reads as the pane having failed to load.
  it('says plainly when a voice has nothing to audition', async () => {
    auditionGroups.mockResolvedValue([])
    render(<AuditionPane cast="Grace" />)

    expect(await screen.findByText('This voice has nothing to audition.')).toBeTruthy()
    expect(screen.queryByText(/groups,/)).toBeNull()
  })

  // With nothing cast there is nobody to audition yet, so nothing is asked for.
  it('asks for nothing until there is a voice to audition', async () => {
    render(<AuditionPane cast="" />)

    await waitFor(() => expect(voices).toHaveBeenCalled())
    expect(auditionGroups).not.toHaveBeenCalled()
  })

  // An empty chooser shrunk to its arrow read as a fault. With no voice to name it
  // holds None instead; it is disabled, so the keyboard ring passes over it.
  it('holds None and stands disabled when no voice exists', async () => {
    voices.mockResolvedValue([])
    render(<AuditionPane cast="" />)

    const chooser = (await screen.findByRole('option', { name: 'None' })).closest(
      'select',
    ) as HTMLSelectElement
    expect(chooser.disabled).toBe(true)
    expect(Array.from(chooser.options).map((option) => option.text)).toEqual(['None'])
  })

  // No voices at all is a different absence from a voice with no takes, so it says
  // which it is and where to go about it.
  it('says no voices were found rather than blaming a voice', async () => {
    voices.mockResolvedValue([])
    render(<AuditionPane cast="" />)

    expect(await screen.findByText(/No voices found/)).toBeTruthy()
    expect(screen.queryByText('This voice has nothing to audition.')).toBeNull()
  })

  // FR-747: a group Chatter has switched off entirely is not offered and not counted.
  it('leaves out a group whose moments Chatter has all switched off', async () => {
    auditionGroups.mockResolvedValue([docked, ...(await auditionGroups('Grace'))])
    await show()

    expect(screen.queryByRole('button', { name: /Docked/ })).toBeNull()
    expect(screen.getByText('2 groups, 5 samples between them.')).toBeTruthy()
  })

  // FR-747: the pane asks for its groups each time it opens, so a group switched back on in Chatter
  // is offered the next time Audition is opened.
  it('asks for the groups again each time it opens', async () => {
    auditionGroups.mockResolvedValue([docked])
    const first = render(<AuditionPane cast="Grace" />)
    await screen.findByText(/Chatter has switched off/)
    first.unmount()

    auditionGroups.mockResolvedValue([{ ...docked, clips: 2, switchedOff: false }])
    render(<AuditionPane cast="Grace" />)

    expect(await screen.findByRole('button', { name: /Docked/ })).toBeTruthy()
    expect(auditionGroups).toHaveBeenCalledTimes(2)
  })

  // FR-750: the groups sit under their category headings in the order they came, a group in no category
  // under This application.
  it('lists the groups under their categories in order', async () => {
    auditionGroups.mockResolvedValue([
      { key: 'Docked', label: 'Docked', clips: 3, switchedOff: false, category: 'Docking and stations' },
      { key: 'ReceiveText', label: 'Receive text', clips: 2, switchedOff: false, category: 'Comms' },
      { key: 'Cast', label: 'Cast', clips: 1, switchedOff: false, category: '' },
    ])
    render(<AuditionPane cast="Grace" />)
    await screen.findByRole('button', { name: /Docked/ })

    const headings = screen.getAllByRole('heading', { level: 3 }).map((heading) => heading.textContent)
    expect(headings).toEqual(['Docking and stations', 'Comms', 'This application'])
    const underComms = screen.getByRole('region', { name: 'Comms' })
    expect(underComms.textContent).toContain('Receive text')
    expect(underComms.textContent).not.toContain('Docked')
    expect(screen.getByRole('region', { name: 'This application' }).textContent).toContain('Cast')
  })

  // FR-754 in the markup: each heading's words stand in a pill. How the pill is drawn is the style
  // sheet's, which jsdom does not compute.
  it('draws each category heading as a pill', async () => {
    auditionGroups.mockResolvedValue([
      { key: 'Docked', label: 'Docked', clips: 3, switchedOff: false, category: 'Docking and stations' },
      { key: 'Cast', label: 'Cast', clips: 1, switchedOff: false, category: '' },
    ])
    render(<AuditionPane cast="Grace" />)
    await screen.findByRole('button', { name: /Docked/ })

    for (const heading of screen.getAllByRole('heading', { level: 3 })) {
      expect(heading.querySelector('.heading-pill')?.textContent).toBe(heading.textContent)
    }
  })

  // FR-748: a voice whose every group Chatter has switched off says so, rather than suggesting the
  // recordings are missing.
  it('says Chatter has switched off everything the voice could be heard on', async () => {
    auditionGroups.mockResolvedValue([docked])
    render(<AuditionPane cast="Grace" />)

    expect(
      await screen.findByText('Chatter has switched off everything this voice could be heard on.'),
    ).toBeTruthy()
    expect(screen.queryByText('This voice has nothing to audition.')).toBeNull()
    expect(screen.queryByText(/groups,/)).toBeNull()
  })

  // Until the list arrives the chooser offers nothing it would have to take back.
  it('offers no None before the voices have been counted', () => {
    voices.mockReturnValue(new Promise<Voice[]>(() => undefined))
    render(<AuditionPane cast="" />)

    expect(screen.queryByRole('option', { name: 'None' })).toBeNull()
  })
})
