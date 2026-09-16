// The fixtures the Missing takes suites share: the three moments they list, the two shapes of
// checklist the facade answers with and the wait that ends once a list has landed.
//
// They sit apart from the suites because both read them: a moment's title or a checklist's shape
// written out twice is two statements that can disagree; the suite that was not edited is then
// the one that passes while it should not.

import { screen } from '@testing-library/react'
import type { Checklist, CueEntry } from './api'

export const docked: CueEntry = {
  id: 'Docked',
  title: 'Docked',
  folder: 'Docked',
  purpose: 'When the ship docks.',
}
export const undocked: CueEntry = {
  id: 'Undocked',
  title: 'Undocked',
  folder: 'Undocked',
  purpose: 'When the ship leaves its pad.',
}
export const hyperspace: CueEntry = {
  id: 'StartJump.JumpType.Hyperspace',
  title: 'Start jump: jump type hyperspace',
  folder: 'StartJump_JumpType_Hyperspace',
  purpose: 'When a hyperspace jump to another system begins.',
}

/** progress builds a voice's checklist over a vocabulary of three moments. */
export const progress = (voice: string, missing: CueEntry[]): Checklist => ({
  voice,
  recorded: 3 - missing.length,
  total: 3,
  missing,
  folder: `D:\\Recordings\\${voice}\\`,
})

/** fromPlugin is the cast plugin voice's checklist: moments with no take and no folder at all. */
export const fromPlugin = (voice: string, missing: CueEntry[]): Checklist => ({
  voice,
  recorded: 3 - missing.length,
  total: 3,
  missing,
  folder: '',
})

/**
 * chooser finds the voice chooser once the lists have landed. findAll rather than find,
 * because the complete state names itself twice: in the chooser and in the pane.
 */
export async function chooser(): Promise<HTMLSelectElement> {
  await screen.findAllByText(
    /has recordings for|has takes for|No voices yet|Every voice is complete/,
  )
  return screen.getByRole('combobox') as HTMLSelectElement
}
