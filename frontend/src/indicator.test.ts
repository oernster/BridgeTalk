// The live indicator's message (FR-719).
//
// The selector is pure, so every row of the table is asserted here without a page. The rows are
// walked from a reading in which every one of them holds at once: taking the condition at the top
// away must reveal the next row, which proves the order as well as the words.

import { describe, expect, it } from 'vitest'
import type { Making, State } from './api'
import { flashMs, indicate, type Reading, type Tone } from './indicator'
import { nothingMade } from './making'
import { watching } from './testState'

/** now is the moment every reading here is taken at. */
const now = 1_000_000

/** makingLines is a machine voice with 13 of its 768 lines made while making goes on. */
const makingLines: Making = { ...nothingMade, voice: 'bf_emma', making: true, current: 13, total: 768 }

/** calm is the application running normally with Grace cast: nothing to say but that it listens. */
const calm: Reading = { state: watching, making: nothingMade, played: null, donateFailedAt: null, now }

/** says asserts the message a reading gives, words and tone together. */
function says(reading: Reading, text: string, tone: Tone) {
  expect(indicate(reading)).toEqual({ text, tone })
}

describe('the live indicator', () => {
  it('takes the first row that holds, in the order the table gives', () => {
    let state: State = {
      ...watching,
      voice: '',
      voiceDisplay: '',
      journalProblem: 'reading the journal directory D:/Nowhere: cannot be found',
      silent: true,
      muted: true,
    }
    let making: Making = { ...makingLines, stopped: 'writing a made line: the disk is full' }
    let reading: Reading = { state, making, played: { at: now, title: 'Docked' }, donateFailedAt: now, now }

    says(reading, 'Could not open a browser for the donation page', 'alert')

    reading = { ...reading, donateFailedAt: null }
    says(reading, 'Not hearing the game: choose a journal folder in Settings', 'alert')

    state = { ...state, journalProblem: '' }
    reading = { ...reading, state }
    says(reading, 'Could not save made lines: see the Cast pane', 'alert')

    making = { ...making, stopped: '' }
    reading = { ...reading, making }
    says(reading, 'No audio device: nothing will be heard', 'alert')

    state = { ...state, silent: false }
    reading = { ...reading, state }
    says(reading, 'Making lines: 13 of 768 ready', 'notice')

    reading = { ...reading, making: nothingMade }
    says(reading, 'Muted', 'muted')

    state = { ...state, muted: false }
    reading = { ...reading, state }
    says(reading, 'Just played: Docked', 'accent')

    reading = { ...reading, played: null }
    says(reading, 'No voice cast', 'muted')

    reading = { ...reading, state: { ...state, voice: 'Grace', voiceDisplay: 'Grace Hart' } }
    says(reading, 'Listening with Grace Hart', 'ambient')
  })

  // FR-719's acceptance: making outranks the mute, so a muted machine voice still says how far it
  // has got.
  it('says how far making has got with playback muted as well', () => {
    says(
      { ...calm, state: { ...watching, muted: true }, making: makingLines },
      'Making lines: 13 of 768 ready',
      'notice',
    )
  })

  // FR-719's acceptance: a refused journal directory is said whatever else holds, a donation
  // failure aside, since that one lasts only a moment.
  it('says the game cannot be heard whatever else holds', () => {
    says(
      {
        ...calm,
        state: { ...watching, journalProblem: 'refused', silent: true, muted: true },
        making: { ...makingLines, stopped: 'the disk is full' },
        played: { at: now, title: 'Docked' },
      },
      'Not hearing the game: choose a journal folder in Settings',
      'alert',
    )
  })

  it('says a donation page could not be opened for four seconds after, then lets it go', () => {
    says(
      { ...calm, donateFailedAt: now - (flashMs - 1) },
      'Could not open a browser for the donation page',
      'alert',
    )
    says({ ...calm, donateFailedAt: now - flashMs }, 'Listening with Grace', 'ambient')
  })

  it('names the moment just played for four seconds after, then lets it go', () => {
    says({ ...calm, played: { at: now - (flashMs - 1), title: 'Docked' } }, 'Just played: Docked', 'accent')
    says({ ...calm, played: { at: now - flashMs, title: 'Docked' } }, 'Listening with Grace', 'ambient')
  })

  it('counts the lines made in the figures a reader expects', () => {
    says(
      { ...calm, making: { ...makingLines, current: 1200, total: 7680 } },
      'Making lines: 1,200 of 7,680 ready',
      'notice',
    )
  })

  // Before the first state arrives nothing about the application is known, so no row reading it
  // can hold; the donation failure reads no state and still says itself.
  it('says nothing before the first state arrives, a donation failure aside', () => {
    says({ ...calm, state: null }, '', 'ambient')
    says(
      { ...calm, state: null, donateFailedAt: now },
      'Could not open a browser for the donation page',
      'alert',
    )
  })
})
