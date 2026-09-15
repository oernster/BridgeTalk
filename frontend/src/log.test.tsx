// The home pane's reaction log.
//
// It is the diagnostic surface, so what is asserted here is that it records the SILENT
// outcomes as well as the played ones: a cue that never speaks explaining itself is the
// whole reason the log exists. It is also a list on the ring, walked with the arrows.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { Reaction } from './api'
import { watching } from './testState'
import { dropped, handlers, played, reactions, resetHome } from './testHome'

vi.mock('./api', async () => (await import('./testHome')).mockedApi)

const { HomePane } = await import('./panes')

beforeEach(resetHome)

describe('the reaction log', () => {
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

  /** docked is a decision arriving after the two the log opens with. */
  const docked: Reaction = {
    at: '09:30:02',
    cue: 'Docked',
    title: 'Docked',
    clip: 'b.mp3',
    outcome: 'played',
  }

  /** selectedRow answers the index of the row the log marks as current; -1 where none is. */
  const selectedRow = () =>
    screen.getAllByRole('option').findIndex((row) => row.getAttribute('aria-selected') === 'true')

  // The log is one ring stop whose rows are walked with the vertical arrows; it
  // wraps at both ends so a long log can be crossed from either direction. Its current
  // row starts on the newest entry, which is where the log keeps its view.
  it('walks its rows with the vertical arrows, wrapping at both ends', async () => {
    reactions.mockResolvedValue([played, dropped])
    render(<HomePane state={watching} />)
    await screen.findByText('StartJump')
    const log = screen.getByRole('listbox', { name: 'Reaction log' })

    expect(selectedRow()).toBe(1)
    fireEvent.keyDown(log, { key: 'ArrowDown' })
    expect(selectedRow()).toBe(0)
    fireEvent.keyDown(log, { key: 'ArrowDown' })
    expect(selectedRow()).toBe(1)
    fireEvent.keyDown(log, { key: 'ArrowUp' })
    expect(selectedRow()).toBe(0)
  })

  // While the keyboard is elsewhere the current row follows the newest entry, so landing on
  // the log starts where the eye already is rather than at the top of a long log.
  it('keeps its current row on the newest entry while the keyboard is elsewhere', async () => {
    reactions.mockResolvedValue([played, dropped])
    render(<HomePane state={watching} />)
    await screen.findByText('StartJump')
    expect(selectedRow()).toBe(1)

    handlers.get('reaction')?.(docked)

    expect(await screen.findByText('Docked')).toBeTruthy()
    expect(selectedRow()).toBe(2)
  })

  // A row walked to stays put while the keyboard is on the log, whatever arrives meanwhile;
  // once the keyboard leaves, the current row follows the newest entry again.
  it('holds the row walked to while the keyboard is on it, then follows the newest again', async () => {
    reactions.mockResolvedValue([played, dropped])
    render(<HomePane state={watching} />)
    await screen.findByText('StartJump')
    const log = screen.getByRole('listbox', { name: 'Reaction log' })

    fireEvent.focus(log)
    fireEvent.keyDown(log, { key: 'ArrowUp' })
    handlers.get('reaction')?.(docked)
    expect(await screen.findByText('Docked')).toBeTruthy()
    expect(selectedRow()).toBe(0)

    fireEvent.blur(log)
    expect(selectedRow()).toBe(2)
  })

  /** scrollsDuring runs act with scrolling recorded, answering with every element scrolled into view. */
  async function scrollsDuring(act: () => Promise<void>): Promise<Element[]> {
    const original = Element.prototype.scrollIntoView
    const scrolled: Element[] = []
    Element.prototype.scrollIntoView = function (this: Element) {
      scrolled.push(this)
    }
    try {
      await act()
    } finally {
      Element.prototype.scrollIntoView = original
    }
    return scrolled
  }

  // The arrows are swallowed, so the log would not scroll to the row they reach on its
  // own. A row walked past the edge of the log is brought back into view.
  it('brings the row it walks to into view', async () => {
    const scrolled = await scrollsDuring(async () => {
      reactions.mockResolvedValue([played, dropped])
      render(<HomePane state={watching} />)
      await screen.findByText('StartJump')

      fireEvent.keyDown(screen.getByRole('listbox', { name: 'Reaction log' }), {
        key: 'ArrowDown',
      })
    })

    expect(scrolled).toEqual([screen.getAllByRole('option')[0]])
  })

  // FR-713: the log is a list, so it shows where focus is by its current row rather than by a
  // ring round the whole of it. That row has to be in sight as focus arrives, so it is brought
  // into view then.
  it('brings its current row into view as the keyboard lands on it', async () => {
    const scrolled = await scrollsDuring(async () => {
      reactions.mockResolvedValue([played, dropped])
      render(<HomePane state={watching} />)
      await screen.findByText('StartJump')

      fireEvent.focus(screen.getByRole('listbox', { name: 'Reaction log' }))
    })

    expect(scrolled).toEqual([screen.getAllByRole('option')[1]])
  })

  it('ignores a key that is not one of its own', async () => {
    reactions.mockResolvedValue([played, dropped])
    render(<HomePane state={watching} />)
    await screen.findByText('StartJump')
    const log = screen.getByRole('listbox', { name: 'Reaction log' })

    fireEvent.keyDown(log, { key: 'ArrowDown' })
    fireEvent.keyDown(log, { key: 'Enter' })

    expect(selectedRow()).toBe(0)
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
