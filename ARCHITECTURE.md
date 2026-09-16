# Bridge Talk Architecture

A background application that watches Elite Dangerous journal and status data, decides that something
worth speaking about has happened, then plays a matching recording from a library the user supplies. It
is one-directional. There is no microphone, no speech recognition and no command and control; the
application talks to the commander and never listens.

It ships no recordings. The application makes no request of its own. The Go source that imports a
network package is the model files download alone: `internal/infrastructure/modelfiles`, its test
support `modelfilestest` and the `tools/models` command. The application imports none of them;
`net/http` reaches it through Wails alone (`go list -deps .`, 2026-09-15). The front end makes no request.

## Invariant

`UI -> Application -> Domain <- Infrastructure`

Dependencies point inward. The Domain is the stable core and depends on nothing. Every rule below is
enforced by a test under `tests/structural`, not by convention. The table is the whole set: a guard
that is not listed here is undiscoverable, so a rule claimed in prose and enforced nowhere reads
exactly like one that holds.

| Invariant | Enforcing test | File |
|---|---|---|
| Domain imports nothing from this module outside `internal/domain` | `TestDomainHasNoOutwardImports` | `boundary_test.go` |
| Domain is pure: no network, filesystem, process or database package; no wall clock or global random source | `TestDomainIsPure` | `boundary_test.go` |
| Application never imports infrastructure or wails | `TestApplicationDoesNotImportInfrastructure` | `boundary_test.go` |
| Only the composition root wires the application services to infrastructure | `TestCompositionRootIsWhitelisted` | `boundary_test.go` |
| No source file exceeds the 400-line limit: the Go, the front end's TypeScript and CSS, the setup page | `TestNoFileExceedsLineLimit` | `boundary_test.go` |
| No source file sits in the danger band of 381 to 400 lines | `TestNoFileInDangerBand` | `boundary_test.go` |
| A file's lines are counted as an editor numbers them, so the newline ending a file adds none | `TestLineCountCountsTheLinesAnEditorShows` | `linecount_test.go` |
| Every exported type carries a doc comment | `TestEveryExportedTypeIsDocumented` | `boundary_test.go` |
| No colour value appears in `frontend/src` outside the theme token file | `TestColoursOnlyInTokens` | `colours_test.go` |
| The secondary lines, the Missing takes purpose line and the Status cards' taglines, share one rule whose colour reads at 7 to 1 or better against the surface and panel grounds in both themes | `TestTheSecondaryLinesContrastInBothThemes` | `contrast_test.go` |
| The thumb of every Chatter switch and the track of a switch while on read at 3 to 1 or better against the surface and panel grounds in both themes | `TestTheChatterSwitchesContrastInBothThemes` | `contrast_test.go` |
| The product is named in one Go file; no Go string literal, front-end source or setup page file spells it | `TestTheProductIsNamedOnce` | `identity_test.go` |
| Both forms of the identity survive being a file name | `TestTheIdentityCanBeAFileName` | `identity_test.go` |
| Every disabled control wears the danger ring at all times | `TestEveryDisabledControlWearsTheDangerRing` | `rings_test.go` |
| Every region that is a keyboard stop because it scrolls wears a focus ring | `TestEveryScrollingRegionRingsForTheKeyboard` | `rings_test.go` |
| No list wears a ring in any state | `TestNoListWearsARing` | `noborder_test.go` |
| No scrolling region wears a ring under the pointer | `TestNoScrollingRegionRingsUnderThePointer` | `noborder_test.go` |
| No container wears a ring | `TestNoContainerWearsARing` | `noborder_test.go` |
| Only a control, a list or a scrolling region is a keyboard stop | `TestOnlyControlsListsAndScrollingRegionsAreStops` | `noborder_test.go` |
| The shipped script and its saved speech sounds break none of the script's rules; every problem is named with its cue and its line | `TestTheShippedScriptHoldsNoProblem` | `script_test.go` |
| Every cue in the table has lines in the script | `TestTheScriptHoldsLinesForEveryCue` | `script_test.go` |
| `pauses.toml` is not stale: every machine voice has a pause for each line the script joins and none for a line it no longer joins, each found in the line's saved speech sounds now with the model and style files the list gives; every problem is named | `TestTheShippedPausesAreNotStale` | `pauses_test.go` |
| The pauses check passes a book found for every joined line of the shipped script with the listed files | `TestPausesFoundForTheShippedScriptWithTheListedFilesPass` | `pauses_test.go` |
| The pauses check names a joined line that has no pause by its voice, its cue and its text | `TestAShippedJoinedLineWithNoPauseIsNamed` | `pauses_test.go` |
| The pauses check names a pause found in speech sounds other than the line's own | `TestAShippedPauseFoundInOtherSoundsIsNamed` | `pauses_test.go` |
| The pauses check names a model other than the listed one with both digests; it names a style file other than the listed one with its voice | `TestShippedPausesFoundWithOtherFilesAreNamed` | `pauses_test.go` |
| `endings.toml` is not stale: every machine voice has an ending for each line ending on a nasal in its accent and none for a line that no longer does, each found in the line's saved speech sounds now with the model and style files the list gives; every problem is named | `TestTheShippedEndingsAreNotStale` | `endings_test.go` |
| The endings check passes a book found for every line of the shipped script ending on a nasal with the listed files | `TestEndingsFoundForTheShippedScriptWithTheListedFilesPass` | `endings_test.go` |
| The endings check names a line ending on a nasal that has no ending by its voice, its cue and its text | `TestAShippedLineEndingOnANasalWithNoEndingIsNamed` | `endings_test.go` |
| The endings check names an ending found in speech sounds other than the line's own | `TestAShippedEndingFoundInOtherSoundsIsNamed` | `endings_test.go` |
| The endings check names a model other than the listed one with both digests; it names a style file other than the listed one with its voice | `TestShippedEndingsFoundWithOtherFilesAreNamed` | `endings_test.go` |
| The setup page applies the boxes it shows and handles the failure of every box that saves at once | `TestSetupAppliesTheBoxesItShows` | `setupchoices_test.go` |
| The setup page header repeats no title beneath the title bar | `TestTheSetupHeaderRepeatsNoTitle` | `setupheader_test.go` |
| The setup page, the scripts beside it and `setupScripts` stay in step: the page loads every script and every script has a place in the list | `TestTheSetupPageLoadsEveryScript` | `setupring_test.go` |
| The setup page's body, a keyboard stop because it scrolls, wears a focus ring | `TestTheSetupBodyRingsForTheKeyboard` | `setupring_test.go` |
| The Status cards widen to share their row: their grid fits its columns to the cards | `TestTheStatusCardsWidenToShareTheRow` | `strip_test.go` |
| The strip is three quarters of the band's height, derived from the sizes the band's own rules draw it with | `TestTheStripIsAShareOfTheBandDrawnFromItsOwnSizes` | `strip_test.go` |
| The strip's labels open above their controls, the left-most from its own left edge | `TestTheStripsLabelsOpenAboveTheirControls` | `strip_test.go` |
| The strip's style part is read after the band's it derives from | `TestTheStripIsReadAfterTheBand` | `strip_test.go` |
| Every tone the live indicator takes is drawn in the colour token of its name | `TestEveryIndicatorToneHasItsColourToken` | `strip_test.go` |
| Every style part is listed in the manifest that reads them | `TestEveryStylePartIsRead` | `styles_test.go` |
| The front end reaches only the methods declared as bound | `TestTheBoundSurfaceIsDeclared` | `surface_test.go` |
| An address handed to a DLL becomes a uintptr only in the argument list of the call into it; no function takes `...uintptr` | `TestAddressesAreConvertedOnlyWhereTheCallIsMade` | `syscall_test.go` |
| The speech sound table matches the model's tokenizer file: every symbol at the same number, the boundary as its marker and nothing more | `TestTheSymbolTableIsTheModelsOwn` | `tokenizer_test.go` |
| Cue ids, journal events and status values stay in the cue table | `TestGameVocabularyStaysInItsHome` | `vocabulary_test.go` |
| No cue id ends in a segment of digits, which the flat form reads as a take number | `TestNoCueIdEndsInDigits` | `vocabulary_test.go` |
| No cue id ends in a dot or a space, which Windows strips from a name | `TestNoCueIdEndsInADotOrASpace` | `vocabulary_test.go` |
| No cue id holds an underscore, which a cue folder writes for a dot | `TestNoCueIdHoldsAnUnderscore` | `vocabulary_test.go` |
| The wire is stated identically in the DTOs and in `api.ts` | `TestTheWireContractMatchesOnBothSides` | `wire_test.go` |

## Layers

- **Domain** (`internal/domain`: `cue`, `ending`, `event`, `machinevoice`, `making`, `measured`, `pause`, `script`, `selection`, `speech`): pure Go. Values are validated on
  construction. No IO and no wall-clock reads: time arrives as a parameter, as the `now` taken by
  `CooldownGate.Open` and `DedupeWindow.Fresh`, while randomness arrives through the injected
  `selection.Chooser`. Cue matching, take selection and the cooldown and dedupe arithmetic live here,
  so all of it is testable without a filesystem, a clock or an audio device. `event` holds the event
  every source emits, tagged with its source: journal, status or application. `machinevoice` holds the
  28 machine voices offered, the accent each speaks with and the name each is shown by; the list is the
  one home of the model's voice ids. `speech` holds the speech sound symbols the model reads with their
  numbers, copied from its tokenizer file by a script; it also reads the spellings a script line gives
  and chooses the row of a voice's style file the model reads beside a line. `speech.Words` is the
  script's table of words, whose speech sounds are given once for every line holding them (FR-549).
  `script` checks the script against the cue table, naming the cue and the line behind every problem;
  it also joins a word the script's `[joins]` table names to the word before it where a line ends with
  a comma then that word (FR-550), with `Joined` listing those lines; `EndingOnNasal` lists the lines
  whose last speech sound is a nasal, which `speech.EndsOnNasal` reads (FR-555).
  `script/scripttest` builds a voiced script for the tests that need one.
  `making` keys each made line by its speech sounds, style file, model and any pause or fade it gets,
  then works out which lines a voice still has to make.
  `measured` holds what the pauses and the endings share: the book of every voice's entries, each tied
  to the speech sounds and the samples it was measured in, what such a book refuses and the check that
  the shipped entries are not stale, each worded by its kind. `pause` holds the pause before a final
  commander: the digest tying a pause to the samples it was found in, inserting its silence and the
  rule that finds a break doubtful (FR-551 to FR-554). `ending` holds the hiss the model adds after a
  final nasal: where each line fades and the fade itself (FR-555 to FR-557).
