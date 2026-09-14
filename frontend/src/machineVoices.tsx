// The machine voices on the Cast pane: every one offered, the one cast with how far making its
// lines has got and whatever went wrong on the way (FR-508, FR-515, FR-518 to FR-522, FR-528,
// FR-530).
//
// It sits apart from cast.tsx because machine voices are cast apart from recorded voices (FR-508).
// A machine voice's row holds the cast control alone: it has no folder and no recordings, so the
// mark that opens what a recorded voice covers has nothing to open.

import { useEffect, useState } from 'react'
import { api, on, type MachineVoice, type Making } from './api'
import { castLabel, counted } from './castWords'
import type { Outcome } from './chooser'
import { nothingMade } from './making'

/** notCast is what an uncast machine voice's row says: its lines are made as they are needed. */
const notCast = 'Its lines are made as they are needed.'

/**
 * progressText reads how far making has got for the cast voice: the lines made out of the lines
 * the script holds (FR-515), then the moments with a made line out of every moment (FR-522).
 */
function progressText(making: Making, total: number): string {
  return (
    `${making.current.toLocaleString()} of ${counted(making.total, 'line', 'lines')} made; ` +
    `${making.cuesServed.toLocaleString()} of ${counted(total, 'moment', 'moments')} spoken`
  )
}

/**
 * MachineVoices lists every machine voice and casts the one chosen.
 *
 * active and machine come from the state: a machine voice is marked cast only while the cast voice
 * is a machine voice, since a recordings folder may carry the same id (FR-540).
 */
export function MachineVoices({
  active,
  machine,
  total,
}: {
  active: string
  machine: boolean
  total: number
}) {
  const [voices, setVoices] = useState<MachineVoice[]>([])
  const [making, setMaking] = useState<Making>(nothingMade)
  const [outcome, setOutcome] = useState<Outcome>(null)

  useEffect(() => {
    void api.machineVoices().then(setVoices)
  }, [])

  // Asked once where making stands, then told of every change, so the figure rises while the pane
  // is open. Asked again on each cast, since a cast starts a making of its own.
  useEffect(() => {
    void api.making().then(setMaking)
    return on('making', (payload) => setMaking(payload as Making))
  }, [active])

  const cast = (id: string) => {
    setOutcome(null)
    void api
      .castMachineVoice(id)
      .catch((reason: unknown) => setOutcome({ refused: true, text: String(reason) }))
  }

  const castID = machine ? active : ''

  return (
    <>
      <h3>Machine voices</h3>
      <p className="lede">
        The application speaks these itself, making each line on this machine the first time it is
        needed and keeping it.
      </p>
      <div className="voices">
        {voices.map((voice) => {
          const isCast = voice.id === castID
          return (
            /* One control a row, in the row a recorded voice uses, so both lists read alike. */
            <div key={voice.id} className="voice" aria-current={isCast}>
              <button className="pick" data-stop type="button" onClick={() => cast(voice.id)}>
                <span className="name">{castLabel(voice.name, isCast)}</span>
                <br />
                <span className="meta">
                  {isCast && making.voice === voice.id ? progressText(making, total) : notCast}
                </span>
              </button>
            </div>
          )
        })}
      </div>

      {outcome !== null && (
        <p className="callout refused" role="alert">
          {outcome.text}
        </p>
      )}
      {making.failed.length > 0 && (
        <p className="callout refused">
          <span>These lines could not be made:</span>
          {making.failed.map((failure) => (
            <span key={`${failure.cue.id}-${failure.line}`}>
              <br />
              <span>{`${failure.cue.title}, line ${failure.line}: ${failure.reason}`}</span>
            </span>
          ))}
        </p>
      )}
      {making.stopped !== '' && (
        <p className="callout refused">{`Making stopped: ${making.stopped}`}</p>
      )}
      {making.notDeleted !== '' && (
        <p className="callout refused">
          {`The voice cast before still has made lines that could not be deleted: ${making.notDeleted}`}
        </p>
      )}
    </>
  )
}
