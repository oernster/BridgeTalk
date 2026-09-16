// The plugin voices on the Cast pane: every voice the loaded plugins offer as a pill, the one
// cast on a card of its own above them and the ones whose audio is not on this machine named
// with the reason they gave (FR-565, FR-568, FR-570).
//
// It sits apart from cast.tsx for the reason machineVoices.tsx does: a plugin voice is cast by
// the plugin that offered it and its own id within that plugin, so it is cast apart from a
// recorded voice. It has no folder and no recordings of this application's own, so the mark that
// opens what a recorded voice covers has nothing to open.
//
// Nothing is drawn while no plugin offers a voice, which is almost every run: the ordinary case
// should mention plugins nowhere (FR-562).

import { useEffect, useState } from 'react'
import { api, type PluginVoice } from './api'
import { castLabel } from './castWords'
import type { Outcome } from './chooser'

/**
 * PluginVoices offers every voice the loaded plugins hold and casts the one chosen.
 *
 * active and plugin come from the state: a plugin voice is cast only while a plugin is named
 * there, since an id identifies a voice within its own plugin alone and a recordings folder may
 * carry the same name (FR-569).
 */
export function PluginVoices({ active, plugin }: { active: string; plugin: string }) {
  const [voices, setVoices] = useState<PluginVoice[]>([])
  const [outcome, setOutcome] = useState<Outcome>(null)

  // Asked again on each cast, because a plugin answers whether its audio is on this machine when
  // it is loaded and a cast is the moment that answer is worth reading again.
  useEffect(() => {
    void api.pluginVoices().then(setVoices)
  }, [active])

  const cast = (voice: PluginVoice) => {
    setOutcome(null)
    void api.castPluginVoice(voice.plugin, voice.id, (reason) =>
      setOutcome({ refused: true, text: reason }),
    )
  }

  if (voices.length === 0) return null

  const isCast = (voice: PluginVoice) =>
    plugin !== '' && voice.plugin === plugin && voice.id === active
  const castVoice = voices.find(isCast)
  const ready = voices.filter((voice) => voice.ready && !isCast(voice))
  const unavailable = voices.filter((voice) => !voice.ready)

  return (
    <>
      <h3>Plugin voices</h3>
      <p className="lede">
        These come from plugins installed beside the application. Their audio belongs to the
        plugin; this application plays it where it stands and never writes to it.
      </p>

      {/* The cast voice stands apart on a card, as a cast machine voice does (FR-721). It is not
          a control: pressing it would cast it again and play its confirmation a second time. */}
      {castVoice !== undefined && (
        <div className="castcard">
          <span className="name">{castLabel(castVoice.display, true)}</span>
        </div>
      )}

      <div className="pills">
        {ready.map((voice) => (
          /* The plugin and the id together key the row, since an id is unique within its own
             plugin alone and two plugins may offer one (FR-569). */
          <button
            key={`${voice.plugin}-${voice.id}`}
            className="pill"
            data-stop
            type="button"
            aria-label={castLabel(voice.display, false)}
            onClick={() => cast(voice)}
          >
            {voice.display}
          </button>
        ))}
      </div>

      {/* A voice that cannot speak is said in words rather than drawn as a control that refuses
          when pressed: the reason is the whole of what the reader can act on (FR-570). */}
      {unavailable.length > 0 && (
        <p className="callout">
          <span>These voices cannot speak on this machine:</span>
          {unavailable.map((voice) => (
            <span key={`${voice.plugin}-${voice.id}`}>
              <br />
              <span>{`${voice.display}: ${voice.reason}`}</span>
            </span>
          ))}
        </p>
      )}

      {outcome !== null && (
        <p className="callout refused" role="alert">
          {outcome.text}
        </p>
      )}
    </>
  )
}
