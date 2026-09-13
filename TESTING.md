# Testing

What is tested, what is not and why the line falls where it does.

This document exists because a coverage figure on its own is a number without a
claim behind it. Every figure here was measured at the moment of writing, by the
commands in [Running it](#running-it); every shortfall is named, with the reason it
is a shortfall rather than an omission.

## The standard

Two rules govern everything below.

**A floor is a measurement, never an aspiration.** A gate set to a number the code
does not reach teaches people to lower it. Every floor in `test.ps1` is taken from
what that package measured, rounded down, so it fails the moment cover is lost,
which is the only moment it is worth being told. One floor sits below its figure on
purpose: part of `internal/infrastructure/setup`'s figure depends on what the
registry of the machine running the tests holds, so its floor leaves room for a
machine whose registry differs.

**A gap is named or it is closed.** Where something cannot be tested, this document
says what it is and what stops it. An unexplained shortfall is indistinguishable
from an oversight; the reader has no way to tell which they are looking at.

There is a third case, neither of the two: code that cannot execute at all. That is not
a gap to name, it is dead code to delete; see [It could not happen, so it is
gone](#it-could-not-happen-so-it-is-gone).

## What the numbers are

### The backend

| Package | Coverage | Floor | Gated by |
|---|---|---|---|
| `internal/domain/...` | 100% | 100% | `test.ps1` |
| `internal/application/...` | 100% | 100% | `test.ps1` |
| `internal/infrastructure/status` | 100% | 100% | `test.ps1` |
| `internal/infrastructure/config` | 100% | 100% | `test.ps1` |
| `internal/infrastructure/journal` | 100% | 100% | `test.ps1` |
| `internal/infrastructure/library` | 100% | 100% | `test.ps1` |
| `internal/infrastructure/audio` | 80.4% | 80% | `test.ps1` |
| the root package (the Wails facade) | 79.5% | 75% | `test.ps1` |
| `internal/infrastructure/setup` | 67.3% | 61% | `test.ps1` |
| `internal/infrastructure/taskbar` | 22.4% | 22% | `test.ps1` |
| `internal/infrastructure/window` | 0% | none | not gated |
| `installer` | 0% | none | not gated |

368 test functions, which expand to 421 runs once their subtests are counted.
Twenty of them are the structural tests in `tests/structural`, which assert the
architecture itself rather than any behaviour: layer direction, domain purity, the
composition-root whitelist, the 400-line cap with its danger band and the rule
that the product is named in exactly one place.

### The front end

| File | Statements | Branches |
|---|---|---|
| `api.ts` | 100% | 100% |
| `audition.tsx` | 100% | 100% |
| `autoscroll.ts` | 100% | 100% |
| `cast.tsx` | 100% | 100% |
| `chooser.tsx` | 100% | 100% |
| `chrome.tsx` | 100% | 94.1% |
| `guideContent.ts` | 100% | 100% |
| `icons.tsx` | 100% | 100% |
| `preferences.ts` | 100% | 100% |
| `missingTakes.tsx` | 100% | 100% |
| `hooks.ts` | 100% | 94.6% |
| `guide.tsx` | 100% | 89.5% |
| `moments.tsx` | 100% | 94.1% |
| `panes.tsx` | 100% | 94.9% |
| `App.tsx` | 100% | 97.3% |
| `dialogs.tsx` | 99.2% | 64.5% |
| `main.tsx` | 0% | 0% |
| **all files** | **99.2%** | **95.6%** |

174 tests across 15 files, run under Vitest with jsdom.

A figure of 100% says every line ran, not that a test would notice the line being wrong.
`missingTakes.tsx` read 100% while six of its behaviours could be broken with every test still
passing. The pane, the menu item that reaches it, the theme item beside it and the bridge
calls in `api.ts` were then proved by planting a violation for each behaviour and reading
the exit code: 43 plants in all, every one caught.

## How each layer is tested

| Layer | Kind of test | Touches |
|---|---|---|
| `internal/domain` | pure unit | nothing |
| `internal/application` | unit, over hand-written fakes | nothing |
| `internal/infrastructure` | integration, over a temporary directory | the filesystem, plus the registry read-only |
| the root package | the facade over a fake device and a fake source | nothing |
| `tests/structural` | source and AST scans | reads files |
| the front end | component and hook tests under jsdom | nothing |

No Go test uses a mocking library. Every Go double is a hand-written fake with the
real interface behind it: `fakePlayer` for the output device, `fakeSource` for an
event source, `fakeSettings` for the settings store, plus a recording double for the
Wails binding. The front end replaces its `api` module with Vitest's own `vi.mock`,
so a component renders without a Wails bridge behind it.

**No test writes to the machine it runs on.** Every filesystem test builds its tree
under `t.TempDir()`. Every test that would otherwise reach the user's own
directories redirects `LOCALAPPDATA`, `APPDATA` or `USERPROFILE` into a temporary
tree first; `redirectUserDirectories` in `internal/infrastructure/setup` proves
the redirection took effect before anything is removed: if it did not take, the test
fails rather than deleting a real shortcut.

## What is not tested and why

### The platform owns it

These need Windows itself or a device; no harness reaches them. A defect in any
of them is found by running the application, which is what the manual pass before a
release is for.

- **`internal/infrastructure/window` (0%).** Win32 focus handling: finding the
  WebView2 child window and giving it the keyboard. Opening a moment's folder in File
  Explorer lives here too. There is no window in a test; the facade reaches the opener
  through a field, so what it opens is tested while Explorer appearing is not.
- **`internal/infrastructure/taskbar` (22.4%).** The tray icon runs its own Win32
  message loop on a locked OS thread. What is testable without one, the command
  vocabulary and the state the menu reads, is tested; the loop, the window
  procedure and the menu construction are not.
- **`internal/infrastructure/audio` (80.4%).** `run` and `playOne` hand a loaded clip
  to the speaker; the speaker itself is the part no harness reaches. Everything around
  them is covered through a SILENT player, which is the same object with the same state
  machine minus the calls into the device, so the sequencing, the cancelling, the volume
  curve and every decoder path are all tested without a sound card. Reading a clip whole
  into memory is tested directly, by deleting the file before streaming what came back:
  a streamer still holding reading to do fails there, which is exactly the reading that
  must not happen on the device's thread. The stall counter is tested over an injected
  clock rather than by waiting.
- **The Wails calls in the root package.** `runtime.EventsEmit`, `WindowShow`,
  `WindowHide`, `WindowCenter`, `Quit` and the directory dialog need a running Wails
  application. Each is reached through a field on the facade, so the behaviour AROUND
  the call is fully tested and only the call itself is not: the tests substitute the
  field and assert what the facade decided.
- **`main`, `run`, `launch` and `startTray` in `main.go`.** The composition root. It
  opens a device, scans the disk, builds a tray and hands the assembled application
  to Wails. Running it in a test would be running the application.

### It would change the machine

- **`installer` (0%).** The setup program's own Wails facade. Every method on it
  writes the uninstall registry key, creates or removes shortcuts, closes a running
  copy or extracts a payload into the user's programs directory. The logic underneath
  it, in `internal/infrastructure/setup`, is tested against a temporary tree; the
  facade that drives it is not, because there is nowhere to redirect the acts to.
- **The registry writes in `internal/infrastructure/setup` (67.3% overall).**
  `WriteUninstallEntry`, `RemoveUninstallEntry` and `SetLaunchOnBoot` write to
  `HKCU`. Unlike a filesystem path there is nothing to point them at, so exercising
  them would register or deregister a real install on the machine running the tests.
  What they write is tested instead: the uninstall entry's text comes from
  `uninstallValues` and the login entry's from `runValue`, both portable.
  The registry READS beside them are exercised, because a read cannot damage
  anything. `SetLaunchOnBoot` is covered on the facade side only in its refusal path:
  a copy running from the temporary directory, which is what a test binary is.
- **Process control in `process_windows.go`.** Closing and launching processes;
  scheduling a directory deletion for the next restart. Enumerating them is tested:
  `processIDs` must find the test binary by its own name, which is the one process a
  test can be certain is running.
- **`createShortcut`.** Shells out to the Windows Script Host to write a `.lnk`. The
  REMOVING half of `ApplyShortcuts` is tested, through `RemoveShortcuts`, against
  redirected directories.

### It could not happen, so it is gone

A branch nothing can reach is not a gap in the tests; it is dead code, so documenting
it would be technical debt wearing a description. Branches like that turned up while
measuring were removed rather than excused, which is part of what took `config` and
`journal` to 100%. Three of them, in code that still exists:

| What it was | Why it could not run |
|---|---|
| the error check after `readAll` | every return in `readAll` was `return out, nil` |
| the empty check in `splitLines` | `bytes.Split` always yields at least one element |
| the encode failure in `Save` | a struct of three strings always marshals |

### The front end

- **`main.tsx` (0%).** Thirteen lines that mount the shell onto the page. There is
  nothing in it to be wrong that rendering the shell in every other test does not
  already prove.
- **The residual branches** in `App.tsx`, `chrome.tsx`, `dialogs.tsx`, `guide.tsx`,
  `hooks.ts`, `moments.tsx` and `panes.tsx` are of two kinds. Some are fallbacks for a value that
  has not arrived, such as an absent credits list or state. The rest depend on real
  layout: whether a region overflows, where a ref points before the node arrives. jsdom
  performs no layout, so what geometry the tests do assert is stated explicitly rather
  than inferred; the remainder is left rather than faked.

## Working with jsdom

jsdom has no layout engine, which decides the shape of several tests.

`src/test-setup.ts` provides two things jsdom lacks: an inert `ResizeObserver` and an
inert `Element.prototype.scrollTo`. Both are inert on purpose. Nothing there invents
a measurement, so a test that needs a region to overflow says so itself by defining
`scrollHeight` and `clientHeight` on the element. A stub that returned a size would
be a test asserting against the stub.

The same applies to the focus ring. Every element in jsdom reports a null
`offsetParent`, which the ring correctly reads as "not on screen", so without help it
finds no stops at all. `hooks.test.tsx` states the shape of the page by making
attached elements report a parent, then asserts what the ring does with it. What is
supplied is the page; what is asserted is the behaviour.

## Running it

The whole backend gate, which `build.ps1` runs before it builds and cannot be told
to skip:

```bash
./test.ps1
```

It checks formatting, runs `go vet`, runs every test, holds the domain and the
application layers at 100%, then holds every other package at its floor. Read the
exit code rather than the last line of output.

The stricter Go analysis, a superset of `go vet`, which `test.ps1` does not run:

```bash
go run honnef.co/go/tools/cmd/staticcheck@latest (go list ./... | Where-Object { $_ -notmatch '/node_modules/' })
```

The package list is narrowed rather than written as `./...`, which reaches into
`frontend/node_modules`, where an npm dependency ships a Go package of its own. It is
nobody here's code and nothing this repository produces contains it, so a future
version of it failing an analyser would break a build over something unowned.
`test.ps1` narrows the same way for `go vet` and `go test`; the formatting check
filters by path instead, because gofmt walks directories rather than packages.

The front end, from the `frontend` directory:

```bash
npx eslint . ; npx tsc --noEmit ; npx vitest run
```

The front-end coverage figures in this document:

```bash
npx vitest run --coverage
```

One package's coverage in detail, when a figure needs explaining:

```bash
go test -coverprofile=cover.out ./internal/infrastructure/library && go tool cover -func=cover.out
```

## Keeping this honest

Three habits, each of them learned by being caught out.

**Prove a new guard bites.** A test or a floor that has never been seen to fail is
not yet a guard. Plant the violation, read the exit code, then restore the file in a
`finally` so a failed run cannot leave the plant behind.

**Re-measure before quoting.** Every figure in this document was measured when it was
written. Copying one forward is how a document starts describing a repository that no
longer exists.

**Read the exit code, never the last line.** A coverage-gated run prints its table
last and emits no summary line, so a `tail` shows coverage rows and a grep for
`failed` matches file names. The exit code is the only answer.

## See also

- [DEVELOPMENT_README.md](DEVELOPMENT_README.md) for building and running on Windows.
- [ARCHITECTURE.md](ARCHITECTURE.md) for the invariants the structural tests enforce.
- [TECH_DEBT.md](TECH_DEBT.md) for what is open, what is deliberately left and what
  only looks like debt.
