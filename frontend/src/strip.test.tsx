// The strip along the foot of the window: the donate button and the live indicator (FR-717 to
// FR-719).
//
// Which message shows is chosen by the pure selector, walked row by row in indicator.test.ts. What
// is asserted here is the wiring around it: a press reaching the application once, a failed press
// saying so for four seconds, a moment just played appearing and clearing on time, making followed
// as it is announced and the timers leaving with the strip.

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'
import type { About, Making, Reaction } from './api'
import { flashMs } from './indicator'
import { nothingMade } from './making'
import { watching } from './testState'

const about = vi.fn<() => Promise<About | null>>()
const making = vi.fn<() => Promise<Making>>()
const openDonation = vi.fn<() => Promise<void>>()
const handlers = new Map<string, (...data: unknown[]) => void>()

vi.mock('./api', () => ({
  api: {
    about: () => about(),
    making: () => making(),
    openDonation: () => openDonation(),
  },
  on: (name: string, handler: (...data: unknown[]) => void) => {
    handlers.set(name, handler)
    return () => handlers.delete(name)
  },
}))

const { Strip } = await import('./strip')

const named: About = {
  name: 'The Product',
  tagline: '',
  version: '9.9.9',
  author: '',
  copyright: '',
  authorship: '',
  attribution: '',
  licence: '',
  credits: [],
}

/** docked is a reaction that sounded. Its full title differs from its id, so which is shown is seen. */
const docked: Reaction = {
  at: '09:30:00',
  cue: 'CarrierDepositFuel',
  title: 'Carrier deposit fuel',
  clip: 'a.mp3',
  outcome: 'played',
}

/** makingLines is a machine voice with 13 of its 768 lines made while making goes on. */
const makingLines: Making = { ...nothingMade, voice: 'bf_emma', making: true, current: 13, total: 768 }

beforeEach(() => {
  handlers.clear()
  for (const spy of [about, making, openDonation]) spy.mockReset()
  about.mockResolvedValue(named)
  making.mockResolvedValue(nothingMade)
  openDonation.mockResolvedValue(undefined)
})

afterEach(() => {
  vi.useRealTimers()
})

/** settle lets the answers the strip asked for arrive. */
async function settle() {
  await act(async () => {})
}

/** said reads the live indicator's message. */
function said(): string {
  return document.querySelector('.strip .indicator')?.textContent ?? ''
}

/** donateButton finds the donate button by the start of its label. */
function donateButton(): HTMLElement {
  return screen.getByRole('button', { name: /^Donate to support/ })
}

describe('the donate button', () => {
  // FR-718: a picture alone does not say that pressing it leaves the application, so the label does.
  it('draws the artwork under a label saying the press opens the browser', async () => {
    render(<Strip state={watching} />)

    const label = 'Donate to support The Product (opens your browser)'
    const button = await screen.findByRole('button', { name: label })
    expect(button.getAttribute('data-label')).toBe(label)
    expect(button.hasAttribute('data-stop')).toBe(true)
    expect(button.querySelector('img')?.getAttribute('alt')).toBe('')
  })

  // The product is named as About gives it, so there is no name to show until it answers.
  it('names no product until About answers', async () => {
    about.mockResolvedValue(null)
    render(<Strip state={watching} />)
    await settle()

    expect(screen.getByRole('button', { name: 'Donate to support (opens your browser)' })).toBeTruthy()
  })

  it('asks the application to open the donation page once for each press', async () => {
    render(<Strip state={watching} />)
    await settle()

    fireEvent.click(donateButton())
    await settle()

    expect(openDonation).toHaveBeenCalledTimes(1)
  })

  // FR-718 and FR-719: a hand-over that failed is said in the indicator for four seconds after.
  it('says the browser could not be opened for four seconds after a failed press', async () => {
    vi.useFakeTimers()
    openDonation.mockRejectedValue(new Error('there is no window to open the browser from'))
    render(<Strip state={watching} />)
    await settle()

    fireEvent.click(donateButton())
    await settle()
    expect(said()).toBe('Could not open a browser for the donation page')

    act(() => vi.advanceTimersByTime(flashMs - 1))
    expect(said()).toBe('Could not open a browser for the donation page')
    act(() => vi.advanceTimersByTime(1))
    expect(said()).toBe('Listening with Grace')
  })
})

describe('the live indicator', () => {
  it('says what the application is doing, politely to a screen reader', async () => {
    render(<Strip state={watching} />)
    await settle()

    const indicator = document.querySelector('.strip .indicator') as HTMLElement
    expect(indicator.getAttribute('aria-live')).toBe('polite')
    expect(said()).toBe('Listening with Grace')
    expect(indicator.classList.contains('tone-ambient')).toBe(true)
  })

  it('says which moment was just played for four seconds, then goes back to listening', async () => {
    vi.useFakeTimers()
    render(<Strip state={watching} />)
    await settle()

    act(() => handlers.get('reaction')?.(docked))
    expect(said()).toBe('Just played: Carrier deposit fuel')

    act(() => vi.advanceTimersByTime(flashMs - 1))
    expect(said()).toBe('Just played: Carrier deposit fuel')
    act(() => vi.advanceTimersByTime(1))
    expect(said()).toBe('Listening with Grace')
  })

  // A second moment played before the first has cleared gets its own four seconds, rather than
  // being taken down on the first one's time.
  it('gives the last moment played its own four seconds', async () => {
    vi.useFakeTimers()
    render(<Strip state={watching} />)
    await settle()
    const half = flashMs / 2

    act(() => handlers.get('reaction')?.(docked))
    act(() => vi.advanceTimersByTime(half))
    act(() => handlers.get('reaction')?.({ ...docked, cue: 'Undocked', title: 'Undocked' }))

    act(() => vi.advanceTimersByTime(half))
    expect(said()).toBe('Just played: Undocked')
    act(() => vi.advanceTimersByTime(half))
    expect(said()).toBe('Listening with Grace')
  })

  it('says nothing of a reaction that played nothing', async () => {
    render(<Strip state={watching} />)
    await settle()

    act(() => handlers.get('reaction')?.({ ...docked, outcome: 'dropped' }))

    expect(said()).toBe('Listening with Grace')
  })

  it('reads where making stands when it opens, then follows it as it is announced', async () => {
    making.mockResolvedValue(makingLines)
    render(<Strip state={watching} />)
    await settle()
    expect(said()).toBe('Making lines: 13 of 768 ready')

    act(() => handlers.get('making')?.({ ...makingLines, current: 14 }))
    expect(said()).toBe('Making lines: 14 of 768 ready')
    expect(document.querySelector('.strip .indicator')?.classList.contains('tone-notice')).toBe(true)
  })

  it('takes its timers with it when it goes', async () => {
    vi.useFakeTimers()
    const { unmount } = render(<Strip state={watching} />)
    await settle()
    const before = vi.getTimerCount()

    act(() => handlers.get('reaction')?.(docked))
    expect(vi.getTimerCount()).toBe(before + 1)
    unmount()

    expect(vi.getTimerCount()).toBe(before)
  })
})
