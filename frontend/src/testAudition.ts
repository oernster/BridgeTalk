// The audition pane's test harness: the calls the pane makes as spies, the events it subscribes to
// and the voices and groups its suites share, so each suite mocks the bridge the same way.

import { vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import type { ReactElement } from 'react'
import type { Audition, Group, MachineVoice, Voice } from './api'

export const voices = vi.fn<() => Promise<Voice[]>>()
export const auditionGroups = vi.fn<(voice: string) => Promise<Group[]>>()
export const audition = vi.fn<(voice: string, group: string) => Promise<Audition | null>>()
export const stopAudition = vi.fn<() => Promise<void>>()
export const playing = vi.fn<() => Promise<boolean>>()
export const machineVoices = vi.fn<() => Promise<MachineVoice[]>>()
export const machineAuditionGroups = vi.fn<() => Promise<Group[]>>()
export const auditionMachineVoice = vi.fn<(id: string, group: string) => Promise<Audition | null>>()

/** handlers holds whatever the pane subscribed to, so a test can raise the event. */
export const handlers = new Map<string, (...data: unknown[]) => void>()

/** mockedApi stands in for ./api: every call reaches its spy and each subscription is kept. */
export const mockedApi = {
  api: {
    voices: () => voices(),
    auditionGroups: (voice: string) => auditionGroups(voice),
    audition: (voice: string, group: string) => audition(voice, group),
    stopAudition: () => stopAudition(),
    playing: () => playing(),
    machineVoices: () => machineVoices(),
    machineAuditionGroups: () => machineAuditionGroups(),
    auditionMachineVoice: (id: string, group: string) => auditionMachineVoice(id, group),
  },
  on: (name: string, handler: (...data: unknown[]) => void) => {
    handlers.set(name, handler)
    return () => handlers.delete(name)
  },
}

export const grace: Voice = { name: 'Grace', display: 'Grace', credit: '', cues: 90, inUse: 100, present: 100 }
export const kate: Voice = { name: 'Kate', display: 'Kate', credit: '', cues: 10, inUse: 12, present: 12 }
export const emma: MachineVoice = { id: 'bf_emma', name: 'Emma (British, female)' }
export const michael: MachineVoice = { id: 'am_michael', name: 'Michael (American, male)' }

const shields: Group = { key: 'shields', label: 'Shields', clips: 4 }
const combat: Group = { key: 'combat', label: 'Combat', clips: 1 }

/**
 * resetAudition clears every spy and subscription. Grace and Kate are then the recorded voices with
 * no machine voice; every voice has the Shields and Combat groups; every audition plays.
 */
export function resetAudition() {
  handlers.clear()
  for (const spy of [
    voices,
    auditionGroups,
    audition,
    stopAudition,
    playing,
    machineVoices,
    machineAuditionGroups,
    auditionMachineVoice,
  ])
    spy.mockReset()
  voices.mockResolvedValue([grace, kate])
  auditionGroups.mockResolvedValue([shields, combat])
  audition.mockResolvedValue({ group: 'shields', clip: 'a.mp3' })
  stopAudition.mockResolvedValue(undefined)
  playing.mockResolvedValue(false)
  machineVoices.mockResolvedValue([])
  machineAuditionGroups.mockResolvedValue([shields, combat])
  auditionMachineVoice.mockResolvedValue({ group: 'shields', clip: 'k.flac' })
}

/** shown renders a pane and waits for its groups. */
export async function shown(pane: ReactElement) {
  render(pane)
  await screen.findByRole('button', { name: /Shields/ })
}

/** groupButton finds one group's play button by the label it shows. */
export function groupButton(name: RegExp): HTMLButtonElement {
  return screen.getByRole('button', { name }) as HTMLButtonElement
}
