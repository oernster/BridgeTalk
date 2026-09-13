// Two rules of the self-reading surface that only show against a real element on a
// timer: the opening focus is not a reader and a frozen surface takes no input.
//
// Kept apart from hooks.test.tsx so neither file approaches the module size cap. As
// there, jsdom performs no layout, so the geometry a surface would have is stated.

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { useRef } from 'react'
import { MANUAL_HOLD_MS, START_HOLD_MS, TICK_MS } from './autoscroll'
import { useAutoScroll, useOverflowStop } from './hooks'

/** Reading renders a focusable surface with the auto-scroll attached. */
function Reading() {
  const surface = useRef<HTMLDivElement>(null)
  useAutoScroll(surface, true)
  return <div ref={surface} tabIndex={0} data-testid="surface" />
}

/** tall states the geometry of a surface whose content overflows. */
function tall(element: HTMLElement) {
  Object.defineProperty(element, 'clientHeight', { configurable: true, value: 100 })
  Object.defineProperty(element, 'scrollHeight', { configurable: true, value: 1000 })
}

// Enough ticks for a resumed descent to have moved a surface several pixels.
const settleMs = TICK_MS * 10

describe('a surface opening', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  // Measured symptom elsewhere: a dialog focusing its own text as it opened turned the
  // start hold into the manual one, so the surface began moving early. The manual hold
  // is the shorter, which is what lets the difference be seen.
  it('keeps the whole start hold when focus arrives as it opens', () => {
    render(<Reading />)
    const surface = screen.getByTestId('surface')
    tall(surface)

    fireEvent.focusIn(surface)
    vi.advanceTimersByTime(MANUAL_HOLD_MS + settleMs)

    expect(surface.scrollTop).toBe(0)

    vi.advanceTimersByTime(START_HOLD_MS)
    expect(surface.scrollTop).toBeGreaterThan(0)
  })

  it('treats focus as a reader once it is already reading', () => {
    render(<Reading />)
    const surface = screen.getByTestId('surface')
    tall(surface)
    vi.advanceTimersByTime(START_HOLD_MS + settleMs)
    const reached = surface.scrollTop

    fireEvent.focusIn(surface)
    vi.advanceTimersByTime(settleMs)

    expect(surface.scrollTop).toBe(reached)
  })
})

describe('a surface beneath a dialog', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  // A frozen surface has no reader, so input reaching it is ignored. Were it heard, the
  // surface would come back suspended when the dialog closed rather than where it was.
  it('ignores input while frozen and resumes in place when the dialog closes', () => {
    render(<Reading />)
    const surface = screen.getByTestId('surface')
    tall(surface)
    vi.advanceTimersByTime(START_HOLD_MS + settleMs)

    const scrim = document.createElement('div')
    scrim.className = 'scrim'
    document.body.appendChild(scrim)
    fireEvent.wheel(surface)
    fireEvent.focusIn(surface)
    const frozenAt = surface.scrollTop
    scrim.remove()

    vi.advanceTimersByTime(settleMs)

    expect(surface.scrollTop).toBeGreaterThan(frozenAt)
  })

  // A dialog's own body is the surface being looked at while its dialog is the topmost;
  // a second dialog opened above it freezes it exactly as it freezes the window beneath.
  it('reads inside the topmost dialog and freezes once another opens above it', () => {
    render(
      <div className="scrim">
        <Reading />
      </div>,
    )
    const surface = screen.getByTestId('surface')
    tall(surface)
    vi.advanceTimersByTime(START_HOLD_MS + settleMs)
    expect(surface.scrollTop).toBeGreaterThan(0)

    const above = document.createElement('div')
    above.className = 'scrim'
    document.body.appendChild(above)
    const frozenAt = surface.scrollTop
    vi.advanceTimersByTime(settleMs)
    above.remove()

    expect(surface.scrollTop).toBe(frozenAt)
  })
})

describe('a region that was never attached', () => {
  // A ref that holds nothing measures as fitting rather than throwing, so a region that
  // has not rendered is simply off the ring.
  it('reports that it does not overflow', () => {
    function Unattached() {
      const region = useRef<HTMLDivElement>(null)
      const overflows = useOverflowStop(region)
      return <span data-testid="unattached" data-overflows={overflows} />
    }
    render(<Unattached />)

    expect(screen.getByTestId('unattached').getAttribute('data-overflows')).toBe('false')
  })
})
