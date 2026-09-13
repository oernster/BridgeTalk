// The guide's text, held apart from the pane that draws it so the pane stays a renderer and the
// words stay one readable document. Two jobs, in this order. It NAMES the furniture, each entry
// carrying the REAL picture the window draws, taken from the same icons module the controls use,
// so a control that is only a picture can be identified by someone who has just met it. Then it
// states the rules the window cannot say for itself: how a recording is found, why it stays quiet
// and what it never touches.
//
// Never a description in words where the control is a picture; never a similar-looking stand-in
// for one. A guide showing something other than the icon is worse than no guide.
//
// It never writes the product's name. The pane reads that from the application, because the name
// is written down in one place and a second copy here is how a rename leaves one behind.

import { artwork } from './icons'

/**
 * GuideEntry is one named piece of furniture: every picture it wears, what it is called and what
 * it does. A toggle wears two pictures, because a reader who has only ever seen one of them still
 * has to recognise the other.
 */
export interface GuideEntry {
  icons: string[]
  name: string
  text: string
}

/** GuideRule is one of the rules behind the behaviour: a claim in bold, then what it means. */
export interface GuideRule {
  title: string
  text: string
}

/**
 * GuideSection is one block of the document. A section carries entries, rules or plain paragraphs;
 * the pane draws whichever are present, in that order.
 */
export interface GuideSection {
  heading: string
  intro?: string
  entries?: GuideEntry[]
  rules?: GuideRule[]
  paragraphs?: string[]
}

export const guideSections: GuideSection[] = [
  {
    heading: 'The buttons along the top',
    intro:
      'The four panes read left to right. The volume slider, Mute, the theme and this guide are held at the far end. Hover any button to see its name.',
    entries: [
      {
        icons: [artwork.cast],
        name: 'Cast',
        text: 'choose the voice that speaks for your ship. The window opens here.',
      },
      { icons: [artwork.audition], name: 'Audition', text: 'hear any voice before casting it.' },
      {
        icons: [artwork.status],
        name: 'Status',
        text: 'what is cast, where it is watching and every decision it has made, including the ones that produced no sound.',
      },
      {
        icons: [artwork.settings],
        name: 'Settings',
        text: 'the recordings directory, the journal directory and whether it starts when you sign in.',
      },
      {
        icons: [artwork.unmute, artwork.mute],
        name: 'Mute',
        text: 'silences what it says in answer to the game. The picture shows the state rather than the action, so a sounding speaker is offering to mute.',
      },
      {
        icons: [artwork.lightMode, artwork.darkMode],
        name: 'Light or dark',
        text: 'the picture shows the mode it will switch INTO, so the moon appears while the window is light.',
      },
      { icons: [artwork.helpInfo], name: 'Guide', text: 'this page.' },
    ],
  },
  {
    heading: 'The cast pane',
    intro:
      'Every directory of recordings found is a voice with a row of its own. Pressing a row casts that voice, which then says something as it takes the part; it stays quiet while the window is muted or where the voice has no line for it.',
    entries: [
      {
        icons: [artwork.moments],
        name: 'Moments spoken for',
        text: 'the mark at the end of a row. It lists every moment that voice has a recording for, then every moment it was never recorded for, each of which stays quiet. It answers for any voice, so one can be weighed before it is cast.',
      },
    ],
  },
  {
    heading: 'The audition pane',
    intro:
      'Choose any voice to hear, whether it is cast or not. The square beside it stops a recording part way through.',
    entries: [
      {
        icons: [artwork.play],
        name: 'Play',
        text: "one button for each part of the game. A press plays one of that voice's recordings for it at random, so pressing again can give a different take. An audition ignores the mute, which silences answers to the game rather than the window.",
      },
    ],
  },
  {
    heading: 'The notification area',
    intro: 'While the window is put away the application keeps watching the game and keeps speaking.',
    entries: [
      {
        icons: [artwork.application],
        name: 'Its icon',
        text: 'a click brings the window back, opening on the cast. Right click for its menu: the voice to speak with, Open, Mute and Quit. Hovering it names the voice that is cast and says when it is muted.',
      },
    ],
    paragraphs: [
      'The cross on the window asks rather than closes. Minimise to the notification area keeps it listening; Quit stops it; Escape leaves everything as it was.',
    ],
  },
  {
    heading: 'The menus',
    paragraphs: [
      'File holds Quit. Audio opens Cast or Audition and holds Mute. Settings opens the settings pane and switches between light and dark. Help holds this guide, the licence and About.',
    ],
  },
  {
    heading: 'Rules behind the behaviour',
    rules: [
      {
        title: 'It talks and never listens.',
        text: "It watches the game's journal and status file, decides that something worth saying has happened, then plays a matching recording. There is no microphone, no speech recognition and no command and control.",
      },
      {
        title: 'The cue table is game truth.',
        text: 'Each cue says which moment in the game it answers and how urgent that is. Your recordings are the other half, matched to a cue by its id alone, so adding a voice never changes what the game means.',
      },
      {
        title: 'A recording is found by its name.',
        text: "Every directory directly inside the recordings directory is one voice. The quickest way to make one is on the Cast pane: type a name and press Make folders, which makes a folder for every moment already named for it; put each recording in the folder for its moment, then press Look again. To name them by hand instead: inside a voice, name a file after the cue it answers, such as DockingGranted.wav, adding .2, .3 and so on for further takes; or make a folder with that name holding any number of takes. A cue's name is the game's own name for the moment, as the journal and status file spell it. A name has to match the id exactly apart from case. WAV, MP3, FLAC and Ogg files are played.",
      },
      {
        title: 'Each moment plays one recording.',
        text: 'Where a voice holds several takes for a cue, one is chosen at random, never the one that cue played last time.',
      },
      {
        title: 'Quiet is often the right answer.',
        text: 'An alert interrupts anything less urgent; a notice waits its turn; an ambient line is dropped when something is already waiting; incidental chatter is dropped whenever anything is waiting or speaking. Many cues also hold a minimum interval between two firings, the game restating itself within a second collapses into one line and a cue the voice has no recording for stays silent rather than borrowing another. While muted, nothing plays in answer to the game. The Status pane logs each of these decisions with its reason.',
      },
      {
        title: 'Your recordings are never changed.',
        text: 'It ships no audio. It never changes or removes anything in the directory you choose; the one thing it writes there is the empty folders you ask it to make. It makes no network requests of its own.',
      },
    ],
  },
  {
    heading: 'Also worth knowing',
    paragraphs: [
      'The cast voice and both directories are kept in a settings file under your own application data folder; the theme and the volume are kept by the window itself. All of them are picked up again at the next start.',
      'The journal directory is found under your own profile until you choose one. Choosing either directory in Settings takes effect at once.',
      'With Start it when I sign in ticked, signing in to Windows starts it hidden in the notification area.',
      'Started from a command line it takes -library, -journal and -voice for that run only, over what Settings holds. -list prints the voices found with their coverage and -unbound the cues the chosen voice cannot serve, each then exiting; -no-tray runs it without the notification-area icon.',
    ],
  },
  {
    heading: 'Keyboard',
    paragraphs: [
      'Everything is reachable from the keyboard. Tab and Right move forward, Shift+Tab and Left move back; the ring wraps at both ends. Up and Down walk the rows of a list. A menu title opens on Down, Enter or Space. Enter or Space activates what is focused and Escape closes a dialog.',
      'The window starts with nothing focused; the first Tab or Right enters the ring.',
    ],
  },
]
