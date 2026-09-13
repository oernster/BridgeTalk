// The self-reading cycle, driven without a browser.
//
// The machine is pure so every phase and every boundary is reachable from a test.
// The fault these exist to prevent was a suspension that could never end, which no
// amount of looking at the code had caught: it needed the cycle run past the hold.

import { describe, expect, it } from 'vitest'
import {
  BOTTOM_HOLD_MS,
  DESCENT_PX,
  DESCENT_TICKS_PER_STEP,
  MANUAL_HOLD_MS,
  REWIND_PX,
  START_HOLD_MS,
  TICK_MS,
  TOP_HOLD_MS,
  focused,
  initialState,
  suspended,
  tick,
  type AutoScrollState,
  type ScrollView,
} from './autoscroll'

const view = (scrollTop: number, maxScrollTop = 1000): ScrollView => ({
  scrollTop,
  maxScrollTop,
})

// run drives the machine for a number of ticks against a view that moves with it,
// recomputed each tick exactly as the hook does against a real element.
function run(state: AutoScrollState, ticks: number, start = 0, max = 1000) {
  let scrollTop = start
  let current = state
  for (let index = 0; index < ticks; index++) {
    const result = tick(current, view(scrollTop, max))
    current = result.state
    scrollTop += result.delta
  }
  return { state: current, scrollTop }
}

/** ticksFor converts a hold into the number of ticks that exhausts it. */
const ticksFor = (ms: number) => Math.ceil(ms / TICK_MS)

const reading = (): AutoScrollState => ({
  phase: 'down',
  waitMs: 0,
  ticksToStep: DESCENT_TICKS_PER_STEP,
  opening: false,
})

describe('opening', () => {
  it('holds still before the first descent', () => {
    const { state, scrollTop } = run(initialState(), ticksFor(START_HOLD_MS) - 1)
    expect(state.phase).toBe('pauseTop')
    expect(scrollTop).toBe(0)
  })

  it('starts reading down once the start hold has run', () => {
    expect(run(initialState(), ticksFor(START_HOLD_MS)).state.phase).toBe('down')
  })

  // A dialog focuses something inside itself as it opens. That is not a reader, so the
  // start hold survives it rather than becoming the shorter manual hold.
  it('ignores focus arriving while the start hold runs', () => {
    const opening = initialState()
    expect(focused(opening)).toBe(opening)
    const partSpent = run(opening, ticksFor(START_HOLD_MS) - 1).state
    expect(focused(partSpent).phase).toBe('pauseTop')
  })

  // The opening ends the moment the hold runs out, not at the first movement, so a
  // reader arriving between the two is still heard.
  it('treats focus as a reader once the start hold has run', () => {
    const started = run(initialState(), ticksFor(START_HOLD_MS)).state
    expect(started.opening).toBe(false)
    expect(focused(started).phase).toBe('manual')
  })

  it('ends the opening when a reader takes over during the start hold', () => {
    const taken = suspended(initialState())
    expect(taken.opening).toBe(false)
    expect(focused(taken).phase).toBe('manual')
  })
})

describe('the reading pass', () => {
  it('advances a pixel every second tick rather than every tick', () => {
    expect(run(reading(), 1).scrollTop).toBe(0)
    expect(run(reading(), 2).scrollTop).toBe(DESCENT_PX)
    expect(run(reading(), 20).scrollTop).toBe(10 * DESCENT_PX)
  })

  it('stops exactly at the end and holds there', () => {
    const { state, scrollTop } = run(reading(), 4, 998, 1000)
    expect(scrollTop).toBe(1000)
    expect(state.phase).toBe('pauseBottom')
  })

  it('rewinds far faster than it reads', () => {
    expect(REWIND_PX).toBeGreaterThan(DESCENT_PX * DESCENT_TICKS_PER_STEP)
  })
})

describe('the far end and the way back', () => {
  it('rewinds after the bottom hold, then holds at the top', () => {
    const atBottom: AutoScrollState = {
      phase: 'pauseBottom',
      waitMs: BOTTOM_HOLD_MS,
      ticksToStep: DESCENT_TICKS_PER_STEP,
      opening: false,
    }
    const started = run(atBottom, ticksFor(BOTTOM_HOLD_MS), 1000)
    expect(started.state.phase).toBe('up')

    // Exactly the ticks the rewind takes, so the top hold is caught at full length
    // rather than part spent. Running longer reads down again, which is correct
    // behaviour and simply a different assertion.
    const home = run(started.state, Math.ceil(1000 / REWIND_PX), 1000)
    expect(home.scrollTop).toBe(0)
    expect(home.state.phase).toBe('pauseTop')
    expect(home.state.waitMs).toBe(TOP_HOLD_MS)
  })
})

describe('a reader taking over', () => {
  // The fault that prompted the port. The cycle suspended correctly and then never
  // came back, because the old driver re-armed the hold on every tick.
  it('RESUMES once the manual hold has run', () => {
    const { state, scrollTop } = run(suspended(reading()), ticksFor(MANUAL_HOLD_MS), 600)
    expect(state.phase).toBe('down')
    expect(scrollTop).toBe(600)
  })

  it('stays still for the whole of the manual hold', () => {
    const { state, scrollTop } = run(
      suspended(reading()),
      ticksFor(MANUAL_HOLD_MS) - 1,
      600,
    )
    expect(state.phase).toBe('manual')
    expect(scrollTop).toBe(600)
  })

  it('resumes from where the reader left it rather than from the top', () => {
    const settled = run(suspended(reading()), ticksFor(MANUAL_HOLD_MS) + 2, 600)
    expect(settled.scrollTop).toBe(600 + DESCENT_PX)
  })

  it('rewinds instead of descending when the reader stopped at the very end', () => {
    const { state } = run(suspended(reading()), ticksFor(MANUAL_HOLD_MS), 1000)
    expect(state.phase).toBe('up')
  })

  it('suspends from any phase without losing the position', () => {
    for (const phase of ['down', 'up', 'pauseTop', 'pauseBottom'] as const) {
      const held = suspended({ phase, waitMs: 0, ticksToStep: 1, opening: false })
      expect(held.phase).toBe('manual')
      expect(held.waitMs).toBe(MANUAL_HOLD_MS)
    }
  })
})

describe('a surface that does not overflow', () => {
  it('consumes nothing at all, so attaching the cycle to it is free', () => {
    const opening = initialState()
    const result = tick(opening, view(0, 0))
    expect(result.delta).toBe(0)
    expect(result.state).toBe(opening)
  })
})
