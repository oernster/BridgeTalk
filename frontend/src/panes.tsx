// The status and settings panes. The pane is switched rather than stacked, so choosing a
// nav-band button replaces what is below it instead of opening a dialog over it.

import { useEffect, useRef, useState } from 'react'
import { api, on, type Reaction, type State } from './api'
import { Chooser, useChooser } from './chooser'
import { useOverflowStop } from './hooks'

/**
 * Card renders one labelled figure, with a line beneath it where the figure would
 * otherwise be read as a fault. A shortfall that is expected has to say so where it
 * is shown; sending the reader to another pane to find out is what a defect feels
 * like.
 */
function Card({
  label,
  value,
  plain,
  hint,
}: {
  label: string
  value: string
  plain?: boolean
  hint?: string
}) {
  return (
    <div className="card">
      <div className="label">{label}</div>
      <div className={plain ? 'value plain' : 'value'}>{value}</div>
      {hint ? <div className="hint">{hint}</div> : null}
    </div>
  )
}

/**
 * HomePane shows what the application is doing and every decision it has made.
 *
 * The reaction log is the diagnostic surface: it records the silent outcomes too, so
 * a cue that never speaks explains itself instead of leaving the user guessing. It is
 * one ring stop whose rows are walked with Up and Down; it does not read itself,
 * because it is a surface to act on rather than one to read through.
 */
export function HomePane({ state }: { state: State | null }) {
  const [log, setLog] = useState<Reaction[]>([])
  const [row, setRow] = useState(0)
  const rowsRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    void api.reactions().then(setLog)
    return on('reaction', (...data: unknown[]) => {
      setLog((previous) => [...previous, data[0] as Reaction].slice(-200))
    })
  }, [])

  useEffect(() => {
    rowsRef.current?.scrollTo({ top: rowsRef.current.scrollHeight })
  }, [log.length])

  // The log earns its place on the ring by scrolling. A log short enough to read
  // whole has nowhere to go, so it is skipped rather than costing a dead press.
  const reachable = useOverflowStop(rowsRef)

  const onKey = (event: React.KeyboardEvent) => {
    if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
    event.preventDefault()
    const delta = event.key === 'ArrowDown' ? 1 : -1
    setRow((current) => (log.length === 0 ? 0 : (current + delta + log.length) % log.length))
  }

  return (
    <>
      <h2>Monitoring</h2>
      <p className="lede">
        The journal and the status file are watched here; every decision is recorded
        below, including the ones that produced no sound.
      </p>

      <div className="cards">
        <Card label="Cast" value={state ? `${state.voice}` : '...'} />
        {/* A shortfall is a gap in the recordings rather than a fault in the
            application; the hint says what the missing cues do and where they are
            named, so it is not read as one. */}
        <Card
          label="Cues served"
          value={state ? `${state.bound} of ${state.total}` : '...'}
          hint="The rest stay silent. The mark beside each voice in the cast pane names them."
        />
        <Card label="Journal" value={state?.journalDir ?? '...'} plain />
        <Card label="Status file" value={state?.statusPath ?? '...'} plain />
      </div>

      {/* The state, not a second control. Muting is done from the band, which is on
          screen whichever pane is open; what cannot be seen there is whether an audio
          device was found at all, so that is what this line is for. */}
      <div className="row">
        <span className="grow">
          {state?.muted ? 'Playback is muted.' : 'Playback is live.'}
          {state?.silent ? ' No audio device was available, so nothing will be heard.' : ''}
          {/* Said only when it has happened. A line reading "0" every run trains the
              eye to skip it, which is the one thing it must not do on the run where
              the number is not zero. */}
          {state && state.stalls > 0
            ? ` The audio device ran dry ${state.stalls} time${state.stalls === 1 ? '' : 's'}` +
              ` this run, the longest for ${state.worstStall} ms, which is heard as the`
              + ' speech breaking up.'
            : ''}
        </span>
      </div>

      <div
        className="log"
        {...(reachable ? { 'data-stop': true } : {})}
        tabIndex={reachable ? 0 : -1}
        role="listbox"
        aria-label="Reaction log"
        onKeyDown={onKey}
      >
        <div className="logrows" ref={rowsRef}>
          {log.length === 0 ? (
            <div className="empty">Nothing yet. Start the game and this will fill up.</div>
          ) : (
            log.map((entry, index) => (
              <div
                className="logrow"
                key={`${entry.at}-${entry.cue}-${index}`}
                role="option"
                aria-selected={index === row}
              >
                <span className="time">{entry.at}</span>
                <span className={`badge ${entry.outcome === 'played' ? 'played' : ''}`}>
                  {entry.outcome}
                </span>
                <span className="cue">{entry.cue}</span>
                <span className="clip">{entry.clip || entry.event}</span>
              </div>
            ))
          )}
        </div>
      </div>
    </>
  )
}


/**
 * SettingsPane holds what can be changed today and states plainly what cannot yet.
 * An empty pane with no explanation reads as a defect; a short honest note does not.
 *
 * Mute and volume are not here. Both live in the band, which is on screen whichever
 * pane is open, so repeating them here would offer the same control twice with no
 * way to tell which one is authoritative. The recordings directory is not here either:
 * it is chosen on the Missing takes pane, beside the voices it holds.
 */
export function SettingsPane({ state }: { state: State | null }) {
  // What the last press of Browse did, shown under its row until the next press.
  const [journalSaid, browseJournal] = useChooser(api.chooseJournalDir, 'The journal directory')

  // Why the login entry could not be written, where it could not be. The box itself
  // is drawn from the state rather than from a local copy, so a refused change simply
  // leaves it where it was; without a reason beside it that would read as a dead
  // control, which is the fault the Browse buttons had.
  const [bootProblem, setBootProblem] = useState('')

  const setBoot = (wanted: boolean) => {
    setBootProblem('')
    void api.setLaunchOnBoot(wanted).catch((reason: unknown) => setBootProblem(String(reason)))
  }

  return (
    <>
      <h2>Settings</h2>
      <p className="lede">Where the application reads from.</p>

      <Chooser
        label="Journal directory"
        path={state?.journalDir ?? '...'}
        outcome={journalSaid}
        onBrowse={browseJournal}
      />

      <div className="row">
        <label className="grow" htmlFor="launch-on-boot">
          Start it when I sign in
          <br />
          <span className="hint">
            It waits in the notification area until the game runs.
          </span>
        </label>
        <input
          id="launch-on-boot"
          className="check"
          data-stop
          type="checkbox"
          checked={state?.launchOnBoot ?? false}
          onChange={(event) => setBoot(event.target.checked)}
          // Space toggles a box on its own; Enter does nothing to one. Every other
          // stop in this window answers both, so a box that ignored Enter would be
          // the single control where the ring's own rule stopped holding.
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault()
              setBoot(!(state?.launchOnBoot ?? false))
            }
          }}
        />
      </div>

      {bootProblem !== '' && (
        <p className="callout refused" role="alert">
          {bootProblem}
        </p>
      )}

      <p className="lede" style={{ marginTop: 18 }}>
        The journal directory is found under your own profile until you choose one.
        Choosing it here takes effect at once and is remembered for next time; the
        command-line flag does the same job for a single run and wins over a choice made
        here. The recordings directory is chosen on the Missing takes pane. Choosing an
        output device is not built: the application speaks through whichever device
        Windows is set to use.
      </p>
    </>
  )
}
