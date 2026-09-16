// The audition pane over plugin voices (FR-585, FR-586): the ones that can speak follow the machine
// voices under the name a flat list shows, are auditioned through calls of their own and are never
// cast by being heard.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import type { PluginVoice } from './api'
import {
  audition,
  auditionGroups,
  auditionMachineVoice,
  auditionPluginVoice,
  emma,
  machineVoices,
  pluginAuditionGroups,
  pluginVoices,
  resetAudition,
  shown,
} from './testAudition'

vi.mock('./api', async () => (await import('./testAudition')).mockedApi)

const { AuditionPane } = await import('./audition')

beforeEach(resetAudition)

/** supplied is one plugin voice, named as a flat list shows it. */
function supplied(plugin: string, id: string, display: string, ready = true): PluginVoice {
  return { plugin, id, name: display, display, section: plugin, group: '', ready, reason: ready ? '' : 'gone' }
}

/** A plugin name holding a slash, which the chooser's value must carry whole. */
const ada = supplied('Crew / Deck', 'ada', 'Ada (Crew / Deck)')
const bo = supplied('Crew / Deck', 'bo', 'Bo', false)

/** options reads the chooser's option texts once it holds the count given. */
async function options(count: number): Promise<HTMLSelectElement> {
  const chooser = screen.getByRole('combobox') as HTMLSelectElement
  await waitFor(() => expect(chooser.options).toHaveLength(count))
  return chooser
}

describe('the audition pane with plugin voices', () => {
  // FR-585: plugin voices that can speak stand after the recorded voices and above the machine voices;
  // one that cannot is not offered.
  it('offers the plugin voices that can speak above the machine voices', async () => {
    machineVoices.mockResolvedValue([emma])
    pluginVoices.mockResolvedValue([ada, bo])
    await shown(<AuditionPane cast="Grace" />)

    const chooser = await options(4)
    expect(Array.from(chooser.options).map((option) => option.text)).toEqual([
      'Grace (cast)',
      'Kate',
      'Ada (Crew / Deck)',
      'Emma (British, female)',
    ])
  })

  // FR-585: the plugin voices stand as the Cast pane stands them: each plugin's voices in no group
  // first, then each group in the order the plugin first names it, whatever order they arrive in.
  it('lists the plugin voices in the order of their sections and groups', async () => {
    const grouped = (id: string, group: string): PluginVoice => ({ ...supplied('Crew', id, id), group })
    machineVoices.mockResolvedValue([])
    pluginVoices.mockResolvedValue([
      grouped('Aster', 'Officers'),
      grouped('Burr', 'Engineers'),
      grouped('Cole', 'Officers'),
      grouped('Dune', ''),
    ])
    await shown(<AuditionPane cast="Grace" />)

    const chooser = await options(6)
    expect(Array.from(chooser.options).map((option) => option.text)).toEqual([
      'Grace (cast)',
      'Kate',
      'Dune',
      'Aster',
      'Cole',
      'Burr',
    ])
  })

  // FR-586: a plugin voice is auditioned through its own calls by its plugin and its id, never through
  // a recorded or machine voice's.
  it('auditions a plugin voice through the plugin voice calls', async () => {
    pluginVoices.mockResolvedValue([ada])
    await shown(<AuditionPane cast="Grace" />)
    const chooser = await options(3)

    fireEvent.change(chooser, { target: { value: chooser.options[2].value } })
    await waitFor(() => expect(pluginAuditionGroups).toHaveBeenCalledWith('Crew / Deck', 'ada'))
    auditionGroups.mockClear()
    fireEvent.click(await screen.findByRole('button', { name: /Shields/ }))

    await waitFor(() =>
      expect(auditionPluginVoice).toHaveBeenCalledWith('Crew / Deck', 'ada', 'shields', expect.any(Function)),
    )
    expect(audition).not.toHaveBeenCalled()
    expect(auditionMachineVoice).not.toHaveBeenCalled()
    expect(auditionGroups).not.toHaveBeenCalled()
  })

  // A cast plugin voice is where the pane opens, marked cast, while a recorded voice sharing its id is not.
  it('opens on the cast plugin voice', async () => {
    pluginVoices.mockResolvedValue([supplied('Crew', 'Grace', 'Grace')])
    await shown(<AuditionPane cast="Grace" plugin="Crew" />)

    const chooser = await options(3)
    expect(chooser.selectedOptions[0].text).toBe('Grace (cast)')
    expect(chooser.options[0].text).toBe('Grace')
    expect(pluginAuditionGroups).toHaveBeenCalledWith('Crew', 'Grace')
  })
})
