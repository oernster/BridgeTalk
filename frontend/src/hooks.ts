// The two behaviours that are the same in every one of these applications: the
// explicit focus ring and the self-reading scroll.

import { useCallback, useEffect, useRef, useState } from 'react'
import {
  TICK_MS,
  focused,
  initialState,
  suspended,
  tick,
  type AutoScrollState,
} from './autoscroll'

/* ------------------------------------------------------------------- ring */

// Elements that own their arrow keys, so the ring is left with Tab.
//
// A text field uses them for the caret. Taking those away to step the ring would
// leave it with no way to move within its own text, which is a worse trade than the
// one rule it breaks; Tab and Shift+Tab still leave it, so nothing is trapped.
//
// Two controls are deliberately NOT in here. A select's arrows are handled below,
// because letting Down change the cast voice without the list ever opening is a
// change nobody asked for. A slider keeps Up and Down for its value, which is the
// internal cursor every group stop has, while Left and Right step the ring: it
// answers to both natively, so taking the horizontal pair costs it nothing and
// leaving them would trap the ring on it.
const ARROW_OWNERS = new Set(['INPUT', 'TEXTAREA'])

/** A slider reads its value from the horizontal arrows too, so it must not keep them. */
function ownsArrows(target: HTMLElement | null): boolean {
  if (target === null || !ARROW_OWNERS.has(target.tagName)) return false
  return !(target instanceof HTMLInputElement && target.type === 'range')
}

/**
 * useRing wires one explicit focus ring over a container.
 *
 * Tab and Right move forward, Shift+Tab and Left move back; both wrap. The
 * horizontal arrows are tested first so they step the ring everywhere and focus is
 * never trapped. Stops are recomputed on every move, so a rebuilt list is handled and
 * a control that has just become disabled is skipped rather than stalling the ring.
 */
export function useRing(container: React.RefObject<HTMLElement | null>, enabled = true) {
  // Where the ring was when it last moved; null while it has never moved. Focus can
  // be somewhere off the ring, most often on an open menu's item, so leaving from
  // there has to continue from the stop that opened it rather than jumping back to
  // the first one. Null is the neutral start: forward means the first stop, back
  // means the last.
  const mark = useRef<number | null>(null)

  const stops = useCallback((): HTMLElement[] => {
    const root = container.current
    if (!root) return []
    const found = root.querySelectorAll<HTMLElement>('[data-stop]')
    return Array.from(found).filter(
      (element) =>
        !element.hasAttribute('disabled') &&
        element.getAttribute('aria-hidden') !== 'true' &&
        element.offsetParent !== null,
    )
  }, [container])

  const step = useCallback(
    (delta: number) => {
      const list = stops()
      if (list.length === 0) return
      const active = document.activeElement as HTMLElement | null
      const found = active ? list.indexOf(active) : -1
      const neutral = delta > 0 ? 0 : list.length - 1
      const raw =
        found >= 0 ? found + delta : mark.current === null ? neutral : mark.current + delta
      const next = (raw + list.length) % list.length
      // Leaving an open popup onto a stop that drops one of its own opens that one too, so
      // a menu bar carries its dropped state along. It happens here, in the press itself:
      // a handler on the popup would never hear it, since the popup is gone by then.
      const leavingPopup = active?.closest('[data-popup]') != null
      mark.current = next
      list[next].focus()
      if (leavingPopup && list[next].hasAttribute('data-drops')) list[next].click()
    },
    [stops],
  )

  useEffect(() => {
    if (!enabled) return
    const onKey = (event: KeyboardEvent) => {
      // A ring answers only while its surface is the one on top. A dialog over the window
      // holds a ring of its own, so the window's must not also move focus behind the scrim.
      const root = container.current
      if (root === null || !isTopmostSurface(root)) return

      const target = event.target as HTMLElement | null
      const forward = !event.shiftKey && (event.key === 'Tab' || event.key === 'ArrowRight')
      const back = event.key === 'ArrowLeft' || (event.key === 'Tab' && event.shiftKey)

      // A select drops its list on Down and retracts on Up. Both are swallowed either
      // way: the browser's own meaning for them is to change the value silently,
      // which is a choice made without ever showing the choices.
      if (target instanceof HTMLSelectElement) {
        if (event.key === 'ArrowDown') {
          event.preventDefault()
          // An older webview may not have it; it also refuses without a recent user
          // gesture. The swallowed key is still the point either way, because the
          // alternative is changing the value with the list shut.
          try {
            target.showPicker?.()
          } catch {
            // Nothing to fall back to; leaving the value alone is the behaviour.
          }
          return
        }
        if (event.key === 'ArrowUp') {
          event.preventDefault()
          return
        }
      }

      // Those elements keep their own arrows; Tab still leaves them.
      if (ownsArrows(target) && event.key !== 'Tab') return

      if (forward) {
        event.preventDefault()
        step(1)
      } else if (back) {
        event.preventDefault()
        step(-1)
      }
    }

    // Capture on the document rather than bubble on the window: a key that something
    // between the two consumes never reaches a bubble listener, while the ring has to
    // see every press to be the ring.
    document.addEventListener('keydown', onKey, true)
    return () => document.removeEventListener('keydown', onKey, true)
  }, [container, step, enabled])

  return { step, stops }
}

