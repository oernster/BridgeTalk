# Testing

What is tested, what is not and why the line falls where it does.

This document exists because a coverage figure on its own is a number without a
claim behind it. Every figure here was measured at the moment of writing, by the
commands in [Running it](#running-it); every shortfall is named, with the reason it
is a shortfall rather than an omission.

## The standard

Two rules govern everything below.

**A floor is a measurement, never an aspiration.** A gate set to a number the code
does not reach teaches people to lower it. Every floor in `test.ps1` sits at or a
little below what that package measured, so it fails once cover is lost, which is
the only moment it is worth being told. `internal/infrastructure/setup`'s floor leaves
room on purpose: its registry reads branch on what the registry of
the machine running the tests holds, so part of its figure moves from one machine to
the next and its floor leaves room for that.

**A gap is named or it is closed.** Where something cannot be tested, this document
says what it is and what stops it. An unexplained shortfall is indistinguishable
from an oversight; the reader has no way to tell which they are looking at.

There is a third case, neither of the two: code that cannot execute at all. That is not
a gap to name; it is dead code to delete. See [It could not happen, so it is
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
| `internal/infrastructure/tomlfile` | 100% | 100% | `test.ps1` |
| `internal/infrastructure/voicefiles` | 100% | 100% | `test.ps1` |
| `internal/infrastructure/appdata` | 100% | 100% | `test.ps1` |
| `internal/infrastructure/reporoot` | 100% | 100% | `test.ps1` |
| `internal/infrastructure/wholefile` | 100% | 100% | `test.ps1` |
| `internal/refusal` | 100% | 100% | `test.ps1` |
| `tools/internal/pyvenv` | 100% | 100% | `test.ps1` |
| `internal/infrastructure/modelfiles` | 99.1% | 99% | `test.ps1` |
| `internal/infrastructure/madelines` | 100% | 98% | `test.ps1` |
| `internal/infrastructure/audio/audiotest` | 86.1% | 86% | `test.ps1` |
| `internal/infrastructure/audio` | 93.6% | 80% | `test.ps1` |
| the root package (the Wails facade) | 82.5% | 75% | `test.ps1` |
| `internal/infrastructure/setup` | 74.2% | 61% | `test.ps1` |
| `internal/infrastructure/speechmodel` | 92.1% | 91% | `test.ps1` |
| `internal/infrastructure/taskbar` | 67.4% | 67% | `test.ps1` |
| `internal/infrastructure/runlog` | 55.2% | 51% | `test.ps1` |
| `tools/models` | 48.3% | 48% | `test.ps1` |
| `tools/payload` | 53.3% | 53% | `test.ps1` |
| `tools/sounds` | 44.9% | 38% | `test.ps1` |
| `tools/pauses` | 64.6% | 64% | `test.ps1` |
| `internal/infrastructure/modelfiles/modelfilestest` | test support, run by the `modelfiles`, `tools/models` and `tools/payload` tests | none | not gated |
| `internal/infrastructure/window` | 0% | none | not gated |
| `installer` | 0% | none | not gated |
| `internal/product` | no statements, constants only | none | not gated |

752 test functions, which expand to 816 runs once their subtests are counted (measured on
2026-09-15: `func Test` in every `_test.go` file bar `TestMain`, then `=== RUN` in a verbose run of
the whole suite; the build-tagged benchmarks are counted as functions but do not run).
Thirty-eight of them are the structural tests in `tests/structural`, which scan the source
rather than run it. They hold the layer direction, domain purity, the
composition-root whitelist, the 400-line cap with its danger band, a doc comment on
every exported type and the rule that the product is named in exactly one place
under an identity that is also a valid file name. They also hold the surface the
facade binds, the wire contract on both sides of it, colours confined to the tokens,
the contrast of the purpose line in both themes, every style part being read, the
setup program applying the boxes it shows with a header that repeats no title, the
setup page loading every script it has with its body ringed for the keyboard, game
vocabulary kept in its home, the shape of every cue id, the speech sound table held
to the model's tokenizer file, `pauses.toml` kept from going stale and every address handed to a DLL converted only where
the call into it is made.

### The front end

| File | Statements | Branches |
|---|---|---|
| `api.ts` | 100% | 100% |
| `audition.tsx` | 100% | 100% |
| `autoscroll.ts` | 100% | 100% |
| `cast.tsx` | 100% | 100% |
| `castWords.ts` | 100% | 100% |
| `chooser.tsx` | 100% | 100% |
| `guideContent.ts` | 100% | 100% |
| `icons.tsx` | 100% | 100% |
| `indicator.ts` | 100% | 100% |
| `machineVoices.tsx` | 100% | 100% |
| `making.ts` | 100% | 100% |
| `missingTakes.tsx` | 100% | 100% |
| `moments.tsx` | 100% | 100% |
| `preferences.ts` | 100% | 100% |
| `productName.ts` | 100% | 100% |
| `statusWords.ts` | 100% | 100% |
| `strip.tsx` | 100% | 100% |
| `testAudition.ts` | 100% | 100% |
| `testLayout.ts` | 100% | 100% |
| `testState.ts` | 100% | 100% |
| `App.tsx` | 100% | 96.3% |
| `hooks.ts` | 100% | 96.3% |
| `panes.tsx` | 100% | 95.8% |
| `chrome.tsx` | 100% | 94.3% |
| `guide.tsx` | 100% | 86.7% |
| `dialogs.tsx` | 99.3% | 69.4% |
| `main.tsx` | 0% | 0% |
| **all files** | **99.4%** | **96.7%** |

234 tests across 21 files, run under Vitest with jsdom.

A figure of 100% says every line ran, not that a test would notice the line being
wrong. The way to find out is to plant a violation for a behaviour and read the exit
code; see [Keeping this honest](#keeping-this-honest).

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
event source, `fakeSettings` for the settings store, plus a recorder that captures
what the facade emits to the front end. The front end replaces its `api` module with
Vitest's own `vi.mock`, so a component renders without a Wails bridge behind it.

**No test writes to the machine it runs on.** Every filesystem test builds its tree
under `t.TempDir()`. Every test that would otherwise reach the user's own
directories redirects `LOCALAPPDATA`, `APPDATA` or `USERPROFILE` into a temporary
tree first; `redirectUserDirectories` in `internal/infrastructure/setup` proves
the redirection took effect before anything is removed. If it did not take, the test
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
- **`internal/infrastructure/taskbar` (67.4%).** The tray icon runs its own Win32
  message loop on a locked OS thread. One test runs that loop for real over a real
  hidden window, replacing only the call that hands the icon to the shell, so the
  hover text being sent again after a change is tested while an icon appearing is
  not. The command vocabulary and the state the menu reads are tested directly; the
  menu as drawn is not.
- **`internal/infrastructure/audio` (93.6%).** `run` and `playOne` hand a loaded clip
  to the speaker; the speaker itself is the part no harness reaches. Everything around
  them is covered through a SILENT player, which is the same object with the same state
  machine minus the calls into the device, so the sequencing, the cancelling, the volume
  curve and every decoder path are all tested without a sound card. Reading a clip whole
  into memory is tested directly, by deleting the file before streaming what came back:
  a streamer still holding reading to do fails there, which is exactly the reading that
  must not happen on the device's thread. The stall counter is tested over an injected
  clock rather than by waiting.
- **The crashes in `internal/infrastructure/runlog` (55.2%).** A crash ends the process that has it,
  so the crash tests start the test binary again as a child that panics or fails fatally, then read
  what the child left in its log. Every line the child runs is in a process coverage does not measure.
  Finding that a run has no error output is not reached at all: a test binary is always given one.
  A windowed probe measured it on 2026-09-14 (FR-715). Writing the start line failing after the file
  opened fails only inside the system.
- **A made lines folder that exists but cannot be listed, in `internal/infrastructure/madelines`.**
  Deleting then says why rather than deleting nothing in silence (FR-530). Only a permission the
  system withholds reaches it: a folder a test makes can always be listed. Windows answers a plain
  file standing where the folder should be as not there at all, measured on 2026-09-14, which is
  nothing to delete.
- **The Wails calls in the root package.** `runtime.EventsEmit`, `WindowShow`,
  `WindowHide`, `WindowCenter`, `Quit` and the directory dialog need a running Wails
  application. Each is reached through a field on the facade, so the behaviour AROUND
  the call is fully tested and only the call itself is not: the tests substitute the
  field and assert what the facade decided.
- **`modelfiles.Dir` refusing where no folder above holds `go.mod` (99.1%).** Only a walk from outside
  any repository reaches it; whether a real drive holds a `go.mod` at its top is the machine's
  business. `reporoot` tests the same walk over a stand-in that answers no; `Dir` only passes its
  refusal on.
- **ONNX Runtime's own failures in `internal/infrastructure/speechmodel` (92.1% with `models/` filled).**
  Making the environment, the memory description, the session options or a tensor fails only inside
  ONNX Runtime; so does reading a made tensor's shape or data. So does a runtime too old to answer
  the version 23 function table. `loadReason` also keeps a fallback for a load error that is not
  `windows.DLLError`, which `golang.org/x/sys/windows` answers every load and lookup failure as. What a
  test can reach is tested: a missing runtime, a library that is not ONNX Runtime, a missing or
  damaged model, a path no file can have and a shipped line made by the real model. The tests that need
  the model files skip where `models/` lacks one; `test.ps1` checks the files before anything else and
  stops where one is missing, so the floor holds the 92.1% measured with them.
- **`main`, `run`, `launch` and `startTray` in `main.go`.** The composition root. It
  opens a device, scans the disk, builds a tray and hands the assembled application
  to Wails. Running it in a test would be running the application.

### It would change the machine

- **`installer` (0%).** The setup program's own Wails facade. Its methods write the
  uninstall registry key, create or remove shortcuts, write the login entry, close or
  launch the application or extract a payload into the user's programs directory; the
  few that do none of that read the machine or drive the Wails window. The logic
  underneath it, in `internal/infrastructure/setup`, is tested against a temporary
  tree. The facade calls that package directly rather than through a field, so there
  is nowhere to redirect its acts to.
- **The registry writes in `internal/infrastructure/setup` (74.2% overall).**
  `WriteUninstallEntry`, `RemoveUninstallEntry` and `SetLaunchOnBoot` write to
  `HKCU`. Unlike a filesystem path there is nothing to point them at, so exercising
  them would register or deregister a real install on the machine running the tests.
  What they write is tested instead: the uninstall entry's text comes from
  `uninstallValues` and the login entry's from `runValue`, both portable.
  The registry READS beside them are exercised, because a read cannot damage
  anything. `SetLaunchOnBoot` is covered on the facade side only in its refusal path:
  a copy running from the temporary directory, which is what a test binary is.
- **Process control in `process_windows.go`.** Closing and launching the application;
  deleting the install directory through a detached shell that outlives the setup
  program. Enumerating processes is tested: `processIDs` must find the test binary by
  its own name, which is the one process a test can be certain is running.
- **`tools/sounds` (44.9%).** The sounds tool. `run` rewrites `sounds.toml` in the repository
  and `python.Make` runs `sounds.py` in the tool's own venv, which a test machine need not have.
  Making every line in each accent from its spelling and matching the answers back to their
  lines is tested in `makeSounds` over a hand-written maker. Finding the venv's Python is
  `tools/internal/pyvenv`'s, tested there for both layouts. What the real tool wrote is checked
  by the structural test over `sounds.toml`.
- **`tools/pauses` (64.6%).** The pauses tool. `main` and `start` find the repository, check
  `models/`, make lines with the real model through ONNX Runtime and rewrite `pauses.toml`;
  `python.Find` runs `pauses.py`, which needs Praat through parselmouth, in the tool's own venv,
  which a test machine need not have. Over a hand-written maker and finder, the rest is tested: the
  digest of each line's samples, the WAV files handed to the finder, which lines are doubtful, the
  flags and a run refused for a failing maker, finder or voice's files, a wrong answer or a changed
  model. Inside a run, a voice's temporary folder that cannot be made and a line that cannot be
  written are not reached; `writeWAV`'s refusal is tested on its own. Nor is the refusal of a line's
  numbers or style row, which lines from the voiced script do not give; both refusals are tested in
  `speech`. What the real tool wrote is checked by the structural test over `pauses.toml`.
- **`tools/models` (48.3%).** The model files tool. `main` and `start` read the real list, find
  `models/` in the repository and hand `run` a client that reaches the internet. `run` itself is tested
  in both modes over a local server, as `internal/infrastructure/modelfiles` is: no test downloads
  anything or writes outside a temporary folder. The figure was 53.1% until checking the folder moved
  into `modelfiles.Verify`, which the payload tool shares; the check is tested there, so the fall is
  statements leaving `run` rather than cover lost.
- **`tools/payload` (53.3%).** The payload tool. `main` and `start` read the real list and find
  `models/` in the repository. `run` is tested over temporary folders: a packing, a models folder that
  does not match, an application folder without the application and a missing flag.

### It could not happen, so it is gone

A branch nothing can reach is not a gap in the tests; it is dead code, so documenting
it would be technical debt wearing a description. Branches like that are removed
rather than excused. Four of them, in code that still exists:

| What it was | Why it could not run |
|---|---|
| the error return from `readAll` | it could only ever have been nil, so `readAll` now returns bytes alone |
| the empty check in `splitLines` | `bytes.Split` always yields at least one element |
| the encode failure in `Save` | a struct of three strings always marshals |
| the encode failures in `madelines.encode` | the FLAC encoder fails only on a write (which memory never refuses) or on a frame shape other than the two the store builds, both round-tripped by its tests |

### The front end

- **`main.tsx` (0%).** Thirteen lines that mount the shell onto the page. There is
  nothing in it to be wrong that rendering the shell in every other test does not
  already prove.
- **The residual branches** in `App.tsx`, `chrome.tsx`, `dialogs.tsx`, `guide.tsx`,
  `hooks.ts` and `panes.tsx` are of two kinds. Some are fallbacks for a value that
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
supplied is the page; what is asserted is the behaviour. `setupRing.test.ts` does the
same for the setup page, whose ring is a script of its own loaded as the page ships it.

## Running it

Every command below is PowerShell, run from the repository root unless it says
otherwise.

The gate needs the model files in `models/`, which the model files tool downloads from
their pinned addresses, fetching only what is missing or different:

```powershell
go run ./tools/models
```

The whole backend gate, which `build.ps1` runs before it builds and cannot be told
to skip:

```powershell
./test.ps1
```

It checks `models/` against the model files list first and stops where a file is
missing or differs, saying to run the tool above. It then checks formatting, runs
`go vet`, runs every test, holds the domain and the application layers at 100%, then
holds each other gated package at its floor. Read the exit code rather than the last
line of output.

The tests that need the real model take minutes, so the gate runs them only when
asked; `build.ps1` always asks:

```powershell
./test.ps1 -Benchmarks
```

It vets and runs the files carrying the `benchmarks` build tag in `tests/machinevoice`
and `internal/infrastructure/speechmodel`. One casts a voice and fails over NFR-P-205's
5 seconds or FR-514's 2 seconds. One makes the shipped script for one voice and fails
over NFR-C-502's 60 MB, skipping while the script lacks lines for any cue. One makes
150 lines while collections shrink goroutine stacks and fails on any line that panics,
fails or comes back empty: on 2026-09-14 it broke 6 of 150 lines before every address
handed to ONNX Runtime was converted where the call is made; none broke after.

The stricter Go analysis, which `test.ps1` does not run:

```powershell
go run honnef.co/go/tools/cmd/staticcheck@latest (go list ./... | Where-Object { $_ -notmatch '/node_modules/' })
```

The package list is narrowed rather than written as `./...`, which reaches into
`frontend/node_modules`, where an npm dependency ships a Go package of its own. It is
nobody here's code and nothing this repository produces contains it, so a future
release of it failing an analyser would break a build over something unowned.
`test.ps1` narrows the same way for `go vet` and `go test`; the formatting check
filters by path instead, because gofmt walks directories rather than packages.

The front end, from the `frontend` directory:

```powershell
npx eslint . ; npx tsc --noEmit ; npx vitest run
```

The front-end coverage figures in this document:

```powershell
npx vitest run --coverage
```

One package's coverage in detail, when a figure needs explaining:

```powershell
go test -coverprofile=cover.out ./internal/infrastructure/library && go tool cover -func=cover.out
```

## Keeping this honest

Three habits.

**Prove a new guard bites.** A test or a floor that has never been seen to fail is
not yet a guard. Plant the violation, read the exit code, then restore the file in a
`finally` so a failed run cannot leave the plant behind.

**Re-measure before quoting.** Every figure in this document was measured when it was
written. Copying one forward is how a document starts describing a repository that no
longer exists.

**Read the exit code, never the last line.** The front-end coverage run prints its
table after the test summary, so a `tail` shows coverage rows rather than the
verdict. The exit code is the only answer.

## See also

- [DEVELOPMENT_README.md](DEVELOPMENT_README.md) for building and running on Windows.
- [ARCHITECTURE.md](ARCHITECTURE.md) for the invariants the structural tests enforce.
- [TECH_DEBT.md](TECH_DEBT.md) for what is open, what is deliberately left and what
  only looks like debt.
