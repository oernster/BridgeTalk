// Geometry for tests of the focus ring.
//
// jsdom performs no layout, so every element reports no offset parent. The ring passes
// over a stop that is not on screen, which it reads from that, so left alone it finds no
// stops at all and no test of it can say anything. layOut states the shape of the page
// instead: an attached element's offset parent is its parent. What is stated is the
// page's shape; what a test asserts is what the ring then does with it.

/** layOut makes attached elements report an offset parent. */
export function layOut(): void {
  Object.defineProperty(HTMLElement.prototype, 'offsetParent', {
    configurable: true,
    get(this: HTMLElement) {
      return this.parentElement
    },
  })
}

/** unlayOut takes the stated geometry away again. */
export function unlayOut(): void {
  delete (HTMLElement.prototype as unknown as Record<string, unknown>).offsetParent
}
