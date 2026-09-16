// The home pane: the cards across it, the lines beneath their figures and what it says about
// the audio device and the journal. The reaction log it holds is tested in log.test.tsx.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import type { State } from './api'
import { watching } from './testState'
import { about, resetHome } from './testHome'

vi.mock('./api', async () => (await import('./testHome')).mockedApi)

const { HomePane } = await import('./panes')

/** taglines reads the lines beneath one card's figure, in order. */
function taglines(label: string): string[] {
  const card = screen.getByText(label).closest('.card') as HTMLElement
  return Array.from(card.querySelectorAll('.tagline')).map((line) => line.textContent ?? '')
}

beforeEach(resetHome)

describe('the home pane', () => {
  it('says what is cast and where it is watching', async () => {
    render(<HomePane state={watching} />)

    expect(screen.getByText('Grace')).toBeTruthy()
    expect(screen.getByText('D:/Journals')).toBeTruthy()
    expect(screen.getByText('D:/Journals/Status.json')).toBeTruthy()
    expect(screen.getByText('Playback is live.')).toBeTruthy()
  })

  // FR-210: the Cast card names the voice as it is shown, not by the directory behind it.
  it('names the cast voice on the Status pane as it is shown', () => {
    render(<HomePane state={{ ...watching, voiceDisplay: 'Grace Hart' }} />)

    expect(screen.getByText('Grace Hart')).toBeTruthy()
    expect(screen.queryByText('Grace')).toBeNull()
  })

  // FR-716: the card once labelled "Cues served" counts moments, the window's own word; its
  // figure keeps the value colour every card uses rather than a warning one.
  it('labels the coverage card Moments covered, its figure in the value colour', () => {
    render(<HomePane state={watching} />)

    const figure = screen.getByText('40 of 60')
    expect(figure.className).toBe('value')
    expect(figure.previousElementSibling?.textContent).toBe('Moments covered')
    expect(screen.queryByText('Cues served')).toBeNull()
  })

  // FR-716: beneath each figure, what its card means, with the product named as About gives it.
  it('says beneath each figure what its card means', async () => {
    render(<HomePane state={watching} />)

    expect(await screen.findByText(/^The folder where Elite Dangerous records/)).toBeTruthy()
    expect(taglines('Cast')).toEqual([
      'The voice that speaks when something happens in the game. Change it on the Cast pane.',
    ])
    expect(taglines('Moments covered')[0]).toBe(
      'A moment is something that happens in the game that a voice can speak for, such as docking.',
    )
    expect(taglines('Journal')).toEqual([
      'The folder where Elite Dangerous records what happens in your game. The Product listens to it for moments to speak.',
    ])
    expect(taglines('Status file')).toEqual([
      "The file the game rewrites as your ship's state changes. The Product reads it for things the journal does not record.",
    ])
  })

  // The product is named in one place, so a tagline naming it waits for About.
  it('names no product beneath a figure until About answers', async () => {
    about.mockResolvedValue(null)
    render(<HomePane state={watching} />)
    await waitFor(() => expect(about).toHaveBeenCalled())

    expect(taglines('Journal')).toEqual([])
    expect(taglines('Status file')).toEqual([])
  })

  // FR-716: the second line beneath Moments covered, one for each situation the voice cast is in.
  it.each<[string, State, string]>([
    [
      'a recorded voice with every moment covered',
      { ...watching, bound: 60, total: 60 },
      'Every game moment has a recording.',
    ],
    [
      'a recorded voice with moments not covered',
      watching,
      'The other 20 moments have no recording yet, so they stay silent. Nothing is wrong: Missing takes lists them and where each recording goes.',
    ],
    [
      'a machine voice',
      { ...watching, voice: 'bf_emma', voiceDisplay: 'Emma', machineVoice: true, bound: 3, total: 256 },
      "A machine voice makes a moment's lines the first time it happens, so this number grows as you play. Nothing is missing.",
    ],
    [
      'no voice cast',
      { ...watching, voice: '', voiceDisplay: '', bound: 0, total: 256 },
      'Cast a voice to hear the game.',
    ],
  ])('says what the figure means for %s', (_situation, state, line) => {
    render(<HomePane state={state} />)

    expect(taglines('Moments covered')).toEqual([
      'A moment is something that happens in the game that a voice can speak for, such as docking.',
      line,
    ])
  })

  // FR-716's acceptance: one take for Docked out of 256 moments; a machine voice with lines made
  // for 3 of them.
  it('reads the acceptance figures with the line for each', () => {
    const { unmount } = render(
      <HomePane state={{ ...watching, voice: 'Oliver', voiceDisplay: 'Oliver', bound: 1, total: 256 }} />,
    )
    expect(screen.getByText('1 of 256')).toBeTruthy()
    expect(taglines('Moments covered')[1]).toMatch(/^The other 255 moments have no recording yet/)
    unmount()

    render(
      <HomePane
        state={{ ...watching, voice: 'bf_emma', voiceDisplay: 'Emma', machineVoice: true, bound: 3, total: 256 }}
      />,
    )
    expect(screen.getByText('3 of 256')).toBeTruthy()
    expect(taglines('Moments covered')[1]).toMatch(/^A machine voice makes a moment's lines/)
  })

  // A break in the speech is otherwise something only the listener knows about;
  // only in words at that. The count turns "it sounded wrong" into a number.
  it('says when the audio device ran dry and for how long', () => {
    render(<HomePane state={{ ...watching, stalls: 3, worstStall: 420 }} />)

    expect(screen.getByText(/ran dry 3 times this run/)).toBeTruthy()
    expect(screen.getByText(/longest for 420 ms/)).toBeTruthy()
  })

  it('counts one stall in the singular', () => {
    render(<HomePane state={{ ...watching, stalls: 1, worstStall: 260 }} />)

    expect(screen.getByText(/ran dry 1 time this run/)).toBeTruthy()
  })

  // A line reading zero every run trains the eye to skip it, which is the one thing
  // it must not do on the run where the number is not zero.
  it('says nothing at all when the device was never starved', () => {
    render(<HomePane state={watching} />)

    expect(screen.queryByText(/ran dry/)).toBeNull()
  })

  // What cannot be seen on the band is whether an audio device was found at all, so
  // that is what this line is for.
  it('says when there is no audio device at all', () => {
    render(<HomePane state={{ ...watching, muted: true, silent: true }} />)

    expect(screen.getByText(/Playback is muted\./)).toBeTruthy()
    expect(screen.getByText(/No audio device was available/)).toBeTruthy()
  })

  // FR-238: the window opens over a journal directory that cannot be watched, so the pane
  // that says what is watched is where the reader learns that nothing is.
  it('says why the journal directory is not being watched and where to put it right', () => {
    const problem = 'reading the journal directory D:/Nowhere: cannot be found'
    render(<HomePane state={{ ...watching, journalDir: 'D:/Nowhere', journalProblem: problem }} />)

    const said = screen.getByRole('alert')
    expect(said.textContent).toContain(problem)
    expect(said.textContent).toContain('Settings pane')
  })

  // FR-742: the loop that watches the game ended, so the window is open and hears nothing. The
  // pane says so and says what to do, since a window that looks alive and answers nothing is
  // worse than one that closed.
  it('says the application has stopped reacting and what to do about it', () => {
    const fault = 'runtime error: invalid memory address or nil pointer dereference'
    render(<HomePane state={{ ...watching, stoppedReacting: fault }} />)

    const said = screen.getByRole('alert')
    expect(said.textContent).toContain(fault)
    expect(said.textContent).toContain('start it again')
  })

  it('says nothing about the journal while it is being watched', () => {
    render(<HomePane state={watching} />)

    expect(screen.queryByRole('alert')).toBeNull()
  })

  // The pane is drawn before the first answer arrives, so every figure needs
  // something to show that is not the word "undefined".
  it('draws placeholders until the first answer arrives', () => {
    render(<HomePane state={null} />)

    expect(screen.getAllByText('...').length).toBeGreaterThan(0)
  })
})
