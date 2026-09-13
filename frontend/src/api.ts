// Typed access to the Go facade.
//
// Wails injects window.go.main.App and window.runtime at load time. Wrapping them
// here keeps the binding shape in one file, so a rename on the Go side is one edit
// rather than a search across components.

export interface State {
  voice: string
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
}

export interface Voice {
  name: string
  /** Clips the application can actually reach, not the size of the voice. */
  inUse: number
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

interface Bridge {
  State(): Promise<State>
  Voices(): Promise<Voice[]>
  SelectVoice(name: string): Promise<void>
  Muted(): Promise<boolean>
  SetMuted(muted: boolean): Promise<void>
  Volume(): Promise<number>
  SetVolume(level: number): Promise<void>
  AuditionGroups(voice: string): Promise<Group[]>
  Audition(voice: string, group: string): Promise<Audition>
  StopAudition(): Promise<void>
  Playing(): Promise<boolean>
  TakeKeyboard(): Promise<void>
  Quit(): Promise<void>
  Reactions(): Promise<Reaction[]>
  CueBreakdown(name: string): Promise<CueBreakdown>
  About(): Promise<About>
  Licence(): Promise<string>
  ChooseLibraryRoot(): Promise<string>
  ChooseJournalDir(): Promise<string>
  MakeVoiceFolders(name: string): Promise<VoiceFolders>
  Rescan(): Promise<number>
  VoiceDirectories(): Promise<string[]>
  Checklist(voice: string): Promise<Checklist>
  OpenMomentFolder(voice: string, id: string): Promise<void>
  SetLaunchOnBoot(enabled: boolean): Promise<void>
  MinimiseToTray(): Promise<void>
  RequestQuit(): Promise<void>
}

interface WailsWindow {
  go?: { main?: { App?: Bridge } }
  runtime?: {
    EventsOn(name: string, handler: (...data: unknown[]) => void): () => void
  }
}

const bridge = (): Bridge | null =>
  (window as unknown as WailsWindow).go?.main?.App ?? null

// A missing bridge means the page is being viewed outside the window, in a browser or
// a test. Every call degrades to an empty value rather than throwing, so the shell
// still renders and the fault shows as empty state rather than a blank screen.
export const api = {
  state: (): Promise<State | null> => bridge()?.State() ?? Promise.resolve(null),
  voices: (): Promise<Voice[]> => bridge()?.Voices() ?? Promise.resolve([]),
  selectVoice: (name: string): Promise<void> =>
    bridge()?.SelectVoice(name) ?? Promise.resolve(),
  setMuted: (muted: boolean): Promise<void> =>
    bridge()?.SetMuted(muted) ?? Promise.resolve(),
  volume: (): Promise<number> => bridge()?.Volume() ?? Promise.resolve(1),
  setVolume: (level: number): Promise<void> =>
    bridge()?.SetVolume(level) ?? Promise.resolve(),
  auditionGroups: (voice: string): Promise<Group[]> =>
    bridge()?.AuditionGroups(voice) ?? Promise.resolve([]),
  audition: (voice: string, group: string): Promise<Audition | null> =>
    bridge()?.Audition(voice, group) ?? Promise.resolve(null),
  stopAudition: (): Promise<void> => bridge()?.StopAudition() ?? Promise.resolve(),
  /** Whether the device is sounding anything, asked once when the audition pane opens. */
  playing: (): Promise<boolean> => bridge()?.Playing() ?? Promise.resolve(false),
  takeKeyboard: (): Promise<void> => bridge()?.TakeKeyboard() ?? Promise.resolve(),
  quit: (): Promise<void> => bridge()?.Quit() ?? Promise.resolve(),
  reactions: (): Promise<Reaction[]> => bridge()?.Reactions() ?? Promise.resolve([]),
  cueBreakdown: (name: string): Promise<CueBreakdown> =>
    bridge()?.CueBreakdown(name) ?? Promise.resolve({ voice: name, served: [], unserved: [] }),
  about: (): Promise<About | null> => bridge()?.About() ?? Promise.resolve(null),
  licence: (): Promise<string | null> => bridge()?.Licence() ?? Promise.resolve(null),

  /**
   * Both open the system's own directory chooser and take the answer at once, then
   * answer with the directory they took. A cancelled dialog is not an error and changes
   * nothing, so neither rejects on it; it comes back as an empty string instead, which
   * is what lets the pane tell a cancel apart from a directory that was accepted.
   */
  chooseLibraryRoot: (): Promise<string> => bridge()?.ChooseLibraryRoot() ?? Promise.resolve(''),
  chooseJournalDir: (): Promise<string> =>
    bridge()?.ChooseJournalDir() ?? Promise.resolve(''),

  /**
   * Makes a voice's folders, one per moment, in the product's own recordings directory
   * when none is chosen yet. It rejects a name that cannot be a folder. With no bridge it
   * makes nothing and answers with an empty path.
   */
  makeVoiceFolders: (name: string): Promise<VoiceFolders> =>
    bridge()?.MakeVoiceFolders(name) ?? Promise.resolve({ path: '', made: 0 }),

  /** Reads the recordings directory again and answers with how many voices it found. */
  rescan: (): Promise<number> => bridge()?.Rescan() ?? Promise.resolve(0),

  /** Every voice folder under the recordings directory, including those still empty. */
  voiceDirectories: (): Promise<string[]> =>
    bridge()?.VoiceDirectories() ?? Promise.resolve([]),

  /** What one voice folder still has no recording for, with its progress. */
  checklist: (voice: string): Promise<Checklist> =>
    bridge()?.Checklist(voice) ??
    Promise.resolve({ voice, recorded: 0, total: 0, missing: [], folder: '' }),

  /**
   * Opens the folder a take for one moment belongs in, making it where it is missing. It
   * rejects with the reason where the folder cannot be made or shown.
   */
  openMomentFolder: (voice: string, id: string): Promise<void> =>
    bridge()?.OpenMomentFolder(voice, id) ?? Promise.resolve(),

  /**
   * Starts or stops the application being launched at sign-in. It rejects rather than
   * failing quietly where the running copy could not be registered.
   */
  setLaunchOnBoot: (enabled: boolean): Promise<void> =>
    bridge()?.SetLaunchOnBoot(enabled) ?? Promise.resolve(),

  /**
   * The two answers to the close question. Minimising leaves the application
   * listening with only its tray icon on screen; quitting ends it.
   */
  minimiseToTray: (): Promise<void> => bridge()?.MinimiseToTray() ?? Promise.resolve(),
  requestQuit: (): Promise<void> => bridge()?.RequestQuit() ?? Promise.resolve(),
}

// on subscribes to a Wails event and returns the unsubscribe function; a no-op
// when there is no runtime to subscribe to.
export function on(name: string, handler: (...data: unknown[]) => void): () => void {
  const runtime = (window as unknown as WailsWindow).runtime
  if (!runtime) return () => undefined
  return runtime.EventsOn(name, handler)
}
