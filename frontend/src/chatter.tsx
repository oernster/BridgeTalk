// The Chatter pane: every moment the game raises, under its category, each with a switch saying
// whether it is spoken for (section 8, FR-725 to FR-741).
//
// The switches live in the application rather than here, because the engine needs them before any
// page loads. So a press never changes the pane itself: it asks, then shows the pane the answer
// describes, which keeps the window from showing a switch the application does not hold.

import { useEffect, useState } from 'react'
import { api, type Chatter, type ChatterCategory, type ChatterMoment } from './api'
import { Dialog, ReadingBody } from './dialogs'

/** switchedOn counts the moments switched on. */
const switchedOn = (moments: ChatterMoment[]) => moments.filter((moment) => moment.on).length

/**
 * Change is a press that may change many moments: the state it switches to, how many moments that
 * changes, where they are for the question to name and the call that makes it.
 */
interface Change {
  on: boolean
  changes: number
  where: string
  make: () => Promise<Chatter>
}

/**
 * Switch is one slider switch (FR-735), announced as a switch by its name with its state (FR-738).
 * It is a button, so Space and Enter press it the way they press every other stop (FR-737).
 */
function Switch({ name, on, onPress }: { name: string; on: boolean; onPress: () => void }) {
  return (
    <button
      className="switch"
      data-stop
      type="button"
      role="switch"
      aria-checked={on}
      aria-label={name}
      onClick={onPress}
    >
      <span className="thumb" aria-hidden="true" />
    </button>
  )
}

/**
 * ChatterPane holds a header that stays put above the list (FR-740): the two buttons that switch every
 * moment (FR-732), then a switch for each category (FR-730, FR-731). Beneath it the list scrolls, every
 * category a group of its own whose heading counts what is on (FR-728) and stays in view while its
 * moments pass (FR-741). A press changing more than one moment asks first (FR-733).
 */
export function ChatterPane() {
  const [chatter, setChatter] = useState<Chatter | null>(null)
  const [problem, setProblem] = useState('')
  const [asking, setAsking] = useState<Change | null>(null)

  useEffect(() => {
    void api
      .chatter()
      .then(setChatter)
      .catch((reason: unknown) => setProblem(String(reason)))
  }, [])

  // settle shows the pane a change was answered with. A switch that applied without being kept
  // comes back with the reason beside it (FR-633); a refusal has only its reason.
  const settle = (made: Promise<Chatter>) => {
    setProblem('')
    void made.then(
      (answer) => {
        setChatter(answer)
        setProblem(answer.problem)
      },
      (reason: unknown) => setProblem(String(reason)),
    )
  }

  // ask makes a change of one moment at once and asks before any larger one (FR-733).
  const ask = (change: Change) => (change.changes > 1 ? setAsking(change) : settle(change.make()))

  const categories = chatter?.categories ?? []
  const moments = categories.flatMap((category) => category.moments)
  const on = switchedOn(moments)
  const off = moments.length - on

  // A category's switch reads on while anything in it is on, so pressing it switches those off;
  // otherwise it switches every moment in it on (FR-731).
  const pressCategory = (category: ChatterCategory) => {
    const lit = switchedOn(category.moments)
    const wanted = lit === 0
    ask({
      on: wanted,
      changes: wanted ? category.moments.length : lit,
      where: ` in ${category.name}`,
      make: () => api.setCategory(category.name, wanted),
    })
  }

  const pressAll = (wanted: boolean) =>
    ask({ on: wanted, changes: wanted ? off : on, where: '', make: () => api.setAllMoments(wanted) })

  const state = asking?.on ? 'on' : 'off'

  return (
    <div className="chatter">
      <div className="chatter-head">
        <h2>Chatter</h2>
        <p className="lede">
          Choose which moments are spoken for. A moment switched off stays quiet whichever voice
          is cast, until it is switched on again.
        </p>

        <div className="chatter-actions">
          {/* FR-734: a button with nothing to change is disabled. */}
          <button className="btn" data-stop type="button" disabled={off === 0} onClick={() => pressAll(true)}>
            Switch all on
          </button>
          <button className="btn" data-stop type="button" disabled={on === 0} onClick={() => pressAll(false)}>
            Switch all off
          </button>
        </div>

        <div className="chatter-categories">
          {categories.map((category) => (
            <div className="chatter-category" key={category.name}>
              {/* The switch comes first so every switch in a column lines up whatever its name's length. */}
              <Switch
                name={category.name}
                on={switchedOn(category.moments) > 0}
                onPress={() => pressCategory(category)}
              />
              <span>{category.name}</span>
            </div>
          ))}
        </div>

        {problem !== '' && (
          <p className="callout refused" role="alert">
            {problem}
          </p>
        )}
      </div>

      <div className="chatter-list">
        {categories.map((category) => (
          <section className="chatter-group" key={category.name} aria-label={category.name}>
            <h3>{`${category.name} (${switchedOn(category.moments)} of ${category.moments.length} on)`}</h3>
            {category.moments.map((moment) => (
              <div className="row" key={moment.cue.id}>
                <span className="grow">
                  {moment.cue.title}
                  <br />
                  <span className="purpose">{moment.cue.purpose}</span>
                </span>
                <Switch
                  name={moment.cue.title}
                  on={moment.on}
                  onPress={() => settle(api.setMoment(moment.cue.id, !moment.on))}
                />
              </div>
            ))}
          </section>
        ))}
      </div>

      <Dialog
        title={`Switch ${asking?.changes} moments ${state}`}
        open={asking !== null}
        onClose={() => setAsking(null)}
        actions={
          <>
            {/* Cancel is written first because the dialog opens focused on its first stop, so
                Enter straight after the press keeps the choice made for each moment. */}
            <button className="btn" data-stop type="button" onClick={() => setAsking(null)}>
              Cancel
            </button>
            <button
              className="btn primary"
              data-stop
              type="button"
              onClick={() => {
                if (asking !== null) settle(asking.make())
                setAsking(null)
              }}
            >
              {`Switch ${asking?.changes} ${state}`}
            </button>
          </>
        }
      >
        <ReadingBody ready>
          <p>
            {`${asking?.changes} moments${asking?.where ?? ''} will be switched ${state}, replacing the choice made for each of them.`}
          </p>
        </ReadingBody>
      </Dialog>
    </div>
  )
}
