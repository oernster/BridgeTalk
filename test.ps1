# Verifies Bridge Talk: formatting, vet, the Go suite, the front end's own checks and the
# coverage floors.
#
#   ./test.ps1              run everything
#   ./test.ps1 -Floor 95    run with a different coverage floor, for a deliberate check
#
# build.ps1 runs this before it builds, so a release cannot be cut from a tree that
# fails it. Run it directly while working.
#
# The floor is the measured number, not a target. Domain and application are at 100%
# today, so that is what it is set to. A floor picked from an aspiration only teaches
# people to lower it; a floor at the measured number fails the moment cover is lost,
# which is the only moment it is worth being told.
#   ./test.ps1 -Benchmarks  also run the tests that need the real model for minutes: the
#                           cast and the complete script (NFR-P-205, NFR-C-502) and lines made
#                           while stacks move; build.ps1 passes it and the everyday gate does
#                           not (Oliver, 2026-09-14)
param(
    [double]$Floor = 100,
    [switch]$Benchmarks
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

# The gate covers the layers that can be tested without a machine: pure decision
# logic with no filesystem, clock or audio device. Infrastructure needs a real audio
# device, a registry and the Windows shell, so it is tested where it can be and is
# deliberately outside the floor rather than dragging it down to a number that means
# nothing.
$gated = './internal/domain/...', './internal/application/...'

# The Go tools below are pointed at this list rather than at ./..., which reaches into
# frontend/node_modules: an npm dependency ships a Go package of its own down there.
# It is not ours, it is not committed and it is not built into anything this repository
# produces, so a future version of it failing vet or staticcheck would break a build
# over code nobody here wrote. The formatting check below filters the same tree by path
# because gofmt walks directories rather than packages.
$packages = go list ./... | Where-Object { $_ -notmatch '/node_modules/' }
if ($LASTEXITCODE -ne 0) { throw "go list failed with exit code $LASTEXITCODE" }

# The model files come first, because the tests that make lines with the real model skip where
# one is missing; a suite that skipped them would pass having proved nothing about the model
# (Oliver, 2026-09-14). The check downloads nothing; the message says how to fill the folder.
Write-Host 'Checking the model files...'
go run ./tools/models -check
if ($LASTEXITCODE -ne 0) { throw "the model files are not all in models/ as the list gives them: run go run ./tools/models, then run this again" }

Write-Host 'Checking formatting...'
$unformatted = gofmt -l . | Where-Object { $_ -notmatch '^frontend' }
if ($unformatted) { throw "gofmt reports unformatted files:`n$($unformatted -join "`n")" }

Write-Host 'Vetting...'
go vet $packages
if ($LASTEXITCODE -ne 0) { throw "go vet failed with exit code $LASTEXITCODE" }

# NFR-M-4. The version is pinned rather than taken as the latest: a checker that moved under the
# gate could fail a change that touched nothing it reads, on the day a new release came out. Raise
# it here on purpose, having read what the new version reports. The first run on a machine fetches
# it; every run after reads it from the module cache.
$staticcheckVersion = 'v0.8.1'
Write-Host "Running staticcheck $staticcheckVersion..."
go run "honnef.co/go/tools/cmd/staticcheck@$staticcheckVersion" $packages
if ($LASTEXITCODE -ne 0) { throw "staticcheck failed with exit code $LASTEXITCODE" }

Write-Host 'Running the whole suite...'
go test $packages
if ($LASTEXITCODE -ne 0) { throw "go test failed with exit code $LASTEXITCODE" }

# The front end is held to the same bar in its own runner, here rather than at build time
# alone. Its checks used to run only inside `wails build`, which meant a lint failure, a type error
# or a broken component test was caught by cutting a release rather than by the everyday gate; the
# type check in particular is what holds a rule the compiler enforces, that a call which can be
# refused cannot be written without a handler for it (ARCHITECTURE.md, Errors).
#
# The commands live in package.json rather than being spelled out here, so the build and this gate
# run the same three. A missing node_modules stops the gate rather than skipping the front end: a
# check that quietly does not run is the one that is not there on the day it would have caught
# something.
Write-Host 'Checking the front end...'
$frontend = Join-Path $root 'frontend'
if (-not (Test-Path (Join-Path $frontend 'node_modules'))) {
    throw "the front end's dependencies are not installed: run npm install in $frontend, then run this again"
}
Push-Location $frontend
try {
    foreach ($check in 'lint', 'typecheck', 'test') {
        npm run $check
        if ($LASTEXITCODE -ne 0) { throw "npm run $check failed with exit code $LASTEXITCODE" }
    }
} finally {
    Pop-Location
}

Write-Host "Measuring coverage of $($gated -join ', ')..."
$profilePath = Join-Path ([System.IO.Path]::GetTempPath()) 'bridge-talk-coverage.out'
try {
    # Each gated package is measured against its own statements. An earlier version
    # passed -coverpkg as well, which go answered with "matched no packages" and then
    # ignored: a flag that silently does nothing is worse than no flag, because the
    # number it produces looks like the one that was asked for.
    go test "-coverprofile=$profilePath" @gated
    if ($LASTEXITCODE -ne 0) { throw "the coverage run failed with exit code $LASTEXITCODE" }

    $summary = go tool cover "-func=$profilePath"
    if ($LASTEXITCODE -ne 0) { throw "go tool cover failed with exit code $LASTEXITCODE" }

    # The last line of the report is the total across the merged profile. Read the
    # exit code and this line, never the run's own output.
    $total = ($summary | Select-Object -Last 1)
    if ($total -notmatch '([0-9]+(?:\.[0-9]+)?)%\s*$') { throw "could not read a total from: $total" }
    $percent = [double]$Matches[1]

    $below = $summary | Where-Object { $_ -notmatch '100\.0%\s*$' -and $_ -notmatch '^total:' }
    if ($percent -lt $Floor) {
        Write-Host 'Not covered:'
        $below | ForEach-Object { Write-Host "  $_" }
        throw "coverage is $percent%, below the floor of $Floor%"
    }
    Write-Host "Coverage $percent%, floor $Floor%."
} finally {
    if (Test-Path $profilePath) { Remove-Item $profilePath -Force }
}

# The rest of the tree, each package held at the number it actually reaches.
#
# These are floors picked from a measurement, never from an aspiration. An infrastructure
# package held at 100% reaches it because everything in it can be exercised over a
# temporary directory. Of those held below it, some need a machine to go further: audio
# needs an output device, taskbar needs the Windows shell, setup needs to write to the
# real registry and the root package needs Wails. What CAN be tested in each of them is;
# these numbers are what that came to. No count is given here because the list below is
# the count; a number written beside it goes stale the first time a floor moves.
#
# A floor fails the moment the cover behind it is lost, which is the only moment it is
# worth being told. Raise a number here when that cover genuinely rises. A figure also
# falls when statements no test can reach are added beside the ones that are covered,
# which is not cover lost: `go tool cover -func` is what tells the two apart, so read it
# before moving a number down and say in TESTING.md which functions the shortfall is in.
#
# TESTING.md names what each shortfall is and why it is where the line falls.
$measured = [ordered]@{
    '.'                                   = 82
    './internal/infrastructure/audio'     = 95
    './internal/infrastructure/audio/audiotest' = 86
    './internal/infrastructure/config'    = 100
    './internal/infrastructure/appdata'   = 100
    './internal/infrastructure/journal'   = 100
    './internal/infrastructure/library'   = 100
    './internal/infrastructure/madelines' = 100
    './internal/infrastructure/modelfiles' = 99
    './internal/infrastructure/plugin'     = 91
    './internal/infrastructure/plugin/plugintest' = 100
    './internal/infrastructure/reporoot'  = 100
    './internal/infrastructure/runlog'    = 48
    './internal/infrastructure/setup'     = 79
    './internal/infrastructure/speechmodel' = 91
    './internal/infrastructure/status'    = 100
    './internal/infrastructure/taskbar'   = 67
    './internal/infrastructure/tomlfile'  = 100
    './internal/infrastructure/voicefiles' = 100
    './internal/infrastructure/wholefile' = 100
    './internal/refusal'                  = 100
    './tools/sounds'                      = 44
    './tools/pauses'                      = 73
    './tools/models'                      = 48
    './tools/payload'                     = 53
    './tools/internal/pyvenv'             = 100
}

Write-Host 'Measuring the rest of the tree...'
foreach ($package in $measured.Keys) {
    $floor = $measured[$package]
    $reported = go test -cover $package
    if ($LASTEXITCODE -ne 0) { throw "$package failed with exit code $LASTEXITCODE" }

    $line = $reported | Where-Object { $_ -match 'coverage: ' } | Select-Object -First 1
    if ($line -notmatch 'coverage: ([0-9]+(?:\.[0-9]+)?)%') {
        throw "could not read a coverage figure for ${package}: $line"
    }
    $reached = [double]$Matches[1]
    if ($reached -lt $floor) {
        throw "$package is at $reached%, below its floor of $floor%"
    }
    Write-Host ("  {0,-38} {1,5}%  floor {2}%" -f $package, $reached, $floor)
}

# Not gated at all, deliberately: internal/infrastructure/window is Win32 focus
# handling; installer is the setup program's Wails facade over acts that change the
# machine. Neither has anything a test can reach without the platform behind it, so a
# floor over either would be a floor at zero, which asserts nothing. TESTING.md says so
# in full rather than leaving the absence to be read as an oversight.
# internal/infrastructure/modelfiles/modelfilestest is test support with no tests of its
# own: the modelfiles and tools/models tests run every part of it.

if ($Benchmarks) {
    # The benchmark files carry the benchmarks build tag, which the vet and the suite above leave
    # out, so they are vetted here as well as run. Making the complete script took 3 m 18 s on
    # 2026-09-14; the timeout leaves it room several times over.
    Write-Host 'Running the tests that need the real model...'
    go vet -tags benchmarks ./tests/machinevoice/ ./internal/infrastructure/speechmodel/
    if ($LASTEXITCODE -ne 0) { throw "go vet of the benchmarks failed with exit code $LASTEXITCODE" }
    go test -tags benchmarks -count=1 -timeout 15m -v ./tests/machinevoice/ ./internal/infrastructure/speechmodel/
    if ($LASTEXITCODE -ne 0) { throw "the benchmarks failed with exit code $LASTEXITCODE" }
}

Write-Host 'All green.'
