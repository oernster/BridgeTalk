// The menu bar: File, Audio, Settings and Help, each repeating a way in the nav band or the
// dialogs also offer (FR-709). It holds which menu is open; everything a choice does is handed in.

import { useCallback, useState } from 'react'
import { api } from './api'
import { MenuTitle } from './chrome'
import type { Theme } from './preferences'

/** Pane names every pane the shell switches between. */
export type Pane = 'home' | 'settings' | 'cast' | 'audition' | 'takes' | 'chatter' | 'guide'

type Menu = 'file' | 'audio' | 'settings' | 'help' | null

/**
 * MenuBar draws the menus and acts on a choice through what it is handed, closing the menu it
 * came from.
 */
export function MenuBar({
  muted,
  theme,
  onPane,
  onToggleMute,
  onTheme,
  onLicence,
  onAbout,
}: {
  muted: boolean
  theme: Theme
  onPane: (pane: Pane) => void
  onToggleMute: () => void
  onTheme: (theme: Theme) => void
  onLicence: () => void
  onAbout: () => void
}) {
  const [menu, setMenu] = useState<Menu>(null)
  const openMenu = (which: Menu) => setMenu((current) => (current === which ? null : which))
  const closeMenu = useCallback(() => setMenu(null), [])

  return (
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
            onPane('cast')
            setMenu(null)
          }}
        >
          Cast
        </button>
        <button
          className="menuitem"
          type="button"
          onClick={() => {
            onPane('audition')
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
            onPane('takes')
            setMenu(null)
          }}
        >
          Missing takes
        </button>
        <button
          className="menuitem"
          type="button"
          onClick={() => {
            onPane('chatter')
            setMenu(null)
          }}
        >
          Chatter
        </button>
        <button
          className="menuitem"
          type="button"
          onClick={() => {
            onToggleMute()
            setMenu(null)
          }}
        >
          {muted ? 'Unmute' : 'Mute'}
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
            onPane('settings')
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
            onTheme(theme === 'dark' ? 'light' : 'dark')
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
            onPane('guide')
            setMenu(null)
          }}
        >
          Guide
        </button>
        <button
          className="menuitem"
          type="button"
          onClick={() => {
            onLicence()
            setMenu(null)
          }}
        >
          Licence
        </button>
        <button
          className="menuitem"
          type="button"
          onClick={() => {
            onAbout()
            setMenu(null)
          }}
        >
          About
        </button>
      </MenuTitle>
    </nav>
  )
}
