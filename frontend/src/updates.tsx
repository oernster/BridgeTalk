// The update check (FR-756 to FR-759): when it runs and what the window says about it.
//
// The facade makes the check and holds the release it offered, so nothing here names an address:
// the page asks, reads the answer and hands Download and Skip back to the facade (FR-757).

import { useCallback, useEffect, useState } from 'react'
import { api, type Update, type UpdateOutcome } from './api'
import { Dialog, ReadingBody } from './dialogs'
import { useProductName } from './productName'

/** How long after the page loads the first check runs, so it never contends with the start (FR-756). */
export const firstCheckMs = 3000

/** How often the check runs again while the application stays open (FR-756). */
export const checkEveryMs = 24 * 60 * 60 * 1000

/** unreached is what a check asked for from Help says when no answer came at all (FR-759). */
const unreached: Update = { outcome: 'unreachable', running: '', latest: '' }

/**
 * useUpdateCheck runs the automatic checks and answers what the dialog should show, with the check
 * Help asks for. An automatic check shows only a release it offers (FR-756); one asked for shows
 * whatever it found (FR-759).
 */
export function useUpdateCheck() {
  const [found, setFound] = useState<Update | null>(null)

  const check = useCallback((manual: boolean) => {
    // A refusal is an answer that did not come. An automatic check says nothing about it; Help is
    // told the check could not reach GitHub, which is what the reader can act on.
    void api
      .checkForUpdates(manual, () => undefined)
      .then((update) => {
        const answer = update ?? (manual ? unreached : null)
        if (answer && (manual || answer.outcome === 'available')) setFound(answer)
      })
  }, [])

  useEffect(() => {
    const first = window.setTimeout(() => check(false), firstCheckMs)
    const daily = window.setInterval(() => check(false), checkEveryMs)
    return () => {
      window.clearTimeout(first)
      window.clearInterval(daily)
    }
  }, [check])

  return { found, checkNow: () => check(true), dismiss: () => setFound(null) }
}

/**
 * said words every outcome a dialog can show but an offer, which is a question of its own (FR-759).
 * A skipped release is never shown: Help sends no skip and an automatic check shows only an offer.
 */
function said(outcome: UpdateOutcome, update: Update): string {
  switch (outcome) {
    case 'current':
      return 'You are running the latest version.'
    case 'uncomparable':
      return `This copy was built from source as ${update.running}, so there is no released version to compare it with.`
    default:
      return 'The update check could not reach GitHub. Please try again later.'
  }
}

/**
 * UpdateDialog says what a check found. An offer asks Download, Skip this version or Later
 * (FR-757, FR-758); anything else is read and closed. A Download or a Skip the facade refused keeps
 * the dialog open saying why.
 */
export function UpdateDialog({ found, onClose }: { found: Update | null; onClose: () => void }) {
  const name = useProductName()
  const [problem, setProblem] = useState('')
  const offered = found?.outcome === 'available'

  const close = () => {
    setProblem('')
    onClose()
  }
  // Each act closes the dialog only where the facade did not refuse it.
  const act = (call: (refused: (reason: string) => void) => Promise<void>) => {
    let refusal = ''
    void call((reason) => {
      refusal = reason
    }).then(() => (refusal ? setProblem(refusal) : close()))
  }

  return (
    <Dialog
      title={offered ? 'Update available' : 'Check for updates'}
      open={found !== null}
      onClose={close}
      actions={
        offered ? (
          <>
            <button className="btn primary" data-stop type="button" onClick={() => act(api.downloadUpdate)}>
              Download
            </button>
            <button className="btn" data-stop type="button" onClick={() => act(api.skipUpdate)}>
              Skip this version
            </button>
            <button className="btn" data-stop type="button" onClick={close}>
              Later
            </button>
          </>
        ) : undefined
      }
    >
      <ReadingBody ready>
        <p>
          {found && offered
            ? `${name} ${found.latest} is available. You are running ${found.running}.`
            : found && said(found.outcome, found)}
        </p>
        {problem && (
          <p className="callout refused" role="alert">
            {problem}
          </p>
        )}
      </ReadingBody>
    </Dialog>
  )
}
