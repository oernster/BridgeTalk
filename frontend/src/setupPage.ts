// The shipped setup page laid out for a test, with the fake setup program that answers it.
//
// The setup page has no build step and no test runner of its own, so the page is laid out here
// from the very file it ships and its scripts are run the way the page runs them. Every suite over
// the page's screens reads it from here, so no two of them can lay the page out differently.

import page from '../../installer/frontend/dist/index.html?raw'
import shell from '../../installer/frontend/dist/setup-shell.js?raw'
import routes from '../../installer/frontend/dist/setup-routes.js?raw'

/** State is the reading of the machine the setup program hands the page. */
export interface State {
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
  keepablePlugins: string
}

/** Options are the choices the page hands an install. */
export interface Options {
  installDir?: string
  startMenu: boolean
  desktop: boolean
  launchOnBoot: boolean
}

/** InstallLocation answers a pick of the install folder. */
export interface InstallLocation {
  dir: string
  refusal: string
}

/** FakeSetup stands in for the setup program, recording the calls that change anything. */
export class FakeSetup {
  running = false
  quits = 0
  /** refusal is what every call that changes the machine answers with in place of doing it. */
  refusal = ''
  /** answer resolves; else it rejects with the refusal as the setup program's bound methods reject. */
  answer = (): Promise<void> => (this.refusal === '' ? Promise.resolve() : Promise.reject(this.refusal))
  /** uninstalls records each removal as whether to forget the settings, then the plugins. */
  uninstalls: [boolean, boolean][] = []
  installs: Options[] = []
  /** picks are the answers the folder picker gives, one per press of Change. */
  picks: InstallLocation[] = []
  /** pickedFrom records the folder each pick was opened from. */
  pickedFrom: string[] = []

  AppRunning = (): Promise<boolean> => Promise.resolve(this.running)
  Quit = (): void => {
    this.quits++
  }
  Uninstall = (forget: boolean, removePlugins: boolean): Promise<void> => {
    this.uninstalls.push([forget, removePlugins])
    return this.answer()
  }
  Repair = (): Promise<void> => this.answer()
  Install = (choices: Options): Promise<void> => {
    this.installs.push(choices)
    return this.answer()
  }
  ChooseInstallLocation = (current: string): Promise<InstallLocation> => {
    this.pickedFrom.push(current)
    return Promise.resolve(this.picks.shift() ?? { dir: '', refusal: '' })
  }
  LaunchApp = (): Promise<void> => Promise.resolve()
  CloseRunningApp = (): Promise<void> => this.answer()
  TakeKeyboard = (): Promise<void> => Promise.resolve()
  SetShortcuts = (): Promise<void> => Promise.resolve()
  SetLaunchOnBoot = (): Promise<void> => Promise.resolve()
}

/** The page's own names, as its scripts leave them on the global scope. */
export interface SetupPage {
  route: (state: State) => void
  init: () => Promise<void>
}

/** Webview is the part of the window the page may close through without the setup program. */
interface Webview {
  go?: unknown
  runtime?: unknown
  chrome?: unknown
}

const bodyOfPage = /<body>([\s\S]*)<\/body>/

/** installed is a machine with this version already on it, opened as a double-click opens it. */
export const installed: State = {
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
  keepablePlugins: '',
}

/**
 * layPage puts the shipped page on the document and runs its scripts once, as the page does. The
 * setup program is the one given; null lays out a page that never reached one. Whatever an earlier
 * test left in place of the Wails runtime or the webview is taken away first.
 */
export function layPage(setup: FakeSetup | null): SetupPage {
  const body = bodyOfPage.exec(page)
  if (body === null) throw new Error('the setup page has no body')
  document.body.innerHTML = body[1]
  const webview = window as unknown as Webview
  delete webview.runtime
  delete webview.chrome
  if (setup === null) delete webview.go
  else webview.go = { main: { App: setup } }
  // An indirect eval runs the scripts in the global scope, as script tags do. They go in as one
  // because the routes read names the shell declares.
  const evaluate = eval
  evaluate(shell + '\n' + routes)
  return window as unknown as SetupPage
}

/** focusedLabel names the button holding focus. */
export function focusedLabel(): string {
  return document.activeElement?.textContent ?? ''
}

/** activeScreen names the screen on show. */
export function activeScreen(): string {
  return document.querySelector('.screen.active')?.id ?? ''
}

/** footerLabels names the footer's buttons in order. */
export function footerLabels(): string[] {
  return Array.from(document.querySelectorAll('#footer .btn')).map((button) => button.textContent ?? '')
}

/** footerButton finds a footer button by its label, failing loudly rather than handing back null. */
export function footerButton(label: string): HTMLButtonElement {
  const found = Array.from(document.querySelectorAll<HTMLButtonElement>('#footer .btn')).find(
    (button) => button.textContent === label,
  )
  if (found === undefined) throw new Error(`no ${label} button in the footer`)
  return found
}

/** pageElement finds one element of the page by id, failing loudly rather than handing back null. */
export function pageElement(id: string): HTMLElement {
  const found = document.getElementById(id)
  if (found === null) throw new Error(`no #${id} on the page`)
  return found
}

/** settle lets the page's awaited calls to the fake answer. */
export function settle(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0))
}
