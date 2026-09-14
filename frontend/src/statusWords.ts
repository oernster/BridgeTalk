// What the Status pane's cards say beneath their figures (FR-716).
//
// Each card carries a tagline saying what it means. Moments covered adds a line for the situation
// the cast voice is in, so a figure short of the whole is never read as a fault the player cannot
// fix. The words live here rather than in the pane so every one of them is read in one place.

import type { State } from './api'

/** castTagline says what the Cast card means. */
export const castTagline =
  'The voice that speaks when something happens in the game. Change it on the Cast pane.'

/** momentsTagline says what the Moments covered card means. */
export const momentsTagline =
  'A moment is something that happens in the game that a voice can speak for, such as docking.'

/** journalTagline says what the Journal card means, naming the product as About gives it. */
export function journalTagline(name: string): string {
  return (
    'The folder where Elite Dangerous records what happens in your game. ' +
    `${name} listens to it for moments to speak.`
  )
}

/** statusTagline says what the Status file card means, naming the product as About gives it. */
export function statusTagline(name: string): string {
  return (
    "The file the game rewrites as your ship's state changes. " +
    `${name} reads it for things the journal does not record.`
  )
}

/**
 * coveredLine says what the Moments covered figure means for the voice cast. A machine voice's
 * figure counts moments with a line made so far, which rises as the game is played; a recorded
 * voice's counts moments with a take, so the rest are the moments with none.
 */
export function coveredLine(state: State): string {
  if (state.voice === '') return 'Cast a voice to hear the game.'
  if (state.machineVoice) {
    return (
      "A machine voice makes a moment's lines the first time it happens, so this number grows as " +
      'you play. Nothing is missing.'
    )
  }
  const uncovered = state.total - state.bound
  if (uncovered <= 0) return 'Every game moment has a recording.'
  if (uncovered === 1) {
    return (
      'The other moment has no recording yet, so it stays silent. ' +
      'Nothing is wrong: Missing takes lists it and where its recording goes.'
    )
  }
  return (
    `The other ${uncovered} moments have no recording yet, so they stay silent. ` +
    'Nothing is wrong: Missing takes lists them and where each recording goes.'
  )
}
