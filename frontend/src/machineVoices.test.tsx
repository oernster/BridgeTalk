// The machine voices on the Cast pane: offered apart from the recorded voices as pills in a panel for
// each accent and sex, the cast one on a card of its own above them, with how far making has got and
// whatever went wrong (FR-508, FR-515, FR-518 to FR-522, FR-528, FR-530, FR-720 to FR-722).

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, within } from '@testing-library/react'
import type { MachineVoice, Making } from './api'
import { nothingMade } from './making'

const machineVoices = vi.fn<() => Promise<MachineVoice[]>>()
const making = vi.fn<() => Promise<Making>>()
const castMachineVoice = vi.fn<(id: string) => Promise<void>>()
const handlers = new Map<string, (...data: unknown[]) => void>()

vi.mock('./api', () => ({
  api: {
    machineVoices: () => machineVoices(),
    making: () => making(),
    castMachineVoice: (id: string) => castMachineVoice(id),
  },
  on: (name: string, handler: (...data: unknown[]) => void) => {
    handlers.set(name, handler)
    return () => handlers.delete(name)
  },
}))

const { MachineVoices } = await import('./machineVoices')

/** voice is a machine voice as the facade offers it. */
function voice(id: string, given: string, group: string): MachineVoice {
  return { id, name: `${given} (${group})`, given, group }
}

const alice = voice('bf_alice', 'Alice', 'British, female')
const emma = voice('bf_emma', 'Emma', 'British, female')
const isabella = voice('bf_isabella', 'Isabella', 'British, female')
const daniel = voice('bm_daniel', 'Daniel', 'British, male')
const george = voice('bm_george', 'George', 'British, male')
const heart = voice('af_heart', 'Heart', 'American, female')
const michael = voice('am_michael', 'Michael', 'American, male')

/** offered is the facade's list in FR-508's group order, each group's voices out of order. */
const offered = [isabella, emma, alice, george, daniel, heart, michael]

/** idle is making with no machine voice cast. */
const idle: Making = nothingMade

/** moments is how many moments there are. */
const moments = 256

/** castLine is what the card says of a voice cast. */
const castLine = /is cast as your ship's voice/

beforeEach(() => {
  handlers.clear()
  machineVoices.mockReset().mockResolvedValue(offered)
  making.mockReset().mockResolvedValue(idle)
  castMachineVoice.mockReset().mockResolvedValue(undefined)
})

/** show renders the machine voices and waits for them to arrive. */
async function show(active = '', machine = false) {
  const shown = render(<MachineVoices active={active} machine={machine} total={moments} />)
  await screen.findByRole('group', { name: 'British, male' })
  return shown
}

/** pills reads the names on the pills in a group's panel, in the order drawn. */
function pills(group: string): string[] {
  return within(screen.getByRole('group', { name: group }))
    .getAllByRole('button')
    .map((pill) => pill.textContent ?? '')
}

describe('the machine voices', () => {
  // FR-508, FR-528 and FR-720: a panel for each accent and sex in the order offered, each headed by
  // it and holding its voices by name alone, sorted.
  it('offers a panel for each accent and sex, its voices sorted by name', async () => {
    await show()

    expect(screen.getByRole('heading', { name: 'Machine voices' })).toBeTruthy()
    expect(screen.getAllByRole('group').map((group) => group.getAttribute('aria-label'))).toEqual([
      'British, female',
      'British, male',
      'American, female',
      'American, male',
    ])
    expect(screen.getByRole('heading', { name: 'British, female' })).toBeTruthy()
    expect(pills('British, female')).toEqual(['Alice', 'Emma', 'Isabella'])
    expect(pills('British, male')).toEqual(['Daniel', 'George'])
    expect(screen.getByRole('button', { name: 'Cast Emma (British, female)' })).toBeTruthy()
    expect(screen.queryByText('Its lines are made as they are needed.')).toBeNull()
  })

  it('casts a machine voice by its id', async () => {
    await show()

    fireEvent.click(screen.getByRole('button', { name: 'Cast Emma (British, female)' }))

    expect(castMachineVoice).toHaveBeenCalledWith('bf_emma')
  })

  // FR-721: the cast voice leaves its panel for a card above them all, which is not a control; a
  // voice cast after it takes the card and puts it back in its panel.
  it('puts the cast machine voice on a card above the panels, out of its own', async () => {
    const shown = await show('bf_emma', true)

    const card = screen.getByText(castLine)
    expect(card.textContent).toBe("Emma (British, female) is cast as your ship's voice")
    expect(card.closest('button, [data-stop]')).toBeNull()
    expect(pills('British, female')).toEqual(['Alice', 'Isabella'])

    shown.rerender(<MachineVoices active="bm_george" machine={true} total={moments} />)

    expect((await screen.findByText(/^George/)).textContent).toBe("George (British, male) is cast as your ship's voice")
    expect(pills('British, female')).toEqual(['Alice', 'Emma', 'Isabella'])
    expect(pills('British, male')).toEqual(['Daniel'])
  })

  // FR-540 and FR-722: a recorded voice cast, even one whose folder carries a machine voice's id,
  // leaves no card and every pill in its panel.
  it('shows no card while a recorded voice is cast, however it is named', async () => {
    await show('bf_emma', false)

    expect(screen.queryByText(castLine)).toBeNull()
    expect(pills('British, female')).toEqual(['Alice', 'Emma', 'Isabella'])
  })

  // FR-515 and FR-522: how far making has got, beneath the card and followed as it is announced.
  it('reads how far making has got for the cast voice and follows it', async () => {
    making.mockResolvedValue({ ...idle, voice: 'bf_emma', making: true, current: 100, total: 768, cuesServed: 12 })
    await show('bf_emma', true)

    expect(await screen.findByText('100 of 768 lines made; 12 of 256 moments spoken')).toBeTruthy()

    act(() => handlers.get('making')?.({ ...idle, voice: 'bf_emma', current: 768, total: 768, cuesServed: 256 }))

    expect(await screen.findByText('768 of 768 lines made; 256 of 256 moments spoken')).toBeTruthy()
  })

  // FR-518, FR-520 and FR-530: each problem said, with its reason.
  it('says which lines could not be made, why making stopped and what could not be deleted', async () => {
    making.mockResolvedValue({
      ...idle,
      voice: 'bf_emma',
      current: 1,
      total: 6,
      cuesServed: 1,
      failed: [{ cue: { id: 'Docked', title: 'Docked', folder: 'Docked', purpose: '' }, line: 2, reason: 'the model failed' }],
      stopped: 'the disk is full',
      notDeleted: 'a made line is in use',
    })
    await show('bf_emma', true)

    expect(await screen.findByText('Docked, line 2: the model failed')).toBeTruthy()
    expect(screen.getByText('Making stopped: the disk is full')).toBeTruthy()
    expect(screen.getByText('The voice cast before still has made lines that could not be deleted: a made line is in use')).toBeTruthy()
  })

  // FR-519: a refused cast says why.
  it('says why a machine voice could not be cast', async () => {
    castMachineVoice.mockRejectedValue('reading bf_emma.bin: the file is missing')
    await show()

    fireEvent.click(screen.getByRole('button', { name: 'Cast Emma (British, female)' }))

    expect((await screen.findByRole('alert')).textContent).toBe('reading bf_emma.bin: the file is missing')
  })
})
