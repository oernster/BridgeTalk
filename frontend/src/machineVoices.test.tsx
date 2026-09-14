// The machine voices on the Cast pane: listed apart from the recorded voices by the name each is
// shown by, with how far making has got and whatever went wrong (FR-508, FR-515, FR-518 to FR-522,
// FR-528, FR-530).

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'
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

/** emma and michael are two machine voices as the facade offers them. */
const emma: MachineVoice = { id: 'bf_emma', name: 'Emma (British, female)' }
const michael: MachineVoice = { id: 'am_michael', name: 'Michael (American, male)' }

/** idle is making with no machine voice cast. */
const idle: Making = nothingMade

/** moments is how many moments there are. */
const moments = 256

beforeEach(() => {
  handlers.clear()
  machineVoices.mockReset().mockResolvedValue([emma, michael])
  making.mockReset().mockResolvedValue(idle)
  castMachineVoice.mockReset().mockResolvedValue(undefined)
})

/** show renders the machine voices and waits for them to arrive. */
async function show(active = '', machine = false) {
  render(<MachineVoices active={active} machine={machine} total={moments} />)
  await screen.findByRole('button', { name: /Emma \(British, female\)/ })
}

describe('the machine voices', () => {
  // FR-508 and FR-528: under a heading of their own, each by the name it is shown by.
  it('lists every machine voice apart, by the name each is shown by', async () => {
    await show()

    expect(screen.getByRole('heading', { name: 'Machine voices' })).toBeTruthy()
    expect(screen.getByRole('button', { name: /^Cast Emma \(British, female\)/ })).toBeTruthy()
    expect(screen.getByRole('button', { name: /^Cast Michael \(American, male\)/ })).toBeTruthy()
    expect(screen.getAllByText('Its lines are made as they are needed.')).toHaveLength(2)
  })

  it('casts a machine voice by its id', async () => {
    await show()

    fireEvent.click(screen.getByRole('button', { name: /^Cast Emma/ }))

    expect(castMachineVoice).toHaveBeenCalledWith('bf_emma')
  })

  // FR-540: a recorded voice whose folder carries a machine voice's id is not taken for it.
  it('marks a machine voice cast only while the cast voice is a machine voice', async () => {
    await show('bf_emma', false)

    expect(screen.getByRole('button', { name: /^Cast Emma/ })).toBeTruthy()
  })

  // FR-515 and FR-522: how far making has got, followed as the backend announces it.
  it('reads how far making has got for the cast voice and follows it', async () => {
    making.mockResolvedValue({ ...idle, voice: 'bf_emma', making: true, current: 100, total: 768, cuesServed: 12 })
    await show('bf_emma', true)

    expect(await screen.findByText('100 of 768 lines made; 12 of 256 moments spoken')).toBeTruthy()
    expect(screen.getByRole('button', { name: /Emma \(British, female\) is cast as your ship's voice/ })).toBeTruthy()

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

    fireEvent.click(screen.getByRole('button', { name: /^Cast Emma/ }))

    expect((await screen.findByRole('alert')).textContent).toBe('reading bf_emma.bin: the file is missing')
  })
})
