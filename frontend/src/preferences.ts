// The two preferences the window keeps for itself: the theme and the playback level.
//
// They live apart from the shell because each is a small contract of its own with
// storage, restoring on load and writing on change, while the shell is a layout.

import { useCallback, useEffect, useState } from 'react'
import { api } from './api'

export type Theme = 'dark' | 'light'

const themeKey = 'bridge-talk.theme'
const volumeKey = 'bridge-talk.volume'

const fullVolume = 1

/** useTheme restores the chosen theme, applies it and writes every change back. */
export function useTheme(): [Theme, (theme: Theme) => void] {
  const [theme, setTheme] = useState<Theme>('dark')

  // Restore the chosen theme. A browser that refuses storage falls back to dark,
  // which is the palette this application is designed around.
  useEffect(() => {
    let stored: Theme = 'dark'
    try {
      const found = window.localStorage.getItem(themeKey)
      if (found === 'light' || found === 'dark') stored = found
    } catch {
      stored = 'dark'
    }
    setTheme(stored)
  }, [])

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    try {
      window.localStorage.setItem(themeKey, theme)
    } catch {
      // A viewer with storage disabled simply keeps the choice for this run.
    }
  }, [theme])

  return [theme, setTheme]
}

/** useVolume restores the chosen level, pushes it into the player and keeps changes. */
export function useVolume(): [number, (level: number) => void] {
  const [volume, setVolume] = useState(fullVolume)

  // Restore the chosen level and push it into the player, which holds no preference
  // of its own. A browser that refuses storage simply starts at full.
  useEffect(() => {
    let stored = fullVolume
    try {
      // Nothing stored has to be tested before converting: Number(null) is zero, not
      // a failure, so a first run would otherwise start silent and read as broken.
      const saved = window.localStorage.getItem(volumeKey)
      const found = saved === null ? fullVolume : Number(saved)
      if (Number.isFinite(found) && found >= 0 && found <= fullVolume) stored = found
    } catch {
      stored = fullVolume
    }
    setVolume(stored)
    void api.setVolume(stored)
  }, [])

  const changeVolume = useCallback((level: number) => {
    setVolume(level)
    void api.setVolume(level)
    try {
      window.localStorage.setItem(volumeKey, String(level))
    } catch {
      // A viewer with storage disabled simply keeps the level for this run.
    }
  }, [])

  return [volume, changeVolume]
}
