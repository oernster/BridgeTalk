// The home pane.
//
// It is the diagnostic surface, so what is asserted here is that it records the SILENT
// outcomes as well as the played ones: a cue that never speaks explaining itself is the
// whole reason the log exists.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { Reaction } from './api'
import { watching } from './testState'

const reactions = vi.fn<() => Promise<Reaction[]>>()
const handlers = new Map<string, (...data: unknown[]) => void>()

vi.mock('./api', () => ({
  api: {
    reactions: () => reactions(),
    chooseLibraryRoot: () => Promise.resolve(''),
    chooseJournalDir: () => Promise.resolve(''),
    setLaunchOnBoot: () => Promise.resolve(),
  },
  on: (name: string, handler: (...data: unknown[]) => void) => {
    handlers.set(name, handler)
    return () => handlers.delete(name)
  },
}))

const { HomePane } = await import('./panes')

const played: Reaction = {
  at: '09:30:00',
  cue: 'StartJump',
  clip: 'a.mp3',
  outcome: 'played',
}
const dropped: Reaction = {
  at: '09:30:01',
  cue: 'ShieldState.ShieldsUp.false',
  clip: '',
  outcome: 'dropped',
}

beforeEach(() => {
  handlers.clear()
  reactions.mockReset()
  reactions.mockResolvedValue([])
})

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

  // A shortfall is a gap in the recordings rather than a fault in the application. The
  // hint beside it is what stops it reading as one.
  it('shows the shortfall with the line that says what it means', () => {
    render(<HomePane state={watching} />)

    expect(screen.getByText('40 of 60')).toBeTruthy()
    expect(screen.getByText(/The rest stay silent/)).toBeTruthy()
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

  it('says the log is empty rather than drawing an empty box', async () => {
    render(<HomePane state={watching} />)

    expect(
      await screen.findByText('Nothing yet. Start the game and this will fill up.'),
    ).toBeTruthy()
  })

  // The silent outcomes are the point. A cue that produced no sound is recorded under
  // its cue with the reason, so a cue that never fires can be diagnosed by looking
  // rather than by guessing.
  it('records the decisions that produced no sound as well as the ones that did', async () => {
    reactions.mockResolvedValue([played, dropped])
    render(<HomePane state={watching} />)

    expect(await screen.findByText('StartJump')).toBeTruthy()
    expect(screen.getByText('a.mp3')).toBeTruthy()
    expect(screen.getByText('played')).toBeTruthy()

    expect(screen.getByText('ShieldState.ShieldsUp.false')).toBeTruthy()
    expect(screen.getByText('dropped')).toBeTruthy()
    // FR-234: with no clip to name, nothing stands in its place, so the cue is shown once.
    expect(screen.getAllByText('ShieldState.ShieldsUp.false')).toHaveLength(1)
    expect(screen.queryByText('ShieldState')).toBeNull()
  })

  it('adds each new decision as it is announced', async () => {
    render(<HomePane state={watching} />)
    await screen.findByText('Nothing yet. Start the game and this will fill up.')

    handlers.get('reaction')?.(played)

    expect(await screen.findByText('StartJump')).toBeTruthy()
  })

  // The log is one ring stop whose rows are walked with the vertical arrows; it
  // wraps at both ends so a long log can be crossed from either direction.
  it('walks its rows with the vertical arrows, wrapping at both ends', async () => {
    reactions.mockResolvedValue([played, dropped])
    render(<HomePane state={watching} />)
    await screen.findByText('StartJump')
    const log = screen.getByRole('listbox', { name: 'Reaction log' })

    const selected = () =>
      screen
        .getAllByRole('option')
        .findIndex((row) => row.getAttribute('aria-selected') === 'true')

    expect(selected()).toBe(0)
    fireEvent.keyDown(log, { key: 'ArrowDown' })
    expect(selected()).toBe(1)
    fireEvent.keyDown(log, { key: 'ArrowDown' })
    expect(selected()).toBe(0)
    fireEvent.keyDown(log, { key: 'ArrowUp' })
    expect(selected()).toBe(1)
  })

  // The arrows are swallowed, so the log would not scroll to the row they reach on its
  // own. A row walked past the edge of the log is brought back into view.
  it('brings the row it walks to into view', async () => {
    const original = Element.prototype.scrollIntoView
    const scrolled = vi.fn()
    Element.prototype.scrollIntoView = scrolled
    try {
      reactions.mockResolvedValue([played, dropped])
      render(<HomePane state={watching} />)
      await screen.findByText('StartJump')

      fireEvent.keyDown(screen.getByRole('listbox', { name: 'Reaction log' }), {
        key: 'ArrowDown',
      })

      expect(scrolled).toHaveBeenCalledTimes(1)
      expect(scrolled.mock.contexts[0]).toBe(screen.getAllByRole('option')[1])
    } finally {
      Element.prototype.scrollIntoView = original
    }
  })

  it('ignores a key that is not one of its own', async () => {
    reactions.mockResolvedValue([played, dropped])
    render(<HomePane state={watching} />)
    await screen.findByText('StartJump')
    const log = screen.getByRole('listbox', { name: 'Reaction log' })

    fireEvent.keyDown(log, { key: 'ArrowDown' })
    fireEvent.keyDown(log, { key: 'Enter' })

    const rows = screen.getAllByRole('option')
    expect(rows[1].getAttribute('aria-selected')).toBe('true')
  })

  // An empty log has no row to move to, so the arrows must not walk off the end of
  // nothing.
  it('does not walk an empty log', async () => {
    render(<HomePane state={watching} />)
    await screen.findByText('Nothing yet. Start the game and this will fill up.')

    fireEvent.keyDown(screen.getByRole('listbox', { name: 'Reaction log' }), {
      key: 'ArrowDown',
    })

    expect(screen.queryAllByRole('option')).toHaveLength(0)
  })
})
