// The Missing takes pane: every voice still missing a recording, then the folder each of
// the chosen voice's missing audio files belongs in.
//
// Recording itself happens in a program built for it. What only this application knows
// is which moments a voice is missing and exactly which folder each take belongs in, so
// that is the whole of this pane.
//
// A cast plugin voice is offered here too, listing the moments it has no take for. Its list
// carries no folder, since a plugin's audio is the plugin's own and this application may not
// write to it: a list with no folder is shown without the path and without the way to open it
// (FR-571). The pane keeps no rule about which kinds of voice have folders; it reads the one
// each list came with.

import { useEffect, useState } from 'react'
import { api, type Checklist } from './api'
import { Chooser, useChooser } from './chooser'

/** stillMissing keeps the checklists with at least one moment unrecorded, in folder order. */
const stillMissing = (lists: Checklist[]) => lists.filter((list) => list.missing.length > 0)

/**
 * MissingTakesPane offers every voice still missing a take and lists, for the one chosen,
 * each moment it has no recording for beside the folder its audio file belongs in.
 *
 * A complete voice is not offered, since it has nothing left to record (FR-316). An empty
 * voice folder is: a voice made with Make folders holds nothing until its first take is
 * saved, so it is not yet a voice anywhere else in the window; it is also exactly the one
 * that needs this list. The chooser is drawn whatever the directory holds, saying why
 * where it has nothing to offer, so the pane is the same window every time (FR-317).
 */
export function MissingTakesPane({
  cast,
  plugin,
  libraryRoot,
}: {
  cast: string
  /** The plugin the cast voice came from, empty for every other kind (FR-569). */
  plugin: string
  // Undefined until the state arrives; empty where no recordings directory is chosen.
  libraryRoot?: string
}) {
  // Every voice folder's checklist, in folder order; null until the first read lands.
  const [lists, setLists] = useState<Checklist[] | null>(null)
  const [voice, setVoice] = useState('')
  const [problem, setProblem] = useState('')
  // How many looks have been taken, so a look reads the folders and their lists again.
  const [looks, setLooks] = useState(0)

  useEffect(() => {
    void api
      .voiceDirectories()
      .then((found) => Promise.all(found.map((name) => api.checklist(name))))
      // The cast plugin voice's list comes first, since it is the voice speaking. Nothing is
      // asked for where no plugin voice is cast, which is almost every run.
      .then(async (folders) =>
        plugin === '' ? folders : [await api.pluginChecklist(), ...folders],
      )
      .then((read) => {
        setLists(read)
        const offered = stillMissing(read).map((list) => list.voice)
        // The voice already chosen survives a look while it still misses something;
        // otherwise the cast voice, else the first voice still missing a take.
        setVoice((current) =>
          offered.includes(current) ? current : offered.includes(cast) ? cast : (offered[0] ?? ''),
        )
      })
      .catch((reason: unknown) => setProblem(String(reason)))
  }, [cast, plugin, looks])

  // A look reads the recordings directory again for the whole window, so a voice filled
  // here appears on the Cast pane too. The lists are read again whatever that finds.
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

  // A directory taken here has already been read by the time it is answered, so its
  // folders are offered straight away rather than after a press of Refresh.
  const [rootSaid, browseRoot] = useChooser(api.chooseLibraryRoot, 'The recordings directory', () =>
    setLooks((count) => count + 1),
  )

  const offered = lists === null ? [] : stillMissing(lists)
  const list = offered.find((each) => each.voice === voice) ?? null
  const noVoices = lists !== null && lists.length === 0
  const allComplete = lists !== null && lists.length > 0 && offered.length === 0
  // What the chooser holds while it has no voice to offer, so it is never left out.
  const standIn =
    lists === null ? 'Reading the recordings' : noVoices ? 'No voices yet' : 'Every voice is complete'

  return (
    <>
      <h2>Missing takes</h2>
      <p className="lede">
        Record in any program you like, such as Audacity; save each take as WAV, MP3,
        FLAC or Ogg. Choose a voice, then press Open folder beside a moment: its folder
        opens, ready for the take to be saved into it under any name. Press Refresh once
        some are saved.
      </p>

      <Chooser
        label="Recordings"
        path={libraryRoot === undefined ? '...' : libraryRoot || 'None chosen yet'}
        outcome={rootSaid}
        onBrowse={browseRoot}
      />

      <div className="row">
        <label className="field">
          <span className="label">Recording for</span>
          <select
            data-stop
            value={offered.length === 0 ? '' : voice}
            disabled={offered.length === 0}
            onChange={(event) => setVoice(event.target.value)}
          >
            {offered.length === 0 ? (
              <option value="">{standIn}</option>
            ) : (
              offered.map((each) => (
                <option key={each.voice} value={each.voice}>
                  {`${each.voice}: incomplete, ${each.missing.length} of ${each.total} missing`}
                </option>
              ))
            )}
          </select>
        </label>
        <button className="btn" data-stop type="button" onClick={look}>
          Refresh
        </button>
      </div>

      {noVoices && (
        <p className="callout">
          <b>No voice folders yet.</b> Make one on the Cast pane: type a name and press Make
          folders.
        </p>
      )}

      {allComplete && (
        <p className="callout">
          {/* FR-234: the chooser above already says every voice is complete. */}
          Each one has a recording for every moment.
        </p>
      )}

      {problem !== '' && (
        <p className="callout refused" role="alert">
          {problem}
        </p>
      )}

      {list !== null && (
        <>
          <p className="meta">
            {list.folder === ''
              ? `${list.voice} has takes for ${list.recorded} of ${list.total} moments.`
              : `${list.voice} has recordings for ${list.recorded} of ${list.total} moments.`}
          </p>
          <p className="lede">
            {list.folder === ''
              ? 'These are the moments this voice has no take for. Its audio belongs to the plugin that offers it, so there is no folder here to open or fill.'
              : 'Each moment below says when it is heard, then the folder its audio file is saved in.'}
          </p>
          {/* FR-233: each moment under its full title alone, so no words are shown twice. */}
          {list.missing.map((item) => (
            <div className="row" key={item.id}>
              <span className="grow">
                {item.title}
                <br />
                {/* FR-318: when the take will be heard, between the title and its folder. */}
                <span className="purpose">{item.purpose}</span>
                {/* A list with no folder shows neither the path nor the way to open it: there is
                    no folder to name and nothing this application may write to (FR-571). */}
                {list.folder !== '' && (
                  <>
                    <br />
                    <span className="hint">{`${list.folder}${item.folder}`}</span>
                  </>
                )}
              </span>
              {list.folder !== '' && (
                <button
                  className="btn"
                  data-stop
                  type="button"
                  aria-label={`Open the folder for ${item.title}`}
                  onClick={() => open(item.id)}
                >
                  Open folder
                </button>
              )}
            </div>
          ))}
        </>
      )}
    </>
  )
}
