# Testing

What is tested, what is not and why the line falls where it does.

This document exists because a coverage figure on its own is a number without a
claim behind it. Every figure here was measured at the moment of writing, by the
commands in [Running it](#running-it); every shortfall is named, with the reason it
is a shortfall rather than an omission.

## The standard

Two rules govern everything below.

**A floor is a measurement, never an aspiration.** A gate set to a number the code
does not reach teaches people to lower it. Every floor in `test.ps1` sits at or below
what that package measured, so it fails once cover is lost, which is
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
| `internal/infrastructure/madelines` | 100% | 100% | `test.ps1` |
| `internal/infrastructure/audio/audiotest` | 86.1% | 86% | `test.ps1` |
| `internal/infrastructure/audio` | 95.3% | 95% | `test.ps1` |
| the root package (the Wails facade) | 82.0% | 82% | `test.ps1` |
| `internal/infrastructure/setup` | 76.3% | 61% | `test.ps1` |
| `internal/infrastructure/speechmodel` | 92.1% | 91% | `test.ps1` |
| `internal/infrastructure/taskbar` | 68.1% | 68% | `test.ps1` |
| `internal/infrastructure/runlog` | 48.8% | 48% | `test.ps1` |
| `tools/models` | 48.3% | 48% | `test.ps1` |
| `tools/payload` | 53.3% | 53% | `test.ps1` |
| `tools/sounds` | 44.9% | 44% | `test.ps1` |
| `tools/pauses` | 73.6% | 73% | `test.ps1` |
| `internal/infrastructure/modelfiles/modelfilestest` | test support with no tests of its own, used by the `modelfiles`, `speechmodel`, `tools/models`, `tools/payload`, `tests/structural` and `tests/machinevoice` tests | none | not gated |
| `internal/infrastructure/window` | 0% | none | not gated |
| `installer` | 0% | none | not gated |
| `internal/product` | no statements, constants only | none | not gated |

836 test functions, which expand to 907 runs once their subtests are counted (measured on
2026-09-15: `func Test` in every `_test.go` file bar `TestMain`, then `=== RUN` in a verbose run of
the whole suite; the build-tagged benchmarks are counted as functions but do not run).
Forty-eight of them are the structural tests in `tests/structural`, which scan the source
rather than run it. They hold the layer direction, domain purity, the
composition-root whitelist, the 400-line cap with its danger band, a doc comment on
every exported type and the rule that the product is named in exactly one place
under an identity that is also a valid file name. They also hold the surface the
facade binds, the wire contract on both sides of it, colours confined to the tokens
with a token for every indicator tone, the contrast of the secondary lines (the
purpose line and the Status cards' taglines) in both themes, every style part being
read, the strip and the Status cards keeping their layout rules, every disabled control
wearing the danger ring, every scrolling region ringed for the keyboard but never under
the pointer, no list or container wearing a ring, nothing on the ring that cannot be acted
on or scrolled, the setup
program applying the boxes it shows with a header that repeats no title, the setup page
loading every script it has with its body ringed for the keyboard, game vocabulary kept
in its home, the shape of every cue id, the shipped script holding no problem with lines
for every cue, the speech sound table held to the model's tokenizer file, `pauses.toml`
and `endings.toml` kept from going stale and every address handed to a DLL converted
only where the call into it is made.

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
| `testHome.ts` | 100% | 100% |
| `testLayout.ts` | 100% | 100% |
| `testState.ts` | 100% | 100% |
| `App.tsx` | 100% | 96.3% |
| `hooks.ts` | 100% | 96.3% |
| `panes.tsx` | 100% | 95.9% |
| `chrome.tsx` | 100% | 94.3% |
| `guide.tsx` | 100% | 86.7% |
| `dialogs.tsx` | 99.3% | 69.4% |
| `main.tsx` | 0% | 0% |
| **all files** | **99.4%** | **96.7%** |

252 tests across 23 files, run under Vitest with jsdom.

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
- **`internal/infrastructure/taskbar` (68.1%).** The tray icon runs its own Win32
  message loop on a locked OS thread. One test runs that loop for real over a real
  hidden window, replacing only the call that hands the icon to the shell, so the
  hover text being sent again after a change is tested while an icon appearing is
  not. The command vocabulary and the state the menu reads are tested directly; the
  menu as drawn is not.
- **`internal/infrastructure/audio` (95.3%).** `run` and `playOne` hand a loaded clip
  to the speaker. What the speaker decides about its queue is tested over a fake of the
  device's queue in `speaker_test.go`: a take that follows another closely waits for its end,
  while a stop, an interrupting take and a take after silence each drop what is queued. The
  device itself is the part no harness reaches. Everything around
  them is covered through a SILENT player, which is the same object with the same state
  machine minus the calls into the device, so the sequencing, the cancelling and the
  volume curve are tested without a sound card. What stays unreached is the device
  failing to open (`NewPlayer`, `outputContext`, `openSpeaker`), a clip at another sample
  rate being resampled, a clip that decodes to no audio, a probe that ends in a stream
  error, an unreadable clip in `playOne` and a cancel landing between clips or during the
  gap in `run`. The figure moves between runs: 95.0%, 95.3% and 95.5% were all
  measured on 2026-09-15, so the floor sits at 95%. Reading a clip whole
  into memory is tested directly, by deleting the file before streaming what came back:
  a streamer still holding reading to do fails there, which is exactly the reading that
  must not happen on the device's thread. The stall counter is tested over an injected
  clock rather than by waiting.
- **`internal/infrastructure/audio/audiotest` (86.1%).** Test support. What is not run is
  its own failure branches: a WAV that cannot be built, a format it cannot make and a
  folder or file that cannot be written. Each fails the test that called it, which no
  passing run does.
- **The crashes in `internal/infrastructure/runlog` (48.8%).** A crash ends the process that has it,
  so the crash tests start the test binary again as a child that panics or fails fatally, then read
  what the child left in its log. Every line the child runs is in a process coverage does not measure.
  Finding that a run has no error output is not reached at all: a test binary is always given one.
  A windowed probe measured it on 2026-09-14 (FR-715). Writing the start line failing after the file
  opened fails only inside the system. Reaching for the terminal a windowed run was started from
  (FR-703) goes past its first check only where the run was given no standard output, which a test
  binary never is; a windowed build of the test binary was seen printing in the terminal it was
  started from on 2026-09-15.
- **The Wails calls in the root package.** `runtime.EventsEmit`, `WindowShow`,
  `WindowHide`, `WindowCenter`, `Quit`, `BrowserOpenURL` and the directory dialog need a running Wails
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
- **`main`, `run`, `startTray` and `newMaking` in `main.go`, `launch` in `window.go`
  and `keepLog`, `runLog` and `reportToTerminal` in `runlog.go`.** The composition root. It keeps the run's
  log, opens a device, scans the disk, builds a tray and the speech model and hands the
  assembled application to Wails. Running it in a test would be running the application.
- **The rest of the root package's shortfall.** The facade's event loop receiving a tray
  command or finding the tray's channel closed is not reached; the commands themselves
  are tested through `handleTray`. Nor is `os.Executable` failing in `SetLaunchOnBoot`.

### It would change the machine

- **`installer` (0%).** The setup program's own Wails facade. Its methods write the
  uninstall registry key, create or remove shortcuts, write the login entry, close or
  launch the application or extract a payload into the user's programs directory; the
  few that do none of that read the machine or drive the Wails window. The logic
  underneath it, in `internal/infrastructure/setup`, is tested against a temporary
  tree. The facade calls that package directly rather than through a field, so there
  is nowhere to redirect its acts to.
- **The registry writes in `internal/infrastructure/setup` (76.3% overall).**
  `WriteUninstallEntry`, `RemoveUninstallEntry` and `SetLaunchOnBoot` write to
  `HKCU`. Unlike a filesystem path there is nothing to point them at, so exercising
  them would register or deregister a real install on the machine running the tests.
  What they write is tested instead: the uninstall entry's text comes from
  `uninstallValues` and the login entry's from `runValue`, both portable.
  The registry READS beside them are exercised, because a read cannot damage
  anything. `SetLaunchOnBoot` is covered on the facade side only in its refusal path:
  a copy running from the temporary directory, which is what a test binary is.
  The same package's shortcuts go through the Windows shell's COM object; COM refusing
  to start, the object refusing to be made and a property or save being refused are not
  reached, nor is packing a payload failing on a read or a write inside the archive.
- **Process control in `process_windows.go`.** Closing and launching the application.
  Deleting the install directory is tested against a temporary directory: a PowerShell
  process started beside it waits for a stand-in for setup to exit, then deletes it, including
  where setup was started inside the directory. The real install directory going once the
  real setup window has closed is not. Enumerating processes is tested: `processIDs` must find the test binary by
  its own name, which is the one process a test can be certain is running.
- **`tools/sounds` (44.9%).** The sounds tool. `run` rewrites `sounds.toml` in the repository
  and `python.Make` runs `sounds.py` in the tool's own venv, which a test machine need not have.
  Making every line in each accent from its spelling and matching the answers back to their
  lines is tested in `makeSounds` over a hand-written maker. Finding the venv's Python is
  `tools/internal/pyvenv`'s, tested there for both layouts. What the real tool wrote is checked
  by the structural test over `sounds.toml`.
- **`tools/pauses` (73.6%).** The pauses tool. `main` and `start` find the repository, check
  `models/`, make lines with the real model through ONNX Runtime and rewrite `pauses.toml` and
  `endings.toml`; `python.run` runs `pauses.py` (Praat through parselmouth) and `endings.py` in the
  tool's own venv, which a test machine need not have. Over hand-written makers and finders, the rest is
  tested: the digest of each line's samples, the WAV files handed to the finders, which lines are
  doubtful, which lines fade, which books a run writes, the
  flags and a run refused for a failing maker, finder or voice's files, a wrong answer or a changed
  model. Inside a run, a voice's temporary folder that cannot be made and a line that cannot be
  written are not reached; `writeWAV`'s refusal is tested on its own. Nor is the refusal of a line's
  numbers or style row, which lines from the voiced script do not give; both refusals are tested in
  `speech`. `measure` is tested writing both books; its passing on a refusal from finding or
  writing a book is not reached, nor are `writeBook`'s own refusals. What the real tool wrote
  is checked by the structural tests over `pauses.toml` and `endings.toml`.
- **`tools/models` (48.3%).** The model files tool. `main` and `start` read the real list, find
  `models/` in the repository and hand `run` a client that reaches the internet. `run` itself is tested
  in both modes over a local server, as `internal/infrastructure/modelfiles` is: no test downloads
  anything or writes outside a temporary folder. Checking the folder lives in `modelfiles.Verify`
  (shared with the payload tool) and is tested there.
- **`tools/payload` (53.3%).** The payload tool. `main` and `start` read the real list and find
  `models/` in the repository. `run` is tested over temporary folders: a packing, a models folder that
  does not match, an application folder without the application and a missing flag. A flag that
  does not parse is not reached.

### It could not happen, so it is gone

A branch nothing can reach is not a gap in the tests; it is dead code, so documenting
it would be technical debt wearing a description. Branches like that are removed
rather than excused. Four of them, in code that still exists:

| What it was | Why it could not run |
|---|---|
| the error return from `readAll` | it could only ever have been nil, so `readAll` now returns bytes alone |
| the empty check in `splitLines` | `bytes.Split` always yields at least one element |
| the encode failure in `Save` | a struct of four strings always marshals |
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
finds no stops at all. `hooks.test.tsx` states the shape of the page through
`testLayout.ts`, which makes attached elements report a parent, then asserts what the ring does with it. What is
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

To hold the domain and the application layers to a different floor, for a deliberate check:

```powershell
./test.ps1 -Floor 95
```

The tests that need the real model take minutes, so the gate runs them only when
asked; `build.ps1` always asks:

```powershell
./test.ps1 -Benchmarks
```

It vets and runs the files carrying the `benchmarks` build tag in `tests/machinevoice`
and `internal/infrastructure/speechmodel`, with a 15 minute timeout. One casts a voice and fails over NFR-P-205's
5 seconds or FR-514's 2 seconds. One makes the shipped script for one voice and fails
over NFR-C-502's 60 MB, skipping while the script lacks lines for any cue. One makes
150 lines while collections shrink goroutine stacks and fails on any line that panics,
fails or comes back empty. Its source records 13 lines in 300 breaking before every
address handed to ONNX Runtime was converted where the call is made.

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

The front end, from the `frontend` directory. The application's build runs the first two;
nothing runs the third for you:

```powershell
npx eslint .
```

```powershell
npx tsc --noEmit
```

```powershell
npx vitest run
```

The front-end coverage figures in this document. The report carries no threshold, so it
fails nothing:

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

- [DEVELOPMENT.md](DEVELOPMENT.md) for building and running on Windows.
- [ARCHITECTURE.md](ARCHITECTURE.md) for the invariants the structural tests enforce.
- [TECH_DEBT.md](TECH_DEBT.md) for what is open, what is deliberately left and what
  only looks like debt.
