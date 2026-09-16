// The order the Cast pane stands its kinds of voice in: the plugin voices above the machine voices
// (FR-592).

import { expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import type { MachineVoice, PluginVoice } from './api'
import { nothingMade } from './making'

const emma: MachineVoice = { id: 'bf_emma', name: 'Emma (British, female)', given: 'Emma', group: 'British, female' }
const ada: PluginVoice = {
  plugin: 'Quartermaster Voices',
  id: 'ada',
  name: 'Ada',
  display: 'Ada',
  section: 'Quartermaster Voices',
  group: 'Crew',
  ready: true,
  reason: '',
}

vi.mock('./api', () => ({
  api: {
    voices: () => Promise.resolve([]),
    machineVoices: () => Promise.resolve([emma]),
    pluginVoices: () => Promise.resolve([ada]),
    making: () => Promise.resolve(nothingMade),
    castMachineVoice: () => Promise.resolve(),
    castPluginVoice: () => Promise.resolve(),
  },
  on: () => () => undefined,
}))

const { CastPane } = await import('./cast')

it('stands the plugin voices above the machine voices', async () => {
  render(
    <CastPane active="" machine={false} plugin="" total={256} libraryRoot="D:/Recordings" onSelect={() => {}} />,
  )

  const plugins = await screen.findByRole('heading', { name: 'Quartermaster Voices' })
  const machines = await screen.findByRole('heading', { name: 'Machine voices' })

  expect(plugins.compareDocumentPosition(machines) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
})
