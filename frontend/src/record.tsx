// The record pane: what a voice still has no recording for, with the way to its folder.
//
// Recording itself happens in a program built for it. What only this application knows
// is which moments a voice is missing and exactly which folder each take belongs in, so
// that is the whole of this pane.

import { useEffect, useState } from 'react'
import { api, type Checklist } from './api'
import { grouped } from './moments'

/**
 * RecordPane lists the moments a voice folder has no recording for, each beside the
 * button that opens the folder its take belongs in.
 *
 * Every voice folder is offered, recorded or not. A voice made with Make folders holds
 * nothing until its first take is saved, so it is not yet a voice anywhere else in the
 * window; it is also exactly the one that needs this list.
 */
export function RecordPane({ cast }: { cast: string }) {
  const [folders, setFolders] = useState<string[]>([])
  const [loaded, setLoaded] = useState(false)
  const [voice, setVoice] = useState('')
  const [list, setList] = useState<Checklist | null>(null)
  const [problem, setProblem] = useState('')
  // How many looks have been taken, so a look fetches the folders and the list again.
  const [looks, setLooks] = useState(0)

  useEffect(() => {
    void api
      .voiceDirectories()
      .then((found) => {
        setFolders(found)
        setLoaded(true)
        // The folder already chosen survives a look; otherwise the cast voice, else the
        // first folder there is.
        setVoice((current) =>
          found.includes(current) ? current : found.includes(cast) ? cast : (found[0] ?? ''),
        )
      })
      .catch((reason: unknown) => setProblem(String(reason)))
  }, [cast, looks])

  useEffect(() => {
    if (voice === '') return
    void api
      .checklist(voice)
      .then(setList)
      .catch((reason: unknown) => setProblem(String(reason)))
  }, [voice, looks])

  // A look reads the recordings directory again for the whole window, so a voice filled
  // here appears on the Cast pane too. The list is fetched again whatever that finds.
  const look = () => {
    setProblem('')
    void api
      .rescan()
      .then(
        () => undefined,
        () => undefined,
      )
      .then(() => setLooks((count) => count + 1))
  }

  const open = (id: string) => {
    setProblem('')
    void api.openMomentFolder(voice, id).catch((reason: unknown) => setProblem(String(reason)))
  }

  const empty = loaded && folders.length === 0

  return (
    <>
      <h2>Record</h2>
      <p className="lede">
        Record in any program you like, such as Audacity; save each take as WAV, MP3,
        FLAC or Ogg. Choose a voice, then press Open folder beside a moment: its folder
        opens, ready for the take to be saved into it under any name. Press Look again once
        some are saved.
      </p>

      {empty ? (
        <p className="callout">
          <b>No voice folders yet.</b> Make one on the Cast pane: type a name and press Make
          folders.
        </p>
      ) : (
        <div className="row">
          <label className="field">
            <span className="label">Recording for</span>
            <select data-stop value={voice} onChange={(event) => setVoice(event.target.value)}>
              {folders.map((name) => (
                <option key={name} value={name}>
                  {name}
                </option>
              ))}
            </select>
          </label>
          <button className="btn" data-stop type="button" onClick={look}>
            Look again
          </button>
        </div>
      )}

      {problem !== '' && (
        <p className="callout refused" role="alert">
          {problem}
        </p>
      )}

      {!empty && list !== null && (
        <>
          <p className="meta">
            {`${list.voice} has recordings for ${list.recorded} of ${list.total} moments.`}
          </p>
          {list.missing.length === 0 ? (
            <p className="lede">Every moment has a recording.</p>
          ) : (
            grouped(list.missing).map(([group, entries]) => (
              <section key={group}>
                <h3>{entries[0].heading}</h3>
                {entries.map((item) => (
                  <div className="row" key={item.id}>
                    <span className="grow">
                      {item.title}
                      <br />
                      <span className="hint">{item.id}</span>
                    </span>
                    <button
                      className="btn"
                      data-stop
                      type="button"
                      aria-label={`Open the folder for ${item.title}`}
                      onClick={() => open(item.id)}
                    >
                      Open folder
                    </button>
                  </div>
                ))}
              </section>
            ))
          )}
        </>
      )}
    </>
  )
}