/**
 * useOverflowStop reports whether a scrolling region currently overflows.
 *
 * A region earns a place on the ring by scrolling: focus that can do nothing is a
 * dead press. One that fits its viewport scrolls nowhere, so it drops off the ring
 * until the window or the content changes and it overflows again.
 *
 * `present` says when the region exists to be measured. A dialog body is rendered
 * only while its dialog is open, so at the moment this hook first runs the ref holds
 * nothing and React has no way to announce that a node arrived later. Without that
 * flag the region is measured once against a body that was never there; it then stays
 * off the ring for the whole life of the window. Panes that are always rendered leave
 * it at its default.
 *
 * The content is watched rather than a snapshot of the children taken at setup.
 * Every region here fills from the backend after it mounts, so the children measured
 * at setup are placeholders that are then REPLACED: a resize observer holding the old
 * nodes hears nothing about the ones that took their place. Watching the subtree for
 * changes catches the arrival either way, while the resize observer on the region
 * itself catches the window being dragged.
 */
export function useOverflowStop(
  region: React.RefObject<HTMLElement | null>,
  present = true,
) {
  const [overflows, setOverflows] = useState(false)

  useEffect(() => {
    const element = region.current
    if (!element) {
      setOverflows(false)
      return
    }
    const measure = () => setOverflows(element.scrollHeight > element.clientHeight)
    measure()
    const resizes = new ResizeObserver(measure)
    resizes.observe(element)
    const changes = new MutationObserver(measure)
    changes.observe(element, { childList: true, subtree: true, characterData: true })
    return () => {
      resizes.disconnect()
      changes.disconnect()
    }
  }, [region, present])

  return overflows
}

/**
 * useFirstStop focuses a dialog's first usable control when it opens.
 *
 * Dialogs are the deliberate opposite of the main window: the window starts neutral
 * with nothing focused, a dialog starts on its first stop. The user opened it to do
 * the one thing it is for, so making them press Tab first costs a keystroke and tells
 * them nothing.
 */
export function useFirstStop(
  container: React.RefObject<HTMLElement | null>,
  open: boolean,
) {
  useEffect(() => {
    if (!open) return
    const first = container.current?.querySelector<HTMLElement>(
      '[data-stop]:not([disabled])',
    )
    first?.focus()
  }, [container, open])
}

/* ------------------------------------------------------------ auto-scroll */

// MANUAL_EVENTS are the inputs that count as reading by hand, so they suspend the
// cycle. mousedown covers a press on the native scrollbar as well as on the content.
// Focus arriving in the surface is watched separately, because the dialog's own
// opening focus is not a reader and has to leave the start hold alone.
//
// Watching for these rather than for a scroll position we did not set is the whole
// correction. Inferring it from the position could not tell the reader's scroll from
// its own, so the suspension it armed stayed armed and the surface never read itself
// again for as long as it was open.
const MANUAL_EVENTS = ['wheel', 'mousedown', 'touchstart', 'keydown'] as const

/**
 * useAutoScroll makes a surface read itself.
 *
 * It holds still on open, descends slowly, holds at the end, rewinds quickly and
 * repeats. Any manual input suspends the cycle for a moment and it then resumes from
 * wherever the reader left it, never switching off. It acts only while the content
 * actually overflows, so attaching it to a surface that currently fits is free.
 *
 * The cycle is NOT gated on prefers-reduced-motion. On Windows that query follows the
 * general animation switch, which people turn off for speed rather than for motion
 * sensitivity, so gating on it would silently remove a feature they never declined.
 * Anyone who does not want a surface to read itself stops it by touching it, which is
 * what the manual suspension is for.
 */
export function useAutoScroll(
  surface: React.RefObject<HTMLElement | null>,
  active: boolean,
) {
  const state = useRef<AutoScrollState>(initialState())

  useEffect(() => {
    if (!active) return
    const element = surface.current
    if (!element) return
    state.current = initialState()

    // A frozen surface takes no input at all. Nothing reaching a surface under a modal
    // can be a reader of it, while acting on it would corrupt the very state the freeze
    // exists to keep: the phase would come back suspended rather than where it was.
    const onManualInput = () => {
      if (!isTopmostSurface(element)) return
      state.current = suspended(state.current)
    }
    const onFocus = () => {
      if (!isTopmostSurface(element)) return
      state.current = focused(state.current)
    }
    for (const type of MANUAL_EVENTS) {
      element.addEventListener(type, onManualInput, { passive: true })
    }
    element.addEventListener('focusin', onFocus)

    const timer = window.setInterval(() => {
      if (!isTopmostSurface(element)) return
      const view = {
        scrollTop: element.scrollTop,
        maxScrollTop: element.scrollHeight - element.clientHeight,
      }
      const { state: next, delta } = tick(state.current, view)
      state.current = next
      if (delta !== 0) element.scrollTop = view.scrollTop + delta
    }, TICK_MS)

    return () => {
      window.clearInterval(timer)
      for (const type of MANUAL_EVENTS) {
        element.removeEventListener(type, onManualInput)
      }
      element.removeEventListener('focusin', onFocus)
    }
  }, [surface, active])
}

/**
 * isTopmostSurface reports whether the surface is the one being looked at. The window's
 * ring stands aside by it for a dialog's ring; a surface reading itself freezes by it.
 *
 * Two surfaces reading at once compete for the same eye, which is reachable here: the
 * guide pane reads itself while a dialog opened over it reads itself too. A surface
 * under a modal is FROZEN rather than suspended, its tick skipped whole, so its phase,
 * its position and the rest of its hold are all still there when the modal closes.
 */
function isTopmostSurface(element: HTMLElement): boolean {
  const scrims = document.querySelectorAll('.scrim')
  const own = element.closest('.scrim')
  if (!own) return scrims.length === 0
  return scrims[scrims.length - 1] === own
}
