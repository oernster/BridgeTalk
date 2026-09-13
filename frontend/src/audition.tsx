// The audition pane: hearing a voice deliberately.
//
// It lives in its own file rather than in panes.tsx for two reasons. It is the only
// pane holding a running action; panes.tsx is already close to the size cap.

import { useCallback, useEffect, useState } from 'react'
import { api, on, type Group, type Playback, type Voice } from './api'
import { PlayIcon } from './icons'

/**
 * AuditionPane plays a random sample from a chosen group.
 *
 * The voice being auditioned starts as the one that is cast but is not tied to it,
 * because hearing a voice before committing to it is the entire point of an
 * audition. Casting stays a separate act on the cast pane.
 */
export function AuditionPane({ cast }: { cast: string }) {
  const [voices, setVoices] = useState<Voice[]>([])
  // Whether the list has arrived. An empty list and an unanswered one look the same,
  // so the pane says no voices exist only once it has been told so.
  const [loaded, setLoaded] = useState(false)
  const [voice, setVoice] = useState(cast)
  const [groups, setGroups] = useState<Group[]>([])
  const [playing, setPlaying] = useState('')
  // Whether anything at all is sounding, the ship's reactions to the game included.
  // Every button is held while it is, because a press must never cut a clip short
  // (FR-236); the backend ignores one that slips through regardless.
  const [busy, setBusy] = useState(false)
  const [failure, setFailure] = useState('')

  useEffect(() => {
    void api.voices().then((found) => {
      setVoices(found)
      setLoaded(true)
    })
  }, [])

  // The pane opens on whoever is cast. Later changes to the cast do not drag the
  // audition along with them, so a switch made here survives.
  useEffect(() => {
    setVoice((current) => current || cast)
  }, [cast])

  useEffect(() => {
    if (!voice) return
    setFailure('')
    void api.auditionGroups(voice).then(setGroups)
  }, [voice])

  // The pulse has to end when the sound does; only the backend knows that. The
  // audition call returns the moment the clip starts. The same event says when
  // something starts, which is how a reaction to the game holds the buttons too.
  //
  // A pane opened part way through a clip asks once, so its buttons are held from the
  // start. An event is newer than that answer, so an answer arriving after one is dropped.
  useEffect(() => {
    let heard = false
    const off = on('playback', (...data: unknown[]) => {
      const state = data[0] as Playback | undefined
      heard = true
      setBusy(state?.playing ?? false)
      if (!state?.playing) setPlaying('')
    })
    void api.playing().then((sounding) => {
      if (!heard) setBusy(sounding)
    })
    return off
  }, [])

  const play = useCallback(
    (group: Group) => {
      setPlaying(group.key)
      setBusy(true)
      setFailure('')
      // Nothing clears the pulse or frees the buttons here. audition resolves when the
      // clip STARTS, so the end of the sound arrives later as a playback event; only a
      // failure is known to mean no sound at all.
      void api.audition(voice, group.key).catch((error: unknown) => {
        setPlaying('')
        setBusy(false)
        setFailure(String(error))
      })
    },
    [voice],
  )

  const total = groups.reduce((sum, group) => sum + group.clips, 0)
  // No voice exists to choose. The chooser still stands at its full width holding
  // None, because an empty control shrunk to its arrow reads as a rendering fault
  // rather than as an answer.
  const none = loaded && voices.length === 0

  return (
    <>
      <h2>Audition</h2>
      <p className="lede">
        Hear what a voice actually has. Each button plays one sample at random from
        that part of the game, so pressing it again can give you a different take. This
        ignores the mute, which silences reactions to the game rather than the
        application.
      </p>

      <div className="row">
        <label className="field">
          <span className="label">Auditioning</span>
          <select
            data-stop
            value={voice}
            disabled={none}
            onChange={(event) => setVoice(event.target.value)}
          >
            {none && <option value="">None</option>}
            {voices.map((item) => (
              <option key={item.name} value={item.name}>
                {item.display}
                {item.name === cast ? ' (cast)' : ''}
              </option>
            ))}
          </select>
        </label>
        <button
          className="stopbtn"
          data-stop
          data-label="Stop"
          type="button"
          aria-label="Stop"
          onClick={() => void api.stopAudition()}
        />
      </div>

      {none ? (
        <p className="lede">
          No voices found, so there is nothing to audition yet. Choose the directory
          holding your recordings on the Missing takes pane.
        </p>
      ) : groups.length === 0 ? (
        <p className="lede">This voice has nothing to audition.</p>
      ) : (
        <>
          <p className="meta">
            {groups.length} groups, {total.toLocaleString()} samples between them.
          </p>
          <div className="groups">
            {groups.map((group) => (
              <button
                className="group"
                key={group.key}
                data-stop
                type="button"
                disabled={busy}
                aria-current={playing === group.key}
                title={`Play a random ${group.label} sample`}
                onClick={() => play(group)}
              >
                <PlayIcon />
                <span>
                  <span className="name">{group.label}</span>
                  <br />
                  <span className="meta">
                    {group.clips.toLocaleString()}{' '}
                    {group.clips === 1 ? 'sample' : 'samples'}
                  </span>
                </span>
              </button>
            ))}
          </div>
        </>
      )}

      {failure !== '' && (
        <p className="failure" style={{ marginTop: 14 }}>
          {failure}
        </p>
      )}
    </>
  )
}
