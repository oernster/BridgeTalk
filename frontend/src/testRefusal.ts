// How a suite stands in for a call the facade refused.
//
// Every api call that can be refused takes a handler and answers with nothing rather than
// rejecting, so a suite that replaces the api module has to answer the same way: a fake that
// rejects is testing a shape the api no longer has; the component under it would be asked to catch
// something that never reaches it.
//
// It lives here rather than in each suite because several read it; a rule written twice is two
// statements that can disagree.

/** Refuses answers a call by telling its handler why, then answering with nothing. */
export function refuses<T>(reason: string, nothing: T) {
  return (...args: unknown[]): Promise<T> => {
    const refused = args[args.length - 1]
    if (typeof refused === 'function') {
      ;(refused as (said: string) => void)(reason)
    }
    return Promise.resolve(nothing)
  }
}
