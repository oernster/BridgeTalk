// The focus ring and the self-reading scroll.
//
// Both hooks read geometry, which jsdom does not have: it performs no layout, so every
// element reports no offset parent and no scroll height. Left alone the ring would find
// no stops and the scroll would find nothing to scroll, so both tests state the
// geometry they mean explicitly. That is the opposite of faking the result: what is
// stated is the SHAPE of the page; what is asserted is what the hook then does
// with it.

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { useRef } from 'react'
import { useAutoScroll, useFirstStop, useOverflowStop, useRing } from './hooks'

/**
 * layOut makes attached elements report an offset parent.
 *
 * The ring skips a stop that is not on screen, which it reads from offsetParent being
 * null. In jsdom that is true of every element, so without this the ring correctly
 * finds nothing and no test of it can say anything.
 */
function layOut() {
  Object.defineProperty(HTMLElement.prototype, 'offsetParent', {
    configurable: true,
    get(this: HTMLElement) {
      return this.parentElement
    },
  })
}

function unlayOut() {
  delete (HTMLElement.prototype as unknown as Record<string, unknown>).offsetParent
}

/** Ringed renders a container of stops with the ring installed over it. */
function Ringed({ enabled = true }: { enabled?: boolean }) {
  const shell = useRef<HTMLDivElement>(null)
  useRing(shell, enabled)
  return (
    <div ref={shell}>
      <button data-stop type="button">
        first
      </button>
      <input data-stop aria-label="text" type="text" />
      <input data-stop aria-label="slider" type="range" />
      <select data-stop aria-label="chooser">
        <option value="a">a</option>
        <option value="b">b</option>
      </select>
      <button data-stop type="button" disabled>
        skipped
      </button>
      <button data-stop type="button">
        last
      </button>
    </div>
  )
}

/**
 * named identifies whatever holds focus, by its label or its words.
 *
 * Nothing focused reads as null rather than as the body: an unfocused document points
 * activeElement at the body, whose text is the whole page, so a test asserting that
 * focus went nowhere would otherwise be handed the page and pass on anything.
 */
function named(): string | null {
  const active = document.activeElement as HTMLElement | null
  if (active === null || active === document.body) return null
  return active.getAttribute('aria-label') ?? active.textContent
}

