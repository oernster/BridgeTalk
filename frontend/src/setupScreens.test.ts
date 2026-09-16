// Where each setup screen puts focus as it opens (FR-808), what Cancel does on the Uninstall screen
// (FR-805), where the install goes (FR-809), what the Uninstall screen offers about the plugins
// folder (FR-578) and what a refused step shows (FR-807).
//
// The page is laid out from the very file it ships, with a hand-written fake setup program behind it
// (setupPage.ts). What is asserted is the screen shown, the button holding focus and the calls made.

import { beforeEach, describe, expect, it } from 'vitest'
import {
  activeScreen,
  FakeSetup,
  focusedLabel,
  footerButton,
  footerLabels,
  installed,
  layPage,
  pageElement,
  settle,
  type SetupPage,
  type State,
} from './setupPage'

let setup: FakeSetup
let setupPage: SetupPage

beforeEach(() => {
  setup = new FakeSetup()
  setupPage = layPage(setup)
})

describe('each setup screen opens on the action it leads with', () => {
  const screens: [string, State, string, string][] = [
    ['Install', { ...installed, mode: 'install', installed: false }, 'screen-install', 'Install'],
    ['Update', { ...installed, relation: 'newer' }, 'screen-update', 'Update'],
    ['Go back', { ...installed, relation: 'older' }, 'screen-update', 'Go back'],
    ['Manage', installed, 'screen-manage', 'Repair'],
    ['Uninstall', { ...installed, mode: 'uninstall' }, 'screen-uninstall', 'Uninstall'],
  ]
  it.each(screens)('the %s screen', (_name, state, screen, lead) => {
    setupPage.route(state)
    expect(activeScreen()).toBe(screen)
    expect(focusedLabel()).toBe(lead)
  })

  it('the Uninstall screen reached from the Manage screen', () => {
    setupPage.route(installed)
    footerButton('Uninstall').click()
    expect(activeScreen()).toBe('screen-uninstall')
    expect(focusedLabel()).toBe('Uninstall')
  })

  it('the screen that offers to close a running copy', async () => {
    setup.running = true
    setupPage.route(installed)
    footerButton('Repair').click()
    await settle()
    expect(activeScreen()).toBe('screen-running')
    expect(focusedLabel()).toBe('Close it and continue')
  })

  it('the screen that says it is done', async () => {
    setupPage.route({ ...installed, mode: 'uninstall' })
    footerButton('Uninstall').click()
    await settle()
    expect(activeScreen()).toBe('screen-done')
    expect(focusedLabel()).toBe('Close')
  })
})

describe('the install location (FR-809)', () => {
  const nothingInstalled: State = { ...installed, mode: 'install', installed: false }

  it('installs where it offers when nothing is changed', async () => {
    setupPage.route(nothingInstalled)
    expect(pageElement('install-path').textContent).toBe(installed.installDir)

    footerButton('Install').click()
    await settle()
    expect(setup.installs.map((choices) => choices.installDir)).toEqual([installed.installDir])
  })

  it('shows the folder picked and installs into it', async () => {
    setup.picks.push({ dir: 'D:\\Games\\Product', refusal: '' })
    setupPage.route(nothingInstalled)

    pageElement('install-change').click()
    await settle()
    expect(setup.pickedFrom).toEqual([installed.installDir])
    expect(pageElement('install-path').textContent).toBe('D:\\Games\\Product')
    expect(pageElement('install-refusal').hidden).toBe(true)

    footerButton('Install').click()
    await settle()
    expect(setup.installs.map((choices) => choices.installDir)).toEqual(['D:\\Games\\Product'])
  })

  it('names a folder that will not do and keeps the last one that would', async () => {
    const reason = 'D:\\Stuff\\Product already holds files, which uninstalling would delete'
    setup.picks.push({ dir: 'D:\\Stuff\\Product', refusal: reason })
    setupPage.route(nothingInstalled)

    pageElement('install-change').click()
    await settle()
    expect(pageElement('install-refusal').hidden).toBe(false)
    expect(pageElement('install-refusal').textContent).toBe(reason)
    expect(pageElement('install-path').textContent).toBe(installed.installDir)

    footerButton('Install').click()
    await settle()
    expect(setup.installs.map((choices) => choices.installDir)).toEqual([installed.installDir])
  })

  it('changes nothing when the picker is closed without a choice', async () => {
    setupPage.route(nothingInstalled)

    pageElement('install-change').click()
    await settle()
    expect(pageElement('install-path').textContent).toBe(installed.installDir)
    expect(pageElement('install-refusal').hidden).toBe(true)
  })

  it('leaves the folder to the setup program on every other screen', async () => {
    setupPage.route({ ...installed, relation: 'newer' })
    footerButton('Update').click()
    await settle()
    expect(setup.installs.map((choices) => choices.installDir)).toEqual([''])
  })
})

