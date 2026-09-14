// The strip along the foot of the window: the donate button at its left and the live indicator at
// its right (FR-717, FR-718, FR-719).
//
// It comes after the pane in the page, so the ring reaches the donate button after everything above
// it (FR-713). The indicator's message is chosen by indicate() in indicator.ts from what this file
// gathers; nothing here decides a message.

import { useEffect, useState } from 'react'
import { api, on, type Making, type Reaction, type State } from './api'
import { NavButton } from './chrome'
import { DonateArt } from './icons'
import { flashMs, indicate, playedOutcome, type Played } from './indicator'
import { nothingMade } from './making'
import { useProductName } from './productName'

/**
 * donateLabel is the donate button's tooltip and accessible name (FR-718). The product is named as
 * About gives it; until then nothing stands in its place.
 */
function donateLabel(name: string): string {
  return ['Donate to support', name, '(opens your browser)'].filter((part) => part !== '').join(' ')
}

/** Strip draws the foot of the window over the state the shell already holds. */
export function Strip({ state }: { state: State | null }) {
  const name = useProductName()
  const [making, setMaking] = useState<Making>(nothingMade)
  const [played, setPlayed] = useState<Played | null>(null)
  const [donateFailedAt, setDonateFailedAt] = useState<number | null>(null)
  const [now, setNow] = useState(() => Date.now())

  // Asked once where making stands, then told of every change (FR-515).
  useEffect(() => {
    void api.making().then(setMaking)
    return on('making', (payload) => setMaking(payload as Making))
  }, [])

  // A reaction that sounded is kept with the moment it arrived. One that played nothing is not news
  // here; the Status pane's log records it.
  useEffect(
    () =>
      on('reaction', (payload) => {
        const reaction = payload as Reaction
        if (reaction.outcome !== playedOutcome) return
        const at = Date.now()
        // FR-233: the moment under its full title, which the reaction carries beside its id.
        setPlayed({ at, title: reaction.title })
        setNow(at)
      }),
    [],
  )

  // A message about one moment is taken down on time. Each still showing sets a timer for when its
  // flashMs is up; the timers go when the strip does or when either moment changes.
  useEffect(() => {
    const timers = [played?.at ?? null, donateFailedAt]
      .filter((at): at is number => at !== null)
      .map((at) => at + flashMs - Date.now())
      .filter((left) => left > 0)
      .map((left) => window.setTimeout(() => setNow(Date.now()), left))
    return () => timers.forEach((timer) => window.clearTimeout(timer))
  }, [played, donateFailedAt])

  const donate = () => {
    void api.openDonation().catch(() => {
      const at = Date.now()
      setDonateFailedAt(at)
      setNow(at)
    })
  }

  const message = indicate({ state, making, played, donateFailedAt, now })

  return (
    <footer className="strip">
      <NavButton label={donateLabel(name)} current={false} onClick={donate}>
        <DonateArt />
      </NavButton>
      <div className={`indicator tone-${message.tone}`} aria-live="polite">
        {message.text}
      </div>
    </footer>
  )
}
