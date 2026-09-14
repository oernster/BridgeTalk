// The nav-band icons and the rest of the window's artwork.
//
// These are the project's own artwork rather than drawn marks, so they carry their
// own colour and do not follow the theme the way a currentColor stroke would. That
// is deliberate. They are steel and orange, which reads against both the light and
// the dark surface; it also keeps the one piece of the interface that is
// illustration looking like illustration.
//
// The files here are generated from the masters in assets/ by tools/genicons.py.
// Never edit them by hand; edit the master and run the script.

import donate from './assets/donate.png'
import appIcon from './assets/icons/application-icon.png'
import audition from './assets/icons/audition.png'
import cast from './assets/icons/cast.png'
import darkMode from './assets/icons/dark-mode.png'
import helpInfo from './assets/icons/help-info.png'
import lightMode from './assets/icons/light-mode.png'
import missingTakes from './assets/icons/missing-takes.png'
import moments from './assets/icons/moments.png'
import mute from './assets/icons/mute.png'
import play from './assets/icons/play.png'
import settings from './assets/icons/settings.png'
import status from './assets/icons/status.png'
import unmute from './assets/icons/unmute.png'

/**
 * Icon draws one piece of artwork at the band's icon size.
 *
 * It is decorative in every use here, because each one sits inside a control that
 * already carries a text label. Marking it aria-hidden keeps a reader from
 * announcing the same button twice.
 */
function Icon({ src, className }: { src: string; className?: string }) {
  return <img className={className ?? 'icon'} src={src} alt="" aria-hidden="true" />
}

/**
 * DonateArt is the donate button's artwork (FR-718). It is wider than it is tall, so the strip
 * sizes it by height alone; the button around it carries the label.
 */
export const DonateArt = () => <Icon src={donate} className="donate" />

export const StatusIcon = () => <Icon src={status} />
export const MissingTakesIcon = () => <Icon src={missingTakes} />
export const SettingsIcon = () => <Icon src={settings} />
export const CastIcon = () => <Icon src={cast} />
export const AuditionIcon = () => <Icon src={audition} />
export const GuideIcon = () => <Icon src={helpInfo} />
export const SunIcon = () => <Icon src={lightMode} />
export const MoonIcon = () => <Icon src={darkMode} />
// These two are named for what they DEPICT, not for what pressing them does. The
// artwork is a state: a slashed speaker reads as "no sound is coming out" wherever it
// is seen. Naming them for the action is what put the crossed speaker on a window
// that was happily playing.
export const MutedIcon = () => <Icon src={mute} />
export const SoundingIcon = () => <Icon src={unmute} />

/**
 * MomentsIcon marks the control that opens a voice's coverage.
 *
 * It is drawn larger than a band icon's neighbours in the cast row because it sits
 * alone against a column of figures rather than in a row of its own kind: at a badge's
 * size it would read as decoration beside the number instead of as the thing to press.
 */
export const MomentsIcon = () => <Icon src={moments} className="icon moments" />

/** PlayIcon is larger, because an audition button is a target rather than a badge. */
export const PlayIcon = () => <Icon src={play} className="icon play" />

/**
 * A crest heads a reading surface, so the reader can see at a glance which of them
 * they opened. It is larger again than a button's icon and carries no label of its
 * own, because the heading beneath it is the label.
 */
export const AppCrest = () => <Icon src={appIcon} className="icon crest" />
export const GuideCrest = () => <Icon src={helpInfo} className="icon crest" />

/**
 * artwork is every picture by what it depicts, for a surface that names the controls rather
 * than being one. The guide draws from here so it shows the same file the control does.
 */
export const artwork = {
  application: appIcon,
  audition,
  cast,
  darkMode,
  helpInfo,
  lightMode,
  missingTakes,
  moments,
  mute,
  play,
  settings,
  status,
  unmute,
}
