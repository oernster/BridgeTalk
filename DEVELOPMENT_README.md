# Development

How to build and run Bridge Talk on Windows, from a machine with nothing
installed to a setup program.

Every command here is PowerShell, one command per block, meant to be pasted as it is.
`README.md` is for somebody using the application; this is for somebody building it.

## What the machine needs

Four things. Each check below prints a version if the tool is on the path, so run them
all before starting: a missing one fails the build several minutes in rather than at
the start.

| Tool | Version | Why |
|---|---|---|
| Go | 1.26.3, which `go.mod` requires | the backend and both Wails applications |
| Node.js | 24.11.1 on the machine this was written on | the React front end and its build |
| Wails CLI | v2.12.0, which `go.mod` requires | packages the Go binary and the web assets into one executable |
| WebView2 runtime | any current | the window the front end is drawn in |

Python 3 with Pillow is optional. It is needed only to regenerate the icons, which are
committed already; nothing in the ordinary build path uses it.

### Go

Install it from [go.dev/dl](https://go.dev/dl/), then check it:

```powershell
go version
```

The module declares `go 1.26.3`.

### Node.js

Install the LTS release from [nodejs.org](https://nodejs.org/) or with winget:

```powershell
winget install OpenJS.NodeJS.LTS
```

```powershell
node --version
```

```powershell
npm --version
```

### The Wails CLI

Wails is a Go program, so it is installed with Go. Install the version the module
requires:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
```

That puts `wails.exe` in `%USERPROFILE%\go\bin`. If the next command is not found,
that directory is not on the path:

```powershell
wails version
```

Add it for the current session with:

```powershell
$env:PATH = "$env:USERPROFILE\go\bin;$env:PATH"
```

Add it permanently through Settings, then System, then About, then Advanced system
settings, then Environment Variables, by appending `%USERPROFILE%\go\bin` to the user
`Path`.

### The WebView2 runtime

Where it is missing, install the Evergreen runtime from Microsoft's
[WebView2 page](https://developer.microsoft.com/microsoft-edge/webview2/).

`wails doctor` reports on all of the above at once:

```powershell
wails doctor
```

## Getting the source

The repository is private. Clone it into your own working directory, not into a
worktree or a second copy:

```powershell
git clone https://github.com/oernster/BridgeTalk.git
```

```powershell
cd BridgeTalk
```

Fetch the Go modules and the front-end packages once:

```powershell
go mod download
```

```powershell
npm --prefix frontend install
```

`wails build` runs `npm install` itself through the `frontend:install` hook in
`wails.json`, so the second command is only needed before running the front-end
tools directly.

## Building

One command builds everything:

```powershell
./build.ps1
```

It does five things in order and stops at the first failure:

1. Reads the version from `VERSION`, then pins `CGO_ENABLED=0` for everything that
   follows, so no machine's default decides how the binary is linked.
2. Runs `test.ps1`. There is no switch to skip it: a gate that can be skipped is a
   gate that is skipped on the day it would have caught something.
3. Refuses to go on without `assets/application-icon.png` and its `.ico`, then copies
   both into the application's and the setup program's build trees.
4. Runs `wails build` for the application. That runs the front end's `npm run build`,
   which runs `eslint` and `tsc --noEmit` before bundling, so a lint or type error
   stops the build here.
5. Zips the result as the setup program's payload, builds the setup program with the
   version passed in through `-ldflags`, then collects it.

| Output | What it is |
|---|---|
| `build/bin/BridgeTalk.exe` | the application |
| `dist-installer/BridgeTalkSetup.exe` | the setup program, application included |

To stop after the application and skip the setup program, which is the faster loop
when only the application has changed:

```powershell
./build.ps1 -SkipInstaller
```

### A note on the payload

The setup program carries the application as an embedded zip at
`installer/payload.zip`. `build.ps1` fills it, builds, then writes an empty zip back
over it, so `go build ./...` and the tests keep working without a full build and a
multi-megabyte payload never reaches a commit. If a build is interrupted between
those steps, run `./build.ps1` again rather than committing what is there.

### A note on `-ldflags`

The version reaches the setup program through `-X main.appVersion=$version`. `-X`
only writes to a **var**; against a **const** it silently does nothing. `appVersion`
is a var for that reason, holding `dev` until the flag replaces it, so a setup program
reporting `dev` was built without the flag. The application itself reads `VERSION`
through an embed rather than a flag.

### A note on cgo

`build.ps1` sets `CGO_ENABLED=0` before it runs either the gate or the build, so the
tests exercise the configuration that ships. Nothing here needs a C toolchain: the
audio path decodes and plays in pure Go. The pin is there so that a machine which does
have one cannot quietly produce a different binary.

## Running it while working

The development loop, with the front end hot-reloading:

```powershell
wails dev
```

That serves the React front end from Vite and rebuilds the Go side on change.

The reporting flags can be run from a plain Go build. `main.go` embeds
`frontend/dist`, which is build output and is not committed, so build the front end
first:

```powershell
npm --prefix frontend run build
```

```powershell
go build -o bridge-talk.exe .
```

```powershell
./bridge-talk.exe -list
```

A flag given on the command line wins over the choice stored from the window, for that
run only.

| Flag | What it does |
|---|---|
| `-library <dir>` | the recordings directory |
| `-journal <dir>` | the journal directory |
| `-voice <name>` | the voice to cast; a name that is not installed falls back to the first voice with a warning |
| `-list` | prints the voices found with their takes and moment coverage, then exits |
| `-unbound` | prints the moments the chosen voice cannot serve, then exits |
| `-no-tray` | runs without a notification-area icon, so closing the window quits |
| `-hidden` | starts in the notification area with no window, as the login entry does; ignored with `-no-tray` |

`-list` and `-unbound` open no window and refuse when no voice is found. With
`-unbound`, a `-voice` that is not installed is refused rather than replaced.

## Verifying

The backend gate, which `build.ps1` runs for you:

```powershell
./test.ps1
```

The stricter Go analysis, which neither script runs:

```powershell
go run honnef.co/go/tools/cmd/staticcheck@latest (go list ./... | Where-Object { $_ -notmatch '/node_modules/' })
```

The package list is narrowed rather than written as `./...`, which reaches into
`frontend/node_modules`, where an npm dependency ships a Go package of its own. It is
nobody here's code and nothing this repository produces contains it, so a future
version of it failing an analyser would break a build over something unowned.
`test.ps1` narrows the same way for `go vet` and `go test`; the formatting check
filters by path instead, because gofmt walks directories rather than packages.

The front end, from the `frontend` directory. The build runs the first two; nothing
runs the third for you:

```powershell
npx eslint .
```

```powershell
npx tsc --noEmit
```

```powershell
npx vitest run
```

Front-end coverage, when a figure needs checking. The report carries no threshold, so
it fails nothing:

```powershell
npx vitest run --coverage
```

Read the exit code of each rather than the last line of its output.

**[TESTING.md](TESTING.md) is the full account**: what every figure is, what is
deliberately not tested and why.

## Installing what you built

Run the setup program:

```powershell
./dist-installer/BridgeTalkSetup.exe
```

Everything it writes is per user, so Windows never asks for administrator rights: the
files under `%LOCALAPPDATA%\Programs\BridgeTalk` with a copy of itself there as
`uninstall.exe` plus the install record under `HKEY_CURRENT_USER`. Where you ask for
them it also writes the Start Menu entry under `%APPDATA%`, the Desktop shortcut on your
own Desktop and the login entry under `HKEY_CURRENT_USER`. With nothing installed it offers an install. Over an
older or a newer version it offers the change on one screen. Over the same version it
opens a manage screen with Repair, Reinstall and Uninstall.

Neither executable is signed: `build.ps1` has no signing step.

To uninstall, use the Apps list. The same program is the `uninstall.exe` in the install
directory. Started with `-uninstall`, as the Apps list starts it, that copy opens on
the removal screen; started bare it opens on the manage screen, which offers Uninstall.

## Regenerating the icons

Only needed after changing an image in `assets/`; the results are committed:

```powershell
python tools/genicons.py
```

That writes the nav band icons into `frontend/src/assets/icons`, composes the muted
speaker from the sounding one and a slash, turns `assets/application-icon.png` into the
`.ico` beside it plus the copy the About dialog shows, then writes the setup page's
header mark and its two theme icons. The `.ico` is committed rather than left for
Wails to derive at build time: Wails only derives one when the file is absent, so
relying on that would mean deleting and hoping.

## Where things live

| Path | What it holds |
|---|---|
| `main.go`, `app.go` | the composition root and the Wails facade |
| `audition.go`, `cast.go`, `checklist.go`, `folders.go`, `settings.go`, `voices.go`, `window_life.go` | the rest of the facade, one pane or concern per file |
| `dto.go`, `identity.go` | the shapes the front end reads, plus the version, credits and licence the About dialog shows |
| `internal/domain` | the cue model, events and selection; no I/O at all |
| `internal/application` | the reaction and scheduling services, over ports |
| `internal/infrastructure` | journal, status, library, audio, config, setup, taskbar, window |
| `internal/product` | the product's name and slug, in one place |
| `internal/refusal` | the wording of a file-system refusal, so each one names its path once |
| `frontend/src` | the React front end |
| `installer/` | the setup program, a Wails application of its own |
| `tests/structural` | the tests that hold the architecture in place |
| `tools/` | icon generation, run by hand |

`ARCHITECTURE.md` explains the layering, the dependency direction and the reasoning
behind each decision; it lists every structural test against the rule it enforces.

## House rules worth knowing before a first change

- **The layer direction is enforced, not suggested.** `internal/domain` imports nothing
  outside the domain, no IO package and reads no clock. `internal/application` may not
  import infrastructure. Only `main.go` and `app.go` wire the two together; a
  structural test holds that whitelist.
- **No file over 400 lines**: Go, the front end's TypeScript and CSS and the setup page,
  tests included. A file refactored down from over the cap lands at 350 or fewer rather
  than at 399; a file in the 381 to 400 band fails the gate. Build scripts are outside
  the rule.
- **No magic numbers.** A literal that needs a comment to say what it represents
  should be a named constant or derived from data.
- **The product is named once**, in `internal/product/product.go`, then read from
  there. A structural test fails when a Go string literal, a front-end source file or a
  setup page file spells it, test fixtures included. The Wails configuration files,
  `frontend/index.html` and the build scripts sit outside that test and still carry it.
- **The version lives in `VERSION`.** The application embeds it; the setup program
  receives it at build time.
- **Every new guard is proved by planting a violation** and reading the exit code,
  with the plant restored in a `finally`. A guard that has never been seen to fail is
  not yet a guard.

## See also

- [README.md](README.md) for what the application is and how to use it.
- [TESTING.md](TESTING.md) for the test suite in full.
- [ARCHITECTURE.md](ARCHITECTURE.md) for the invariants and the design decisions.
- [TECH_DEBT.md](TECH_DEBT.md) for what is open and what only looks like debt.
