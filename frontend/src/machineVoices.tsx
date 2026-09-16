// The machine voices on the Cast pane: every one offered as a pill in a panel for its accent and
// sex, the one cast on a card of its own above them with how far making its lines has got and
// whatever went wrong on the way (FR-508, FR-515, FR-518 to FR-522, FR-528, FR-530, FR-720 to
// FR-722).
//
// It sits apart from cast.tsx because machine voices are cast apart from recorded voices (FR-508).
// A machine voice has no folder and no recordings, so the mark that opens what a recorded voice
// covers has nothing to open; a pill holds the cast control alone.

import { useEffect, useState } from 'react'
import { api, on, type MachineVoice, type Making } from './api'
import { castLabel, counted } from './castWords'
import type { Outcome } from './chooser'
import { nothingMade } from './making'

/** Panel is one group of machine voices: its heading and its voices in the order drawn. */
interface Panel {
  group: string
  voices: MachineVoice[]
}

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
 * panelsOf gathers the voices offered into their panels (FR-720). The groups keep the order their
 * first voices arrive in, which is FR-508's, so that order has its one home in the facade; within a
 * group the voices are sorted by name alone, ignoring case.
 */
function panelsOf(voices: MachineVoice[]): Panel[] {
  const groups = new Map<string, MachineVoice[]>()
  for (const voice of voices) {
    groups.set(voice.group, [...(groups.get(voice.group) ?? []), voice])
  }
  return [...groups].map(([group, members]) => ({
    group,
    voices: [...members].sort((a, b) => a.given.localeCompare(b.given, undefined, { sensitivity: 'base' })),
  }))
}

/**
 * MachineVoices offers every machine voice and casts the one chosen.
 *
 * active and machine come from the state: a machine voice is cast only while the cast voice is a
 * machine voice, since a recordings folder may carry the same id (FR-540, FR-722).
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
    void api.castMachineVoice(id, (reason) => setOutcome({ refused: true, text: reason }))
  }

  const castVoice = machine ? voices.find((voice) => voice.id === active) : undefined

  return (
    <>
      <h3>Machine voices</h3>
      <p className="lede">
        The application speaks these itself, making each line on this machine the first time it is
        needed and keeping it.
      </p>

      {/* The cast voice stands apart on a card (FR-721). It is not a control: pressing it would cast
          it again, making and playing its confirmation a second time. */}
      {castVoice !== undefined && (
        <div className="castcard">
          <span className="name">{castLabel(castVoice.name, true)}</span>
          {making.voice === castVoice.id && (
            <>
              <br />
              <span className="meta">{progressText(making, total)}</span>
            </>
          )}
        </div>
      )}

      <div className="machinegroups">
        {panelsOf(voices).map((panel) => (
          <div key={panel.group} className="machinegroup" role="group" aria-label={panel.group}>
            <h4>{panel.group}</h4>
            <div className="pills">
              {panel.voices
                .filter((voice) => voice.id !== castVoice?.id)
                .map((voice) => (
                  /* The pill shows the name alone beneath the group's heading; the whole name
                     reaches a reader through the label (FR-528). */
                  <button
                    key={voice.id}
                    className="pill"
                    data-stop
                    type="button"
                    aria-label={castLabel(voice.name, false)}
                    onClick={() => cast(voice.id)}
                  >
                    {voice.given}
                  </button>
                ))}
            </div>
          </div>
        ))}
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
      {/* A cast deletes the lines no longer current of the voice it casts (FR-527), so what could
          not be deleted is the cast voice's own; a recorded voice cast clears it. */}
      {making.notDeleted !== '' && (
        <p className="callout refused">
          {`The cast voice still has made lines no longer current that could not be deleted: ${making.notDeleted}`}
        </p>
      )}
    </>
  )
}
