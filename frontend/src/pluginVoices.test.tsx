// The plugin voices on the Cast pane: offered apart from every other kind, a section for each
// plugin with its groups inside it, the cast one on a card of its own, a voice with no audio behind
// it named with the reason it gave and nothing at all drawn where no plugin offers a voice (FR-562,
// FR-565, FR-568, FR-569, FR-570, FR-583, FR-584).

import { beforeEach, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, within } from '@testing-library/react'
import { refuses } from './testRefusal'
import type { PluginVoice, Refused } from './api'

const pluginVoices = vi.fn<() => Promise<PluginVoice[]>>()
const castPluginVoice =
  vi.fn<(plugin: string, id: string, refused: Refused) => Promise<void>>()

vi.mock('./api', () => ({
  api: {
    pluginVoices: () => pluginVoices(),
    castPluginVoice: (plugin: string, id: string, refused: Refused) =>
      castPluginVoice(plugin, id, refused),
  },
}))

const { PluginVoices } = await import('./pluginVoices')

/** voice is one voice a plugin offers in no group, with its audio on this machine. */
function voice(plugin: string, id: string, name: string, group = ''): PluginVoice {
  return { plugin, id, name, display: name, section: plugin, group, ready: true, reason: '' }
}

const officer = voice('Bridge Crew', 'one', 'The First Officer')
const pilot = voice('Flight Deck', 'two', 'The Pilot')

/** absent is a voice whose audio is not on this machine, with the reason the plugin gave. */
const absent: PluginVoice = {
  plugin: 'Bridge Crew',
  id: 'three',
  name: 'The Engineer',
  display: 'The Engineer',
  section: 'Bridge Crew',
  group: '',
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

  expect(screen.queryByRole('heading')).toBeNull()
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

  expect(castPluginVoice).toHaveBeenCalledWith('Flight Deck', 'two', expect.any(Function))
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

// Two plugins may honestly choose one name for a voice. Inside its own plugin's section each is
// shown by its plain name, since the heading already says which plugin offers it (FR-583, which
// amends how FR-568 reads on this pane).
it('shows a shared name plainly inside each plugin section', async () => {
  const shared = {
    ...officer,
    plugin: 'Flight Deck',
    section: 'Flight Deck',
    display: 'The First Officer (Flight Deck)',
  }
  await show([{ ...officer, display: 'The First Officer (Bridge Crew)' }, shared])

  for (const heading of ['Bridge Crew', 'Flight Deck']) {
    const section = screen.getByRole('region', { name: heading })
    expect(within(section).getByRole('button').textContent).toBe('The First Officer')
  }
})

// FR-583: each plugin's voices stand in a section of their own, headed by the name the facade gave
// the section, in the order the plugins arrive.
it('draws a section for each plugin headed by its name', async () => {
  await show([officer, pilot])

  const headings = screen.getAllByRole('heading', { level: 3 }).map((each) => each.textContent)
  expect(headings).toEqual(['Bridge Crew', 'Flight Deck'])
  expect(within(screen.getByRole('region', { name: 'Flight Deck' })).getByRole('button').textContent).toBe(
    'The Pilot',
  )
})

// FR-584: inside a section, the voices in no group come first, then a panel for each group in the
// order the plugin first names it.
it('stands ungrouped voices first then each group in the order it is first named', async () => {
  await show([
    voice('Quartermaster Voices', 'ada', 'Ada', 'Crew'),
    voice('Quartermaster Voices', 'bo', 'Bo'),
    voice('Quartermaster Voices', 'cy', 'Cy', 'Stations'),
    voice('Quartermaster Voices', 'di', 'Di', 'Crew'),
  ])

  const section = screen.getByRole('region', { name: 'Quartermaster Voices' })
  const order = within(section)
    .getAllByRole('button')
    .map((each) => each.textContent)
  expect(order).toEqual(['Bo', 'Ada', 'Di', 'Cy'])
  expect(within(section).getAllByRole('heading', { level: 4 }).map((each) => each.textContent)).toEqual([
    'Crew',
    'Stations',
  ])
  expect(within(screen.getByRole('group', { name: 'Crew' })).getAllByRole('button')).toHaveLength(2)
})

// FR-583: the cast voice's card and the voices that cannot speak stand inside their own plugin's
// section rather than above or below every section.
it('keeps the cast card and the voices that cannot speak inside their own section', async () => {
  await show([pilot, officer, absent], 'two', 'Flight Deck')

  const deck = screen.getByRole('region', { name: 'Flight Deck' })
  const crew = screen.getByRole('region', { name: 'Bridge Crew' })
  expect(within(deck).getByText(/The Pilot/)).toBeTruthy()
  expect(within(deck).queryByRole('button')).toBeNull()
  expect(within(crew).getByText('The Engineer: its recordings are not on this machine')).toBeTruthy()
  expect(within(deck).queryByText(/cannot speak/)).toBeNull()
})

// A refusal is the reason the facade gave, said where the reader is looking rather than swallowed.
it('says why a cast was refused', async () => {
  castPluginVoice.mockImplementation(
    refuses('The First Officer cannot speak: no audio', undefined),
  )
  await show([officer])

  fireEvent.click(screen.getByRole('button', { name: /The First Officer/ }))
  await screen.findByRole('alert')

  expect(screen.getByRole('alert').textContent).toContain('cannot speak')
})