- **Application** (`internal/application`: `ports`, `services`): the reaction, scheduling, making and Chatter services plus the ports they
  depend on (`EventSource`, `AudioPlayer`, `VoiceCatalogue`, `AudioSource`, `Clock`, `SettingsStore`,
  `Reporter`, `Switchboard`). `Switchboard` answers whether a moment is switched off, which the
  reaction service and the scheduler ask at each decision rather than holding a copy (FR-622, FR-627). `AudioSource` answers the takes for a cue id and nothing else, so the catalogue serves
  any kind of voice without knowing where its audio came from (FR-501, FR-502). A machine voice is made
  through three more: `SpeechMaker` loads the model and turns a line's numbers and style row into samples, `VoiceFiles`
  reads the files a voice is made from, refusing one that is missing (FR-519); `MadeLines` keeps the
  made lines (FR-517, FR-523, FR-527). A line the endings give a sample is faded over 30 ms from it,
  then a line the pauses give a sample has 40 ms of silence inserted at it, each only where the digest
  of the samples as made equals the one saved; where a digest differs, that change is left out and the
  line is logged through `RunLog` (FR-553, FR-556). A cast makes only its confirmation's lines (FR-511); every
  other line is made the first time its cue fires. The audio source a cast answers with is also a `CueMaker`, through which the
  reaction service has a cue's lines made next when it fires with none; the cue waits for them for up
  to 2 seconds, handed over on the poll tick (FR-514). It never imports Infrastructure or the Wails
  runtime. `services/makingtest` holds hand-written fakes of the making ports, so every suite that
  makes lines over fakes builds them the same way.
- **Infrastructure** (`internal/infrastructure`): concrete adapters behind those ports. The journal tail
  reader (`journal`), the status-flag watcher (`status`), the voice library scanner, catalogue and
  folder maker (`library`), the audio engine (`audio`), the cue table, the script with its saved speech sounds, the pauses, the endings and the settings store (`config`),
  the strict reading both TOML files share (`tomlfile`), the files a machine voice is made from
  (`voicefiles`), the made lines kept as 16-bit FLAC (`madelines`), the product's local data folder
  both it and the default recordings directory sit in (`appdata`), the log each run leaves in that
  folder with the `RunLog` written to the run's error output (`runlog`),
  the Windows tray (`taskbar`), keyboard focus for the web view plus opening a folder in File Explorer
  (`window`), the model run through ONNX Runtime's C API with cgo disabled (`speechmodel`), the one rule
  for putting a file in place whole or not at all, which the made lines, the stored settings, the model
  files and the payload archive are written through (`wholefile`) and the per-user install work behind
  the setup program (`setup`). `audio/audiotest` lays out the smallest playable take in each format for
  the tests. Never imported by Domain or Application.
- **Development support** (`internal/infrastructure/modelfiles`, `internal/infrastructure/reporoot`):
  `modelfiles` holds the pinned list of model files with the download that fills a folder from it and
  the check that downloads nothing; `modelfilestest` is its test support. `reporoot` finds the
  repository root by walking up to `go.mod`. The tools and the structural tests read them; the
  application imports neither (`go list -deps .`).
- **UI**: the React front end plus a Wails facade in package `main`, which calls the Application
  services and maps what they return into the shapes in `dto.go`.
- **Outside the layers**: `internal/product` holds the product's name and `internal/refusal` words a
  refusal over a path. Each is a leaf that several layers read, so it belongs to none of them.
- **The sounds tool** (`tools/sounds`): a command run while developing, never shipped. It reads the
  script through `config` and the `script` and `machinevoice` packages, asks misaki in the tool's own Python venv for every line's
  speech sounds in each accent with the table of words spelled in (FR-549) and writes `sounds.toml`.
  A line the `[joins]` table joins is saved with the comma before its final word left out (FR-550).
  Python only turns text into speech sounds;
  the spelling rules and the file's shape stay in Go, so each keeps one home.
- **The pauses tool** (`tools/pauses`): a command run while developing, never shipped. For each
  machine voice it makes every line the script joins with the model in `models/`, digests the samples
  and has `pauses.py`, run in the tool's own Python venv, find the break before commander with Praat;
  it then writes `pauses.toml` and prints each voice's doubtful lines (FR-551, FR-552). In the same
  run it makes every line ending on a nasal, has `endings.py` find where each ends on a burst and
  writes `endings.toml`, printing each voice's faded lines (FR-555). Python only reads the sound; every
  setting, the doubtful rule and the files' shapes stay in Go. Making, digesting and writing the lines
  a finder is handed is one step both finders share. `-only` names some voices and needs `-out` and
  `-endings`, since books missing voices are never written over the shipped files. `-endings-only` finds
  and writes the endings alone, leaving `pauses.toml` untouched.
  Both tools find their venv's Python through `tools/internal/pyvenv`.
- **The models tool** (`tools/models`): a command run while developing, never shipped. It fills
  `models/` at the repository root from the list in `modelfiles`; with `-check` it downloads nothing
  and fails naming each file missing or different.
- **The payload tool** (`tools/payload`): run by `build.ps1`. It checks `models/` against the list,
  then packs the built application with every model file setup installs into `installer/payload.zip`
  (see The setup program).
- **The scripts beside them**: `tools/genicons.py` writes the icons (see Icons under UI);
  `tools/gensocialcard.py` writes the site's link card, `docs/social-card.png`, reading its words from
  the page, its colours from the site's stylesheet and its picture from the icon's master; it is not
  part of the build and its output is committed;
  `stamp_version.py`, which `build.ps1` runs first, stamps the version from `VERSION` into the site's
  delimited tokens.

## Composition root

`main.go` is the composition root. It constructs the infrastructure adapters, injects them into the
application services by constructor and hands the assembled facade to Wails. `newMaking` there builds
the making service over the model files beside the executable and the made lines' folder; where
either cannot be found, every machine voice is refused with why rather than the application refusing
to start. It also loads the script with its saved speech sounds, `pauses.toml` and `endings.toml`, any of which
failing to load stops the run, then hands the service `runlog.Lines` over the run's error output as its
`RunLog`. No service is held in a
package-level variable and there is no service locator or auto-wiring. The structural test whitelists
`main.go` and `app.go`: no other file may import both the application services and infrastructure. The
facade is spread over the root files beside them, `settings.go`, `cast.go`, `machine.go`, `folders.go`, `checklist.go`, `audition.go`, `audition_machine.go`,
`chatter.go`, `donate.go`, `journaldir.go`, `reactions.go`, `runlog.go`, `voices.go`, `identity.go` and `window_life.go`, each a slice of the surface it would otherwise outgrow the size limit
carrying; the wire shapes are in `dto.go`. `window.go` holds the window `run` launches: its assets,
its geometry and `launch`.

## Dependency direction

```
             +-------------------+
   Wails/UI  |   app.go (facade) |
             +---------+---------+
                       | calls
             +---------v---------+
             |    application    |  ports (interfaces) + services
             +----+---------+----+
        depends on |         ^ implements
             +-----v----+    |
             |  domain  |    |
             +----------+    |
                       +-----+-----------------------+
                       |      infrastructure         |
                       | journal, status, library,   |
                       | audio, config, tomlfile,    |
                       | voicefiles, madelines,      |
                       | speechmodel, appdata,       |
                       | runlog, wholefile, taskbar, |
                       | window, setup               |
                       +-----------------------------+
```

## The cue model

Game semantics and a person's recordings are separate axes, so they are kept apart. `FSDJump` is a
fact about Elite Dangerous and is true for every voice; which file answers it is a fact about one
person's library. The two meet at the cue id and nowhere else.

**The cue table, `internal/infrastructure/config/cues.toml`, embedded in the binary.** Each entry names
a source, the journal event or status flag it listens for, an optional edge, an optional predicate over
the payload or a key stem or key beginnings in its place (below), a priority and a cooldown in seconds.
A cue the game raises also names the category Chatter lists it under. A missing priority reads as
`ambient`; a missing cooldown leaves the cue limited by the dedupe window alone. The table lists its
twelve categories first, as `[[category]]` entries in the order Chatter shows them.

```toml
[[cue]]
id = "Cast.Confirmed"
purpose = "When you cast this voice, to confirm it is now the one speaking."
source = "application"
event = "cast"
priority = "notice"

[[cue]]
id = "StartJump.JumpType.Hyperspace"
purpose = "When a hyperspace jump to another system begins."
source = "journal"
event = "StartJump"
match = { JumpType = "Hyperspace" }
priority = "ambient"
cooldown = 30

[[cue]]
id = "LightsOn.Cleared"
purpose = "When the ship's lights are switched off."
source = "status"
flag = "LightsOn"
edge = "falling"
priority = "ambient"
```

**Every id is spelled in the game's own words.** A journal cue's id is the event name, followed by a
field and a value where one payload field narrows it. A status cue's id is the flag name followed by
`Set` or `Cleared`; a status value's id is its name followed by what it became (`GuiFocus.GalaxyMap`),
except the fire group, whose one cue is `FireGroup.Changed`. The one cue with no name from the game is
the application's own `Cast.Confirmed`. The first segment is therefore the moment the cue listens for.
The audition pane groups by it rather than keeping a taxonomy of its own in step. Chatter needs a
grouping a player reads by subject instead, which no part of an id gives, so every cue the game raises
names its category in the table beside its purpose, from the set the table lists in order (FR-634,
FR-635). A list of cues reads neither: the Missing takes pane and the Moments spoken for dialog show each cue
under its full title alone (FR-233), since a heading read from the first segment would repeat the start
of every title beneath it.

