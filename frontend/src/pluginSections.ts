// The order plugin voices stand in, kept in one place for every list that shows them: one section for
// each plugin in the order it first appears, its voices in no group first, then a group for each name
// it gives in the order it first names each one (FR-583, FR-584, FR-585).
//
// The Cast pane draws the sections; the Audition chooser lists the voices that can speak in the same
// order, so a voice stands in the same place on both.

import type { PluginVoice } from './api'

/** Section is one plugin's voices as the Cast pane draws them. */
export interface Section {
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
export function sectionsOf(voices: PluginVoice[], isCast: (voice: PluginVoice) => boolean): Section[] {
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

/** speakingInSectionOrder answers the voices that can speak, in the order the Cast pane stands them (FR-585). */
export function speakingInSectionOrder(voices: PluginVoice[]): PluginVoice[] {
  return sectionsOf(voices, () => false).flatMap((section) => [
    ...section.ungrouped,
    ...section.groups.flatMap((group) => group.voices),
  ])
}
