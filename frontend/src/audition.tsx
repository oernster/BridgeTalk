// The audition pane: hearing a voice deliberately.
//
// It lives in its own file rather than in panes.tsx for two reasons. It is the only
// pane holding a running action; panes.tsx is already close to the size cap.

import { useCallback, useEffect, useState } from 'react'
import { api, on, type Group, type MachineVoice, type Playback, type Voice } from './api'
import { PlayIcon } from './icons'

/**
 * machinePrefix marks a machine voice's value in the chooser. A recordings folder may carry
 * a machine voice's id as its name, so the two are kept apart; no folder name holds a slash.
 */
const machinePrefix = 'machine/'

/** applicationHeading heads the groups Chatter lists under no category, the cue from the application's (FR-750). */
const applicationHeading = 'This application'

/** Section is one category heading with the groups listed under it. */
interface Section {
  heading: string
  groups: Group[]
}

/**
 * sectionsOf gathers the groups offered under their category headings. The backend sends them in
 * Chatter's category order, a group in no category last, so each run of one category is one section
 * (FR-750).
 */
function sectionsOf(offered: Group[]): Section[] {
  const sections: Section[] = []
  for (const group of offered) {
    const heading = group.category || applicationHeading
    const last = sections[sections.length - 1]
    if (last?.heading === heading) last.groups.push(group)
    else sections.push({ heading, groups: [group] })
  }
  return sections
}

/** chosenFor is the chooser's value for a voice: a folder's name as it is; a machine voice's id marked. */
function chosenFor(name: string, machine: boolean): string {
  if (!name) return ''
  return machine ? machinePrefix + name : name
}

/**
 * AuditionPane plays a random sample from a chosen group.
 *
 * The voice being auditioned starts as the one that is cast but is not tied to it,
 * because hearing a voice before committing to it is the entire point of an
 * audition. Casting stays a separate act on the cast pane. The recorded voices come
 * first, then every machine voice (FR-545); `machine` says the cast voice is one.
 */
export function AuditionPane({ cast, machine = false }: { cast: string; machine?: boolean }) {
  const [voices, setVoices] = useState<Voice[]>([])
  const [machines, setMachines] = useState<MachineVoice[]>([])
  // Whether the lists have arrived. An empty list and an unanswered one look the same,
  // so the pane says no voices exist only once it has been told so.
  const [loaded, setLoaded] = useState(false)
  const [voice, setVoice] = useState(chosenFor(cast, machine))
  const [groups, setGroups] = useState<Group[]>([])
  const [playing, setPlaying] = useState('')
  // Whether anything at all is sounding or being made, the ship's reactions to the game
  // included. Every button is held while it is, because a press must never cut a clip
  // short (FR-236) or set a second line making (FR-547); the backend ignores one that
  // slips through regardless.
  const [busy, setBusy] = useState(false)
  const [failure, setFailure] = useState('')

  useEffect(() => {
    void Promise.all([api.voices(), api.machineVoices()]).then(([found, offered]) => {
      setVoices(found)
      setMachines(offered)
      setLoaded(true)
    })
  }, [])

  // The pane opens on whoever is cast. Later changes to the cast do not drag the
  // audition along with them, so a switch made here survives.
  useEffect(() => {
    setVoice((current) => current || chosenFor(cast, machine))
  }, [cast, machine])

  // The machine voice chosen, by id; empty while a recorded voice is.
  const machineId = voice.startsWith(machinePrefix) ? voice.slice(machinePrefix.length) : ''

  useEffect(() => {
    if (!voice) return
    setFailure('')
    const asked = machineId ? api.machineAuditionGroups() : api.auditionGroups(voice)
    void asked.then(setGroups)
  }, [voice, machineId])

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
      // Nothing clears the pulse or frees the buttons on the way out. An audition answers when
      // the clip STARTS, so the end of the sound arrives later as a playback event; only a
      // refusal is known to mean no sound at all, which is what this handler is for.
      const refused = (reason: string) => {
        setPlaying('')
        setBusy(false)
        setFailure(reason)
      }
      void (machineId
        ? api.auditionMachineVoice(machineId, group.key, refused)
        : api.audition(voice, group.key, refused))
    },
    [voice, machineId],
  )

  // A group whose moments Chatter has all switched off is not offered (FR-747). The pane asks for its
  // groups each time it opens, so switching one back on shows the group next time.
  const offered = groups.filter((group) => !group.switchedOff)
  // Everything the voice has is switched off, which reads differently from a voice with nothing
  // (FR-748).
  const allSwitchedOff = offered.length === 0 && groups.length > 0
  const total = offered.reduce((sum, group) => sum + group.clips, 0)
  // No voice exists to choose. The chooser still stands at its full width holding
  // None, because an empty control shrunk to its arrow reads as a rendering fault
  // rather than as an answer.
  const none = loaded && voices.length === 0 && machines.length === 0

  return (
    <>
      <h2>Audition</h2>
      <p className="lede">
        Hear what a voice actually has. Each button plays one sample at random from
        that part of the game, so pressing it again can give you a different take. A
        machine voice makes a line it has not made yet first, which takes a moment. This
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
                {!machine && item.name === cast ? ' (cast)' : ''}
              </option>
            ))}
            {machines.map((item) => (
              <option key={machinePrefix + item.id} value={machinePrefix + item.id}>
                {item.name}
                {machine && item.id === cast ? ' (cast)' : ''}
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
      ) : allSwitchedOff ? (
        <p className="lede">Chatter has switched off everything this voice could be heard on.</p>
      ) : offered.length === 0 ? (
        <p className="lede">This voice has nothing to audition.</p>
      ) : (
        <>
          <p className="meta">
            {offered.length} groups, {total.toLocaleString()} samples between them.
          </p>
          {sectionsOf(offered).map((section) => (
            <section key={section.heading} aria-label={section.heading}>
              <h3>
                <span className="heading-pill">{section.heading}</span>
              </h3>
              <div className="groups">
                {section.groups.map((group) => (
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
            </section>
          ))}
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
