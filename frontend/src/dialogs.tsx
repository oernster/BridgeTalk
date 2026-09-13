// The modal shell, the body every dialog reads through and the dialogs themselves.
//
// Every dialog that carries text reads itself, through one body. A body that fits its
// dialog never moves, since the cycle only acts on content that overflows, so attaching
// it to a short dialog costs nothing and keeps every dialog on the same rules.

import { useEffect, useRef, useState } from 'react'
import { api, type About } from './api'
import { useAutoScroll, useFirstStop, useOverflowStop, useRing } from './hooks'
import { AppCrest } from './icons'

/**
 * ReadingBody is the scrolling body of a dialog.
 *
 * It holds still on open, reads itself down once its content has arrived and gives way
 * to the reader's own input. It takes a place on the ring only while it overflows,
 * because focus on a region that cannot scroll is a dead press.
 *
 * `ready` says when the content has arrived. A dialog filled from the backend is empty
 * until the answer lands, so the start hold is counted from then rather than from a
 * placeholder the reader never saw.
 */
export function ReadingBody({
  ready,
  children,
}: {
  ready: boolean
  children: React.ReactNode
}) {
  const body = useRef<HTMLDivElement>(null)
  useAutoScroll(body, ready)
  const overflows = useOverflowStop(body)
  return (
    <div
      className="body"
      ref={body}
      {...(overflows ? { 'data-stop': true } : {})}
      tabIndex={overflows ? 0 : -1}
    >
      {children}
    </div>
  )
}

/**
 * Dialog is the shared modal shell.
 *
 * Exported, because the moments dialog in moments.tsx is built on it: a second modal
 * that opened, trapped and closed by its own rules would be a second set of rules to
 * keep in step with this one.
 *
 * It opens focused on its first stop, the deliberate opposite of the main window's
 * neutral start: the user opened it to do the one thing it is for. Escape closes it
 * and focus returns to whatever opened it.
 */
export function Dialog({
  title,
  open,
  onClose,
  children,
  actions,
}: {
  title: string
  open: boolean
  onClose: () => void
  children: React.ReactNode
  /**
   * The footer's buttons, where the dialog asks something rather than reporting it.
   * Left out, the footer carries Close alone, which is the whole answer a surface
   * that is read and dismissed needs.
   */
  actions?: React.ReactNode
}) {
  const frame = useRef<HTMLDivElement>(null)

  // Whatever held focus when the dialog opened is given it back when the dialog closes, so
  // the ring carries on from where the reader was. Declared before useFirstStop, which moves
  // focus into the dialog, so it reads the opener rather than the dialog's own first stop.
  useEffect(() => {
    if (!open) return
    const opener = document.activeElement as HTMLElement | null
    return () => opener?.focus()
  }, [open])

  useFirstStop(frame, open)
  // The dialog's own ring. The window's ring stands aside while a scrim is over it.
  useRing(frame, open)

  useEffect(() => {
    if (!open) return
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey, true)
    return () => window.removeEventListener('keydown', onKey, true)
  }, [open, onClose])

  if (!open) return null
  return (
    <div className="scrim" role="presentation" onMouseDown={onClose}>
      <div
        className="dialog"
        ref={frame}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header>{title}</header>
        {children}
        <footer>
          {actions ?? (
            <button className="btn primary" data-stop type="button" onClick={onClose}>
              Close
            </button>
          )}
        </footer>
      </div>
    </div>
  )
}

/** AboutDialog shows the identity, the version and the dependency credits. */
export function AboutDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [about, setAbout] = useState<About | null>(null)
  useEffect(() => {
    if (open) void api.about().then(setAbout)
  }, [open])

  return (
    <Dialog title="About" open={open} onClose={onClose}>
      <ReadingBody ready={about !== null}>
        <AppCrest />
        {/* The name comes from the application rather than being written here as
            well. A fallback would be a second place the product is named, which is
            the thing that let a rename leave a stale name behind; the dialog waits
            for the answer instead, which arrives in the same breath as the version
            and the credits it sits above. */}
        <div className="identity">{about?.name ?? ''}</div>
        <p>{about?.tagline}</p>
        <p>
          <b>Version:</b> {about?.version}
          <br />
          <b>Author:</b> {about?.author}
        </p>
        <h2>Authorship</h2>
        <p>{about?.authorship}</p>
        <h2>Licence</h2>
        <p>{about?.licence}</p>
        <h2>Voices</h2>
        <p>{about?.attribution}</p>
        <h2>Open source credits</h2>
        <ul>
          {(about?.credits ?? []).map((credit) => (
            <li key={credit}>{credit}</li>
          ))}
        </ul>
        <p>Built on the Go and web ecosystems, with thanks to their communities.</p>
        <p className="copyright">{about?.copyright}</p>
      </ReadingBody>
    </Dialog>
  )
}

/**
 * LicenceDialog shows the full terms the application is released under.
 *
 * The text is the licence file the source carries, embedded when the application is
 * built, so the dialog cannot show terms other than the ones that apply. It is long and
 * read rather than acted on, which is exactly the surface the reading body is for.
 */
export function LicenceDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [text, setText] = useState<string | null>(null)
  useEffect(() => {
    if (!open) {
      setText(null)
      return
    }
    void api.licence().then(setText)
  }, [open])

  return (
    <Dialog title="Licence" open={open} onClose={onClose}>
      <ReadingBody ready={text !== null}>
        <pre className="licence">{text ?? ''}</pre>
      </ReadingBody>
    </Dialog>
  )
}

/**
 * CloseChoiceDialog answers the window's cross.
 *
 * The cross is ambiguous in a resident application: it means "put it away" at least
 * as often as it means "stop it"; it closed the application outright before, so a
 * commander who expected to still be listened to was not. Dismissing the dialog, by
 * Escape or by the scrim, cancels the close and changes nothing, since an accidental
 * press should cost nothing.
 *
 * Minimise is the primary because it is the answer that keeps the application doing
 * what it is for; quitting is the deliberate act and is one plain button away.
 */
export function CloseChoiceDialog({
  open,
  onMinimise,
  onQuit,
  onCancel,
}: {
  open: boolean
  onMinimise: () => void
  onQuit: () => void
  onCancel: () => void
}) {
  return (
    <Dialog
      title="Close the window"
      open={open}
      onClose={onCancel}
      actions={
        <>
          {/* Minimise is written first because the dialog opens focused on its first
              stop, so Enter straight after the cross has to mean "put it away" rather
              than "stop it". Measured before this: the dialog opened on Quit. */}
          <button className="btn primary" data-stop type="button" onClick={onMinimise}>
            Minimise to the notification area
          </button>
          <button className="btn" data-stop type="button" onClick={onQuit}>
            Quit
          </button>
        </>
      }
    >
      <ReadingBody ready>
        <p>
          Leave it running in the notification area? Stopping it is the other button.
          While it runs there it
          keeps watching the journal and keeps speaking; its icon summons this window
          back.
        </p>
      </ReadingBody>
    </Dialog>
  )
}
