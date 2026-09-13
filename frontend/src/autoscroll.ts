// The gentle self-reading cycle for long reading surfaces: hold still on open, read
// down slowly, hold at the tail, rewind fast, repeat, then step aside the moment the
// reader takes over.
//
// This file is the state machine only, pure and free of the DOM, so every phase and
// every boundary can be driven by a test without a browser. useAutoScroll in hooks.ts
// drives it against a real element on a timer.
//
// It is a port of the same machine in PigeonPost, which is the house reference for
// this behaviour. The earlier version here inferred manual input by comparing the
// scroll position against the one it last set, which is what put the cycle into a
// suspension it could never leave: the mismatch that armed the hold was still true on
// the next tick, so the hold re-armed forever. The driver watches for input EVENTS
// instead, which is what the reference does and why it has never had that fault.

// The pace is the application's, not the surface's. One standard everywhere: if a
// surface seems to want a different pace, the pace is wrong everywhere.
export const TICK_MS = 40

// START_HOLD_MS is the stillness before the first descent, so the reader orients
// before anything moves. It is the opening phase's wait rather than a special case.
export const START_HOLD_MS = 5000

// The descent is one pixel every second tick. The divider is a countdown of ticks
// rather than a slower timer, so holds keep their TICK_MS granularity.
export const DESCENT_PX = 1
export const DESCENT_TICKS_PER_STEP = 2

// BOTTOM_HOLD_MS is long enough to finish reading the tail before the rewind takes
// it away.
export const BOTTOM_HOLD_MS = 5000

// REWIND_PX is a reposition rather than a reading pass, so it travels fast. Never
// read at this pace; never rewind at the reading pace.
export const REWIND_PX = 15

// TOP_HOLD_MS is the breath before the next pass.
export const TOP_HOLD_MS = 2000

// MANUAL_HOLD_MS is the stillness required after any manual reading input before the
// cycle picks up again, from wherever the reader left it. Manual input suspends the
// cycle; it never switches it off.
export const MANUAL_HOLD_MS = 2500

// Two reading movements and three holds. manual is a hold like the others, differing
// only in what it resumes into.
export type Phase = 'down' | 'pauseBottom' | 'up' | 'pauseTop' | 'manual'

export interface AutoScrollState {
  phase: Phase
  /** What is left of the current hold, ignored by the movement phases. */
  waitMs: number
  /** Counts down to the next descent pixel. */
  ticksToStep: number
  /**
   * True until the start hold has run out. A dialog focuses something inside itself as
   * it opens, which is not a reader taking hold: there is nothing yet to take over from.
   * Honouring that arrival would turn the start hold into the shorter manual hold.
   */
  opening: boolean
}

/** ScrollView is the part of a scrollable element the machine reads. */
export interface ScrollView {
  scrollTop: number
  maxScrollTop: number
}

/**
 * initialState opens in the top hold seeded with the start hold, so a fresh surface
 * stands still before it first moves.
 */
export function initialState(): AutoScrollState {
  return {
    phase: 'pauseTop',
    waitMs: START_HOLD_MS,
    ticksToStep: DESCENT_TICKS_PER_STEP,
    opening: true,
  }
}

/**
 * suspended is the state a manual reading input puts the cycle into: a hold, which it
 * leaves at the reader's own position rather than by restarting.
 */
export function suspended(state: AutoScrollState): AutoScrollState {
  return { ...state, phase: 'manual', waitMs: MANUAL_HOLD_MS, opening: false }
}

/**
 * focused is what focus arriving in the surface does: nothing while the start hold is
 * still running, since that is the dialog's own opening focus rather than a reader;
 * the manual suspension once it has run out.
 */
export function focused(state: AutoScrollState): AutoScrollState {
  return state.opening ? state : suspended(state)
}

/**
 * tick advances the cycle by one tick and reports how far the surface should move,
 * already clamped to its bounds.
 *
 * Content that does not overflow consumes nothing: no wait counts down and no phase
 * changes, so attaching the cycle to a surface that currently fits is free.
 */
export function tick(
  state: AutoScrollState,
  view: ScrollView,
): { state: AutoScrollState; delta: number } {
  if (view.maxScrollTop <= 0) return { state, delta: 0 }
  if (state.phase === 'down') return descend(state, view)
  if (state.phase === 'up') return rewind(state, view)
  return hold(state, view)
}

/**
 * hold counts down the current wait; when it runs out it chooses what to resume
 * into: the rewind after the bottom hold, the rewind again after a manual hold left
 * at the very end, otherwise the reading pass.
 */
function hold(
  state: AutoScrollState,
  view: ScrollView,
): { state: AutoScrollState; delta: number } {
  const waitMs = state.waitMs - TICK_MS
  if (waitMs > 0) return { state: { ...state, waitMs }, delta: 0 }
  // Every way out of a hold ends the opening, at the moment the hold runs out rather
  // than at the first movement, so a reader arriving between the two is not missed.
  if (state.phase === 'pauseBottom') {
    return { state: { ...state, phase: 'up', waitMs: 0, opening: false }, delta: 0 }
  }
  if (state.phase === 'manual' && view.scrollTop >= view.maxScrollTop) {
    return { state: { ...state, phase: 'up', waitMs: 0, opening: false }, delta: 0 }
  }
  return {
    state: { phase: 'down', waitMs: 0, ticksToStep: DESCENT_TICKS_PER_STEP, opening: false },
    delta: 0,
  }
}

/** descend advances the reading pass, handing over to the bottom hold on arrival. */
function descend(
  state: AutoScrollState,
  view: ScrollView,
): { state: AutoScrollState; delta: number } {
  const ticksToStep = state.ticksToStep - 1
  if (ticksToStep > 0) return { state: { ...state, ticksToStep }, delta: 0 }
  const remaining = view.maxScrollTop - view.scrollTop
  if (remaining <= DESCENT_PX) {
    return {
      state: {
        phase: 'pauseBottom',
        waitMs: BOTTOM_HOLD_MS,
        ticksToStep: DESCENT_TICKS_PER_STEP,
        opening: false,
      },
      delta: Math.max(0, remaining),
    }
  }
  return { state: { ...state, ticksToStep: DESCENT_TICKS_PER_STEP }, delta: DESCENT_PX }
}

/** rewind travels back at the repositioning pace, handing over to the top hold. */
function rewind(
  state: AutoScrollState,
  view: ScrollView,
): { state: AutoScrollState; delta: number } {
  if (view.scrollTop <= REWIND_PX) {
    return {
      state: {
        phase: 'pauseTop',
        waitMs: TOP_HOLD_MS,
        ticksToStep: DESCENT_TICKS_PER_STEP,
        opening: false,
      },
      delta: -view.scrollTop,
    }
  }
  return { state, delta: -REWIND_PX }
}
