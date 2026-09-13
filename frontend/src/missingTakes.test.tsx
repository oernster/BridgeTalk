// The Missing takes pane: which voices are still missing takes and where each file goes.
//
// What these guard is that only a voice still missing a take is offered, an empty one
// included; that the chooser is there whatever the directory holds; that the list says
// what is missing, each moment under its full title alone, with the folder its file belongs in;
// that the button opens the folder for the moment it sits beside.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { Checklist, CueEntry } from './api'

const voiceDirectories = vi.fn<() => Promise<string[]>>()
const checklist = vi.fn<(voice: string) => Promise<Checklist>>()
const openMomentFolder = vi.fn<(voice: string, id: string) => Promise<void>>()
const rescan = vi.fn<() => Promise<number>>()
const chooseLibraryRoot = vi.fn<() => Promise<string>>()

vi.mock('./api', () => ({
  api: {
    voiceDirectories: () => voiceDirectories(),
    checklist: (voice: string) => checklist(voice),
    openMomentFolder: (voice: string, id: string) => openMomentFolder(voice, id),
    rescan: () => rescan(),
    chooseLibraryRoot: () => chooseLibraryRoot(),
  },
}))

const { MissingTakesPane } = await import('./missingTakes')

const docked: CueEntry = {
  id: 'Docked',
  title: 'Docked',
  folder: 'Docked',
  purpose: 'When the ship docks.',
}
const undocked: CueEntry = {
  id: 'Undocked',
  title: 'Undocked',
  folder: 'Undocked',
  purpose: 'When the ship leaves its pad.',
}
const hyperspace: CueEntry = {
  id: 'StartJump.JumpType.Hyperspace',
  title: 'Start jump: jump type hyperspace',
  folder: 'StartJump_JumpType_Hyperspace',
  purpose: 'When a hyperspace jump to another system begins.',
}

/** renderedPurpose opens the pane on Oliver and returns the hyperspace moment's purpose. */
const renderedPurpose = async () => {
  render(<MissingTakesPane cast="Oliver" />)
  return screen.findByText(hyperspace.purpose)
}

/** progress builds a voice's checklist over a vocabulary of three moments. */
const progress = (voice: string, missing: CueEntry[]): Checklist => ({
  voice,
  recorded: 3 - missing.length,
  total: 3,
  missing,
  folder: `D:\\Recordings\\${voice}\\`,
})

// What each voice misses. Built afresh before every test, so a test that changes it for a
// look cannot leave the change behind for the next one even where it fails part way.
let fixture: Record<string, CueEntry[]> = {}

beforeEach(() => {
  // Grace is complete, Hugo is an empty folder and Oliver holds one take of three.
  fixture = { Grace: [], Hugo: [docked, hyperspace, undocked], Oliver: [docked, hyperspace] }
  for (const spy of [voiceDirectories, checklist, openMomentFolder, rescan, chooseLibraryRoot]) {
    spy.mockReset()
  }
  voiceDirectories.mockResolvedValue(['Grace', 'Hugo', 'Oliver'])
  checklist.mockImplementation((voice) => Promise.resolve(progress(voice, fixture[voice] ?? [])))
  openMomentFolder.mockResolvedValue(undefined)
  rescan.mockResolvedValue(1)
  chooseLibraryRoot.mockResolvedValue('')
})

/**
 * chooser finds the voice chooser once the lists have landed. findAll rather than find,
 * because the complete state names itself twice: in the chooser and in the pane.
 */
async function chooser(): Promise<HTMLSelectElement> {
  await screen.findAllByText(/has recordings for|No voices yet|Every voice is complete/)
  return screen.getByRole('combobox') as HTMLSelectElement
}

/** browse presses the recordings row's Browse button. */
function browse() {
  fireEvent.click(screen.getByRole('button', { name: 'Browse' }))
}

