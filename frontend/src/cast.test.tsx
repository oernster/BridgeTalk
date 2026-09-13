// The cast pane and the breakdown behind each row.
//
// What these guard is the pane's honesty about what it found. Every voice listed can be
// cast, because a directory that resolved no recording never becomes a voice; a row
// that refused would be a refusal nothing could explain. An empty library root says where
// it looked, because an absence with an address is something the reader can act on.

import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { CueBreakdown, Voice } from './api'

const voices = vi.fn<() => Promise<Voice[]>>()
const cueBreakdown = vi.fn<(name: string) => Promise<CueBreakdown>>()

vi.mock('./api', () => ({
  api: {
    voices: () => voices(),
    cueBreakdown: (name: string) => cueBreakdown(name),
  },
}))

const { CastPane } = await import('./cast')

/** grace and kate are two voices as the scan found them. */
const grace: Voice = { name: 'Grace', inUse: 1234 }
const kate: Voice = { name: 'Kate', inUse: 40 }

beforeEach(() => {
  voices.mockReset()
  cueBreakdown.mockReset()
  cueBreakdown.mockResolvedValue({ voice: '', served: [], unserved: [] })
})

/** show renders the pane over a given set of voices and waits for them to arrive. */
async function show(found: Voice[], active = '') {
  voices.mockResolvedValue(found)
  const onSelect = vi.fn<(name: string) => void>()
  render(<CastPane active={active} libraryRoot="D:/Recordings" onSelect={onSelect} />)
  if (found.length > 0) {
    // findAllByRole, because every row carries two buttons naming the same voice:
    // the one that casts it and the mark that opens its breakdown.
    await screen.findAllByRole('button', { name: new RegExp(found[0].name) })
  }
  return { onSelect }
}

describe('the cast pane', () => {
  it('counts the recordings each voice can reach', async () => {
    await show([grace, kate])

    expect(screen.getByText('1,234 usable recordings')).toBeTruthy()
    expect(screen.getByText('40 usable recordings')).toBeTruthy()
  })

  // A row that could not be pressed would need a reason, which nothing on the wire can
  // supply; so no row is ever drawn disabled and every one offers its breakdown.
  it('offers every voice it lists, refusing none', async () => {
    await show([grace, kate])

    const buttons = screen.getAllByRole('button') as HTMLButtonElement[]
    expect(buttons.filter((button) => button.disabled)).toEqual([])
    expect(screen.getByRole('button', { name: 'Moments Grace speaks for' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Moments Kate speaks for' })).toBeTruthy()
  })

  it('says which voice is cast, in the part it is cast as', async () => {
    await show([grace, kate], 'Grace')

    expect(
      screen.getByRole('button', { name: /Grace is cast as your ship's voice/ }),
    ).toBeTruthy()
    expect(screen.getByRole('button', { name: /^Cast Kate/ })).toBeTruthy()
  })

  it('casts the voice whose row is pressed', async () => {
    const { onSelect } = await show([grace, kate])

    fireEvent.click(screen.getByRole('button', { name: /^Cast Kate/ }))

    expect(onSelect).toHaveBeenCalledWith('Kate')
  })

  // The application cannot install a voice and cannot guess where one went, so the one
  // useful thing it can say is exactly where it looked.
  it('says where it looked when nothing was found', async () => {
    voices.mockResolvedValue([])
    render(<CastPane active="" libraryRoot="D:/Recordings" onSelect={vi.fn()} />)

    await screen.findByText('No voices found.')
    expect(screen.getByText('D:/Recordings')).toBeTruthy()
  })

  // An empty list and an unanswered one look the same on screen. Saying the voices are
  // missing for the fraction of a second before the answer lands is worse than
  // saying nothing.
  it('says nothing about an absence until the answer has arrived', () => {
    voices.mockReturnValue(new Promise(() => undefined))
    render(<CastPane active="" libraryRoot="D:/Recordings" onSelect={vi.fn()} />)

    expect(screen.queryByText('No voices found.')).toBeNull()
  })

  it('names the directory generically when there is not even a root', async () => {
    voices.mockResolvedValue([])
    render(<CastPane active="" libraryRoot="" onSelect={vi.fn()} />)

    await screen.findByText('No voices found.')
    expect(screen.getByText('the chosen directory')).toBeTruthy()
  })
})

describe('the breakdown behind a row', () => {
  it('opens for the voice whose mark was pressed, then closes again', async () => {
    cueBreakdown.mockResolvedValue({
      voice: 'Grace',
      served: [
        {
          id: 'StartJump.JumpType.Hyperspace',
          title: 'Start jump: jump type hyperspace',
          group: 'StartJump',
          heading: 'Start jump',
        },
      ],
      unserved: [
        { id: 'Disembark', title: 'Disembark', group: 'Disembark', heading: 'Disembark on foot' },
      ],
    })
    await show([grace])

    fireEvent.click(screen.getByRole('button', { name: 'Moments Grace speaks for' }))

    await waitFor(() => expect(cueBreakdown).toHaveBeenCalledWith('Grace'))
    expect(await screen.findByText('Start jump: jump type hyperspace')).toBeTruthy()
    expect(screen.getByText('Disembark')).toBeTruthy()

    // The group headings are the ones the backend generated rather than the id's segment.
    expect(screen.getByText('Start jump')).toBeTruthy()
    expect(screen.getByText('Disembark on foot')).toBeTruthy()

    // Each half is counted in its own heading, so the pair reads as a whole.
    expect(screen.getByRole('heading', { name: 'Moments spoken for (1)' })).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Moments with no lines (1)' })).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    await waitFor(() =>
      expect(screen.queryByText('Start jump: jump type hyperspace')).toBeNull(),
    )
  })

  // A voice that covers everything or nothing still has to render both halves: a
  // count on its own is not an answer without the list it counts.
  it('says plainly when one half of the breakdown is empty', async () => {
    cueBreakdown.mockResolvedValue({ voice: 'Grace', served: [], unserved: [] })
    await show([grace])

    fireEvent.click(screen.getByRole('button', { name: 'Moments Grace speaks for' }))

    await screen.findByRole('heading', { name: 'Moments spoken for (0)' })
    expect(screen.getAllByText('Nothing here.')).toHaveLength(2)
  })
})
