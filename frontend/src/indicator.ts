// The live indicator's one message (FR-719).
//
// A pure choice from what the window is already sent: the state, how far making has got, the last
// moment played with when it arrived, when a donation page last failed to open and the time now.
// The table's rows are tried in its order; the first that holds is the message. The tone names the
// colour token its row gives. The stylesheet draws it, so no colour is written here.

import type { Making, State } from './api'

/**
 * flashMs is how long a message about one moment stays: a donation page that could not be opened
 * (FR-718) and a moment just played (FR-719) each show for this long after it happened.
 */
export const flashMs = 4000

/** playedOutcome is the outcome a reaction carries when its moment sounded. */
export const playedOutcome = 'played'

/** tones are the colours a message can take, each named for its colour token. */
export const tones = ['alert', 'notice', 'muted', 'accent', 'ambient'] as const

/** Tone is one of the colours a message can take. */
export type Tone = (typeof tones)[number]

/** Played is a moment that sounded: when it arrived on the page's clock with its full title. */
export interface Played {
  at: number
  title: string
}

/** Reading is everything the message is chosen from. */
export interface Reading {
  state: State | null
  making: Making
  played: Played | null
  /** When handing the donation page to the browser last failed; null where it has not. */
  donateFailedAt: number | null
  now: number
}

/** Message is what the indicator says with the tone it says it in. */
export interface Message {
  text: string
  tone: Tone
}

/** lately reports whether something that happened at `at` is still within flashMs of now. */
function lately(at: number | null, now: number): boolean {
  return at !== null && now - at < flashMs
}

/**
 * indicate chooses the message from the first row of FR-719's table that holds. A row meaning
 * nothing will be heard comes before one that only describes. Before the first state arrives no row
 * reading it can hold, so only a donation failure is said.
 */
export function indicate({ state, making, played, donateFailedAt, now }: Reading): Message {
  if (lately(donateFailedAt, now)) {
    return { text: 'Could not open a browser for the donation page', tone: 'alert' }
  }
  if (state === null) return { text: '', tone: 'ambient' }
  if (state.journalProblem !== '') {
    return { text: 'Not hearing the game: choose a journal folder in Settings', tone: 'alert' }
  }
  if (making.stopped !== '') {
    return { text: 'Could not save made lines: see the Cast pane', tone: 'alert' }
  }
  if (state.silent) return { text: 'No audio device: nothing will be heard', tone: 'alert' }
  if (making.making) {
    const ready = `${making.current.toLocaleString()} of ${making.total.toLocaleString()}`
    return { text: `Making lines: ${ready} ready`, tone: 'notice' }
  }
  if (state.muted) return { text: 'Muted', tone: 'muted' }
  if (played !== null && lately(played.at, now)) {
    return { text: `Just played: ${played.title}`, tone: 'accent' }
  }
  if (state.voice === '') return { text: 'No voice cast', tone: 'muted' }
  return { text: `Listening with ${state.voiceDisplay}`, tone: 'ambient' }
}
