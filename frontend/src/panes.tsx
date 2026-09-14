// The status and settings panes. The pane is switched rather than stacked, so choosing a
// nav-band button replaces what is below it instead of opening a dialog over it.

import { useEffect, useRef, useState } from 'react'
import { api, on, type Reaction, type State } from './api'
import { Chooser, useChooser } from './chooser'
import { useOverflowStop } from './hooks'
import { playedOutcome } from './indicator'
import { useProductName } from './productName'
import {
  castTagline,
  coveredLine,
  journalTagline,
  momentsTagline,
  statusTagline,
} from './statusWords'

/**
 * Card renders one labelled figure with the lines saying what it means beneath it, drawn
 * smaller than the figure in the secondary text colour (FR-716). A shortfall that is
 * expected has to say so where it is shown; sending the reader to another pane to find
 * out is what a defect feels like.
 */
function Card({
  label,
  value,
  plain,
  lines,
}: {
  label: string
  value: string
  plain?: boolean
  lines: string[]
}) {
  return (
    <div className="card">
      <div className="label">{label}</div>
      <div className={plain ? 'value plain' : 'value'}>{value}</div>
      {lines.map((line) => (
        <div className="tagline" key={line}>
          {line}
        </div>
      ))}
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
  const name = useProductName()
  // A tagline naming the product waits for About to name it; the page keeps no copy of the name.
  const naming = (tagline: (named: string) => string) => (name === '' ? [] : [tagline(name)])

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
    if (log.length === 0) return
    const delta = event.key === 'ArrowDown' ? 1 : -1
    const next = (row + delta + log.length) % log.length
    setRow(next)
    // The arrows are swallowed above, so the log would not scroll to the row they reach
    // on its own; a row walked past the edge would be selected out of sight.
    rowsRef.current?.children[next]?.scrollIntoView?.({ block: 'nearest' })
  }

  return (
    <>
      <h2>Status</h2>
      <p className="lede">
        The journal and the status file are watched here; every decision is recorded
        below, including the ones that produced no sound.
      </p>

      <div className="cards">
        <Card label="Cast" value={state ? state.voiceDisplay : '...'} lines={[castTagline]} />
        {/* FR-716: a figure short of the whole is a gap in the recordings or a machine voice
            still making its lines, never a fault, so the line beneath says which. The figure
            keeps the value colour every card uses rather than a warning one. */}
        <Card
          label="Moments covered"
          value={state ? `${state.bound} of ${state.total}` : '...'}
          lines={state ? [momentsTagline, coveredLine(state)] : [momentsTagline]}
        />
        <Card
          label="Journal"
          value={state?.journalDir ?? '...'}
          plain
          lines={naming(journalTagline)}
        />
        <Card
          label="Status file"
          value={state?.statusPath ?? '...'}
          plain
          lines={naming(statusTagline)}
        />
      </div>

      {/* FR-238: the window opens over a journal directory that cannot be watched, so
          this is where the run says it is hearing nothing from the game. */}
      {state?.journalProblem ? (
        <p className="callout refused" role="alert">
          {state.journalProblem}
          <br />
          Nothing the game does will be heard until another journal directory is chosen on
          the Settings pane.
        </p>
      ) : null}

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
                <span className={`badge ${entry.outcome === playedOutcome ? 'played' : ''}`}>
                  {entry.outcome}
                </span>
                <span className="cue">{entry.cue}</span>
                {/* FR-234: the clip alone, with nothing standing in for a missing one. An
                    event name here only repeated the start of the cue id beside it. */}
                <span className="clip">{entry.clip}</span>
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

      {/* FR-238: until a press answers, the row carries why startup could not watch the
          directory it names. A press's own answer takes its place. */}
      <Chooser
        label="Journal directory"
        path={state?.journalDir ?? '...'}
        outcome={
          journalSaid ??
          (state?.journalProblem ? { refused: true, text: state.journalProblem } : null)
        }
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