describe('the Uninstall screen', () => {
  it('leaves forgetting the settings unticked', async () => {
    setupPage.route({ ...installed, mode: 'uninstall' })
    footerButton('Uninstall').click()
    await settle()
    expect(setup.uninstalls).toEqual([[false, false]])
  })

  it('returns to the screen setup opened on when Cancel is pressed', () => {
    setupPage.route(installed)
    footerButton('Uninstall').click()
    footerButton('Cancel').click()
    expect(activeScreen()).toBe('screen-manage')
    expect(setup.quits).toBe(0)
  })

  it('closes setup on Cancel when opened with -uninstall', () => {
    setupPage.route({ ...installed, mode: 'uninstall' })
    footerButton('Cancel').click()
    expect(setup.quits).toBe(1)
  })
})

describe('the plugins folder on the Uninstall screen (FR-578)', () => {
  const plugins = 'C:\\Programs\\Product\\plugins'

  /** optionLabels names the boxes the Uninstall screen offers, in order. */
  function optionLabels(): string[] {
    return Array.from(document.querySelectorAll('#uninstall-options .label')).map(
      (label) => label.textContent ?? '',
    )
  }

  it('offers nothing about plugins where the folder holds nothing', async () => {
    setupPage.route({ ...installed, mode: 'uninstall' })
    expect(optionLabels()).toEqual(['Also forget my settings'])
    footerButton('Uninstall').click()
    await settle()
    expect(setup.uninstalls).toEqual([[false, false]])
  })

  it('keeps the plugins unless asked, saying where they are', async () => {
    setupPage.route({ ...installed, mode: 'uninstall', keepablePlugins: plugins })
    expect(optionLabels()).toEqual(['Also forget my settings', 'Also remove my plugins'])
    footerButton('Uninstall').click()
    await settle()
    expect(setup.uninstalls).toEqual([[false, false]])
    expect(document.getElementById('done-msg')?.textContent).toContain(plugins)
  })

  it('removes the plugins when the box is ticked, saying nothing is kept', async () => {
    setupPage.route({ ...installed, mode: 'uninstall', keepablePlugins: plugins })
    const boxes = document.querySelectorAll<HTMLInputElement>('#uninstall-options input')
    boxes[1].checked = true
    footerButton('Uninstall').click()
    await settle()
    expect(setup.uninstalls).toEqual([[false, true]])
    expect(document.getElementById('done-msg')?.textContent).not.toContain(plugins)
  })
})

describe('a failure says why (FR-807)', () => {
  const refusal = 'extract files: writing C:\\Programs\\Product\\Product.exe: the disk is full'

  /** failures names each screen, the machine it opens on and the go-ahead pressed while every change is refused. */
  const failures: [string, State, string][] = [
    ['an install', { ...installed, mode: 'install', installed: false }, 'Install'],
    ['an update', { ...installed, relation: 'newer' }, 'Update'],
    ['a repair', installed, 'Repair'],
    ['an uninstall', { ...installed, mode: 'uninstall' }, 'Uninstall'],
  ]
  it.each(failures)('%s that is refused shows the reason and Close alone', async (_name, state, press) => {
    setup.refusal = refusal
    setupPage.route(state)
    footerButton(press).click()
    await settle()
    await settle()
    expect(activeScreen()).toBe('screen-error')
    expect(pageElement('screen-error').querySelector('h1')?.textContent).toBe('Something went wrong')
    expect(pageElement('error-msg').textContent).toBe(refusal)
    expect(footerLabels()).toEqual(['Close'])
    footerButton('Close').click()
    expect(setup.quits).toBe(1)
  })

  it('a running copy that cannot be closed shows the reason and Close alone', async () => {
    setup.running = true
    setup.refusal = 'the application is still running after 5 seconds; close it by hand'
    setupPage.route(installed)
    footerButton('Repair').click()
    await settle()
    footerButton('Close it and continue').click()
    await settle()
    expect(activeScreen()).toBe('screen-error')
    expect(pageElement('error-msg').textContent).toBe(setup.refusal)
    footerButton('Close').click()
    expect(setup.quits).toBe(1)
  })
})