describe('the focus ring', () => {
  beforeEach(layOut)
  afterEach(unlayOut)

  // Tab and Right move forward, Shift+Tab and Left move back; both wrap. The
  // horizontal arrows are tested first so they step the ring everywhere rather than
  // being trapped by whatever holds focus.
  it('steps forward on Tab and on Right, from a neutral start', () => {
    render(<Ringed />)

    fireEvent.keyDown(document, { key: 'Tab' })
    expect(named()).toBe('first')
    fireEvent.keyDown(document, { key: 'ArrowRight' })
    expect(named()).toBe('text')
  })

  it('steps back on Shift+Tab and on Left, from a neutral start', () => {
    render(<Ringed />)

    fireEvent.keyDown(document, { key: 'ArrowLeft' })
    expect(named()).toBe('last')
    fireEvent.keyDown(document, { key: 'Tab', shiftKey: true })
    expect(named()).toBe('chooser')
  })

  // The ring wraps at both ends, so it can be crossed from either direction and never
  // stalls at an end.
  it('wraps at both ends', () => {
    render(<Ringed />)
    screen.getByText('last').focus()

    fireEvent.keyDown(document, { key: 'Tab' })
    expect(named()).toBe('first')
    fireEvent.keyDown(document, { key: 'Tab', shiftKey: true })
    expect(named()).toBe('last')
  })

  // A control that has just become disabled is skipped rather than stalling the ring
  // on a stop that can do nothing.
  it('skips a stop that cannot be used', () => {
    render(<Ringed />)
    screen.getByLabelText('chooser').focus()

    fireEvent.keyDown(document, { key: 'Tab' })

    expect(named()).toBe('last')
  })

  // A text field uses its arrows for the caret. Taking those away to step the ring
  // would leave it with no way to move within its own text; Tab still leaves it.
  it('leaves a text field its own arrows and still lets Tab out', () => {
    render(<Ringed />)
    const field = screen.getByLabelText('text')
    field.focus()

    fireEvent.keyDown(field, { key: 'ArrowRight' })
    expect(named()).toBe('text')

    fireEvent.keyDown(field, { key: 'Tab' })
    expect(named()).toBe('slider')
  })

  // A slider answers to both pairs of arrows natively, so taking the horizontal pair
  // costs it nothing while leaving them would trap the ring on it.
  it('takes the horizontal arrows back from a slider', () => {
    render(<Ringed />)
    const slider = screen.getByLabelText('slider')
    slider.focus()

    fireEvent.keyDown(slider, { key: 'ArrowRight' })

    expect(named()).toBe('chooser')
  })

  // Letting Down change the cast voice without the list ever opening is a choice made
  // without showing the choices, so both vertical arrows are swallowed on a select.
  it('swallows the vertical arrows on a chooser rather than changing it silently', () => {
    render(<Ringed />)
    const chooser = screen.getByLabelText('chooser') as HTMLSelectElement
    chooser.focus()
    const picker = vi.fn()
    ;(chooser as unknown as { showPicker: () => void }).showPicker = picker

    fireEvent.keyDown(chooser, { key: 'ArrowDown' })
    expect(picker).toHaveBeenCalled()
    expect(named()).toBe('chooser')

    fireEvent.keyDown(chooser, { key: 'ArrowUp' })
    expect(named()).toBe('chooser')
  })

  // An older webview may not have it; it also refuses without a recent user gesture.
  // The swallowed key is still the point either way: the alternative is changing the
  // value with the list shut.
  it('leaves the value alone where the list cannot be dropped', () => {
    render(<Ringed />)
    const chooser = screen.getByLabelText('chooser') as HTMLSelectElement
    chooser.focus()
    ;(chooser as unknown as { showPicker: () => void }).showPicker = () => {
      throw new Error('a gesture is required')
    }

    fireEvent.keyDown(chooser, { key: 'ArrowDown' })

    expect(named()).toBe('chooser')
    expect(chooser.value).toBe('a')
  })

  it('does nothing at all while it is switched off', () => {
    render(<Ringed enabled={false} />)

    fireEvent.keyDown(document, { key: 'Tab' })

    expect(named()).toBeFalsy()
  })

  it('leaves a key it has no meaning for alone', () => {
    render(<Ringed />)

    fireEvent.keyDown(document, { key: 'q' })

    expect(named()).toBeFalsy()
  })

  // A container with nothing on the ring cannot be stepped, which must be a quiet
  // nothing rather than an index off the end of an empty list.
  it('is quiet over a container holding no stops', () => {
    function Empty() {
      const shell = useRef<HTMLDivElement>(null)
      useRing(shell)
      return <div ref={shell} />
    }
    render(<Empty />)

    fireEvent.keyDown(document, { key: 'Tab' })

    expect(named()).toBeFalsy()
  })
})

describe('the first stop of a dialog', () => {
  beforeEach(layOut)
  afterEach(unlayOut)

  // Dialogs are the deliberate opposite of the main window: the window starts neutral
  // with nothing focused, a dialog starts on its first stop, because the user opened
  // it to do the one thing it is for.
  it('takes focus when the dialog opens, skipping a control that cannot be used', () => {
    function Framed({ open }: { open: boolean }) {
      const frame = useRef<HTMLDivElement>(null)
      useFirstStop(frame, open)
      return (
        <div ref={frame}>
          <button data-stop type="button" disabled>
            skipped
          </button>
          <button data-stop type="button">
            usable
          </button>
        </div>
      )
    }
    const { rerender } = render(<Framed open={false} />)
    expect(named()).toBeFalsy()

    rerender(<Framed open />)

    expect(named()).toBe('usable')
  })
})

