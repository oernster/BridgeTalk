// The shell: menu bar, nav band, switched pane and the modal dialogs.

import { useCallback, useEffect, useRef, useState } from 'react'
import { api, on, type State } from './api'
import { AboutDialog, CloseChoiceDialog, LicenceDialog } from './dialogs'
import { MenuTitle, NavButton, Volume } from './chrome'
import { useRing } from './hooks'
import { useTheme, useVolume } from './preferences'
import {
  AuditionIcon,
  CastIcon,
  ChatterIcon,
  GuideIcon,
  MissingTakesIcon,
  MoonIcon,
  MutedIcon,
  SettingsIcon,
  SoundingIcon,
  StatusIcon,
  SunIcon,
} from './icons'
import { AuditionPane } from './audition'
import { CastPane } from './cast'
import { ChatterPane } from './chatter'
import { GuidePane } from './guide'
import { MissingTakesPane } from './missingTakes'
import { HomePane, SettingsPane } from './panes'
import { Strip } from './strip'

type Pane = 'home' | 'settings' | 'cast' | 'audition' | 'takes' | 'chatter' | 'guide'
type Menu = 'file' | 'audio' | 'settings' | 'help' | null

// How long the keyboard is given to settle on the window before the page decides
// it has not arrived. Long enough for the webview to be handed focus in the normal
// case, short enough that nobody types into a window that is ignoring them.
const keyboardSettleMs = 400

