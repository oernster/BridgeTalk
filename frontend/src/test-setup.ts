// What jsdom does not provide, provided.
//
// jsdom performs no layout, so the two observers the overflow hook uses are either
// absent or answer about a page with no geometry. Both are stubbed rather than
// worked around in each test: a hook that constructs an observer would otherwise
// throw the moment any component using it is rendered, which is most of them.
//
// The stubs are inert on purpose. Nothing here fakes a measurement, so a test that
// wants a region to overflow says so by setting the two properties the hook reads;
// see the overflow tests. A stub that invented a size would be a test asserting
// against the stub rather than against the component.

class InertObserver {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}

if (!('ResizeObserver' in globalThis)) {
  ;(globalThis as unknown as { ResizeObserver: unknown }).ResizeObserver = InertObserver
}

// jsdom performs no scrolling either, so scrollTo is absent from an element rather
// than being a no-op. The log scrolls itself to the newest line whenever one arrives,
// which would throw on every render without this.
if (typeof Element.prototype.scrollTo !== 'function') {
  Element.prototype.scrollTo = function scrollTo(): void {}
}