// The recordings directory lives here rather than in Settings, beside the voices it holds.
describe('the recordings directory on the missing takes pane', () => {
  // Each waits for the lists to land before it ends, so no update arrives after the test
  // and the run stays free of the warning that hides real ordering faults.
  it('shows the directory it reads from', async () => {
    render(<MissingTakesPane cast="Oliver" libraryRoot="D:/Recordings" />)

    expect(screen.getByText('D:/Recordings')).toBeTruthy()
    await chooser()
  })

  it('says when no directory is chosen and waits while the state is on its way', async () => {
    const { unmount } = render(<MissingTakesPane cast="Oliver" libraryRoot="" />)
    expect(screen.getByText('None chosen yet')).toBeTruthy()
    await chooser()
    unmount()

    render(<MissingTakesPane cast="Oliver" />)
    expect(screen.getByText('...')).toBeTruthy()
    await chooser()
  })

  it('confirms the directory it took, then reads its folders at once', async () => {
    chooseLibraryRoot.mockResolvedValue('D:/Takes')
    render(<MissingTakesPane cast="Oliver" libraryRoot="D:/Recordings" />)
    await waitFor(() => expect(voiceDirectories).toHaveBeenCalledTimes(1))

    browse()

    const said = await screen.findByRole('status')
    expect(said.textContent).toContain('The recordings directory is now D:/Takes')
    expect(said.className).toContain('taken')
    await waitFor(() => expect(voiceDirectories).toHaveBeenCalledTimes(2))
  })

  // The fault this guards was silence: a directory with no voices refused in the body
  // colour read as a button that did nothing.
  it('says why a directory was refused, as a refusal under its own row', async () => {
    chooseLibraryRoot.mockRejectedValue('no voices in D:/Empty')
    render(<MissingTakesPane cast="Oliver" libraryRoot="D:/Recordings" />)

    browse()

    const said = await screen.findByRole('alert')
    expect(said.textContent).toContain('no voices in D:/Empty')
    expect(said.className).toContain('refused')
    expect(said.previousElementSibling?.textContent).toContain('Recordings')
  })

  it('says nothing at all when the dialog is cancelled', async () => {
    render(<MissingTakesPane cast="Oliver" libraryRoot="D:/Recordings" />)
    await screen.findByText('Oliver has recordings for 1 of 3 moments.')

    browse()

    // Waiting for the press to settle first, so this cannot pass by being early.
    await waitFor(() => expect(chooseLibraryRoot).toHaveBeenCalled())
    expect(screen.queryByRole('alert')).toBeNull()
    expect(screen.queryByRole('status')).toBeNull()
    expect(voiceDirectories).toHaveBeenCalledTimes(1)
  })

  it('clears what the last press said before the next one answers', async () => {
    chooseLibraryRoot.mockRejectedValue('no voices in D:/Empty')
    render(<MissingTakesPane cast="Oliver" libraryRoot="D:/Recordings" />)
    browse()
    await screen.findByRole('alert')

    chooseLibraryRoot.mockResolvedValue('')
    browse()

    await waitFor(() => expect(screen.queryByRole('alert')).toBeNull())
  })
})

