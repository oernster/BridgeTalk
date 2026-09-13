// The three pieces the nav band is assembled from: a menu title with its popup, one
// band button and the volume slider.
//
// They are tested apart from the shell for the reason they live apart from it: each
// owns a slice of the keyboard contract that reads better on its own. What is guarded
// here is that contract rather than the layout: a menu moves into itself as it opens,
// walks with the vertical arrows, wraps at both ends and closes whenever focus lands
// anywhere outside it.

import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { MenuTitle, NavButton, Volume } from './chrome'

describe('the pieces the band is assembled from', () => {
  // A menu that drops open with nothing highlighted makes the user press Down before
  // anything has happened.
  it('moves into a menu as it opens', () => {
    render(
      <MenuTitle label="File" open onOpen={vi.fn()} onClose={vi.fn()}>
        <button className="menuitem" type="button">
          Quit
        </button>
      </MenuTitle>,
    )

    expect(document.activeElement?.textContent).toBe('Quit')
  })

  it('opens on Down, on Enter and on Space as well as on a click', () => {
    const onOpen = vi.fn()
    render(
      <MenuTitle label="File" open={false} onOpen={onOpen} onClose={vi.fn()}>
        <button className="menuitem" type="button">
          Quit
        </button>
      </MenuTitle>,
    )
    const title = screen.getByRole('button', { name: 'File' })

    fireEvent.click(title)
    for (const key of ['ArrowDown', 'Enter', ' ']) fireEvent.keyDown(title, { key })

    expect(onOpen).toHaveBeenCalledTimes(4)
  })

  it('leaves a key it has no meaning for alone', () => {
    const onOpen = vi.fn()
    render(
      <MenuTitle label="File" open={false} onOpen={onOpen} onClose={vi.fn()}>
        <span />
      </MenuTitle>,
    )

    fireEvent.keyDown(screen.getByRole('button', { name: 'File' }), { key: 'q' })

    expect(onOpen).not.toHaveBeenCalled()
  })

  it('walks its items with the vertical arrows, wrapping at both ends', () => {
    render(
      <MenuTitle label="File" open onOpen={vi.fn()} onClose={vi.fn()}>
        <button className="menuitem" type="button">
          First
        </button>
        <button className="menuitem" type="button">
          Second
        </button>
      </MenuTitle>,
    )
    const popup = screen.getByRole('button', { name: 'First' }).parentElement as HTMLElement

    expect(document.activeElement?.textContent).toBe('First')
    fireEvent.keyDown(popup, { key: 'ArrowDown' })
    expect(document.activeElement?.textContent).toBe('Second')
    fireEvent.keyDown(popup, { key: 'ArrowDown' })
    expect(document.activeElement?.textContent).toBe('First')
    fireEvent.keyDown(popup, { key: 'ArrowUp' })
    expect(document.activeElement?.textContent).toBe('Second')
  })

  it('closes on Escape and hands the keyboard back to its title', () => {
    const onClose = vi.fn()
    render(
      <MenuTitle label="File" open onOpen={vi.fn()} onClose={onClose}>
        <button className="menuitem" type="button">
          Quit
        </button>
      </MenuTitle>,
    )
    const popup = screen.getByRole('button', { name: 'Quit' }).parentElement as HTMLElement

    fireEvent.keyDown(popup, { key: 'Escape' })

    expect(onClose).toHaveBeenCalled()
    expect(document.activeElement?.textContent).toBe('File')
  })

  // Focus leaving the menu closes it, whichever key took it away. A blur's
  // relatedTarget is empty on some routes out, so where focus ARRIVES is watched.
  it('closes when focus lands anywhere outside it', () => {
    const onClose = vi.fn()
    render(
      <>
        <MenuTitle label="File" open onOpen={vi.fn()} onClose={onClose}>
          <button className="menuitem" type="button">
            Quit
          </button>
        </MenuTitle>
        <button type="button">Elsewhere</button>
      </>,
    )

    // Focus arriving back inside the menu is not a reason to close.
    fireEvent.focusIn(screen.getByRole('button', { name: 'Quit' }))
    fireEvent.focusIn(screen.getByRole('button', { name: 'File' }))
    expect(onClose).not.toHaveBeenCalled()

    fireEvent.focusIn(screen.getByRole('button', { name: 'Elsewhere' }))
    expect(onClose).toHaveBeenCalled()
  })

  // A button whose label names a state keeps its tooltip after the press, because the
  // label is still worth reading; every other button lets its name retire on time.
  it('marks a button whose label names a state rather than a place', () => {
    render(
      <>
        <NavButton label="Mute" current={false} toggles onClick={vi.fn()}>
          <span />
        </NavButton>
        <NavButton label="Cast" current onClick={vi.fn()}>
          <span />
        </NavButton>
      </>,
    )

    const mute = screen.getByRole('button', { name: 'Mute' })
    const cast = screen.getByRole('button', { name: 'Cast' })
    expect(mute.hasAttribute('data-toggle')).toBe(true)
    expect(cast.hasAttribute('data-toggle')).toBe(false)
    expect(cast.getAttribute('aria-current')).toBe('page')
    expect(mute.getAttribute('aria-current')).toBeNull()
  })

  it('acts when a band button is pressed', () => {
    const onClick = vi.fn()
    render(
      <NavButton label="Cast" current={false} onClick={onClick}>
        <span />
      </NavButton>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Cast' }))

    expect(onClick).toHaveBeenCalled()
  })

  it('reads the volume back as a percentage and reports what it is moved to', () => {
    const onChange = vi.fn()
    render(<Volume level={0.5} onChange={onChange} />)

    expect(screen.getByText('50%')).toBeTruthy()
    fireEvent.change(screen.getByRole('slider', { name: 'Playback volume' }), {
      target: { value: '0.25' },
    })

    expect(onChange).toHaveBeenCalledWith(0.25)
  })
})
