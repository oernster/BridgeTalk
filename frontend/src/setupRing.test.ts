// The setup page's keyboard ring (FR-808).
//
// The setup page has no build step and no test runner of its own, so its ring is loaded here as
// the very script the page ships, run once the way the page runs it, then driven with real key
// presses over a page shaped like one of its screens. What is asserted is where focus goes. That
// the page loads the script at all is held by the structural suite.

import { afterAll, beforeAll, beforeEach, describe, expect, it } from 'vitest'
import ring from '../../installer/frontend/dist/setup-ring.js?raw'
import { layOut, unlayOut } from './testLayout'

// An indirect eval runs the script in the global scope, as a script tag does: its key listener
// goes on the document and nothing of it leaks into this module's own names.
const evaluate = eval
evaluate(ring)

/** screen is the Install screen reduced to its stops: the theme button, two boxes, the footer. */
const screen = `
  <div class="head"><button id="theme">Theme</button></div>
  <div class="body">
    <label class="option"><input type="checkbox" id="start-menu" checked /></label>
    <label class="option"><input type="checkbox" id="desktop" checked /></label>
  </div>
  <div class="footer"><button id="cancel">Cancel</button><button id="install">Install</button></div>
`

/** byId finds one element of the screen, failing loudly rather than handing back null. */
function byId(id: string): HTMLElement {
  const found = document.getElementById(id)
  if (found === null) throw new Error(`no #${id} on the screen`)
  return found
}

/** press sends one key to whatever holds focus and answers with the event it sent. */
function press(key: string, shiftKey = false): KeyboardEvent {
  const event = new KeyboardEvent('keydown', { key, shiftKey, bubbles: true, cancelable: true })
  const target = document.activeElement ?? document.body
  target.dispatchEvent(event)
  return event
}

/** focusedId names what holds focus. */
function focusedId(): string {
  return (document.activeElement as HTMLElement | null)?.id ?? ''
}

beforeAll(layOut)
afterAll(unlayOut)

beforeEach(() => {
  document.body.innerHTML = screen
})

describe('the setup ring', () => {
  it('steps forward on Tab and on Right, wrapping at the end', () => {
    byId('install').focus()

    const right = press('ArrowRight')
    expect(focusedId()).toBe('theme')
    expect(right.defaultPrevented).toBe(true)

    press('Tab')
    expect(focusedId()).toBe('start-menu')
  })

  it('steps back on Shift+Tab and on Left, wrapping at the start', () => {
    byId('theme').focus()

    const left = press('ArrowLeft')
    expect(focusedId()).toBe('install')
    expect(left.defaultPrevented).toBe(true)

    press('Tab', true)
    expect(focusedId()).toBe('cancel')
  })

  it('passes over a control that is disabled or hidden', () => {
    ;(byId('desktop') as HTMLInputElement).disabled = true
    Object.defineProperty(byId('cancel'), 'offsetParent', { get: () => null })
    byId('start-menu').focus()

    press('ArrowRight')
    expect(focusedId()).toBe('install')

    press('ArrowLeft')
    expect(focusedId()).toBe('start-menu')
  })

  it('ticks a box on Enter as Space does', () => {
    const box = byId('desktop') as HTMLInputElement
    let changes = 0
    box.addEventListener('change', () => {
      changes++
    })
    box.focus()

    const enter = press('Enter')
    expect(box.checked).toBe(false)
    expect(changes).toBe(1)
    expect(enter.defaultPrevented).toBe(true)
  })

  it('offers the body only while it holds more than fits', () => {
    const body = document.querySelector('.body') as HTMLElement
    byId('theme').focus()
    press('ArrowRight')
    expect(focusedId()).toBe('start-menu')
    expect(body.tabIndex).toBe(-1)

    Object.defineProperty(body, 'scrollHeight', { configurable: true, get: () => 500 })
    Object.defineProperty(body, 'clientHeight', { configurable: true, get: () => 100 })
    byId('theme').focus()
    press('ArrowRight')
    expect(document.activeElement).toBe(body)
    expect(body.tabIndex).toBe(0)
  })
})
