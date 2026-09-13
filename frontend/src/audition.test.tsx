// The audition pane: hearing a voice deliberately.
//
// The behaviour worth guarding here is the pulse. An audition call returns the moment
// the clip STARTS, so nothing on this side knows when the sound stopped: the pulse is
// cleared by a playback event from the backend. Clearing it on the call's own
// resolution would put the light out while the clip was still audible; never
// clearing it would leave every button lit for the rest of the session.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { Audition, Group, Voice } from './api'

const voices = vi.fn<() => Promise<Voice[]>>()
const auditionGroups = vi.fn<(voice: string) => Promise<Group[]>>()
const audition = vi.fn<(voice: string, group: string) => Promise<Audition | null>>()
const stopAudition = vi.fn<() => Promise<void>>()

/** handlers holds whatever the pane subscribed to, so a test can raise the event. */
const handlers = new Map<string, (...data: unknown[]) => void>()

vi.mock('./api', () => ({
  api: {
    voices: () => voices(),
    auditionGroups: (voice: string) => auditionGroups(voice),
    audition: (voice: string, group: string) => audition(voice, group),
    stopAudition: () => stopAudition(),
  },
  on: (name: string, handler: (...data: unknown[]) => void) => {
    handlers.set(name, handler)
    return () => handlers.delete(name)
  },
}))

const { AuditionPane } = await import('./audition')

const grace: Voice = { name: 'Grace', inUse: 100 }
const kate: Voice = { name: 'Kate', inUse: 12 }

const shields: Group = { key: 'shields', label: 'Shields', clips: 4 }
const combat: Group = { key: 'combat', label: 'Combat', clips: 1 }

beforeEach(() => {
  handlers.clear()
  for (const spy of [voices, auditionGroups, audition, stopAudition]) spy.mockReset()
  voices.mockResolvedValue([grace, kate])
  auditionGroups.mockResolvedValue([shields, combat])
  audition.mockResolvedValue({ group: 'shields', clip: 'a.mp3' })
  stopAudition.mockResolvedValue(undefined)
})

/** show renders the pane over whoever is cast and waits for its groups. */
async function show(cast = 'Grace') {
  render(<AuditionPane cast={cast} />)
  await screen.findByRole('button', { name: /Shields/ })
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

    fireEvent.click(screen.getByRole('button', { name: /Shields/ }))

    await waitFor(() => expect(audition).toHaveBeenCalledWith('Grace', 'shields'))
  })

  // The pulse stays on until the backend says the sound has stopped. Clearing it when
  // the call resolves would put the light out while the clip was still playing.
  it('keeps the pulse on until the backend says the sound stopped', async () => {
    await show()
    const button = screen.getByRole('button', { name: /Shields/ })

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
    const button = screen.getByRole('button', { name: /Shields/ })
    fireEvent.click(button)
    await waitFor(() => expect(button.getAttribute('aria-current')).toBe('true'))

    handlers.get('playback')?.(undefined)

    await waitFor(() => expect(button.getAttribute('aria-current')).toBe('false'))
  })

  // A refused clip is the one case this side knows means no sound at all, so it both
  // clears the pulse and says what went wrong.
  it('reports a refused clip and puts the pulse out', async () => {
    await show()
    audition.mockRejectedValue(new Error('the device is busy'))
    const button = screen.getByRole('button', { name: /Shields/ })

    fireEvent.click(button)

    expect(await screen.findByText(/the device is busy/)).toBeTruthy()
    expect(button.getAttribute('aria-current')).toBe('false')
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
})