**Comms moments (FR-617 to FR-620).** A `ReceiveText` message the game sends carries a key beside the
words it generated, such as `$Pirate_OnDeclarePiracyAttack07;`. One moment arrives under many keys that
differ only in a variant number, some with values after it, so a comms moment names the key stem they
share rather than any one key, with no match field beside it:

```toml
[[cue]]
id = "ReceiveText.Pirate.OnDeclarePiracyAttack"
purpose = "When a pirate declares it is attacking you for your cargo."
source = "journal"
event = "ReceiveText"
stem = { Message = "Pirate_OnDeclarePiracyAttack" }
priority = "alert"
```

`cue.KeyStem` reads a stem out of a key by dropping the leading `$`, any values from the first `:#`,
the closing `;` and the variant digits; a message with no leading `$` is text a player typed and
reaches no comms moment (FR-618). The id is the event's name followed by the stem with each underscore
written as a dot, so it holds no underscore (FR-230) and stays in the game's own words (FR-619). What
is heard is the cast voice's take for the moment; the message's own words are never spoken.

**Station traffic (FR-637, FR-638).** A station, settlement or carrier speaks under many stems sharing
a few beginnings, so its one moment names those beginnings rather than a stem:

```toml
[[cue]]
id = "ReceiveText.StationTraffic"
source = "journal"
event = "ReceiveText"
begins = { Message = ["STATION_", "DockingChatter_", "DockingFailed_"] }
priority = "ambient"
cooldown = 30
```

No one stem names the family, so its id is not spelled from its beginnings. A key names three widths
of moment and `Table` sorts the narrowest first: a whole stem ahead of beginnings, beginnings ahead of
any cue naming no key, such as `ReceiveText.Channel.npc`, whatever their match fields
(`cue.narrower`). The shipped table holds seven comms moments: six naming a stem, one for each pirate
stem FR-620 names; one naming the station traffic beginnings.

**The words a reader sees are generated, with one exception.** `cue.ID.Title` reads an id as words:
each segment breaks where its capitals begin a new word, a run of capitals stays an initialism and
whatever narrows the moment follows a colon, so `StartJump.JumpType.Hyperspace` reads "Start jump: jump
type hyperspace". `cue.ID.Heading` reads the group the same way. The table has no title key. A table
that writes one is refused by name, as is a table writing any other key the loader does not hold.

**The exception is the purpose (FR-231).** A title can only restate its id, which does not tell someone
recording a take when it will be heard. Each cue therefore carries one sentence saying so, written in the
table by hand; `cue.Cue.Purpose` returns it unchanged. The Missing takes pane shows it beneath each title
in the secondary colour (FR-318), which `tests/structural/contrast_test.go` holds to 7 to 1 against
the surface and panel grounds in both themes.

**The shipped set.** The journal cues leave out the snapshots the game writes at login or when a screen
opens (`Cargo`, `Loadout`, `Market` and the like) and two bulk listings (`Music`,
`FSSSignalDiscovered`). At most one payload field narrows an event; a comms moment is narrowed by its key
stem or its key beginnings instead. The status cues are every flag the
status watcher decodes on both edges, every `GuiFocus` value, every pip distribution and the fire group.
`config_test.go` holds every shipped id to that spelling: its first segment is what the cue listens for.

Every table is built through `cue.New`, which refuses a definition it cannot honour: an empty id, an
unknown source or edge, a journal or application cue naming no event, a status cue naming no flag, an
unknown priority, a negative cooldown, an id ending in a segment of digits (FR-219, below), an id
ending in a dot or a space (FR-222, below), an id holding an underscore (FR-230, below) and a comms
moment naming more than one stem, a stem beside match fields, a stem that is no key stem or an id not
spelled from its stem (FR-619); a station traffic moment naming beginnings for more than one field,
beginnings beside a stem or match fields, no beginning or a beginning no key stem can start with (FR-638).
`config.LoadCueTable` also refuses a duplicate id, a key it does not hold and a cue whose purpose is missing or blank (FR-231).
It builds the table through `cue.NewCategorisedTable`, which refuses a category left blank or listed
twice, a cue the game raises whose category is missing or not listed, an application cue naming one and
a listed category no cue sits in (FR-634, FR-635). It accepts a path to a table on disk, which would
replace the shipped one whole under exactly the same rules; no flag supplies such a path today, so the
running application always loads the embedded table and only the tests exercise the other route.

## The voice library

A voice is one person's recordings, held in a directory under a library root the user chooses.
`internal/infrastructure/library` finds them by scanning. The names on disk are the mapping; an optional
manifest only adds to it (below).

```
<library root>/
  Alice/
    StartJump/               folder form: every audio file directly inside is a take
      best-one.wav
      second-try.wav
  Bob/
    DockingGranted.wav       flat form: the base name is the cue id
    DockingGranted.2.wav     a trailing dot and digits tells one take from another
```

- **Every immediate subdirectory of the root is a candidate.** It becomes a voice when at least one take
  resolves inside it. One that resolves nothing is reported rather than cast; the Missing takes pane
  still lists it as a folder to record into.
- **Matching is exact apart from case.** A file name matches a cue id only when the two are equal compared
  case insensitively; a folder name matches only the id with each dot written as an underscore
  (FR-229), compared the same way. `cue.ID.Folder` is the one home of that form. Nothing else is
  normalised, so a folder named in prose or named with dots resolves nothing, which is what makes it
  safe to point the application at a directory and simply see what happens.
- **Four formats are recognised**, `.wav`, `.mp3`, `.flac` and `.ogg`, matched case insensitively.
  Every other file is ignored without a word.
- **Nothing deeper than a cue folder is read.** A folder inside a cue folder is not a take.
- **Two cue folders differing only in case are merged** and the duplication is reported (FR-218). Linux
  permits such a pair; Windows refuses to create the second, so the test that holds the merge hands the
  scan a tree held in memory through its injected directory lister.
- **No cue id may end in a segment of digits** (FR-219). The flat form reads a trailing dot and digits as
  a take number, so such an id would give a file name two meanings. A structural test holds the shipped
  table to this; `cue.New` holds every table to it.
- **No cue id may end in a dot or a space** (FR-222). Windows strips either from the end of a name as it is
  created, so a folder made for such an id would arrive under another name and never be found. The same
  pair of guards holds it.
- **No cue id may hold an underscore** (FR-230). A folder writes each dot as an underscore, so an id
  already holding one could share a folder with another id. The same pair of guards holds it.
- **A voice may hold a manifest**, `voice.toml` (FR-210). It gives the voice the name it is shown by and a
  credit line; its `[takes]` table declares takes the names cannot reach, each a path inside the voice's
  directory. The directory name stays the identity: settings store it, `-voice` matches it and a cast
  sends it back, so `library.Voice` carries `Name` and `Display` apart. The file is read through
  `tomlfile`, the one home of the rule the cue table follows too: a key the shape does not hold is
  refused. A manifest that cannot be read or used is set aside whole and reported (FR-211); an entry
  that cannot be used costs only itself. Nothing the manifest reaches is reported as unmatched.

Every scan returns a report beside the voices, naming what it passed over with the path it was found at
and the reason: directories and audio files whose names match no cue id, candidates that resolved
nothing, case duplicates, takes that will not play and whatever a manifest held that could not be used.
The report is the difference between "nothing here" and "here is what I found and could not use". At
startup it is written to standard error. Voices are listed by the name they are shown by, ignoring case.

The scan reads names, decodes the start of every take a name resolves to and writes nothing. A take
that will not play is left out and named in the report (FR-204). The formats and the decoding belong
to the audio package; the scan asks it rather than keeping a list of extensions of its own. There is
no cache: the scan runs at startup, whenever a
recordings directory is chosen on the Missing takes pane and whenever Refresh is pressed on the Cast or
Missing takes pane.

**Making folders.** The one write the library performs is making empty folders, on request. Make folders
on the Cast pane checks the typed name before anything is made: it refuses a blank name, one starting or
ending with a space, one ending in a dot, one holding a character Windows refuses in a folder name or a
control character and a name Windows keeps for a device. It then makes the voice's folder under the
recordings directory plus one folder for every cue id that has none, answering with how many it made. It
adds and never replaces: a folder already there is left with its recordings, as is a file standing where
a folder would go. Where no recordings directory is chosen yet the folders go in the default recordings
directory, which is then kept as the recordings directory and written to the settings file (FR-228).
Open folder on the Missing takes pane makes the one cue folder a take belongs in where it is missing,
then opens it in File Explorer. Neither changes or removes a file.

**The default recordings directory** is `%LOCALAPPDATA%\BridgeTalk\Recordings` on Windows; elsewhere it
is `BridgeTalk/Recordings` under `$XDG_DATA_HOME`, else under `~/.local/share`. It is made when Make
folders needs it and when the recordings Browse opens with no directory chosen, since that dialog opens
inside it (FR-227).

## Plugins

A plugin is a native library in the `plugins` folder inside the install directory, offering voices
whose audio is already on the user's machine. It is the third implementation of `ports.AudioSource`,
after a scanned recorded voice and a made machine voice, so the catalogue, the picker, the scheduler
and the player learn nothing about it. Section 6.3 of `REQUIREMENTS.md` says what the application
promises; **`PLUGINS-GUIDE.md` is the contract** and states the exported functions, the buffer rules
and the byte layouts. They are not repeated here.

**Why a C ABI.** A plugin is built by somebody else, in a language of their choosing, at a time of
their choosing. The only interface every language agrees on across separately compiled binaries is
the platform's C calling convention. Go has no stable ABI between binaries and its own `plugin`
package does not work on Windows at all, so a plugin cannot be a Go plugin. The technique is one the
application has already proved: `internal/infrastructure/speechmodel` loads ONNX Runtime with
`windows.LoadDLL` and calls it with `syscall.SyscallN`, with cgo disabled.

**Three functions; why so few.** The version, one description of the plugin with its voices, then one
answer per cue. Oliver chose this shape on 2026-09-16 over a dozen smaller calls, which would have
carried more interface surface in exchange for nothing to decode. Every buffer is asked for its size first, then filled, so nothing is allocated on
one side of the boundary and freed on the other; a negative return is always a refusal rather than a
size, so the two can never be confused. Strings carry their own length, so no encoding of the answer
depends on a separator that a path might contain.

