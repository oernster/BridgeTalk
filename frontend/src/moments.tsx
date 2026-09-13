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
 * grouped gathers cues under the moment in the game they belong to.
 *
 * The backend already decided each cue's group and its heading in words; each list also
 * arrives sorted by id, so this only has to keep first-seen order rather than sort again.
 * No name for a group is written here: the heading is generated from the game's own
 * spelling beside the title, which is what keeps a second list of names from existing.
 */
function grouped(cues: CueEntry[]): [string, CueEntry[]][] {
  const order: string[] = []
  const bucket = new Map<string, CueEntry[]>()
  for (const item of cues) {
    if (!bucket.has(item.group)) {
      order.push(item.group)
      bucket.set(item.group, [])
    }
    bucket.get(item.group)?.push(item)
  }
  return order.map((group) => [group, bucket.get(group) ?? []])
}

/** Half renders one side of the breakdown, grouped; or says that it is empty. */
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
          {grouped(cues).map(([group, entries]) => (
            <div className="cuegroup" key={group}>
              <div className="grouphead">{entries[0].heading}</div>
              {entries.map((item) => (
                <div className="cuerow" key={item.id}>
                  {item.title}
                </div>
              ))}
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
 */
export function MomentsDialog({
  voice,
  open,
  onClose,
}: {
  voice: string
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
    <Dialog title={`What ${voice} speaks for`} open={open} onClose={onClose}>
      <ReadingBody ready={breakdown !== null}>
        <Half
          title="Moments spoken for"
          lede={`Moments ${voice} has a recording for.`}
          cues={breakdown?.served ?? []}
        />
        <Half
          title="Moments with no lines"
          lede={`Never recorded for ${voice}, so each one stays quiet.`}
          cues={breakdown?.unserved ?? []}
        />
      </ReadingBody>
    </Dialog>
  )
}