describe('the missing takes pane', () => {
  // FR-316: a complete voice has nothing to record; an empty folder has everything.
  it('offers only the voices still missing takes, each with what it misses', async () => {
    render(<MissingTakesPane cast="Oliver" />)
    const select = await chooser()

    expect(Array.from(select.options).map((option) => option.textContent)).toEqual([
      'Hugo: incomplete, 3 of 3 missing',
      'Oliver: incomplete, 2 of 3 missing',
    ])
    expect(select.value).toBe('Oliver')
  })

  it('opens on the first voice still missing a take when the cast voice is complete', async () => {
    render(<MissingTakesPane cast="Grace" />)

    expect((await chooser()).value).toBe('Hugo')
  })

  // FR-311 and FR-313: each missing moment says where its audio file goes. FR-233: under its
  // full title alone, with no heading repeating the start of that title.
  it('lists each missing moment under its full title with the folder its file belongs in', async () => {
    render(<MissingTakesPane cast="Oliver" />)

    expect(await screen.findByText('Oliver has recordings for 1 of 3 moments.')).toBeTruthy()
    expect(screen.getByText(hyperspace.title)).toBeTruthy()
    expect(screen.queryByText('Start jump')).toBeNull()
    expect(screen.queryAllByRole('heading', { level: 3 })).toHaveLength(0)
    // FR-229: the folder writes each dot of the id as an underscore.
    expect(screen.getByText('D:\\Recordings\\Oliver\\StartJump_JumpType_Hyperspace')).toBeTruthy()
    expect(screen.queryByText('D:\\Recordings\\Oliver\\Undocked')).toBeNull()
  })

  // FR-318: each missing moment says when it is heard, drawn as a purpose between its title
  // and its folder.
  it('says when each missing take is heard, between its title and its folder', async () => {
    const purpose = await renderedPurpose()
    expect(purpose.className).toBe('purpose')
    const row = purpose.parentElement?.textContent ?? ''
    const order = [hyperspace.title, hyperspace.purpose, hyperspace.folder].map((part) => row.indexOf(part))
    expect(order).not.toContain(-1)
    expect(order).toEqual([...order].sort((first, second) => first - second))
  })

  // FR-314. The button pressed is the second moment's, so one wired to whichever moment
  // happens to be missing first is caught rather than passing by coincidence.
  it('opens the folder for the moment whose button was pressed', async () => {
    render(<MissingTakesPane cast="Oliver" />)

    fireEvent.click(
      await screen.findByRole('button', {
        name: 'Open the folder for Start jump: jump type hyperspace',
      }),
    )

    expect(openMomentFolder).toHaveBeenCalledWith('Oliver', 'StartJump.JumpType.Hyperspace')
  })

  // FR-315.
  it('draws a folder that could not be opened as a refusal', async () => {
    openMomentFolder.mockRejectedValue('that folder could not be made')
    render(<MissingTakesPane cast="Oliver" />)

    fireEvent.click(await screen.findByRole('button', { name: 'Open the folder for Docked' }))

    expect((await screen.findByRole('alert')).textContent).toMatch(/could not be made/)
  })

  it('switches the list to the voice chosen', async () => {
    render(<MissingTakesPane cast="Oliver" />)

    fireEvent.change(await chooser(), { target: { value: 'Hugo' } })

    expect(await screen.findByText('Hugo has recordings for 0 of 3 moments.')).toBeTruthy()
    expect(screen.getByText('D:\\Recordings\\Hugo\\Undocked')).toBeTruthy()
  })

  // A look reads everything again; it must not move the reader off the voice they chose
  // while that voice still misses something. Hugo's count changes on the look, so the
  // assertion can only pass once the look has landed.
  it('keeps the voice chosen across a look', async () => {
    render(<MissingTakesPane cast="Oliver" />)
    fireEvent.change(await chooser(), { target: { value: 'Hugo' } })
    await screen.findByText('Hugo has recordings for 0 of 3 moments.')
    fixture.Hugo = [docked, hyperspace]

    fireEvent.click(screen.getByRole('button', { name: 'Look again' }))

    expect(await screen.findByText('Hugo has recordings for 1 of 3 moments.')).toBeTruthy()
    expect(screen.getByRole<HTMLSelectElement>('combobox').value).toBe('Hugo')
  })

  // FR-316 over time: a voice finished between looks leaves the chooser.
  it('drops a voice from the chooser once a look finds it complete', async () => {
    render(<MissingTakesPane cast="Oliver" />)
    await screen.findByText('Oliver has recordings for 1 of 3 moments.')
    fixture.Oliver = []

    fireEvent.click(screen.getByRole('button', { name: 'Look again' }))

    expect(await screen.findByText('Hugo has recordings for 0 of 3 moments.')).toBeTruthy()
    expect(Array.from((await chooser()).options).map((option) => option.value)).toEqual(['Hugo'])
  })

  // FR-317: the chooser stays and says why it has nothing to offer.
  it('keeps the chooser, saying there are no voices yet, when there is no voice folder', async () => {
    voiceDirectories.mockResolvedValue([])
    render(<MissingTakesPane cast="" />)
    const select = await chooser()

    expect(select.disabled).toBe(true)
    expect(Array.from(select.options).map((option) => option.textContent)).toEqual(['No voices yet'])
    expect(screen.getByText('No voice folders yet.')).toBeTruthy()
    expect(checklist).not.toHaveBeenCalled()
  })

  it('keeps the chooser, saying every voice is complete, when none misses a take', async () => {
    checklist.mockImplementation((voice) => Promise.resolve(progress(voice, [])))
    render(<MissingTakesPane cast="Oliver" />)
    const select = await chooser()

    expect(select.disabled).toBe(true)
    expect(Array.from(select.options).map((option) => option.textContent)).toEqual([
      'Every voice is complete',
    ])
    expect(screen.getByText('Every voice is complete.')).toBeTruthy()
    expect(screen.queryByText(/has recordings for/)).toBeNull()
  })

  it('draws voice folders that could not be read as a refusal', async () => {
    voiceDirectories.mockRejectedValue('the recordings directory could not be read')
    render(<MissingTakesPane cast="Oliver" />)

    expect((await screen.findByRole('alert')).textContent).toMatch(/directory could not be read/)
  })

  it('draws a checklist that could not be read as a refusal', async () => {
    checklist.mockRejectedValue('that voice folder could not be read')
    render(<MissingTakesPane cast="Oliver" />)

    expect((await screen.findByRole('alert')).textContent).toMatch(/folder could not be read/)
  })

  // A refusal belongs to the attempt that met it, so the next attempt takes it down
  // rather than leaving an old reason on screen beside a list that has since loaded.
  it('takes a refusal down when the next look begins', async () => {
    checklist.mockRejectedValueOnce('that voice folder could not be read')
    render(<MissingTakesPane cast="Oliver" />)
    await screen.findByRole('alert')

    fireEvent.click(screen.getByRole('button', { name: 'Look again' }))

    expect(await screen.findByText('Oliver has recordings for 1 of 3 moments.')).toBeTruthy()
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('takes a refusal down when the next open begins', async () => {
    openMomentFolder.mockRejectedValueOnce('that folder could not be made')
    render(<MissingTakesPane cast="Oliver" />)
    const button = await screen.findByRole('button', { name: 'Open the folder for Docked' })
    fireEvent.click(button)
    await screen.findByRole('alert')

    fireEvent.click(button)

    await waitFor(() => expect(openMomentFolder).toHaveBeenCalledTimes(2))
    expect(screen.queryByRole('alert')).toBeNull()
  })

  // FR-214 from here: a look rescans for the whole window, then reads the lists again.
  it('looks again and reads the lists again', async () => {
    render(<MissingTakesPane cast="Oliver" />)
    await screen.findByText('Oliver has recordings for 1 of 3 moments.')
    fixture.Oliver = [hyperspace]

    fireEvent.click(screen.getByRole('button', { name: 'Look again' }))

    expect(await screen.findByText('Oliver has recordings for 2 of 3 moments.')).toBeTruthy()
    expect(rescan).toHaveBeenCalled()
  })

  // A rescan refused for want of a voice is no reason to leave a stale list on screen.
  it('reads the lists again even when the rescan is refused', async () => {
    rescan.mockRejectedValue('no voice was found')
    render(<MissingTakesPane cast="Oliver" />)
    await screen.findByText('Oliver has recordings for 1 of 3 moments.')
    fixture.Oliver = [hyperspace]

    fireEvent.click(screen.getByRole('button', { name: 'Look again' }))

    expect(await screen.findByText('Oliver has recordings for 2 of 3 moments.')).toBeTruthy()
  })
})