**Every address is converted inside the call.** `uintptr(unsafe.Pointer(...))` appears only in the
argument list of `syscall.SyscallN` itself, as it does for ONNX Runtime and for the same measured
reason: through a Go helper, a moving goroutine stack left the library writing to the old copy.
`TestAddressesAreConvertedOnlyWhereTheCallIsMade` holds that rule over the whole tree, so it already
covers the plugin adapter.

**One thread, one call at a time.** Every call into every plugin is made from a single goroutine
holding one operating system thread with `runtime.LockOSThread`, fed over a channel. It costs one
goroutine and a channel round trip on a path that runs once per cue firing. It buys two things: a
plugin author needs no locking; a plugin that initialises something belonging to a thread, such
as a COM apartment, finds that thread again on the next call. No plugin exists to measure, so this is
a precaution rather than a finding; it is taken now because it cannot be retrofitted once plugins are
in the wild. Oliver chose it on 2026-09-16 on those terms, with the cost and the absence of a
measurement both stated.

**Never unloaded.** A plugin once loaded stays loaded until the process ends, for the reason ONNX
Runtime is never unloaded: whether it can be unloaded safely while its own threads may still run has
not been measured.

**Where the code sits.** `internal/infrastructure/plugin` is the adapter. Its portable half owns the
byte layouts, the version check, the buffer protocol and the walk of the plugins folder, all in
plain Go with unit tests over hand-built answers; its Windows half sits behind a build tag with a
no-op stub beside it, so the package builds and vets on every platform. That is the split
`internal/infrastructure/setup` already uses. The composition root wires loaded voices in as audio
sources; nothing in the Application layer learns that a plugin exists.

The seam between the two halves is `Library`, one plugin's three functions. `Load` takes an
`Opener` rather than reaching for the real one, so every refusal a loader can reach is exercised
with no library file in existence. The directory read is handed in for the same reason the voice
scanner takes a lister: a folder that exists and cannot be read is a case Windows will not let a
test produce, since reading a file as a directory answers that the path is not there, measured on
2026-09-16.

**What is measured about the Windows half.** Not the three calls themselves, which need a plugin
file that cannot be built here. What is measured is everything they rest on, against libraries
Windows itself ships: that a library loads by path, that a real library exporting none of the three
functions is refused by the function it lacks rather than called, that a file which is no library is
refused, then that a call through `syscall.SyscallN` fills a Go buffer and answers a size the way the
plugin protocol does. The thread is measured too: every call arrives on one thread id that is not
the caller's. The three `dllLibrary` methods are the package's only uncovered statements, which is
why its floor is the measured 91 percent rather than 100.

**Proved without a plugin; the limit of that.** A real plugin cannot be built here: a library
file exporting C functions needs cgo and a C toolchain; this machine has neither, measured on
2026-09-16. Requiring one would put a C compiler on every build machine, which is exactly what was
turned down when ONNX Runtime was called through its C API instead of through cgo. So the contract
is stood up in Go: `internal/infrastructure/plugin/plugintest` answers in the real layouts, obeying
every rule the guide states, with invented voices and invented paths, so no test, fixture or
document needs any real content to exist.

That proves the layouts, the protocol and everything read back from them. It does not prove the
call into a library file. That belongs to the Windows half and is measured there, against a library
every Windows machine already has rather than one this repository cannot build.

## Resolving a cue

An event resolves to a cue through the table. The reaction service asks the catalogue for the cast
voice's takes for that cue; the domain `Picker` chooses one, avoiding the take last handed over to be spoken for
the same cue; the scheduler hands that take's parts to the player.

**A take is one or more parts.** A take is one alternative answer to a cue; its parts are the files
that answer is made of, played in order with no added gap (FR-573). Most takes have one part. The
port answers takes of parts for every kind of voice, so no part of the application asks where a take
came from before deciding what a take is. The player has always played a sequence: `Play` takes a
list of clips and a gap, with `takeGap` at zero. So the change was small: the scheduler no longer
keeps only the first clip, while the picker chooses among takes rather than among files, identifying
the take it last chose by its first part's path (`take.Take.Key`).

There is no fallback chain. A voice that recorded nothing for a cue answers it with silence and the
reaction list records why, because a wrong line delivered confidently is worse than silence. A voice is
expected to be complete, so a gap is something to record rather than something to paper over.

**The acknowledgement.** Casting a voice plays one take of the single cue whose source is the application
rather than the game. The catalogue finds that cue by its source rather than by its id, so the id is
written in the cue table and nowhere else. A voice that recorded no acknowledgement is silent when cast
rather than broken. It goes straight to the player rather than through the scheduler, so it cannot queue
behind the ship; unlike an audition it respects the mute.

**Coverage.** The Moments spoken for dialog behind each recorded voice's cast row lists the cues a voice serves and the cues it does
not, as exact complements over the table: a cue is in one list or the other, never both and never
neither.

## Event sources

Both sources satisfy one `EventSource` port, so the reaction service sees a single stream and neither
file format reaches the Application layer. The facade polls both every 250 milliseconds.

**Journal (`internal/infrastructure/journal`).** Elite Dangerous appends one JSON object per line to
`Journal.*.log`. The reader stores a byte offset, seeks to it, reads to end of file, prepends any
partial line carried from the previous pass, splits on newline and carries the trailing fragment
forward. A file smaller than the stored offset means rotation or truncation, so the reader restarts at
zero. A line that does not parse is dropped, as is one naming no event.

- **Start at the end, never at the beginning.** On launch the newest journal file's current size becomes
  the starting offset and nothing is read. Replaying an existing session would fire hundreds of clips at
  once. There is no read-all path in this application.
- **Follow the newest file.** Each poll re-resolves the newest journal by name, since the names embed a
  sortable timestamp. A changed path starts at offset zero, because the new file's contents genuinely are
  new.

**Status (`internal/infrastructure/status`).** `Status.json` is rewritten in place rather than appended,
so it needs a different reader: read the whole file, parse it and compare it with the previous reading.
Each changed bit in `Flags` or `Flags2` emits one event tagged rising or falling; a change of GUI focus,
fire group or dominant power distribution emits one value event. The first reading only primes the
baseline, because a flag already set at startup is state rather than news. A reading identical to the
last emits nothing; a read that fails or does not parse is discarded.

**A reading with no flag set in either word is not the ship doing anything.** Wherever a commander can
be, in the main ship, a fighter, an SRV or on foot, one of those flags is set. Both words at zero
therefore means there is no session, so nothing is emitted and the baseline is dropped: the next reading
primes afresh. Diffed instead, leaving the game would announce every flag that had been set as falling.

The status cues (landing gear, hardpoints, cargo scoop, mass lock and the rest) are reachable only
through this source.

## Playback

**Decoding and output.** The audio engine decodes MP3, WAV, FLAC and Ogg Vorbis through the pure-Go
decoders of one library and writes to the device directly. It does not use Windows Media Foundation,
DirectShow, any installed system codec or any external process. `build.ps1` pins cgo off before the gate
and the build, so a machine carrying a C toolchain cannot quietly produce a different binary.

**Nothing is decoded on the device's thread.** A clip is read whole into memory on the player's own
goroutine, resampled to the device rate, before the speaker is given any of it. The device's request for
its next buffer is then a copy out of memory rather than a decode, where being slow would be heard
rather than merely slow. One clip is loaded at a time.

**The player owns its speaker (`speaker.go`).** beep's speaker package kept its oto player to itself, so
audio queued ahead of a new take could never be dropped: measured on 2026-09-13, 240 ms sat ahead of
every take. The player now holds a quarter of a second against a late refill and asks Windows for
100 ms. Stop and a take that cuts in both drop what is queued; a take that starts after silence drops
the queued silence; a take that follows another closely waits for its end rather than cutting it off.
oto opens its device once per process, so that context is the one package-level value in the package.
`TestPlaybackBeginsWithinTheLatencyBudget` holds NFR-P-202 on a real device.

**A late refill is counted.** `levelled.go` times the interval between the device's requests for
samples; one longer than half a second means the buffer emptied before it was refilled. The count and
the longest wait are reported on the status pane, only once the count is above zero. The timing restarts
with each clip.

**An unreadable clip is skipped.** A clip that fails to open or decode plays nothing and raises no error.
The scheduler has already logged the request as played by then, so the reaction list shows it as played.

**A button never cuts a clip short.** An audition starts through `PlayIfIdle`, which asks whether anything
is playing and claims the device under the one lock, so a press while a clip sounds is ignored rather than
started over it (FR-236). The page is told when something starts as well as when it ends: from an
audition, from the cast confirmation and from any poll that set a reaction playing. The audition buttons
are held for as long as anything plays; a pane opened part way through a clip asks whether anything is
playing, so its buttons are held from the start. A machine voice's line being made for an audition counts
as playing too (FR-547). Casting still ends what is playing through `Play`; so
does an alert over a reaction of lower priority. Muting stops the player too, as does the audition
pane's Stop button.

## Scheduling

Each event goes through one decision in `ReactionService.Handle`, in this order: resolve the cue; record
a cue switched off on Chatter as `off`, before anything else is asked of it, so it opens no repeat window,
starts no cooldown, makes no line and reads as `off` while muted too (FR-622 to FR-624); record
a cue still waiting for its line to be made as a repeat; drop a repeat of the same cue inside the 900
millisecond dedupe window; drop a cue still inside its cooldown; ask the catalogue for takes, where there
are none either wait for a line the making service will make next or record the cue as unserved; drop
it while muted; pick one take; submit it to the scheduler. Every step after the first that ends in silence is reported
to the reaction list. An event no cue claims is dropped at the first step and leaves no record.

The arithmetic of the cooldown and the dedupe window lives in the Domain (`selection`). The dedupe width
and the priority policy live in the Application (`services`).

- **Priority**, in descending order `alert`, `notice`, `ambient`, `flavour`. The queue is kept in
  priority order; equal priorities keep their arrival order.