/**
 * Region renders a scrolling area with the overflow hook over it.
 *
 * lines controls how many children it holds, because the hook watches the child list:
 * an attribute changing on one child is not a change it hears about; every region
 * in the application grows by having its children REPLACED once the backend answers.
 */
function Region({ lines }: { lines: number }) {
  const body = useRef<HTMLDivElement>(null)
  const overflows = useOverflowStop(body)
  return (
    <div ref={body} data-testid="region" data-overflows={overflows}>
      {Array.from({ length: lines }, (_, index) => (
        <span key={index}>a line</span>
      ))}
    </div>
  )
}

/** measure states the geometry an element would have if the page had layout. */
function measure(element: HTMLElement, clientHeight: number, scrollHeight: number) {
  Object.defineProperty(element, 'clientHeight', { configurable: true, value: clientHeight })
  Object.defineProperty(element, 'scrollHeight', { configurable: true, value: scrollHeight })
}

describe('a scrolling region on the ring', () => {
  // A region earns its place on the ring by scrolling: focus that can do nothing is a
  // dead press. One that fits its viewport scrolls nowhere and drops off.
  it('is off the ring while its content fits', () => {
    render(<Region lines={1} />)

    expect(screen.getByTestId('region').getAttribute('data-overflows')).toBe('false')
  })

  // The content is watched rather than snapshotted, because every region here fills
  // from the backend after it mounts: the children measured at setup are placeholders
  // that are then REPLACED.
  it('joins the ring once its content overflows', async () => {
    const { rerender } = render(<Region lines={1} />)
    const region = screen.getByTestId('region')
    measure(region, 100, 400)

    // A change to the child list is what the hook watches for, so one is made.
    rerender(<Region lines={20} />)

    await vi.waitFor(() =>
      expect(region.getAttribute('data-overflows')).toBe('true'),
    )
  })
})

describe('a surface that reads itself', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  /** Reading renders a surface with the auto-scroll attached. */
  function Reading({ active }: { active: boolean }) {
    const surface = useRef<HTMLDivElement>(null)
    useAutoScroll(surface, active)
    return <div ref={surface} data-testid="surface" />
  }

  // It holds still on open, then descends. The descent is what proves the cycle is
  // running rather than merely attached.
  it('descends on its own once it has held still', () => {
    render(<Reading active />)
    const surface = screen.getByTestId('surface')
    measure(surface, 100, 1000)

    vi.advanceTimersByTime(20000)

    expect(surface.scrollTop).toBeGreaterThan(0)
  })

  // Any input of the reader's own suspends the cycle and hands the surface back. It
  // never switches off; it resumes from wherever they left it.
  it('suspends when the reader touches it', () => {
    render(<Reading active />)
    const surface = screen.getByTestId('surface')
    measure(surface, 100, 1000)
    vi.advanceTimersByTime(20000)
    const reached = surface.scrollTop

    fireEvent.wheel(surface)
    vi.advanceTimersByTime(1000)

    expect(surface.scrollTop).toBe(reached)
  })

  it('does nothing at all while it is switched off', () => {
    render(<Reading active={false} />)
    const surface = screen.getByTestId('surface')
    measure(surface, 100, 1000)

    vi.advanceTimersByTime(20000)

    expect(surface.scrollTop).toBe(0)
  })

  // Two surfaces reading at once compete for the same eye, which is reachable: the
  // guide pane reads itself while a dialog opened over it reads itself too. The one
  // underneath is FROZEN rather than suspended, so its place is still there when the
  // dialog closes.
  it('freezes while a dialog is open over it', () => {
    render(<Reading active />)
    const surface = screen.getByTestId('surface')
    measure(surface, 100, 1000)

    const scrim = document.createElement('div')
    scrim.className = 'scrim'
    document.body.appendChild(scrim)
    vi.advanceTimersByTime(20000)
    expect(surface.scrollTop).toBe(0)

    scrim.remove()
    vi.advanceTimersByTime(20000)
    expect(surface.scrollTop).toBeGreaterThan(0)
  })
})
