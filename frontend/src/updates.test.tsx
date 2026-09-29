// The update check (FR-756 to FR-759): when it runs, when it stays silent and what each answer says.

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { Refused, Update } from './api'

const checkForUpdates = vi.fn<(manual: boolean, refused: Refused) => Promise<Update | null>>()
const downloadUpdate = vi.fn<(refused: Refused) => Promise<void>>()
const skipUpdate = vi.fn<(refused: Refused) => Promise<void>>()

vi.mock('./api', () => ({
  api: {
    about: () => Promise.resolve({ name: 'the application' }),
    checkForUpdates: (manual: boolean, refused: Refused) => checkForUpdates(manual, refused),
    downloadUpdate: (refused: Refused) => downloadUpdate(refused),
    skipUpdate: (refused: Refused) => skipUpdate(refused),
  },
}))

import { checkEveryMs, firstCheckMs, UpdateDialog, useUpdateCheck } from './updates'

/** Harness is the shell's part in the check: the hook, a way to ask from Help and the dialog. */
function Harness() {
  const updates = useUpdateCheck()
  return (
    <>
      <button type="button" onClick={updates.checkNow}>
        Ask
      </button>
      <UpdateDialog found={updates.found} onClose={updates.dismiss} />
    </>
  )
}

/** answer is an update check's answer with the given outcome, running 1.4.2 against 1.5.0. */
const answer = (outcome: Update['outcome']): Update => ({ outcome, running: '1.4.2', latest: '1.5.0' })

beforeEach(() => {
  for (const spy of [checkForUpdates, downloadUpdate, skipUpdate]) spy.mockReset()
  downloadUpdate.mockResolvedValue(undefined)
  skipUpdate.mockResolvedValue(undefined)
})

afterEach(() => {
  vi.useRealTimers()
})

/** launched renders the harness and lets the first automatic check run and answer. */
async function launched(found: Update | null) {
  vi.useFakeTimers()
  checkForUpdates.mockResolvedValue(found)
  render(<Harness />)
  await act(async () => {
    await vi.advanceTimersByTimeAsync(firstCheckMs)
  })
  vi.useRealTimers()
}

/** asked renders the harness and asks from Help, answering with found. */
async function asked(found: Update | null) {
  checkForUpdates.mockResolvedValue(found)
  render(<Harness />)
  fireEvent.click(screen.getByRole('button', { name: 'Ask' }))
  return screen.findByRole('dialog')
}

describe('the automatic check', () => {
  it('checks 3 seconds after the page loads, then once a day', async () => {
    vi.useFakeTimers()
    checkForUpdates.mockResolvedValue(answer('current'))
    render(<Harness />)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(firstCheckMs - 1)
    })
    expect(checkForUpdates).not.toHaveBeenCalled()

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1)
    })
    expect(checkForUpdates).toHaveBeenCalledTimes(1)
    expect(checkForUpdates.mock.calls[0][0]).toBe(false)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(checkEveryMs)
    })
    expect(checkForUpdates).toHaveBeenCalledTimes(2)
    expect(firstCheckMs).toBe(3000)
    expect(checkEveryMs).toBe(24 * 60 * 60 * 1000)
  })

  it('stays silent when an automatic check offers nothing', async () => {
    for (const found of [answer('current'), answer('skipped'), answer('unreachable'), answer('uncomparable'), null]) {
      await launched(found)
      expect(screen.queryByRole('dialog')).toBeNull()
      cleanup()
    }
    expect(checkForUpdates).toHaveBeenCalledTimes(5)
  })
})

describe('the prompt', () => {
  it('offers Download, Skip this version and Later for a newer release', async () => {
    await launched(answer('available'))

    const dialog = await screen.findByRole('dialog', { name: 'Update available' })
    expect(dialog.textContent).toContain('the application 1.5.0 is available. You are running 1.4.2.')
    const buttons = Array.from(dialog.querySelectorAll('footer button')).map((each) => each.textContent)
    expect(buttons).toEqual(['Download', 'Skip this version', 'Later'])
    expect(document.activeElement?.textContent).toBe('Download')
  })

  it('hands the download to the facade and closes', async () => {
    await launched(answer('available'))
    fireEvent.click(await screen.findByRole('button', { name: 'Download' }))

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(downloadUpdate).toHaveBeenCalledTimes(1)
    expect(skipUpdate).not.toHaveBeenCalled()
  })

  it('keeps the dialog open to say why a download could not be handed over', async () => {
    downloadUpdate.mockImplementation((refused) => {
      refused('Error: there is no window to open the browser from')
      return Promise.resolve()
    })
    await launched(answer('available'))
    fireEvent.click(await screen.findByRole('button', { name: 'Download' }))

    expect((await screen.findByRole('alert')).textContent).toBe(
      'Error: there is no window to open the browser from',
    )
    expect(screen.getByRole('dialog')).toBeTruthy()
  })

  it('skips the offered version and closes', async () => {
    await launched(answer('available'))
    fireEvent.click(await screen.findByRole('button', { name: 'Skip this version' }))

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(skipUpdate).toHaveBeenCalledTimes(1)
    expect(downloadUpdate).not.toHaveBeenCalled()
  })

  it('keeps the dialog open to say why a skip could not be kept', async () => {
    skipUpdate.mockImplementation((refused) => {
      refused('Error: remembering the choice: the disk is full')
      return Promise.resolve()
    })
    await launched(answer('available'))
    fireEvent.click(await screen.findByRole('button', { name: 'Skip this version' }))

    expect((await screen.findByRole('alert')).textContent).toContain('the disk is full')
  })

  it('closes on Later, changing nothing', async () => {
    await launched(answer('available'))
    fireEvent.click(await screen.findByRole('button', { name: 'Later' }))

    expect(screen.queryByRole('dialog')).toBeNull()
    expect(downloadUpdate).not.toHaveBeenCalled()
    expect(skipUpdate).not.toHaveBeenCalled()
  })
})

describe('Help, then Check for updates', () => {
  it('tells every outcome of Help, then Check for updates', async () => {
    const told: Array<[Update | null, string]> = [
      [answer('current'), 'You are running the latest version.'],
      [answer('unreachable'), 'The update check could not reach GitHub. Please try again later.'],
      [null, 'The update check could not reach GitHub. Please try again later.'],
      [
        { outcome: 'uncomparable', running: '0.0.0-dev', latest: '1.5.0' },
        'This copy was built from source as 0.0.0-dev, so there is no released version to compare it with.',
      ],
    ]
    for (const [found, words] of told) {
      const dialog = await asked(found)
      expect(dialog.getAttribute('aria-label')).toBe('Check for updates')
      expect(dialog.textContent).toContain(words)
      cleanup()
    }
    expect(checkForUpdates.mock.calls.every(([manual]) => manual)).toBe(true)
  })

  it('offers a newer release as the prompt', async () => {
    const dialog = await asked(answer('available'))
    expect(dialog.getAttribute('aria-label')).toBe('Update available')
  })
})