- **Policy per priority.** An `alert` stops what is playing when that request is of lower priority, then
  joins the queue behind any alert already waiting and ahead of everything less urgent; a second alert
  arriving while one plays waits for it. A firing counts against its cooldown and the repeat window
  only once the scheduler takes it. A `notice`
  queues. An `ambient` is dropped when anything is already queued. A `flavour` is dropped when anything is
  queued or playing.
- **Cooldown per cue**, from the table. A cue that fires again inside its cooldown is dropped.
- **Dedupe window.** The journal can restate a situation in quick succession, so a short window collapses
  repeats of one cue into one utterance.
- **Switches (FR-625, FR-626).** A request switched off while it waits in the queue is let go and
  recorded as `off` when its turn comes, the one behind it taken instead; a cue switched off while it
  waits for its line is let go the same way on the next tick. A take already playing is never stopped
  by its switch.

Pre-emption is a cut, not a fade. The player stops the current clip outright and the next starts; there
is no mixer, no ducking and no crossfade. What the player does carry is a single gain applied per audio
buffer, which is what makes the volume slider audible mid-clip.

An audition and the acknowledgement go to the player directly. The acknowledgement starts through `Play`,
so it stops whatever the player was doing, a scheduled clip included; the cut request is not resumed. An
audition starts through `PlayIfIdle`, so it never stops anything.

**Threading.** The audio library runs its own output thread and nothing on it touches the UI. The player
reports completion over a channel. The facade's poll loop selects over that channel, the tray's command
channel and the poll ticker; on completion it tells the scheduler where a voice is cast (an audition
plays with none, when no scheduler exists) then announces the playback state to the
page, so nothing on the audio path reaches Wails directly. The switches are read on the poll loop's
goroutine and changed from the window's own, so `ChatterService` holds them as one value swapped in
whole: a reader always has a complete set without taking a lock, while changes are taken one at a time.
The service is built once at start and handed to each reaction service and scheduler a cast builds,
so casting another voice leaves every switch as it stands (FR-630).

## Choosing where to read from

**Recordings.** At startup, two sources in order: the `-library` flag, then the stored directory. There
is no third. Nothing is detected, because only the user knows where their recordings are. A root that is
missing or unreadable warns on standard error; a readable root holding no voice notes on standard error
each directory the scan passed over. Either way the application runs with nothing cast and the cast pane
names where it looked.

**Journal.** The `-journal` flag, then the stored directory, then the game's saved-games directory under
the user's profile. Where that directory cannot be found, cannot be read, holds no journal file or has no
`Status.json`, the window opens anyway and says why on the Status pane and the Settings pane (FR-238);
nothing is watched until Browse takes a directory that can be.

**Changing either.** The recordings directory is chosen on the Missing takes pane and the journal directory
in Settings. Each Browse opens the system's directory chooser and takes effect at
once; the recordings chooser opens in the chosen directory, else in the default recordings directory. A
recordings directory is rescanned and refused, with the reason drawn in the alert colour, when it
cannot be read or holds no voice; otherwise the cast voice is re-cast by name where it survives the move
and falls back to the first voice where it does not. A journal directory is refused unless both sources
can be built over it, then both are swapped in together. Either choice writes both directories to the
settings file. Each chooser answers with the directory it took, else with nothing where the dialog was
cancelled, which is what lets the pane tell a cancel from a refusal.

**Refresh.** Refresh on the Cast pane and on the Missing takes pane reads the recordings directory again
without restarting (FR-214). It is refused where no recordings directory is chosen or the directory
cannot be read. A scan that finds voices takes them and re-casts as a new directory does; one that finds
none changes nothing. It writes nothing to the settings file.

**The cast voice.** Casting is the one act that writes the voice's name down, so the next run opens
speaking with it. The re-cast that follows a new recordings directory or a Refresh does not write it. A
name is matched case insensitively, whole or as an unambiguous prefix. A stored or flagged name that is
no longer found falls back to the first voice with a warning on standard error.

**The command line.** `-voice` names a voice for one run; `-list` prints each voice with its takes and
the cues it covers, then exits; `-unbound` lists the cues the chosen voice cannot serve, then exits;
`-no-tray` runs without a notification-area icon; `-hidden` starts in the tray with no window. `-list`
and `-unbound` exit with an error where the recordings directory holds no voice. A windowed build started
from a terminal attaches to it first (`runlog.ReportToTerminal`), so both reports print there.

## Idle remarks: not built

**Nothing below runs today.** The application registers exactly two event sources, journal and status.
It speaks only in answer to the game or to a press of its own controls. This section records what the
shape allows, so that the absence reads as a decision rather than as an omission.

The seam is the port, not a component. The composition root hands the facade a list of `EventSource`,
so a third source would be added to that list and nothing in the scheduler would change. It would feed
the same queue at `flavour` priority, which is already the priority that is always outranked, always
droppable and never interrupts.

What is missing is the source itself, along with the rate limit and quiet-period rule any such source
would need. Without those it turns from characterful to grating quickly, which is the reason it has not
been built on the way past.

## UI

**Shape.** The application is resident rather than foreground. A tray icon carries the menu; the main
window is summoned. The cross asks rather than acts: it is ambiguous in a resident application, meaning
"put it away" at least as often as "stop it", so `OnBeforeClose` cancels the close and the page draws the
choice. Minimising leaves the application listening with its tray icon on screen; quitting is deliberate;
dismissing the dialog changes nothing. A quit already chosen, from the File menu, the tray or the dialog,
passes straight through rather than being asked about twice. With no tray icon there is nowhere to hide,
so the close is allowed through instead of leaving a running application with nothing on screen to
summon it.

```
+-----------------------------------------------------------------------------------------------------+
| File   Audio   Settings   Help                                                                      |
+-----------------------------------------------------------------------------------------------------+
| [Cast] [Audition] [Status] [Missing takes] [Chatter] [Settings]   [Volume] [Mute] [Theme] [Guide]   |
+-----------------------------------------------------------------------------------------------------+
|                                                                                                     |
|   main pane: a switched view, not a stack of modal dialogs                                          |
|                                                                                                     |
+-----------------------------------------------------------------------------------------------------+
| [Donate]                                                                             live indicator |
+-----------------------------------------------------------------------------------------------------+
```

The nav band is one flat row with the two groups separated by a stretch, so layout order is reading
order. The main pane switches between Cast, Audition, Status, Missing takes, Chatter, Settings and Guide;
it opens on Cast. The menu bar repeats the ways in: File holds Quit; Audio holds Cast, Audition, Missing
takes, Chatter and Mute; Settings holds the
pane and the theme; Help holds the guide, the licence and About.

**Machine voices on the Cast pane.** `frontend/src/machineVoices.tsx` offers them under their own heading
after the recorded voices, as pills in a panel for each accent and sex, the panels sharing the row
(FR-720). The panels keep the group order the facade offers, which is FR-508's; each sorts its pills
by name. `machinevoice.Voice` gives each voice's name alone and its group beside the full name, so the
id's format keeps one home. The cast machine voice leaves its panel for a card above them, which is not a
control, since pressing it would cast it again (FR-721); a recorded voice cast leaves no card, even one
whose folder carries a machine voice's id (FR-722). The card reads how far making has got: it asks `Making` once, then follows the `making` event, which the poll loop
sends only when the answer has moved, so an idle tick says nothing. Failed lines, a stopped making,
undeleted lines and a refused cast each get a callout beneath the list. `StateDTO.MachineVoice` says
which kind of voice is cast, since a recordings folder may carry a machine voice's id. The words both
lists share, the part a cast voice plays and the counted figures, live in `frontend/src/castWords.ts`;
where making stands before anything is made lives in `frontend/src/making.ts`, apart from `api.ts`,
because a test replaces that module whole.

Five surfaces are modal, all built on one dialog shell so none arrives with rules of its own: About, the
licence, the close choice, the Moments spoken for dialog and the question Chatter asks before changing
more than one moment. Each opens focused on its first control; the close choice lists Minimise first,
since Enter straight after pressing the cross must not mean stop. Chatter's question lists Cancel first
for the same reason: Enter straight after the press keeps the choice made for each moment. The licence dialog shows the `LICENSE` file itself, embedded at build time, so the terms shown
and the terms the source carries cannot differ; About names the licence in a sentence.

**Status.** Four cards widen to share their row: the cast voice, Moments covered, the journal directory
and the status file. Beneath each figure a tagline in the secondary text colour, the colour the purpose
line on Missing takes is drawn in, says what the card means; a tagline naming the product waits for
About to name it. Moments covered adds the line for the cast voice's situation, so a figure short of the
whole reads as a gap in the recordings or a machine voice still making its lines rather than as a fault
(FR-716). The words live in `frontend/src/statusWords.ts`. Then whether playback is live or muted, a line
where no audio device was found and a line where the device has run dry. Beneath them, the reaction list: each decision with its time, its outcome and its cue,
beside the clip's file name, left blank where there was none. The facade keeps the last 200. That list is
how a cue that never speaks gets diagnosed.

**Casting, not selecting.** Choosing a voice is casting a part, so each row names the act it offers:
"Cast Iris"; "Grace is cast as your ship's voice" for the one already in the role. Beneath the name
the row gives the number of recordings the voice holds that answer a cue. Every voice listed can be cast,
because a directory that resolved no recording never becomes a voice. A button at the end of the row
opens the Moments spoken for dialog. Beneath the rows, Make a voice holds a name box with Make folders and Refresh.

**Missing takes.** The recordings directory row with its Browse, then a chooser offering every voice
folder still missing a recording, empty folders included, since a voice made with Make folders holds
nothing until its first take (FR-316). It opens on the voice already chosen, else the cast voice, else the
first. For the voice chosen it lists each missing moment by title, its purpose beneath and the folder its
take belongs in, with Open folder beside it.

