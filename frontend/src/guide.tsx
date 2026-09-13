// The guide pane.
//
// Its words live in guideContent.ts; this file only draws them, so a change to what the
// application says is a change to one document rather than to markup. The guide used to be
// HTML embedded on the Go side, which could not reach the artwork: the band's pictures had to
// be spliced in beside it and every control beyond the band went without one.

import { useEffect, useRef, useState } from 'react'
import { api } from './api'
import { guideSections } from './guideContent'
import { useAutoScroll, useOverflowStop } from './hooks'
import { GuideCrest } from './icons'

/**
 * GuidePane renders the guide and reads itself, because it is a surface to read through rather
 * than one to act on.
 *
 * The title names the product, which arrives from the application rather than being written
 * here: a fallback would be a second place the name is kept, which is what lets a rename leave
 * a stale one behind. The sections need no answer, so they are drawn at once.
 */
export function GuidePane() {
  const [name, setName] = useState('')
  const bodyRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    void api.about().then((about) => setName(about?.name ?? ''))
  }, [])
  useAutoScroll(bodyRef, true)
  const overflows = useOverflowStop(bodyRef)

  return (
    <div
      className="guide-body"
      ref={bodyRef}
      {...(overflows ? { 'data-stop': true } : {})}
      tabIndex={overflows ? 0 : -1}
      style={{ height: '100%', overflowY: 'auto' }}
    >
      <GuideCrest />
      {name !== '' && <h2 className="guide-title">{`How ${name} works`}</h2>}
      {guideSections.map((section) => (
        <section className="guide-section" key={section.heading}>
          <h3>{section.heading}</h3>
          {section.intro && <p className="guide-intro">{section.intro}</p>}
          {section.entries?.map((entry) => (
            <div className="guide-entry" key={entry.name}>
              <span className="marks">
                {entry.icons.map((icon) => (
                  <img className="guide-icon" key={icon} src={icon} alt="" aria-hidden="true" />
                ))}
              </span>
              <span>
                <b>{entry.name}</b>: {entry.text}
              </span>
            </div>
          ))}
          {section.rules?.map((rule) => (
            <p className="guide-rule" key={rule.title}>
              <b>{rule.title}</b> {rule.text}
            </p>
          ))}
          {section.paragraphs?.map((text) => (
            <p className="guide-para" key={text}>
              {text}
            </p>
          ))}
        </section>
      ))}
    </div>
  )
}
