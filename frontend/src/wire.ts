// The shapes the Go facade answers with, stated here by hand.
//
// Wails generates the same shapes into frontend/wailsjs at build time. That file is gitignored
// and imported by nothing, so it is not the contract; this is, typed out by a person and compared
// against the facade's own structs by tests/structural/wire_test.go.
//
// They sit apart from api.ts because they are the wire rather than the calls, which is the split
// the Go side already makes between dto.go and the facade beside it.

export interface State {
  /** Identifies the cast voice; empty while none is cast. */
  voice: string
  /** The name the cast voice is shown by: its manifest's name, else its directory's (FR-210). */
  voiceDisplay: string
  bound: number
  total: number
  muted: boolean
  silent: boolean
  journalDir: string
  statusPath: string
  libraryRoot: string
  version: string
  /** Whether Windows starts the application at sign-in, read from the entry itself. */
  launchOnBoot: boolean
  /**
   * How many times the output device ran out of audio before it was refilled,
   * plus the longest such wait in milliseconds. A break in the speech is otherwise
   * something only the listener knows about; a machine that keeps the device fed
   * reports nothing at all.
   */
  stalls: number
  worstStall: number
  /**
   * Why the journal directory is not being watched, naming it once; empty while it is.
   * The window opens either way, so the panes are where it is said (FR-238).
   */
  journalProblem: string
  /** Whether the cast voice is a machine voice, since a recordings folder may carry its id (FR-540). */
  machineVoice: boolean
  /**
   * The plugin the cast voice came from; empty for every other kind. With voice, which then holds
   * the voice's id within that plugin, it says which plugin voice is cast (FR-569).
   */
  plugin: string
}

/** MachineVoice is one machine voice the Cast pane offers (FR-508). */
export interface MachineVoice {
  /** The voice's id, which a cast sends back. */
  id: string
  /** The name the screen shows, such as "Emma (British, female)" (FR-528). */
  name: string
  /** The name alone, such as "Emma", which its pill shows (FR-720). */
  given: string
  /** The accent and sex it is offered under, such as "British, female": its panel's heading (FR-720). */
  group: string
}

/** PluginVoice is one voice a plugin offers, as the Cast pane shows it (FR-565). */
export interface PluginVoice {
  /** The plugin's own name, which a cast sends back with the id (FR-569). */
  plugin: string
  /** The voice's id within that plugin, which a cast sends back. */
  id: string
  /** The name the plugin gave the voice. */
  name: string
  /** The name the screen shows: the name above, unless another plugin offers one like it (FR-568). */
  display: string
  /** Whether the audio the voice needs is on this machine (FR-570). */
  ready: boolean
  /** Why it is not; empty while it is (FR-570). */
  reason: string
}

/** LineFailure is one line that could not be made (FR-518). */
export interface LineFailure {
  cue: CueEntry
  /** The line's place among its moment's lines, counting from one. */
  line: number
  reason: string
}

/**
 * Making is how far making the cast machine voice's lines has got: current of total lines made
 * (FR-515), cuesServed moments with a made line (FR-522), the lines that failed (FR-518), why making
 * stopped (FR-520) and why old lines were not deleted (FR-530). The two reasons are empty where
 * nothing went wrong.
 */
export interface Making {
  voice: string
  making: boolean
  current: number
  total: number
  cuesServed: number
  failed: LineFailure[]
  stopped: string
  notDeleted: string
}

export interface Voice {
  /** Identifies the voice: its directory's name, which a cast sends back. */
  name: string
  /** The name the voice is shown by: its manifest's name, else its directory's (FR-210). */
  display: string
  /** The manifest's credit line; empty where there is none. */
  credit: string
  /** Moments the voice has a recording for (FR-215). */
  cues: number
  /** Distinct files the voice uses, not the size of its folder. */
  inUse: number
  /** Recordings in the voice's folder, whether any moment reaches them or not. */
  present: number
}

/** Group is one auditionable area of the game, named by the cue vocabulary itself. */
export interface Group {
  key: string
  label: string
  clips: number
}

/** Playback reports whether the output device is sounding anything. */
export interface Playback {
  playing: boolean
}

/** Audition reports which clip an audition actually played. */
export interface Audition {
  group: string
  clip: string
}

export interface Reaction {
  at: string
  cue: string
  /** The moment's full title, as every list names it (FR-233); the live indicator says it (FR-719). */
  title: string
  clip: string
  outcome: string
}

/**
 * CueEntry is one cue named for a reader.
 *
 * The title is what reaches the screen, since an id names the cue for the code rather
 * than for a commander. The id travels with it because it keys the list and because
 * the reaction log reports ids, so a silence seen there has something to match.
 */
export interface CueEntry {
  id: string
  title: string
  /** The name of the folder holding the cue's takes, dots written as underscores on the Go side. */
  folder: string
  /** When the cue is heard, written by hand in the cue table (FR-231). */
  purpose: string
}

/** CueBreakdown is one voice's whole relationship with the cue table. */
export interface CueBreakdown {
  voice: string
  served: CueEntry[]
  unserved: CueEntry[]
}

/** Checklist is what one voice folder still has no recording for, with its progress. */
export interface Checklist {
  voice: string
  recorded: number
  total: number
  missing: CueEntry[]
  /** The voice folder's full path, ending in the separator: a moment's folder is this plus its id. */
  folder: string
}

/** VoiceFolders reports what making a voice's folders did; an empty path is a cancel. */
export interface VoiceFolders {
  path: string
  made: number
}

export interface About {
  name: string
  tagline: string
  version: string
  author: string
  copyright: string
  /** Who wrote the application. */
  authorship: string
  /** Whose voices it speaks with. */
  attribution: string
  /** The terms it is released under, in a sentence; the full text is Licence(). */
  licence: string
  credits: string[]
}

/**
 * Chatter is what the Chatter pane shows: every category in order with its moments (FR-727).
 * problem says why the last switch pressed could not be kept, the switch applying all the same;
 * empty while it was kept (FR-633).
 */
export interface Chatter {
  categories: ChatterCategory[]
  problem: string
}

/** ChatterCategory is one category Chatter lists, its moments in the table's order. */
export interface ChatterCategory {
  name: string
  moments: ChatterMoment[]
}

/** ChatterMoment is one moment named for a reader, with whether it is switched on. */
export interface ChatterMoment {
  cue: CueEntry
  on: boolean
}