**Chatter.** The pane lists every cue the game raises; the application's own `Cast.Confirmed` answers
the player's act rather than the game, so it has no switch (`cue.Cue.Switchable`, FR-621). A header
that does not scroll holds Switch all on and Switch all off, each disabled while
it would change nothing, then a switch for every category in the table's order, named beside it and
reading on while any moment in it is on (FR-740). Beneath it the list scrolls on its own: every category
a group ruled down its side, headed by the category with how many of its moments are on. The heading is
sticky, so it stays at the top of the list while its moments pass and gives way to the next group's
(FR-741); the list's scroll padding is the heading's height, so a switch the keyboard reaches is never
brought into view under it. Each moment shows its title with its purpose beneath and its own switch. A switch is a button with the switch role, named by its moment or its category, drawn
as a track holding a thumb whose position tells on from off (FR-735, FR-736). A press that would change
more than one moment asks first, naming how many. The page keeps no copy of the switches: each press is
answered with the pane as the application then holds it, with the reason beside it where a change
applied without being kept (FR-633). The pane is `frontend/src/chatter.tsx`; its style part is
`theme/chatter.css`.

**Audition.** The cast pane says what a voice covers; the audition pane lets it be heard. Groups come
from the cue vocabulary's own first segment, the moment in the game's own words, so the audition keeps no
grouping of its own in step with the cue table; a group with no takes is not offered. Each group button plays one clip
drawn at random from the union of its cues' takes, deduplicated, so one take answering several cues is not
weighted by them; a press can repeat the previous clip. A Stop button ends whatever is playing, a reaction
to the game included. The voice being auditioned starts as the cast one but is not tied to it: hearing a
voice before committing to it is what an audition is for. An audition ignores the mute, which silences
reactions to the game rather than the application, because answering a deliberate press with silence
would read as a fault.

A machine voice is auditioned on the script's groups instead, each counting its lines (FR-546). A press
draws one line; one not yet made is made on the model's next turn, ahead of the cast voice's next line,
then kept as every made line is, so casting the voice later does not make it again. The making service
hands the model to a run or an audition one turn at a time, a turn covering the line's write as well, so
lines are written in the order the turns were taken. While the line is made the buttons are held and a
further press is ignored; Stop lets it go, so it is kept once written yet never played (FR-547). A
recordings folder may carry a machine voice's id as its name, so the chooser keeps the two apart and a
machine voice goes through `MachineAuditionGroups` and `AuditionMachineVoice` rather than the recorded
voice's pair.

**Volume.** A slider in the nav band, from silence to the clip as recorded, in twenty steps. Perceived
loudness is roughly logarithmic in gain, so the slider position picks a point up to six halvings below
full rather than scaling gain directly: mapped straight onto gain, most of the travel sounds the same.
The level is read once per audio buffer rather than once per clip, so moving the slider is heard
immediately. The page stores the level and pushes it in on load, so the player persists nothing.

**Icons.** The band's icons are the project's own artwork rather than drawn marks, so they carry their
own colour and do not follow the theme. `tools/genicons.py` writes them from the masters in `assets/`
into `frontend/src/assets/icons`, each centred on a square canvas; it composes the muted speaker from
the sounding one and a slash so the two states cannot drift. The script is not part of the build and its
output is committed, so a clone needs neither Python nor Pillow to build the application.

`assets/donate.png` is not a band icon and the script stops rather than square it into one. It trims the
artwork to its content and scales it by height alone to four times the height the strip draws it at, then
writes that render to `frontend/src/assets/donate.png` for the window (FR-718). The site's
`docs/images/donate.png` is the donate mark every project site shares, 133 by 116 pixels, which the
script never writes (Oliver, 2026-09-15).

`assets/application-icon.png` is the whole identity. The same script turns it into a multi-size `.ico`
beside it, a copy for the About crest and the setup page's header mark, alongside the setup page's two
theme icons. `build.ps1` stops before building where the `.png` or the `.ico` is missing, then copies both
onto the application and onto the setup program. The shortcuts point their icon at the executable and the
tray loads its icon out of it.

The icons are drawn large, which leaves no room for a label beside each one, so the buttons carry the
artwork alone and the name arrives on hover or focus as a tooltip drawn by the stylesheet from the
button's own label. Each button also carries its name as an accessible label, so nothing depends on the
tooltip being seen.

**Keyboard.** One explicit ring, recomputed on each move. Its stops are the controls marked as stops that
are enabled, not hidden from assistive technology and actually rendered; the nav button for the pane
already open stays a stop. Tab and Right step forward; Shift+Tab and Left step back; both wrap. Text
fields keep their own arrows and leave the ring by Tab; the volume slider gives Left and Right to the
ring and keeps Up and Down. A drop-down list opens on Down; Up and Down never change its value with the
list shut. A scrolling region joins the ring only while it overflows and rings on focus alone, never
under the pointer. The reaction list is a single stop whose rows are walked with Up and Down; being a
list, it wears no ring and shows where focus is by its current row, brought into view as focus lands.
That row is the newest entry while the keyboard is elsewhere and stays where the arrows put it while
the keyboard is on the log. No container is ever a stop. A menu title opens on Down, Enter or Space;
Up and Down walk its items; Escape closes it back to the title. The main window starts neutral with
nothing focused and no menu open; every dialog opens focused on its first enabled control and Escape
closes it.

Ring colours follow three states and no others: no ring at rest, one colour while hovered or focused
and enabled, a permanent second colour while disabled. The brand accent is never a ring colour, because
it carries meaning rather than state.

**Getting the keyboard at all.** Wails gives the web view its keyboard only from the main window's own
focus event, which a window that never took focus never raises. `internal/infrastructure/window` finds
this process's visible window, attaches to its thread's input and focuses the WebView2 child directly,
which is what a click would do. Both programs call it once the page exists; each page then checks
`document.hasFocus()` after 400 milliseconds and asks the facade for the keyboard again where it has none.

**Reading surfaces.** The guide and every dialog carrying text hold still on open, then read themselves
down slowly, hold at the end, rewind quickly and repeat. Every dialog reads through one `ReadingBody`, so
none keeps rules of its own; a body that fits its dialog never moves, since the cycle acts only on content
that overflows. A wheel, a press, a touch, a key or focus arriving suspends the cycle and it resumes from
where the reader left it rather than switching off. Focus arriving while the start hold still runs is the
dialog's own opening focus rather than a reader, so it leaves the hold alone. A surface beneath a modal
freezes in place rather than suspending and takes no input while frozen, so it resumes exactly where it
was. The cycle is not gated on reduced motion. Surfaces to act on, such as the reaction list, do not read
themselves.

**Theme.** Light and dark, chosen from the nav band or the Settings menu, stored by the page and dark by
default. Every colour value in `frontend/src` lives in one token file with a set per theme and no
component names a colour directly, so the three ring states hold in both themes by construction. The
setup page and the window backgrounds set in Go carry colour values of their own, outside that rule.

**The strip.** Along the foot of the window, below whichever pane is open, `frontend/src/strip.tsx`
draws the donate button at the left and the live indicator at the right (FR-717). Its height is three
quarters of the band's, derived in `theme/footer.css` from the sizes `theme/navband.css` names once and
draws the band with. A number of its own would read right until the band changed, then leave the strip
measuring a band that is no longer on screen; derived, it follows the band, which keeps it reading as
subordinate. `tools/genicons.py` reads the artwork's drawn height from the same tokens. The band's
tooltip opens below its control, which at the foot would fall outside the window, so a rule scoped to the
strip opens its tooltips above their controls, the left-most from its own left edge. The strip comes
after the pane in the page, so the ring reaches the donate button last.

The donate button calls `OpenDonation` in `donate.go`, which hands `product.DonateURL` to the desktop
through Wails' `BrowserOpenURL` (FR-718). An unexported helper refuses any address not beginning
`https://` with a sentinel error before anything is handed over; the Wails call sits behind a field a test
replaces, so the tests see the one address handed over with no browser opened. The application opens no
connection for the button and fetches nothing: the browser does the asking, so no network code enters the
application for it. Wails reports nothing back from the hand-over, so the one failure the facade can see is
having no window to open from; the page shows a rejection in the indicator for four seconds.

The indicator's message is chosen by `indicate` in `frontend/src/indicator.ts` (FR-719). It is a pure
function of the state, the last `making` announcement, the last moment played with the time it arrived,
the time a donation hand-over last failed and the time now, so every row of the table and the order
between them are tested without a page. Each row names a tone; a class per tone maps it to its colour
token, so no colour leaves the palette. The strip holds the times. A timer for each message still showing
wakes the strip when its four seconds are up, which `flashMs` states once for both uses; the timers are
cleared when the strip goes. The moment is named by its full title, which `ReactionDTO.Title` carries:
`record` in `reactions.go` finds it in the cue table for any voice, recorded or machine, since before it
only a recorded voice's breakdown brought titles to the page.

**One control, one place.** Mute and volume live in the band, which is on screen whichever pane is
open. Neither is repeated in a pane. The status pane still reports the playback state in words, because
whether an audio device was found at all is something the band cannot show.

**Crests.** About and the guide are headed by the artwork they belong to, the application icon and the
help icon respectively, so a reader can see at a glance which surface they opened.

**Guide.** Its words live in `frontend/src/guideContent.ts` as typed sections: entries naming a control
by the pictures it wears, rules stated as a claim followed by what it means, then plain paragraphs.
`guide.tsx` only draws them. Every entry takes its pictures from the `artwork` export in `icons.tsx`, the
module the controls themselves are drawn from, so the guide cannot show a picture the window does not.
Its title is read from About rather than written into the page, because the product is named in one
place. `guide.test.tsx` renders the nav band in both states of Mute and the theme; it fails when the band
draws a picture that no entry shows.

## The tray

The notification-area icon is the application's only visible surface while the window is put away. It
owns a hidden window and a message loop on a thread locked for their lifetime, which is a hard
requirement of the Win32 windowing model.

Nothing crosses that thread boundary by callback. The tray reports what the user chose on a channel that
the facade's loop selects over, dropping a choice rather than blocking when nothing is reading; the loop
pushes the mute state and the cast voice back through atomics that the menu builder and the tooltip
read. Each push also posts a message to the tray's own window, so the hover text is sent again from the
tray thread once the state has changed. A menu choice never sends it itself: the choice has not been
acted on when the menu returns. A callback invoked from the tray thread would be running on the wrong thread for everything it
wanted to touch, which is a failure mode this design removes rather than manages.

