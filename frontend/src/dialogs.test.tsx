// The close choice, which is the one dialog that acts rather than reports.
//
// The fault it exists to prevent was the cross ending a resident application outright:
// a commander who meant "put it away" got "stop it" and was not listened to again
// until they noticed. Each of the three answers is asserted, including the one that
// deliberately does nothing.

import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { CloseChoiceDialog } from './dialogs'

/** open renders the dialog with spies for its three answers. */
function open() {
  const onMinimise = vi.fn()
  const onQuit = vi.fn()
  const onCancel = vi.fn()
  render(
    <CloseChoiceDialog
      open
      onMinimise={onMinimise}
      onQuit={onQuit}
      onCancel={onCancel}
    />,
  )
  return { onMinimise, onQuit, onCancel }
}

describe('the close choice', () => {
  it('offers both answers rather than acting on the cross', () => {
    open()
    expect(screen.getByRole('button', { name: /minimise/i })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Quit' })).toBeTruthy()
  })

  it('opens focused on minimising, so Enter after the cross does not quit', () => {
    open()
    expect(document.activeElement?.textContent).toContain('Minimise')
  })

  it('puts the window away without ending the run', () => {
    const { onMinimise, onQuit } = open()
    fireEvent.click(screen.getByRole('button', { name: /minimise/i }))
    expect(onMinimise).toHaveBeenCalled()
    expect(onQuit).not.toHaveBeenCalled()
  })

  it('ends the run when that is the answer chosen', () => {
    const { onMinimise, onQuit } = open()
    fireEvent.click(screen.getByRole('button', { name: 'Quit' }))
    expect(onQuit).toHaveBeenCalled()
    expect(onMinimise).not.toHaveBeenCalled()
  })

  it('costs nothing when it is dismissed, since the press may have been an accident', () => {
    const { onMinimise, onQuit, onCancel } = open()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onCancel).toHaveBeenCalled()
    expect(onMinimise).not.toHaveBeenCalled()
    expect(onQuit).not.toHaveBeenCalled()
  })

  it('draws nothing at all while it is closed', () => {
    render(
      <CloseChoiceDialog
        open={false}
        onMinimise={vi.fn()}
        onQuit={vi.fn()}
        onCancel={vi.fn()}
      />,
    )
    expect(screen.queryByRole('button', { name: 'Quit' })).toBeNull()
  })
})
