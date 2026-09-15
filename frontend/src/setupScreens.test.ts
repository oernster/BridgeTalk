// Where each setup screen puts focus as it opens (FR-808) and what Cancel does on the Uninstall
// screen (FR-805).
//
// The setup page has no build step and no test runner of its own, so the page is laid out here
// from the very file it ships and its scripts are run the way the page runs them. The setup
// program behind the page is a hand-written fake that answers the calls the screens make. What is
// asserted is the screen shown, the button holding focus and the calls made.

import { beforeEach, describe, expect, it } from 'vitest'
import page from '../../installer/frontend/dist/index.html?raw'
import shell from '../../installer/frontend/dist/setup-shell.js?raw'
import routes from '../../installer/frontend/dist/setup-routes.js?raw'

/** State is the reading of the machine the setup program hands the page. */
interface State {
  appName: string
  mode: string
  relation: string
  installed: boolean
  installedVersion: string
  thisVersion: string
  installDir: string
  launchOnBoot: boolean
  startMenu: boolean
  desktop: boolean
  prefersDark: boolean
}

/** Options are the choices the page hands an install. */
interface Options {
  installDir?: string
  startMenu: boolean
  desktop: boolean
  launchOnBoot: boolean
}

/** InstallLocation answers a pick of the install folder. */
interface InstallLocation {
  dir: string
  refusal: string
}

/** FakeSetup stands in for the setup program, recording the calls that change anything. */
class FakeSetup {
  running = false
  quits = 0
  uninstalls: boolean[] = []
  installs: Options[] = []
  /** picks are the answers the folder picker gives, one per press of Change. */
  picks: InstallLocation[] = []
  /** pickedFrom records the folder each pick was opened from. */
  pickedFrom: string[] = []

  AppRunning = (): Promise<boolean> => Promise.resolve(this.running)
  Quit = (): void => {
    this.quits++
  }
  Uninstall = (forget: boolean): Promise<void> => {
    this.uninstalls.push(forget)
    return Promise.resolve()
  }
  Repair = (): Promise<void> => Promise.resolve()
  Install = (choices: Options): Promise<void> => {
    this.installs.push(choices)
    return Promise.resolve()
  }
  ChooseInstallLocation = (current: string): Promise<InstallLocation> => {
    this.pickedFrom.push(current)
    return Promise.resolve(this.picks.shift() ?? { dir: '', refusal: '' })
  }
  LaunchApp = (): Promise<void> => Promise.resolve()
  TakeKeyboard = (): Promise<void> => Promise.resolve()
  SetShortcuts = (): Promise<void> => Promise.resolve()
  SetLaunchOnBoot = (): Promise<void> => Promise.resolve()
}

/** The page's own names, as its scripts leave them on the global scope. */
interface SetupPage {
  route: (state: State) => void
}

const bodyOfPage = /<body>([\s\S]*)<\/body>/

/** installed is a machine with this version already on it, opened as a double-click opens it. */
const installed: State = {
  appName: 'Product',
  mode: 'manage',
  relation: 'same',
  installed: true,
  installedVersion: '1.0.0',
  thisVersion: '1.0.0',
  installDir: 'C:\\Programs\\Product',
  launchOnBoot: false,
  startMenu: true,
  desktop: true,
  prefersDark: false,
}

let setup: FakeSetup

/** layPage puts the shipped page on the document and runs its scripts once, as the page does. */
function layPage(): SetupPage {
  const body = bodyOfPage.exec(page)
  if (body === null) throw new Error('the setup page has no body')
  document.body.innerHTML = body[1]
  setup = new FakeSetup()
  ;(window as unknown as { go: unknown }).go = { main: { App: setup } }
  // An indirect eval runs the scripts in the global scope, as script tags do. They go in as one
  // because the routes read names the shell declares.
  const evaluate = eval
  evaluate(shell + '\n' + routes)
  return window as unknown as SetupPage
}

/** focusedLabel names the button holding focus. */
function focusedLabel(): string {
  return document.activeElement?.textContent ?? ''
}

/** activeScreen names the screen on show. */
function activeScreen(): string {
  return document.querySelector('.screen.active')?.id ?? ''
}

/** footerButton finds a footer button by its label, failing loudly rather than handing back null. */
function footerButton(label: string): HTMLButtonElement {
  const found = Array.from(document.querySelectorAll<HTMLButtonElement>('#footer .btn')).find(
    (button) => button.textContent === label,
  )
  if (found === undefined) throw new Error(`no ${label} button in the footer`)
  return found
}

/** settle lets the page's awaited calls to the fake answer. */
function settle(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

let setupPage: SetupPage

beforeEach(() => {
  setupPage = layPage()
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

/** pageElement finds one element of the page by id, failing loudly rather than handing back null. */
function pageElement(id: string): HTMLElement {
  const found = document.getElementById(id)
  if (found === null) throw new Error(`no #${id} on the page`)
  return found
}

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
    expect(setup.uninstalls).toEqual([false])
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