The left button asks for the window on a single click and on a double; the right button opens the menu:
a Voice submenu, Open, Mute and Quit. The tooltip names the cast voice and says when it is muted. The
Voice submenu lists the voices found at startup; it is not rebuilt when a new recordings directory is
chosen or Refresh finds more. The machine voices follow them under a separator. A choice carries whether
it is a machine voice, since a recordings folder may carry a machine voice's id: the check mark matches
a voice by name and kind, the kind pushed beside the name through an atomic of its own. The tooltip
shows the voice by the label pushed with it, so a voice found after startup is still shown by the name
its manifest gives (FR-210). A machine voice chosen there reaches `CastMachineVoice` rather than
`SelectVoice`.

The window comes back centred and on the cast pane, whatever pane it was left on. Centred, because a
window put away for hours may return to a different arrangement of screens and the middle is the one
place it is certainly reachable. On the cast pane, because hiding a window does not reload the page. The
facade says so on the way back rather than the page guessing, since only the facade can tell a summons
from an ordinary raise. Showing it precedes taking focus: `SetForegroundWindow` acts on a window that is
already visible.

A tray that cannot be created is not fatal: the application still watches the journal and still speaks.
Non-Windows builds compile a no-op tray rather than taking on a desktop toolkit dependency for one icon.

## The setup program

Delivery is a second Wails application. `installer/` is its own `main` package inside the same module,
embedding the built application as a zip and the setup page as assets, so one file is the whole
distribution. `build.ps1` runs `tools/payload`, which checks `models/` against the list, downloading
nothing, then packs the built application with every model file the application reads beside itself into
`installer/payload.zip` (FR-543). It builds the setup program with the version from `VERSION` passed in
through `-ldflags`, then puts the empty placeholder zip back whether or not that build succeeded, so a full
payload never reaches a commit. A setup program built without that flag reports its version as `dev`. The
payload is embedded as a string rather than a byte slice, which a stand-in program measured at 12.8 MB of
private memory at start against 323.6 MB (FR-524).

It is split the way the application is. `internal/infrastructure/setup` holds the machine work: the
paths, the payload extraction with its fence against an archive entry that climbs out of the install
directory, the version comparison, the registry writes, the shortcut handling and the process work. The
portable half carries unit tests; the Windows half is behind a build tag with no-op stubs beside it, so
the package builds and vets on every platform, while its own tests run only on Windows.
`installer/app.go` is the facade over it and owns the sequencing: the order of the install and uninstall
steps and their progress figures, the name of the uninstaller copy and reading the `-uninstall`
argument. It has no tests of its own.

**The setup page names nothing.** The setup front end is hand written with no build step, so nothing
compiles it and nothing type checks it. The product's name arrives on the state the page is already
given and the static markup carries none of it. `TestTheProductIsNamedOnce` reads the page's directory to
hold it to that; the line-limit tests read the same directory for size.

**It is five files beside its three images.** `index.html` carries the markup, `setup.css` the palette and layout, `setup-ring.js`
the keyboard ring (FR-808), `setup-shell.js` the page plumbing and `setup-routes.js` the screens. No
bundler is involved: the whole directory is embedded already, so a stylesheet link and three script tags
resolve as they stand. The scripts are classic, sharing one global scope in load order, which is why the
state the screens read is declared in `setup-shell.js` ahead of them.

The ring is the window's keyboard model written again, since the page has no build step to share the
window's code through. It has no test runner either, so `frontend/src/setupRing.test.ts` loads the script
the page ships and presses keys against a page shaped like one of its screens;
`TestTheSetupPageLoadsEveryScript` holds the page to loading it.

Which screen opens is decided by two readings of the machine: whether the uninstall registry entry
exists and how the payload's version compares with the one recorded there.

| Reading | Screen |
|---|---|
| started with `-uninstall` | uninstall |
| nothing installed | install |
| payload is newer | update, with Uninstall also offered |
| payload is older | the update screen, worded as going back a version |
| versions match | manage: Repair, Reinstall, Uninstall or Close |

Repair and reinstall are genuinely different acts rather than two words for one. Repair writes the files
again and leaves the shortcuts and the login entry as they stand; reinstall writes them again with the
manage screen's boxes applied as they stand, never a set of choices of its own (FR-235). Both act on one
press and both run the same single install path, which is what keeps them from drifting apart. The boxes
on the manage screen act the moment they change; a box whose change fails is put back and the error is
shown. On the install screen both shortcut boxes start ticked; on the update screen they show the
shortcuts the machine has. Every screen that writes files ends with a box, ticked by default, that starts
the application and closes setup once the work succeeds.

Everything is per user, so no step needs administrator rights: the files in the install folder, the
Start Menu shortcut under `%APPDATA%`, the Desktop shortcut in the user's own Desktop directory and the
install record and login entry under `HKEY_CURRENT_USER`. The shortcut boxes apply in both directions,
so unticking one removes a shortcut rather than leaving a stale one behind. Setup copies itself into the
install directory as `uninstall.exe` and registers that copy as the uninstaller and as the Modify
target, with `NoModify` and `NoRepair` both zero.

The install folder is chosen on the Install screen alone (FR-809). Setup offers
`%LOCALAPPDATA%\Programs\BridgeTalk`; Change opens a folder picker and the install goes into a folder
named for the product inside the folder picked, never into the picked folder itself, because uninstall
deletes the install folder whole. `setup.CheckInstallDir` refuses a folder that already holds anything
but the application, a path that is not full or stands beneath a file and a folder whose nearest
existing folder this account cannot write to. The facade asks it when the picker returns and again
before it writes. The folder is recorded as the Apps list entry's `InstallLocation`, which
`setup.InstallDir` reads back for every later run; where nothing is recorded, the offered folder
answers.

Uninstall removes the shortcuts, the login entry and the install record, then the folder the made lines
are kept in and the run log whatever is ticked (FR-525, FR-715), leaving the product's data folder around it, which can hold the
default recordings directory. It then hands the install directory to a detached shell that deletes it once
setup has exited. `setup.RemoveLeftovers` holds the rule for what goes outside the install directory, so
the facade only finds the folders and passes the box on. Its one box, Also forget my settings, is
unticked by default; ticked, it also removes the web view's folder under `%APPDATA%`, which holds the
theme and the volume, plus the settings file and its working file, then the settings directory where
that leaves it empty. The recordings are never touched, the default recordings directory included.

An install or an uninstall refuses to run while the application is open, because writing over a locked
executable fails part way and leaves a half-written install. The setup window offers to close it instead.
Closing is forced rather than a polite window close, since a polite one only raises the application's own
close question, which setup cannot answer; setup then waits up to five seconds for the process to go.

The setup page carries the application's palette in both themes and opens in whichever one Windows is set
to for applications, with a toggle in its header. The window's background colour is chosen before the
page loads so setup never flashes the wrong ground. Its own web view data is kept under the temporary
directory, so running setup leaves no folder beside the application's.

## Data locations

| What | Where |
|---|---|
| Journal and status files | `-journal`, else the stored choice, else the game's saved-games directory under the user's profile |
| Recordings | `-library`, else the stored choice; at startup nothing is detected |
| Default recordings directory | `%LOCALAPPDATA%\BridgeTalk\Recordings` on Windows, `BridgeTalk/Recordings` under `$XDG_DATA_HOME` or `~/.local/share` elsewhere; made on first use and never removed by setup |
| Settings | `settings.json` in `BridgeTalk` under Go's user configuration directory (`%APPDATA%` on Windows): both directories, the cast voice (a recorded voice's name or a machine voice's id, the other forgotten) and the ids of the moments switched off on Chatter. Choosing either directory writes both, as does Make folders adopting the default recordings directory; casting writes the voice; a switch writes the switches |
| Cue table | embedded in the binary |
| Script | `script.toml`, embedded in the binary beside the cue table |
| Saved speech sounds | `sounds.toml`, embedded in the binary beside the script; written by `go run ./tools/sounds`, never by hand |
| Pauses | `pauses.toml`, embedded in the binary beside the saved speech sounds; written by `go run ./tools/pauses`, never by hand |
| Endings | `endings.toml`, embedded in the binary beside the pauses; written by `go run ./tools/pauses`, never by hand |
| Model files on the build machine | `models/` at the repository root, found by walking up to `go.mod` and filled by `go run ./tools/models` from the list embedded in `internal/infrastructure/modelfiles`; not committed |
| Model files when running | `models` beside the executable, the one folder the application reads them from; `voicefiles.Folder` names it for the build machine's folder too |
| Made lines | `Made lines` in the product's local data folder (`%LOCALAPPDATA%\BridgeTalk` on Windows), one folder for the cast machine voice; never under the recordings directory |
| Run log | `Log.txt` in the product's local data folder, started afresh once it passes 1 MB; removed by uninstall |
| Theme and volume | the page's own storage, inside the web view's folder `%APPDATA%\BridgeTalk.exe` |
| Installed files | `%LOCALAPPDATA%\Programs\BridgeTalk` unless the Install screen picked another folder, per user; recorded in the install record |
| Shortcuts | `%APPDATA%\Microsoft\Windows\Start Menu\Programs` and the user's Desktop |
| Login entry | `HKCU\...\CurrentVersion\Run`, written by setup or by Settings, one entry either way |
| Login entry value | the quoted path plus `-hidden`, so a sign-in start waits in the tray |
| Install record | `HKCU\...\Uninstall\BridgeTalk`, per user |

Nothing is written to the game's directories. Under the recordings directory the application writes only
the empty folders described in Making folders; it never changes or removes a file there. The game is the
single writer of the journal; this application is one of several readers.

## Errors

Errors are wrapped with context at each boundary using `%w`. One comparison is made by string: the
journal reader tests a read error's text against `"EOF"`.

- **Fatal, before any window:** a cue table, a script with its saved speech sounds, a `pauses.toml` or
  an `endings.toml` that fails to load. All four are embedded in the binary.
