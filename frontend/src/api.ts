// Typed access to the Go facade.
//
// Wails injects window.go.main.App and window.runtime at load time. Wrapping them
// here keeps the binding shape in one file, so a rename on the Go side is one edit
// rather than a search across components.

import { nothingMade } from './making'
import type {
  About,
  Audition,
  Chatter,
  Checklist,
  CueBreakdown,
  Group,
  MachineVoice,
  Making,
  PluginVoice,
  Reaction,
  State,
  Voice,
  VoiceFolders,
} from './wire'

// The wire shapes are re-exported here so a component reads its shapes and its calls from
// one place, which is how every one of them already imports them.
export type {
  About,
  Audition,
  Chatter,
  ChatterCategory,
  ChatterMoment,
  Checklist,
  CueBreakdown,
  CueEntry,
  Group,
  LineFailure,
  MachineVoice,
  Making,
  Playback,
  PluginVoice,
  Reaction,
  State,
  Voice,
  VoiceFolders,
} from './wire'


/** noChatter is a Chatter pane with nothing to list, the answer outside the window. */
const noChatter = (): Chatter => ({ categories: [], problem: '' })

interface Bridge {
  State(): Promise<State>
  Voices(): Promise<Voice[]>
  SelectVoice(name: string): Promise<void>
  MachineVoices(): Promise<MachineVoice[]>
  CastMachineVoice(id: string): Promise<void>
  PluginVoices(): Promise<PluginVoice[]>
  CastPluginVoice(plugin: string, id: string): Promise<void>
  Making(): Promise<Making>
  Muted(): Promise<boolean>
  SetMuted(muted: boolean): Promise<void>
  Volume(): Promise<number>
  SetVolume(level: number): Promise<void>
  AuditionGroups(voice: string): Promise<Group[]>
  Audition(voice: string, group: string): Promise<Audition>
  StopAudition(): Promise<void>
  MachineAuditionGroups(): Promise<Group[]>
  AuditionMachineVoice(id: string, group: string): Promise<Audition>
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
  PluginChecklist(): Promise<Checklist>
  OpenMomentFolder(voice: string, id: string): Promise<void>
  OpenDonation(): Promise<void>
  SetLaunchOnBoot(enabled: boolean): Promise<void>
  MinimiseToTray(): Promise<void>
  RequestQuit(): Promise<void>
  Chatter(): Promise<Chatter>
  SetMoment(id: string, on: boolean): Promise<Chatter>
  SetCategory(name: string, on: boolean): Promise<Chatter>
  SetAllMoments(on: boolean): Promise<Chatter>
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
  /** Every machine voice offered, by the name the screen shows (FR-508, FR-528). */
  machineVoices: (): Promise<MachineVoice[]> => bridge()?.MachineVoices() ?? Promise.resolve([]),
  /** Casts a machine voice by id; rejects with the reason where it is refused (FR-519). */
  castMachineVoice: (id: string): Promise<void> =>
    bridge()?.CastMachineVoice(id) ?? Promise.resolve(),
  /** Every voice every loaded plugin offers; none where no plugin is loaded (FR-562). */
  pluginVoices: (): Promise<PluginVoice[]> => bridge()?.PluginVoices() ?? Promise.resolve([]),
  /**
   * Casts a plugin voice by the plugin that offered it and its id within that plugin; rejects with
   * the reason where it is refused, which a voice with no audio behind it is (FR-570).
   */
  castPluginVoice: (plugin: string, id: string): Promise<void> =>
    bridge()?.CastPluginVoice(plugin, id) ?? Promise.resolve(),
  /** How far making the cast machine voice's lines has got. */
  making: (): Promise<Making> => bridge()?.Making() ?? Promise.resolve(nothingMade),
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
  /** The script's groups a machine voice is auditioned on, each counting its lines (FR-546). */
  machineAuditionGroups: (): Promise<Group[]> =>
    bridge()?.MachineAuditionGroups() ?? Promise.resolve([]),
  /** Plays a line of a group for a machine voice, making it first where it is not yet made (FR-546). */
  auditionMachineVoice: (id: string, group: string): Promise<Audition | null> =>
    bridge()?.AuditionMachineVoice(id, group) ?? Promise.resolve(null),
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
   * What the cast plugin voice has no take for, with no folder at all: a plugin's audio is the
   * plugin's own and there is nothing here to open or create (FR-571). It is empty where the cast
   * voice is of any other kind.
   */
  pluginChecklist: (): Promise<Checklist> =>
    bridge()?.PluginChecklist() ??
    Promise.resolve({ voice: '', recorded: 0, total: 0, missing: [], folder: '' }),

  /**
   * Opens the folder a take for one moment belongs in, making it where it is missing. It
   * rejects with the reason where the folder cannot be made or shown.
   */
  openMomentFolder: (voice: string, id: string): Promise<void> =>
    bridge()?.OpenMomentFolder(voice, id) ?? Promise.resolve(),

  /**
   * Hands the donation page to the desktop to open in the browser (FR-718). It rejects where that
   * could not be done. With no bridge there is no desktop to hand it to, so it does nothing.
   */
  openDonation: (): Promise<void> => bridge()?.OpenDonation() ?? Promise.resolve(),

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

  /** What the Chatter pane shows (FR-727); with no bridge it lists nothing. */
  chatter: (): Promise<Chatter> => bridge()?.Chatter() ?? Promise.resolve(noChatter()),
  /** Switches one moment on or off, answering the pane as it now stands (FR-729). */
  setMoment: (id: string, on: boolean): Promise<Chatter> =>
    bridge()?.SetMoment(id, on) ?? Promise.resolve(noChatter()),
  /** Switches every moment in one category on or off (FR-731). */
  setCategory: (name: string, on: boolean): Promise<Chatter> =>
    bridge()?.SetCategory(name, on) ?? Promise.resolve(noChatter()),
  /** Switches every moment on or off (FR-732). */
  setAllMoments: (on: boolean): Promise<Chatter> =>
    bridge()?.SetAllMoments(on) ?? Promise.resolve(noChatter()),
}

// on subscribes to a Wails event and returns the unsubscribe function; a no-op
// when there is no runtime to subscribe to.
export function on(name: string, handler: (...data: unknown[]) => void): () => void {
  const runtime = (window as unknown as WailsWindow).runtime
  if (!runtime) return () => undefined
  return runtime.EventsOn(name, handler)
}
