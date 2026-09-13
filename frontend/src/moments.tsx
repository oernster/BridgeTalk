// What a voice can and cannot speak for, plus the words used to say it.
//
// The vocabulary lives here rather than in panes.tsx because the dialog is its only
// reader now: the cast pane used to list the unserved cues along its bottom, which
// showed half the answer to somebody who had not asked the question. Both halves are
// behind one button on the row they describe instead.

import { useEffect, useState } from 'react'
import { api, type CueBreakdown, type CueEntry } from './api'
import { Dialog, ReadingBody } from './dialogs'

/**
 * Half renders one side of the breakdown; or says that it is empty.
 *
 * Each cue stands under its full title alone (FR-233). A heading read from the id's first
 * segment would repeat the start of every title beneath it, word for word wherever the id
 * has only the one segment. The list arrives sorted by id, so related cues still sit together.
 */
function Half({ title, lede, cues }: { title: string; lede: string; cues: CueEntry[] }) {
  return (
    <>
      <h2>
        {title} ({cues.length})
      </h2>
      <p className="lede">{lede}</p>
      {cues.length === 0 ? (
        <p className="empty">Nothing here.</p>
      ) : (
        <div className="cuelist">
          {cues.map((item) => (
            <div className="cuerow" key={item.id}>
              {item.title}
            </div>
          ))}
        </div>
      )}
    </>
  )
}

/**
 * MomentsDialog shows both halves of one voice's coverage.
 *
 * It reads itself, as the guide and the About dialog do, because it is a surface to
 * read through rather than one to act on: the lists run past a screen for every voice
 * in the library root; a reader who opened it to see what a voice covers should not
 * have to drive it. Any key or pointer of their own suspends that and hands it back.
 *
 * voice identifies whose coverage is asked for; shown is the name they are shown by, which is
 * the one the words use (FR-210).
 */
export function MomentsDialog({
  voice,
  shown,
  open,
  onClose,
}: {
  voice: string
  shown: string
  open: boolean
  onClose: () => void
}) {
  const [breakdown, setBreakdown] = useState<CueBreakdown | null>(null)

  useEffect(() => {
    if (!open) {
      setBreakdown(null)
      return
    }
    void api.cueBreakdown(voice).then(setBreakdown)
  }, [open, voice])

  return (
    <Dialog title={`What ${shown} speaks for`} open={open} onClose={onClose}>
      <ReadingBody ready={breakdown !== null}>
        <Half
          title="Moments spoken for"
          lede={`Moments ${shown} has a recording for.`}
          cues={breakdown?.served ?? []}
        />
        <Half
          title="Moments with no lines"
          lede={`Never recorded for ${shown}, so each one stays quiet.`}
          cues={breakdown?.unserved ?? []}
        />
      </ReadingBody>
    </Dialog>
  )
}
