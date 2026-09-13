// The pieces the shell is assembled from: a menu title with its popup, one band
// button and the volume slider.
//
// They live beside App rather than inside it for two reasons: the shell is already
// at the size cap; each one owns a slice of the keyboard contract that reads better
// on its own than buried in the layout.

import { useEffect, useRef } from 'react'

// The slider runs from silence to the clip as recorded, in steps fine enough to
// find a level and coarse enough that the keyboard crosses the range in a few
// presses rather than a hundred.
const volumeSteps = 20

/** MenuTitle highlights on focus and drops its popup on Down, Enter or Space. */
export function MenuTitle({
  label,
  open,
  onOpen,
  onClose,
  children,
}: {
  label: string
  open: boolean
  onOpen: () => void
  onClose: () => void
  children: React.ReactNode
}) {
  const title = useRef<HTMLButtonElement>(null)
  const popup = useRef<HTMLDivElement>(null)

  // Opening a menu moves into it. A menu that drops open with nothing highlighted
  // makes the user press Down before anything has happened.
  useEffect(() => {
    if (!open) return
    popup.current?.querySelector<HTMLElement>('.menuitem')?.focus()
  }, [open])

  // Focus leaving the menu closes it, whichever key took it away. This watches where
  // focus ARRIVES rather than where it left from: a blur's relatedTarget is empty on
  // some routes out, so a menu could hang open behind the focus that walked away.
  useEffect(() => {
    if (!open) return
    const onFocusIn = (event: FocusEvent) => {
      const arrived = event.target as Node | null
      if (popup.current?.contains(arrived) || title.current === arrived) return
      onClose()
    }
    document.addEventListener('focusin', onFocusIn)
    return () => document.removeEventListener('focusin', onFocusIn)
  }, [open, onClose])

  const walk = (delta: number) => {
    const items = Array.from(popup.current?.querySelectorAll<HTMLElement>('.menuitem') ?? [])
    if (items.length === 0) return
    const index = items.indexOf(document.activeElement as HTMLElement)
    items[(index + delta + items.length) % items.length].focus()
  }

  return (
    <div style={{ position: 'relative' }}>
      <button
        className="menutitle"
        data-stop
        ref={title}
        type="button"
        aria-expanded={open}
        onClick={onOpen}
        onKeyDown={(event) => {
          if (event.key === 'ArrowDown' || event.key === 'Enter' || event.key === ' ') {
            event.preventDefault()
            onOpen()
          }
        }}
      >
        {label}
      </button>
      {open && (
        <div
          className="menupopup"
          ref={popup}
          onKeyDown={(event) => {
            if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
              event.preventDefault()
              walk(event.key === 'ArrowDown' ? 1 : -1)
            } else if (event.key === 'Escape') {
              event.preventDefault()
              onClose()
              title.current?.focus()
            }
          }}
        >
          {children}
        </div>
      )}
    </div>
  )
}

/**
 * NavButton is one icon stop on the band.
 *
 * toggles marks a button whose label names a state rather than a place: mute against
 * unmute, light against dark. Pressing one of those changes what its own label says,
 * so the label keeps its worth after the press in a way that Cast or Guide never do,
 * and the tooltip stays for as long as the button is the one last pressed. Every other
 * button lets its label retire on time, which is what stops a name sitting over the
 * pane after a click.
 */
export function NavButton({
  label,
  current,
  toggles,
  onClick,
  children,
}: {
  label: string
  current: boolean
  toggles?: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      className="navbtn"
      data-stop
      data-label={label}
      {...(toggles ? { 'data-toggle': true } : {})}
      type="button"
      aria-label={label}
      aria-current={current ? 'page' : undefined}
      onClick={onClick}
    >
      {children}
    </button>
  )
}

/**
 * Volume is the playback slider in the nav band.
 *
 * It is a native range input, so it arrives with the keyboard and the screen reader
 * already working. That also means it keeps the horizontal arrows for its own value
 * rather than stepping the ring, which is why the ring hook lists it among the
 * elements that own their arrows: Tab still leaves it in both directions.
 */
export function Volume({ level, onChange }: { level: number; onChange: (level: number) => void }) {
  const percent = Math.round(level * 100)
  return (
    <label className="volume" title={`Playback volume, ${percent} per cent`}>
      <input
        data-stop
        type="range"
        min={0}
        max={1}
        step={1 / volumeSteps}
        value={level}
        aria-label="Playback volume"
        onChange={(event) => onChange(Number(event.target.value))}
      />
      <span className="reading">{percent}%</span>
    </label>
  )
}
