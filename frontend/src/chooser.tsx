// A directory the application reads from, with the button that moves it and the answer
// the last press got.
//
// Two panes hold one: Missing takes for the recordings, Settings for the journal. Each
// row draws its own answer directly beneath it. Collected at the foot of a pane instead,
// a refusal about the recordings once sat under the journal directory and read as though
// the journal were the thing that had gone wrong.

import { useState } from 'react'
import type { Refused } from './api'

/**
 * Outcome is what a press of Browse came to; null where the row has not answered.
 *
 * Refusal is a field rather than two separate pieces of state, so a pane cannot render a
 * message without knowing which kind it is; that pairing is the whole point, since the
 * fault being fixed was a refusal drawn in the colour of an explanation.
 */
export type Outcome = { refused: boolean; text: string } | null

/**
 * useChooser runs one directory chooser and remembers what its last press came to.
 *
 * Both outcomes are reported, not only the refusal: a directory holding no voices is kept
 * out; saying nothing on the way out made the button read as broken rather than
 * strict. A cancelled dialog is the one quiet case, since nothing changed and the reader
 * knows they cancelled. onTaken runs once a directory is accepted.
 */
export function useChooser(
  pick: (refused: Refused) => Promise<string | null>,
  what: string,
  onTaken?: () => void,
): [Outcome, () => void] {
  const [outcome, setOutcome] = useState<Outcome>(null)

  const browse = () => {
    setOutcome(null)
    void pick((reason) => setOutcome({ refused: true, text: reason })).then((taken) => {
      // Nothing back is a directory that was not taken: refused, which the handler above has
      // already said, else no window to open a chooser in. An empty string is the third quiet
      // case, a dialog the reader cancelled.
      if (taken === null || taken === '') {
        return
      }
      // FR-234: the row above already shows the new path, since both choosers announce
      // the new state before they answer; saying it again here would repeat it.
      setOutcome({ refused: false, text: `${what} was changed.` })
      onTaken?.()
    })
  }

  return [outcome, browse]
}

/**
 * Chooser is one directory row: what it is called, where it points now and the button
 * that moves it, with the last answer drawn beneath it. A refusal is drawn as a refusal,
 * since an explanation styled as body prose sits among a pane's other grey paragraphs
 * and is read as one of them.
 */
export function Chooser({
  label,
  path,
  outcome,
  onBrowse,
}: {
  label: string
  path: string
  outcome: Outcome
  onBrowse: () => void
}) {
  return (
    <>
      <div className="row">
        <span className="grow">
          {label}
          <br />
          <span className="hint">{path}</span>
        </span>
        <button className="btn" data-stop type="button" onClick={onBrowse}>
          Browse
        </button>
      </div>

      {outcome !== null && (
        <p
          className={outcome.refused ? 'callout refused' : 'callout taken'}
          role={outcome.refused ? 'alert' : 'status'}
        >
          {outcome.text}
        </p>
      )}
    </>
  )
}
