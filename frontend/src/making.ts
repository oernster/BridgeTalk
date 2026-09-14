// Where making stands before anything has been made.
//
// It lives apart from api.ts because a test replaces that module whole; the page and every test
// read this one value from here instead.

import type { Making } from './api'

/**
 * nothingMade is making with no machine voice cast: nothing made and nothing gone wrong. The Cast
 * pane starts from it; a page with no bridge behind it stays on it.
 */
export const nothingMade: Making = {
  voice: '',
  making: false,
  current: 0,
  total: 0,
  cuesServed: 0,
  failed: [],
  stopped: '',
  notDeleted: '',
}
