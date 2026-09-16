// The plugin voices on the Cast pane: offered apart from every other kind, the cast one on a card
// of its own, a voice with no audio behind it named with the reason it gave and nothing at all
// drawn where no plugin offers a voice (FR-562, FR-565, FR-568, FR-569, FR-570).

import { beforeEach, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'
import type { PluginVoice } from './api'

const pluginVoices = vi.fn<() => Promise<PluginVoice[]>>()
const castPluginVoice = vi.fn<(plugin: string, id: string) => Promise<void>>()

vi.mock('./api', () => ({
  api: {
    pluginVoices: () => pluginVoices(),
    castPluginVoice: (plugin: string, id: string) => castPluginVoice(plugin, id),
  },
}))

const { PluginVoices } = await import('./pluginVoices')

/** voice is one voice a plugin offers, with its audio on this machine. */
function voice(plugin: string, id: string, name: string): PluginVoice {
  return { plugin, id, name, display: name, ready: true, reason: '' }
}

const officer = voice('Bridge Crew', 'one', 'The First Officer')
const pilot = voice('Flight Deck', 'two', 'The Pilot')

/** absent is a voice whose audio is not on this machine, with the reason the plugin gave. */
const absent: PluginVoice = {
  plugin: 'Bridge Crew',
  id: 'three',
  name: 'The Engineer',
  display: 'The Engineer',
  ready: false,
  reason: 'its recordings are not on this machine',
}

/** show draws the section over the voices given and waits for them to arrive. */
async function show(offered: PluginVoice[], active = '', plugin = '') {
  pluginVoices.mockResolvedValue(offered)
  render(<PluginVoices active={active} plugin={plugin} />)
  // The list arrives after a tick even where it is empty, so the draw that follows it is waited
  // for either way: a section that draws nothing has nothing to find by name.
  await act(async () => {})
}

beforeEach(() => {
  vi.clearAllMocks()
  castPluginVoice.mockResolvedValue(undefined)
})

// Almost every run has no plugin at all. Saying nothing is the whole of what the ordinary case
// should say about them (FR-562).
it('draws nothing where no plugin offers a voice', async () => {
  await show([])

  expect(screen.queryByText('Plugin voices')).toBeNull()
})

it('offers every voice every plugin holds', async () => {
  await show([officer, pilot])

  expect(screen.getByRole('button', { name: /The First Officer/ })).toBeTruthy()
  expect(screen.getByRole('button', { name: /The Pilot/ })).toBeTruthy()
})

// A cast sends back the plugin as well as the id, since an id identifies a voice within its own
// plugin alone (FR-569).
it('casts a voice by its plugin and its id within it', async () => {
  await show([officer, pilot])

  fireEvent.click(screen.getByRole('button', { name: /The Pilot/ }))

  expect(castPluginVoice).toHaveBeenCalledWith('Flight Deck', 'two')
})

// The cast voice stands on a card rather than among the pills, so pressing it again is not
// something the page offers (FR-721).
it('stands the cast voice apart from the rest', async () => {
  await show([officer, pilot], 'one', 'Bridge Crew')

  expect(screen.queryByRole('button', { name: /The First Officer/ })).toBeNull()
  expect(screen.getByText(/The First Officer/)).toBeTruthy()
  expect(screen.getByRole('button', { name: /The Pilot/ })).toBeTruthy()
})

// A voice cast from a plugin and a recorded voice may carry one id. The plugin named in the state
// is what tells them apart, so an id alone never marks a card (FR-569).
it('marks no voice cast while the state names no plugin', async () => {
  await show([officer], 'one')

  expect(screen.getByRole('button', { name: /The First Officer/ })).toBeTruthy()
})

// A voice whose audio is not on this machine is shown with the reason it gave rather than left
// out; it is never drawn as something that can be pressed (FR-570).
it('names a voice that cannot speak with the reason it gave', async () => {
  await show([officer, absent])

  expect(screen.queryByRole('button', { name: /The Engineer/ })).toBeNull()
  expect(
    screen.getByText('The Engineer: its recordings are not on this machine'),
  ).toBeTruthy()
})

// Two plugins may honestly choose one name for a voice. The facade works out the name each is
// shown by; the pane shows what it was given rather than the plain name (FR-568).
it('shows each voice by the name the facade worked out', async () => {
  const shared = { ...officer, plugin: 'Flight Deck', display: 'The First Officer (Flight Deck)' }
  await show([{ ...officer, display: 'The First Officer (Bridge Crew)' }, shared])

  expect(screen.getByRole('button', { name: /The First Officer \(Bridge Crew\)/ })).toBeTruthy()
  expect(screen.getByRole('button', { name: /The First Officer \(Flight Deck\)/ })).toBeTruthy()
})

// A refusal is the reason the facade gave, said where the reader is looking rather than swallowed.
it('says why a cast was refused', async () => {
  castPluginVoice.mockRejectedValue('The First Officer cannot speak: no audio')
  await show([officer])

  fireEvent.click(screen.getByRole('button', { name: /The First Officer/ }))
  await screen.findByRole('alert')

  expect(screen.getByRole('alert').textContent).toContain('cannot speak')
})
