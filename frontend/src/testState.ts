// The state the shell's tests start from.
//
// One voice cast and speaking, a journal being watched and nothing gone wrong: the ordinary
// running application. A test that needs something else spreads this and changes the one field
// it is about, so the fixture has one home rather than a copy in every suite.

import type { State } from './api'

/** watching is the application running normally, with Grace cast. */
export const watching: State = {
  voice: 'Grace',
  voiceDisplay: 'Grace',
  bound: 40,
  total: 60,
  muted: false,
  silent: false,
  journalDir: 'D:/Journals',
  statusPath: 'D:/Journals/Status.json',
  libraryRoot: 'D:/Recordings',
  version: '9.9.9',
  launchOnBoot: false,
  stalls: 0,
  worstStall: 0,
  journalProblem: '',
  machineVoice: false,
}
