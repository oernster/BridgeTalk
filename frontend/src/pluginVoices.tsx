// The plugin voices on the Cast pane: one section for each plugin, its voices in no group first and
// then a panel for each group it names; the one cast on a card of its own at the top of its section
// and the ones whose audio is not on this machine named with the reason they gave (FR-565, FR-570,
// FR-583, FR-584).
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

/** Section is one plugin's voices as the pane draws them. */
interface Section {
  heading: string
  ungrouped: PluginVoice[]
  groups: { name: string; voices: PluginVoice[] }[]
  unavailable: PluginVoice[]
  cast: PluginVoice | undefined
}

/**
 * sectionsOf gathers the voices into their plugins' sections, each in the order its plugin first
 * appears, with the groups in the order the plugin first names each one (FR-584).
 */
function sectionsOf(voices: PluginVoice[], isCast: (voice: PluginVoice) => boolean): Section[] {
  const sections: Section[] = []
  for (const voice of voices) {
    let section = sections.find((each) => each.heading === voice.section)
    if (section === undefined) {
      section = { heading: voice.section, ungrouped: [], groups: [], unavailable: [], cast: undefined }
      sections.push(section)
    }
    if (isCast(voice)) {
      // The cast voice stands on the card and keeps its pill in its group too (FR-593).
      section.cast = voice
    }
    if (!voice.ready) {
      section.unavailable.push(voice)
    } else if (voice.group === '') {
      section.ungrouped.push(voice)
    } else {
      let group = section.groups.find((each) => each.name === voice.group)
      if (group === undefined) {
        group = { name: voice.group, voices: [] }
        section.groups.push(group)
      }
      group.voices.push(voice)
    }
  }
  return sections
}

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

  /* A voice is shown by its own name inside its section, since the heading already says which
     plugin offers it (FR-583). The plugin and the id together key a pill, since an id is unique
     within its own plugin alone and two plugins may offer one (FR-569). The cast voice's pill is
     disabled: seen where it belongs, never cast twice (FR-593). */
  const pills = (offered: PluginVoice[]) => (
    <div className="pills">
      {offered.map((voice) => (
        <button
          key={`${voice.plugin}-${voice.id}`}
          className="pill"
          data-stop
          type="button"
          disabled={isCast(voice)}
          aria-label={castLabel(voice.name, isCast(voice))}
          onClick={() => cast(voice)}
        >
          {voice.name}
        </button>
      ))}
    </div>
  )

  return (
    <>
      {sectionsOf(voices, isCast).map((section, index) => (
        <section key={section.heading} aria-label={section.heading}>
          <h3>{section.heading}</h3>
          {/* Said once, under the first plugin: it is true of every plugin alike. */}
          {index === 0 && (
            <p className="lede">
              These come from plugins installed beside the application. Their audio belongs to the
              plugin; this application plays it where it stands and never writes to it.
            </p>
          )}

          {/* The cast voice stands apart on a card, as a cast machine voice does (FR-721). It is
              not a control: pressing it would cast it again and play its confirmation a second
              time. */}
          {section.cast !== undefined && (
            <div className="castcard">
              <span className="name">{castLabel(section.cast.name, true)}</span>
            </div>
          )}

          {section.ungrouped.length > 0 && pills(section.ungrouped)}

          {section.groups.length > 0 && (
            <div className="voicegroups">
              {section.groups.map((group) => (
                <div key={group.name} className="voicegroup" role="group" aria-label={group.name}>
                  <h4>{group.name}</h4>
                  {pills(group.voices)}
                </div>
              ))}
            </div>
          )}

          {/* A voice that cannot speak is said in words rather than drawn as a control that
              refuses when pressed: the reason is the whole of what the reader can act on
              (FR-570). */}
          {section.unavailable.length > 0 && (
            <p className="callout">
              <span>These voices cannot speak on this machine:</span>
              {section.unavailable.map((voice) => (
                <span key={`${voice.plugin}-${voice.id}`}>
                  <br />
                  <span>{`${voice.name}: ${voice.reason}`}</span>
                </span>
              ))}
            </p>
          )}
        </section>
      ))}

      {outcome !== null && (
        <p className="callout refused" role="alert">
          {outcome.text}
        </p>
      )}
    </>
  )
}
