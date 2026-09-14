// FR-716: the Moments covered line counts what is missing in words that read right for one or many.

import { describe, expect, it } from 'vitest'
import type { State } from './api'
import { coveredLine } from './statusWords'

/** recorded is a recorded voice cast with bound of total moments covered. */
function recorded(bound: number, total: number): State {
  return { ...({} as State), voice: 'Oliver', machineVoice: false, bound, total }
}

describe('coveredLine', () => {
  it('says one missing moment in the singular', () => {
    expect(coveredLine(recorded(255, 256))).toBe(
      'The other moment has no recording yet, so it stays silent. ' +
        'Nothing is wrong: Missing takes lists it and where its recording goes.',
    )
  })

  it('says many missing moments in the plural', () => {
    expect(coveredLine(recorded(1, 256))).toBe(
      'The other 255 moments have no recording yet, so they stay silent. ' +
        'Nothing is wrong: Missing takes lists them and where each recording goes.',
    )
  })
})
