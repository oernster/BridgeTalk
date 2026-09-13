// The keyboard on the setup page: the application's own ring (FR-713), written for a page
// that has no build step to share the application's code through (FR-808).
//
// Tab and Right step forward; Shift+Tab and Left step back; both wrap. The stops are read
// again on every press, because each screen rebuilds its boxes and its footer: a list kept
// from the last screen would name buttons that no longer exist. Enter ticks a box as Space
// does; a button already answers to both.

// setupRing holds the stop the ring last stood on, so a press made while focus rests on no
// stop, as it does while a footer is being rebuilt, carries on from there rather than from
// the first stop.
const setupRing = { mark: null }

// ringStops lists what the keyboard can reach in the order it is drawn: every button and box
// that can be used, plus the body while it holds more than fits. A screen running past the
// window can be read from the keyboard only through its body.
function ringStops() {
    const body = document.querySelector('.body')
    const overflows = body !== null && body.scrollHeight > body.clientHeight
    if (body !== null) body.tabIndex = overflows ? 0 : -1
    return Array.from(document.querySelectorAll('button, input[type="checkbox"], .body')).filter(
        (el) => el.offsetParent !== null && !el.disabled && (el !== body || overflows),
    )
}

// stepRing moves focus one stop along the ring: delta is 1 forward, -1 back.
function stepRing(delta) {
    const stops = ringStops()
    if (stops.length === 0) return
    const found = stops.indexOf(document.activeElement)
    const from = found >= 0 ? found : setupRing.mark
    const next = from === null
        ? (delta > 0 ? 0 : stops.length - 1)
        : (((from + delta) % stops.length) + stops.length) % stops.length
    setupRing.mark = next
    stops[next].focus()
}

// onRingKey answers the keys the ring owns. It listens in the capture phase, so nothing on
// the page can take a press before the ring sees it.
function onRingKey(event) {
    const forward = !event.shiftKey && (event.key === 'Tab' || event.key === 'ArrowRight')
    const back = event.key === 'ArrowLeft' || (event.key === 'Tab' && event.shiftKey)
    if (forward || back) {
        event.preventDefault()
        stepRing(forward ? 1 : -1)
        return
    }
    const target = event.target
    if (event.key === 'Enter' && target instanceof HTMLInputElement && target.type === 'checkbox') {
        event.preventDefault()
        target.click()
    }
}

document.addEventListener('keydown', onRingKey, true)
