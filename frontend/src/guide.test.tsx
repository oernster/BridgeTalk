// The guide pane.
//
// The guide names the furniture by its REAL picture, so two things are guarded above the rest.
// Every entry draws exactly the pictures it declares. Every picture the band draws has an entry:
// a button the guide never shows is a picture with no caption anywhere in the window, which is
// how a guide falls behind the window it describes. The band is rendered in both of its states,
// since Mute and the theme each draw one picture at a time.

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import type { About, State } from './api'
import { guideSections } from './guideContent'

const about = vi.fn<() => Promise<About | null>>()
const state = vi.fn<() => Promise<State | null>>()

vi.mock('./api', () => ({
  api: {
    about: () => about(),
    state: () => state(),
    setMuted: () => Promise.resolve(),
    setVolume: () => Promise.resolve(),
    selectVoice: () => Promise.resolve(),
    takeKeyboard: () => Promise.resolve(),
    quit: () => Promise.resolve(),
    minimiseToTray: () => Promise.resolve(),
    requestQuit: () => Promise.resolve(),
    voices: () => Promise.resolve([]),
    cueBreakdown: (name: string) =>
      Promise.resolve({ voice: name, served: [], unserved: [] }),
    auditionGroups: () => Promise.resolve([]),
    audition: () => Promise.resolve(null),
    stopAudition: () => Promise.resolve(),
    playing: () => Promise.resolve(false),
    reactions: () => Promise.resolve([]),
    chooseLibraryRoot: () => Promise.resolve(''),
    chooseJournalDir: () => Promise.resolve(''),
    setLaunchOnBoot: () => Promise.resolve(),
  },
  on: () => () => undefined,
}))

const { GuidePane } = await import('./guide')
const { App } = await import('./App')

const named: About = {
  name: 'The Product',
  tagline: '',
  version: '9.9.9',
  author: '',
  copyright: '',
  authorship: '',
  attribution: '',
  licence: '',
  credits: [],
}

const watching: State = {
  voice: 'Grace',
  bound: 40,
  total: 60,
  muted: false,
  silent: false,
  journalDir: 'D:/Journals',
  statusPath: 'D:/Journals/Status.json',
  libraryRoot: 'D:/Recordings',
  version: '9.9.9',
  launchOnBoot: false,
  stalls: 0,
  worstStall: 0,
}

beforeEach(() => {
  window.localStorage.clear()
  about.mockReset()
  about.mockResolvedValue(named)
  state.mockReset()
  state.mockResolvedValue(watching)
})

describe('the guide pane', () => {
  // The name is written down in one place and read from there, so the title waits for it.
  it('titles itself with the name the application gives', async () => {
    render(<GuidePane />)

    expect(await screen.findByRole('heading', { name: 'How The Product works' })).toBeTruthy()
  })

  // A fallback would be a second place the product is named, so no answer means no title.
  it('writes no title while the application has not named itself', async () => {
    about.mockResolvedValue(null)
    render(<GuidePane />)

    await waitFor(() => expect(about).toHaveBeenCalled())
    expect(screen.queryByRole('heading', { name: /^How / })).toBeNull()
  })

  it('draws every section of the guide', () => {
    render(<GuidePane />)

    for (const section of guideSections) {
      expect(screen.getByRole('heading', { name: section.heading })).toBeTruthy()
    }
  })

  // An entry without its picture is the defect the whole screen exists to avoid.
  it('gives every entry the pictures it declares, in order', () => {
    const { container } = render(<GuidePane />)

    const declared = guideSections
      .flatMap((section) => section.entries ?? [])
      .flatMap((entry) => entry.icons)
    const drawn = Array.from(container.querySelectorAll('.guide-entry .guide-icon')).map(
      (img) => img.getAttribute('src'),
    )
    expect(drawn).toEqual(declared)
  })

  it('draws every rule and every paragraph', () => {
    render(<GuidePane />)

    for (const section of guideSections) {
      for (const rule of section.rules ?? []) {
        expect(screen.getByText(rule.title)).toBeTruthy()
      }
      for (const text of section.paragraphs ?? []) {
        expect(screen.getByText(text)).toBeTruthy()
      }
    }
  })
})

/** bandPictures renders the window in one state and reads every picture its band draws. */
async function bandPictures(muted: boolean, theme: 'dark' | 'light'): Promise<string[]> {
  window.localStorage.setItem('bridge-talk.theme', theme)
  state.mockResolvedValue({ ...watching, muted })
  const { container, unmount } = render(<App />)
  const band = container.querySelector('.navband') as HTMLElement
  await within(band).findByRole('button', { name: muted ? 'Unmute' : 'Mute' })
  const pictures = Array.from(band.querySelectorAll('img')).map(
    (img) => img.getAttribute('src') ?? '',
  )
  unmount()
  return pictures
}

describe('the guide against the band', () => {
  it('shows every picture the band draws, in both of its states', async () => {
    const shown = new Set(
      guideSections.flatMap((section) => section.entries ?? []).flatMap((entry) => entry.icons),
    )

    const sounding = await bandPictures(false, 'dark')
    const silenced = await bandPictures(true, 'light')

    // The second state has to draw something the first did not; otherwise the guard below would
    // be checking one state twice.
    expect(silenced.some((picture) => !sounding.includes(picture))).toBe(true)
    for (const picture of [...sounding, ...silenced]) {
      expect(shown.has(picture), `the band draws ${picture} but the guide never shows it`).toBe(
        true,
      )
    }
  })
})