export function App() {
  const [state, setState] = useState<State | null>(null)
  // Cast on launch rather than status. Choosing a voice is the first thing anybody
  // does and the last thing they change; the status pane is where you go once
  // something is already speaking, which makes it the second question rather than
  // the first. Status keeps its place in the band as the leftmost of the pair.
  const [pane, setPane] = useState<Pane>('cast')
  const [menu, setMenu] = useState<Menu>(null)
  const [about, setAbout] = useState(false)
  const [licence, setLicence] = useState(false)
  // Raised by the backend when the cross is pressed, since the window's own close is
  // cancelled and turned into a question rather than acted on.
  const [closing, setClosing] = useState(false)
  const [theme, setTheme] = useTheme()
  const [volume, changeVolume] = useVolume()

  const shell = useRef<HTMLDivElement>(null)
  const sink = useRef<HTMLDivElement>(null)
  // One ring over the window. Every dialog holds a ring of its own; this one stands aside
  // while any scrim is over the window, so no dialog has to be listed here.
  useRing(shell)

  const refresh = useCallback(() => {
    void api.state().then(setState)
  }, [])

  useEffect(() => {
    refresh()
    return on('state', () => refresh())
  }, [refresh])

  // The cross does not close the window; the backend cancels that and asks here
  // instead, because in a resident application the cross means "put it away" as often
  // as it means "stop it".
  useEffect(() => on('close-request', () => setClosing(true)), [])

  // A window summoned back from the notification area opens on the cast, whatever pane
  // it was left on. Hiding it never reloads the page, so it would otherwise return to
  // whichever pane was open when it was put away.
  useEffect(() => on('window-shown', () => setPane('cast')), [])

  // The main window starts neutral: nothing focused, no menu open. A zero-size sink
  // absorbs the initial focus and the first Tab or Right enters the ring.
  //
  // Setting activeElement is not the same as the document HAVING focus. Where the
  // host window keeps the keyboard, every press goes to it, the ring never sees one
  // and no control can paint its ring, which together read as a dead keyboard rather
  // than as a focus that landed elsewhere. The page is the only place that can tell,
  // so it checks once the focus has settled and asks the window to take it back.
  useEffect(() => {
    window.focus()
    sink.current?.focus()
    const timer = window.setTimeout(() => {
      if (document.hasFocus()) return
      void api.takeKeyboard().then(() => {
        window.focus()
        sink.current?.focus()
      })
    }, keyboardSettleMs)
    return () => window.clearTimeout(timer)
  }, [])

  const toggleMute = useCallback(() => {
    void api.setMuted(!state?.muted).then(refresh)
  }, [state?.muted, refresh])

  // Refreshed either way. A cast can succeed and still report a failure, when the
  // choice was made but could not be written down, so leaving the state unread after a
  // refusal would leave the pane naming the voice that has stopped speaking. The refusal
  // itself is said by the pane the cast was made from, which is the Cast pane's own row.
  const selectVoice = useCallback(
    (name: string) => {
      void api.selectVoice(name, refresh).then(refresh)
    },
    [refresh],
  )

  const openMenu = (which: Menu) => setMenu((current) => (current === which ? null : which))
  const closeMenu = useCallback(() => setMenu(null), [])

  return (
    <div className="shell" ref={shell}>
      <div ref={sink} tabIndex={-1} style={{ width: 0, height: 0, outline: 'none' }} />

      <nav className="menubar" onMouseLeave={() => setMenu(null)}>
        <MenuTitle label="File" open={menu === 'file'} onOpen={() => openMenu('file')} onClose={closeMenu}>
          <button className="menuitem" type="button" onClick={() => void api.quit()}>
            Quit
          </button>
        </MenuTitle>
        <MenuTitle
          label="Audio"
          open={menu === 'audio'}
          onOpen={() => openMenu('audio')}
          onClose={closeMenu}
        >
          <button
            className="menuitem"
            type="button"
            onClick={() => {
              setPane('cast')
              setMenu(null)
            }}
          >
            Cast
          </button>
          <button
            className="menuitem"
            type="button"
            onClick={() => {
              setPane('audition')
              setMenu(null)
            }}
          >
            Audition
          </button>
          {/* Missing takes and Chatter are on the band too; the menu repeats them as it
              repeats Cast and Audition. */}
          <button
            className="menuitem"
            type="button"
            onClick={() => {
              setPane('takes')
              setMenu(null)
            }}
          >
            Missing takes
          </button>
          <button
            className="menuitem"
            type="button"
            onClick={() => {
              setPane('chatter')
              setMenu(null)
            }}
          >
            Chatter
          </button>
          <button
            className="menuitem"
            type="button"
            onClick={() => {
              toggleMute()
              setMenu(null)
            }}
          >
            {state?.muted ? 'Unmute' : 'Mute'}
          </button>
        </MenuTitle>
        <MenuTitle
          label="Settings"
          open={menu === 'settings'}
          onOpen={() => openMenu('settings')}
          onClose={closeMenu}
        >
          <button
            className="menuitem"
            type="button"
            onClick={() => {
              setPane('settings')
              setMenu(null)
            }}
          >
            Open settings
          </button>
          {/* One item, not two: it names the theme it would switch to, so there is
              never a choice between the mode you are in and the one you are not. */}
          <button
            className="menuitem"
            type="button"
            onClick={() => {
              setTheme(theme === 'dark' ? 'light' : 'dark')
              setMenu(null)
            }}
          >
            {theme === 'dark' ? 'Light mode' : 'Dark mode'}
          </button>
        </MenuTitle>
        <MenuTitle label="Help" open={menu === 'help'} onOpen={() => openMenu('help')} onClose={closeMenu}>
          <button
            className="menuitem"
            type="button"
            onClick={() => {
              setPane('guide')
              setMenu(null)
            }}
          >
            Guide
          </button>
          <button
            className="menuitem"
            type="button"
            onClick={() => {
              setLicence(true)
              setMenu(null)
            }}
          >
            Licence
          </button>
          <button
            className="menuitem"
            type="button"
            onClick={() => {
              setAbout(true)
              setMenu(null)
            }}
          >
            About
          </button>
        </MenuTitle>
      </nav>

      {/* A flat row: the two groups are separated by a stretch, so layout order is
          reading order and the ring needs no declared override.

          Each of these picks a pane and nothing more. Pressing the one already
          showing leaves the window where it is, because the mark saying which pane is
          open belongs to a chooser rather than to a switch: pressing Cast while
          reading the cast is a way of asking where you are, so answering it by moving
          somewhere else is the one thing it must not do. Mute and Light are the
          switches in this row; their labels say as much. */}
      <div className="navband">
        <NavButton
          label="Cast"
          current={pane === 'cast'}
          onClick={() => setPane('cast')}
        >
          <CastIcon />
        </NavButton>
        <NavButton
          label="Audition"
          current={pane === 'audition'}
          onClick={() => setPane('audition')}
        >
          <AuditionIcon />
        </NavButton>
        <NavButton
          label="Status"
          current={pane === 'home'}
          onClick={() => setPane('home')}
        >
          <StatusIcon />
        </NavButton>
        <NavButton
          label="Missing takes"
          current={pane === 'takes'}
          onClick={() => setPane('takes')}
        >
          <MissingTakesIcon />
        </NavButton>
        <NavButton
          label="Chatter"
          current={pane === 'chatter'}
          onClick={() => setPane('chatter')}
        >
          <ChatterIcon />
        </NavButton>
        <NavButton
          label="Settings"
          current={pane === 'settings'}
          onClick={() => setPane('settings')}
        >
          <SettingsIcon />
        </NavButton>

        <span className="spacer" />

        <Volume level={volume} onChange={changeVolume} />
        <NavButton
          label={state?.muted ? 'Unmute' : 'Mute'}
          current={false}
          toggles
          onClick={toggleMute}
        >
          {/* The icon is the state, the name is the action: a muted application
              shows a silenced speaker and offers to unmute. */}
          {state?.muted ? <MutedIcon /> : <SoundingIcon />}
        </NavButton>
        <NavButton
          label={theme === 'dark' ? 'Light' : 'Dark'}
          current={false}
          toggles
          onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
        >
          {theme === 'dark' ? <SunIcon /> : <MoonIcon />}
        </NavButton>
        <NavButton
          label="Guide"
          current={pane === 'guide'}
          onClick={() => setPane('guide')}
        >
          <GuideIcon />
        </NavButton>
      </div>

      <main className="pane">
        {pane === 'home' && <HomePane state={state} />}
        {pane === 'settings' && <SettingsPane state={state} />}
        {pane === 'cast' && (
          <CastPane
            active={state?.voice ?? ''}
            machine={state?.machineVoice ?? false}
            plugin={state?.plugin ?? ''}
            total={state?.total ?? 0}
            libraryRoot={state?.libraryRoot ?? ''}
            onSelect={selectVoice}
          />
        )}
        {pane === 'audition' && (
          <AuditionPane cast={state?.voice ?? ''} machine={state?.machineVoice ?? false} />
        )}
        {pane === 'takes' && (
          <MissingTakesPane
            cast={state?.voice ?? ''}
            plugin={state?.plugin ?? ''}
            libraryRoot={state?.libraryRoot}
          />
        )}
        {pane === 'chatter' && <ChatterPane />}
        {pane === 'guide' && <GuidePane />}
      </main>

      {/* FR-717: the strip sits after the pane, so the ring reaches its donate button last. */}
      <Strip state={state} />

      <AboutDialog open={about} onClose={() => setAbout(false)} />
      <LicenceDialog open={licence} onClose={() => setLicence(false)} />

      <CloseChoiceDialog
        open={closing}
        onMinimise={() => {
          setClosing(false)
          void api.minimiseToTray()
        }}
        onQuit={() => {
          setClosing(false)
          void api.requestQuit()
        }}
        onCancel={() => setClosing(false)}
      />
    </div>
  )
}
