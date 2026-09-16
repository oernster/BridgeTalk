// The recording double that stands where Wails puts the real binding, beside the page with no
// binding at all.
//
// Both halves of the api suite read it: one asks what a call reaches, the other what a call
// answers when it does not happen. Written twice they would be two statements that can disagree,
// and the half nobody edited would be the half that passes while it should not.

/** installBridge puts a recording double where Wails would put the real binding. */
export function installBridge(overrides: Record<string, unknown> = {}) {
  const calls: Array<{ name: string; args: unknown[] }> = []
  const record =
    (name: string, answer: unknown) =>
    (...args: unknown[]) => {
      calls.push({ name, args })
      return Promise.resolve(answer)
    }

  const App: Record<string, unknown> = {
    State: record('State', { voice: 'Alpha' }),
    Voices: record('Voices', [{ name: 'Alpha' }]),
    SelectVoice: record('SelectVoice', undefined),
    MachineVoices: record('MachineVoices', [{ id: 'bf_emma', name: 'Emma (British, female)' }]),
    CastMachineVoice: record('CastMachineVoice', undefined),
    PluginVoices: record('PluginVoices', [{ plugin: 'Bridge Crew', id: 'one' }]),
    CastPluginVoice: record('CastPluginVoice', undefined),
    Making: record('Making', { voice: 'bf_emma' }),
    Muted: record('Muted', true),
    SetMuted: record('SetMuted', undefined),
    Volume: record('Volume', 0.5),
    SetVolume: record('SetVolume', undefined),
    AuditionGroups: record('AuditionGroups', [{ key: 'ShieldState' }]),
    Audition: record('Audition', { group: 'ShieldState', clip: 'a.mp3' }),
    StopAudition: record('StopAudition', undefined),
    MachineAuditionGroups: record('MachineAuditionGroups', [{ key: 'Docked' }]),
    AuditionMachineVoice: record('AuditionMachineVoice', { group: 'Docked', clip: 'k.flac' }),
    PluginAuditionGroups: record('PluginAuditionGroups', [{ key: 'Docked' }]),
    AuditionPluginVoice: record('AuditionPluginVoice', { group: 'Docked', clip: 'p.mp3' }),
    TakeKeyboard: record('TakeKeyboard', undefined),
    Quit: record('Quit', undefined),
    Reactions: record('Reactions', [{ cue: 'ShieldState.ShieldsUp.false' }]),
    CueBreakdown: record('CueBreakdown', { voice: 'Alpha', served: [], unserved: [] }),
    About: record('About', { name: 'the application' }),
    Licence: record('Licence', 'the terms'),
    ChooseLibraryRoot: record('ChooseLibraryRoot', 'D:/Recordings'),
    ChooseJournalDir: record('ChooseJournalDir', 'D:/Journals'),
    MakeVoiceFolders: record('MakeVoiceFolders', { path: 'D:/Recordings/Alpha', made: 3 }),
    Rescan: record('Rescan', 2),
    VoiceDirectories: record('VoiceDirectories', ['Alpha', 'Beta']),
    Checklist: record('Checklist', {
      voice: 'Alpha',
      recorded: 1,
      total: 3,
      missing: [],
      folder: 'D:/Recordings/Alpha/',
    }),
    PluginChecklist: record('PluginChecklist', {
      voice: 'The First Officer',
      recorded: 1,
      total: 3,
      missing: [],
      folder: '',
    }),
    OpenMomentFolder: record('OpenMomentFolder', undefined),
    OpenDonation: record('OpenDonation', undefined),
    SetLaunchOnBoot: record('SetLaunchOnBoot', undefined),
    MinimiseToTray: record('MinimiseToTray', undefined),
    RequestQuit: record('RequestQuit', undefined),
    Chatter: record('Chatter', { categories: [], problem: '' }),
    SetMoment: record('SetMoment', { categories: [], problem: '' }),
    SetCategory: record('SetCategory', { categories: [], problem: '' }),
    SetAllMoments: record('SetAllMoments', { categories: [], problem: '' }),
    ...overrides,
  }
  ;(window as unknown as { go: unknown }).go = { main: { App } }
  return calls
}

/** removeBridge leaves the page as a browser or a test sees it. */
export function removeBridge() {
  delete (window as unknown as { go?: unknown }).go
  delete (window as unknown as { runtime?: unknown }).runtime
}

