// The cast pane: every voice found and the one currently speaking.
//
// It sits in its own file for the reason the audition pane and the guide do. A
// pane is a slice that comes out whole: it carries its own row, its own label and the
// dialog it opens, none of which the other panes have any use for.

import { useEffect, useState } from 'react'
import { api, type Voice } from './api'
import { MomentsIcon } from './icons'
import { MomentsDialog } from './moments'

/** casting is the part every cast voice holds, whoever it is. */
const casting = "your ship's voice"

/**
 * castLabel names the act a row offers.
 *
 * A cast row names the part as well as the fact, because "Grace is cast" says the job
 * has been given out without saying what the job is. The part is the same for every
 * voice. That it does not vary is why it is written here beside
 * the sentence it belongs to rather than carried across with the voice.
 */
function castLabel(voice: Voice, cast: boolean): string {
  return cast ? `${voice.name} is cast as ${casting}` : `Cast ${voice.name}`
}

/**
 * VoiceRow draws one voice: the control that casts it and the control that shows what
 * it covers.
 *
 * Every row can be cast. A directory that resolved no recording never becomes a voice,
 * so there is no row here to refuse and nothing to explain a refusal with.
 */
function VoiceRow({
  voice,
  cast,
  onSelect,
  onShowMoments,
}: {
  voice: Voice
  cast: boolean
  onSelect: (name: string) => void
  onShowMoments: (name: string) => void
}) {
  return (
    /* A row is a container holding two controls rather than one control, so it
       neither takes focus nor draws a ring of its own: a ring belongs to the thing
       that is focused; here that is one of the two buttons inside. It is also what a
       button inside a button would have been, which no browser accepts. */
    <div className="voice" aria-current={cast}>
      <button
        className="pick"
        data-stop
        type="button"
        title={castLabel(voice, cast)}
        onClick={() => onSelect(voice.name)}
      >
        <span className="name">{castLabel(voice, cast)}</span>
        <br />
        <span className="meta">{`${voice.inUse.toLocaleString()} usable recordings`}</span>
      </button>
      <span className="coverage">
        {/* An icon rather than words, so it reads as a control at the end of the row
            it acts on. The name it needs arrives twice: aria-label for a reader, who
            wants to know WHOSE moments; the band's own tooltip for an eye, which
            wants to know what pressing it does. */}
        <button
          className="momentsbtn"
          data-stop
          data-label="Moments spoken for"
          type="button"
          aria-label={`Moments ${voice.name} speaks for`}
          onClick={() => onShowMoments(voice.name)}
        >
          <MomentsIcon />
        </button>
      </span>
    </div>
  )
}

/**
 * CastPane lists every voice found and casts the one chosen, each with the mark that
 * opens what it covers.
 *
 * The metaphor is casting rather than selecting, because that is what choosing a
 * voice for a role is.
 */
export function CastPane({
  active,
  libraryRoot,
  onSelect,
}: {
  active: string
  libraryRoot: string
  onSelect: (name: string) => void
}) {
  const [voices, setVoices] = useState<Voice[]>([])
  // Whether the answer has arrived. An empty list means nothing was found; so does a
  // list that has not been fetched yet. Telling somebody their voices are missing for
  // the fraction of a second before the real answer lands is worse than saying nothing
  // at all.
  const [loaded, setLoaded] = useState(false)
  // The voice whose breakdown is open. Empty means no dialog, which is one piece of
  // state rather than a flag and a name that could disagree with each other.
  const [showing, setShowing] = useState('')

  useEffect(() => {
    void api.voices().then((found) => {
      setVoices(found)
      setLoaded(true)
    })
  }, [active])

  return (
    <>
      <h2>Cast</h2>
      <p className="lede">
        Cast the voice that speaks for your ship. The mark beside each one opens the
        whole account of what they speak for and what they were never recorded for.
      </p>

      {/* Nothing found is a state the window has to be able to sit in. The application
          cannot make a recording appear and cannot guess where one went, so the only
          useful thing it can do is say exactly where it looked: an absence with an
          address is something the reader can act on, while an absence on its own reads
          as a fault in the application. It waits for the answer before saying so,
          because an empty list and an unanswered one look the same on screen. */}
      {loaded && voices.length === 0 ? (
        <p className="callout">
          <b>No voices found.</b> Nothing under{' '}
          <code>{libraryRoot || 'the chosen directory'}</code> is named for a cue. A voice
          is one directory per person, holding their recordings; name each recording (or
          the folder it sits in) after the cue it answers. Another directory can be
          pointed at with the <code>-library</code> option.
        </p>
      ) : null}

      <div className="voices">
        {voices.map((voice) => (
          <VoiceRow
            key={voice.name}
            voice={voice}
            cast={voice.name === active}
            onSelect={onSelect}
            onShowMoments={setShowing}
          />
        ))}
      </div>

      <MomentsDialog
        voice={showing}
        open={showing !== ''}
        onClose={() => setShowing('')}
      />
    </>
  )
}