- **Said in the window, which opens anyway (FR-238):** a journal directory that cannot be found, cannot be
  read, holds no journal file or has no `Status.json`, the game's usual one included where nothing was
  chosen. `openJournal` carries the reason rather than returning it; the Status pane and the Journal
  directory row on the Settings pane show it, standard error prints it and nothing is polled until Browse
  takes a directory that can be watched.
- **A warning on standard error, then carry on:** no audio device, which runs silent and says so on the
  status pane; no tray; a recordings root that is missing or unreadable; a voice name that is not
  found; the scan report.
- **Passed over while running:** a poll that fails is printed to standard error and skipped until the next
  tick; a malformed journal line is dropped; a status read that fails to parse is discarded; a clip that
  fails to decode is skipped after being logged as played; a made line whose samples differ from those
  its pause or its ending was found in is written without that pause or fade and logged (FR-553,
  FR-556).
- **Recorded in the reaction list:** a cue switched off on Chatter, a repeat inside the dedupe window, a
  cue in cooldown, a cue the voice has no takes for and a request dropped by the mute or by the priority
  policy, beside what was queued, what is being made and what played.
- **Refused with the reason, beneath the control that was pressed:** on the Cast pane, a voice name Make
  folders will not use, a folder it cannot make and a Refresh with no readable recordings directory; on
  the Missing takes pane, a recordings directory that cannot be read or holds no voice and a moment's
  folder that cannot be made or opened; in Settings, a journal directory the sources cannot be built over
  and a login entry that could not be written, which includes turning it on from a copy running under the
  temporary directory.

**A refusal names its path once (FR-237).** A file-system error from the standard library already
carries the path and the system call behind it, so wrapping one beneath words that name the path showed
the path twice, with a call such as `GetFileAttributesEx` between. `internal/refusal` is the one home for
the fix: `Reason` keeps only the system's reason, which the site that names the path wraps with `%w`;
`Check` is the one statement of the rule, which the refusal tests of the facade and of `config`,
`setup`, `madelines`, `modelfiles`, `runlog`, `speechmodel`, `voicefiles` and `wholefile` hold their
refusals to. It sits under `internal` beside `product` for the same reason: the facade, `tools/pauses`
and the `audio`, `config`, `journal`, `library`, `madelines`, `modelfiles`, `runlog`, `setup`, `status`,
`voicefiles` and `wholefile` packages all read it, so it belongs to no layer. A refusal the window
shows writes its path with `%s` rather than `%q`, which doubles every Windows separator.

## Quality enforcement

- Structural tests enforce the layer direction, domain purity, the module-size limit and its danger
  band, the composition-root whitelist, the declared bound surface, the wire contract and the rules that
  keep a value in one home: colours in the theme tokens, the product name in `internal/product`, the
  game's own words in the cue table. They also hold the shape of every cue id, the secondary lines'
  contrast, the Status cards' grid, the strip's height, labels and tones, the rings a control, a list
  or a scrolling region may wear, the setup page's boxes and header and the speech sound table in `internal/domain/speech`
  against the model's tokenizer file in `models/`. Another lets an address handed to a DLL become a
  uintptr only where the call into it is made; another fails where `pauses.toml` or `endings.toml` is stale against the
  script, the machine voices or the listed model files, with four more for each proving that check names
  what it should. The invariant table above lists every one of them with
  the test that enforces it.
- The wire is written twice by necessity, as Go structs with json tags and as TypeScript interfaces in
  `frontend/src/api.ts`. Wails generates the same shapes into `frontend/wailsjs` at build time; that
  output is gitignored and imported by nothing, so it is not the contract and it goes stale silently.
  The hand-written file is the one the application compiles against; a structural test compares it to
  the DTOs so a rename on one side alone cannot pass.
- The game's vocabulary has two homes and no others: `internal/infrastructure/config` holds the cue
  table and `internal/infrastructure/status/flags.go` declares the status values the game writes.
  Elsewhere a cue id, a journal event or a status value is read as data rather than written down. The
  guard knows the real words because it reads the table, so it never has to guess which strings look
  like one.
- `test.ps1` first checks `models/` against the model files list, stopping where a file is missing or
  differs (FR-538). It then checks formatting, vets and runs the whole Go suite, leaving out the Go package an npm
  dependency ships inside `frontend/node_modules`. It holds `internal/domain` and `internal/application`
  to a combined 100% coverage, then holds each other measured package to a floor of its own, set from
  what that package measured rather than from a target. `internal/infrastructure/window` and
  `installer` carry no floor, since neither has anything a test can reach without the platform behind
  it; `modelfilestest` is test support with no tests of its own; `internal/product` holds constants
  alone and is not measured. TESTING.md tabulates every figure beside its floor and names what each
  shortfall is, so the numbers are stated there once.
- `build.ps1` first stamps the version into the site through `stamp_version.py`, then runs
  `test.ps1 -Benchmarks` before it builds and offers no switch to skip it. `-Benchmarks` vets and runs
  `tests/machinevoice` and `internal/infrastructure/speechmodel` under the `benchmarks` build tag, so
  every build also times a machine voice's cast against NFR-P-205 and measures making a complete script
  against NFR-C-502. `wails build` runs the
  front end's own build script, which runs `eslint` and `tsc --noEmit` before bundling, so a lint or type
  error stops the build too.
- Neither script runs `staticcheck` or the front-end test runner. Both are run by hand, the tests with
  `npx vitest run` from `frontend/`; the front-end coverage report carries no threshold.

## Design decisions

| Decision | Why | Rejected alternative |
|---|---|---|
| Go with a web front end | A single binary with no runtime to ship; the same web view serves the setup program | A Python and Qt desktop stack |
| Pure-Go audio, cgo disabled | No system codec, no external process | A system media framework; a bundled transcoder, too heavy for the job |
| The names on disk are the mapping | Game semantics and a person's recordings change independently, so a voice needs no mapping file | A mapping file per voice, kept in step by hand |
| One cue plays one take | A cue is answered once, by one alternative chosen from what the voice holds | Playing every take a voice has for a cue |
| A take is one or more parts, played in order with no added gap | A line is sometimes recorded in pieces; offering the pieces as separate takes would let the picker speak the middle of a line on its own (Oliver, 2026-09-16) | One file per take, which was the rule until plugins needed otherwise |
| A plugin is reached through the C ABI, loaded by full path from one folder | It is the only interface every language agrees on across separately built binaries; Go has no stable ABI between binaries and its plugin package does not run on Windows | A Go plugin; a helper process speaking over a pipe, which is a second program to install and keep alive |
| Buffers are owned by Bridge Talk, sized by a first call | Nothing is allocated on one side of the boundary and freed on the other, so the two need not share an allocator | The plugin allocating and a fourth function freeing |
| A negative return is always a refusal, never a size | A size and an error code sharing a range is how a one byte answer becomes an error | Negative meaning the bytes needed, with a sentinel carved out of the range |
| Every call into a plugin is made from one locked operating system thread, one at a time | A plugin author needs no locking; a plugin that initialises something belonging to a thread finds that thread again. No plugin exists to measure; this is a precaution taken while it is still cheap, chosen by Oliver on 2026-09-16 | A mutex alone, which serialises without giving thread affinity |
| A plugin is never unloaded | Whether it can be unloaded safely while its own threads may still run has not been measured, the same reason ONNX Runtime is never unloaded | Freeing the library when the last voice it offered is dropped |
| A plugin is trusted because the user put it in the folder | Checking a signature would mean deciding whose signature counts, which is a promise the application cannot keep for other people's work (Oliver, 2026-09-16) | Refusing an unsigned plugin; asking the user to confirm each one |
| No fallback chain | A wrong line delivered confidently is worse than silence | Substituting another cue's take |
| No cue id ends in digits | The flat form reads a trailing dot and digits as a take number | Letting a file name carry two meanings |
| Status flags as a first-class source | A large block of cues describes ship state that never appears in the journal | Journal only, which would leave those cues permanently unreachable |
| Start reading at end of file | Replaying a session on launch would fire hundreds of clips at once | Reading from the beginning; a heuristic time cutoff |
| Priority, cooldown and dedupe | Without them the application is a slot machine rather than a voice worth listening to | A plain queue |
| The facade polls a list of event sources | A third trigger source is a line at the composition root rather than surgery on a finished scheduler | Wiring the two sources in directly |
| No audio shipped | The recordings belong to the user; the application plays them | Bundling audio |
| ONNX Runtime called through its C API table with cgo disabled, loaded by full path | The build stays pure Go; the full path keeps the older copy Windows ships in System32 from standing in | cgo bindings, which need a C toolchain on every build machine |
| The model is loaded at the earlier of a machine voice being cast and its first line being made, then kept until the maker is closed | A player who casts only recorded voices never pays for loading 310 MB; a cast loads it without waiting, so the first cue made on call is spared the 539 ms load (FR-544); a load that fails is tried again on the next line, so a folder Repair put right is used without a restart | Loading at start whatever voice is cast; remembering a failed load |
| ONNX Runtime is never unloaded | Whether it can be unloaded safely while its own threads may still run has not been measured | Freeing the library on Close |
| Every address is converted to uintptr in the argument list of `syscall.SyscallN` itself | Only there does Go keep the variable where ONNX Runtime was told it is; through a Go helper, a moving goroutine stack left ONNX Runtime writing the old copy, which broke a build on 2026-09-14 | A helper taking `...uintptr`; pinning every out-parameter instead |
| The Chatter switches are kept in the settings file | The engine needs them before any page loads, which the theme and the volume do not | The page's own storage, beside the theme and the volume |
| The switches are one value swapped in whole | The poll loop reads them while the window changes them from another goroutine, so a reader holds a whole set without a lock on the path every firing takes | A map edited in place under a lock taken on every firing |
| A moment switched off while it waits is let go when it is reached | Nothing waiting is edited from the window's goroutine; the scheduler and the tick ask the switch as they come to it | Removing it from the queue at the press |
| Model files found by Go in `models/` beside `go.mod`, filled from a pinned list | Every machine finds them the same way with nothing to set; the rule that a file must match its published SHA-256 lives once, in Go (Oliver, 2026-09-14) | An environment variable naming a folder, with the checksums checked a second time in PowerShell |
