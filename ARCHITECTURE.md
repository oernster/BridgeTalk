# Bridge Talk Architecture

A background application that watches Elite Dangerous journal and status data, decides that something
worth speaking about has happened, then plays a matching recording from a library the user supplies. It
is one-directional. There is no microphone, no speech recognition and no command and control; the
application talks to the commander and never listens.

It ships no audio of its own. Nothing in its own source reaches the network: no Go package here imports
one and the front end makes no request.

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
| Every exported type carries a doc comment | `TestEveryExportedTypeIsDocumented` | `boundary_test.go` |
| No colour value appears in `frontend/src` outside the theme token file | `TestColoursOnlyInTokens` | `colours_test.go` |
| The Missing takes purpose line reads at 7 to 1 or better against the surface and panel grounds in both themes | `TestThePurposeLineContrastsInBothThemes` | `contrast_test.go` |
| The product is named in one Go file; no Go string literal, front-end source or setup page file spells it | `TestTheProductIsNamedOnce` | `identity_test.go` |
| Both forms of the identity survive being a file name | `TestTheIdentityCanBeAFileName` | `identity_test.go` |
| Every disabled control wears the danger ring at all times | `TestEveryDisabledControlWearsTheDangerRing` | `rings_test.go` |
| Every region that is a keyboard stop because it scrolls wears a focus ring | `TestEveryScrollingRegionRingsForTheKeyboard` | `rings_test.go` |
| The shipped script and its saved speech sounds break none of the script's rules; every problem is named with its cue and its line | `TestTheShippedScriptHoldsNoProblem` | `script_test.go` |
| Every cue in the table has lines in the script | `TestTheScriptHoldsLinesForEveryCue` | `script_test.go` |
| The setup page applies the boxes it shows and handles the failure of every box that saves at once | `TestSetupAppliesTheBoxesItShows` | `setupchoices_test.go` |
| The setup page header repeats no title beneath the title bar | `TestTheSetupHeaderRepeatsNoTitle` | `setupheader_test.go` |
| The setup page, the scripts beside it and `setupScripts` stay in step: the page loads every script and every script has a place in the list | `TestTheSetupPageLoadsEveryScript` | `setupring_test.go` |
| The setup page's body, a keyboard stop because it scrolls, wears a focus ring | `TestTheSetupBodyRingsForTheKeyboard` | `setupring_test.go` |
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

- **Domain** (`internal/domain`: `cue`, `event`, `machinevoice`, `making`, `script`, `selection`, `speech`): pure Go. Values are validated on
  construction. No IO and no wall-clock reads: time arrives as a parameter, as the `now` taken by
  `CooldownGate.Allow` and `DedupeWindow.Fresh`, while randomness arrives through the injected
  `selection.Chooser`. Cue matching, take selection and the cooldown and dedupe arithmetic live here,
  so all of it is testable without a filesystem, a clock or an audio device. `machinevoice` holds the
  28 machine voices offered, the accent each speaks with and the name each is shown by; the list is the
  one home of the model's voice ids. `speech` holds the speech sound symbols the model reads with their
  numbers, copied from its tokenizer file by a script; it also reads the spellings a script line gives
  and chooses the row of a voice's style file the model reads beside a line.
  `script` checks the script against the cue table, naming the cue and the line behind every problem;
  `script/scripttest` builds a voiced script for the tests that need one.
  `making` keys each made line by its speech sounds, style file and model, then works out which lines
  a voice still has to make.
- **Application** (`internal/application`): the reaction, scheduling and making services plus the ports they
  depend on (`EventSource`, `AudioPlayer`, `VoiceCatalogue`, `AudioSource`, `Clock`, `SettingsStore`,
  `Reporter`). `AudioSource` answers the takes for a cue id and nothing else, so the catalogue serves
  any kind of voice without knowing where its audio came from (FR-501, FR-502). A machine voice is made
  through three more: `SpeechMaker` loads the model and turns a line's numbers and style row into samples, `VoiceFiles`
  reads the files a voice is made from, refusing one that is missing (FR-519); `MadeLines` keeps the
  made lines (FR-517, FR-523, FR-527). A cast makes only its confirmation's lines (FR-511); every
  other line is made the first time its cue fires. The audio source a cast answers with is also a `CueMaker`, through which the
  reaction service has a cue's lines made next when it fires with none; the cue waits for them for up
  to 2 seconds, handed over on the poll tick (FR-514). It never imports Infrastructure or the Wails
  runtime.
- **Infrastructure** (`internal/infrastructure`): concrete adapters behind those ports. The journal tail
  reader (`journal`), the status-flag watcher (`status`), the voice library scanner, catalogue and
  folder maker (`library`), the audio engine (`audio`), the cue table, the script with its saved speech sounds and the settings store (`config`),
  the strict reading both TOML files share (`tomlfile`), the files a machine voice is made from
  (`voicefiles`), the made lines kept as 16-bit FLAC (`madelines`), the product's local data folder
  both it and the default recordings directory sit in (`appdata`), the log each run leaves in that
  folder (`runlog`),
  the Windows tray (`taskbar`), keyboard focus for the web view plus opening a folder in File Explorer
  (`window`) and the per-user install work behind the setup program (`setup`). Never imported by
  Domain or Application.
- **UI**: the React front end plus a Wails facade in package `main`, which calls the Application
  services and maps what they return into the shapes in `dto.go`.
- **Outside the layers**: `internal/product` holds the product's name and `internal/refusal` words a
  refusal over a path. Each is a leaf that several layers read, so it belongs to none of them.
- **The sounds tool** (`tools/sounds`): a command run while developing, never shipped. It reads the
  script through `config` and `speech`, asks misaki in the tool's own Python venv for every line's
  speech sounds in each accent and writes `sounds.toml`. Python only turns text into speech sounds;
  the spelling rules and the file's shape stay in Go, so each keeps one home.

## Composition root

`main.go` is the composition root. It constructs the infrastructure adapters, injects them into the
application services by constructor and hands the assembled facade to Wails. `newMaking` there builds
the making service over the model files beside the executable and the made lines' folder; where
either cannot be found, every machine voice is refused with why rather than the application refusing
to start. No service is held in a
package-level variable and there is no service locator or auto-wiring. The structural test whitelists
`main.go` and `app.go`: no other file may import both the application services and infrastructure. The
facade is spread over the root files beside them, `settings.go`, `cast.go`, `machine.go`, `folders.go`, `checklist.go`, `audition.go`, `audition_machine.go`,
`voices.go`, `identity.go` and `window_life.go`, each a slice of the surface it would otherwise outgrow the size limit
carrying; the wire shapes are in `dto.go`.

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
                       | audio, config, taskbar,     |
                       | window, setup               |
                       +-----------------------------+
```

## The cue model

Game semantics and a person's recordings are separate axes, so they are kept apart. `FSDJump` is a
fact about Elite Dangerous and is true for every voice; which file answers it is a fact about one
person's library. The two meet at the cue id and nowhere else.

**The cue table, `internal/infrastructure/config/cues.toml`, embedded in the binary.** Each entry names
a source, the journal event or status flag it listens for, an optional edge, an optional predicate over
the payload, a priority and a cooldown in seconds. A missing priority reads as `ambient`; a missing
cooldown leaves the cue limited by the dedupe window alone.

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
It is the only grouping the vocabulary needs: the audition pane reads it rather than keeping a second
taxonomy in step. A list of cues does not: the Missing takes pane and the breakdown dialog show each cue
under its full title alone (FR-233), since a heading read from the first segment would repeat the start
of every title beneath it.

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
`FSSSignalDiscovered`). At most one payload field narrows an event. The status cues are every flag the
status watcher decodes on both edges, every `GuiFocus` value, every pip distribution and the fire group.
`config_test.go` holds every shipped id to that spelling: its first segment is what the cue listens for.

Every table is built through `cue.New`, which refuses a definition it cannot honour: an empty id, an
unknown source or edge, a journal or application cue naming no event, a status cue naming no flag, an
unknown priority, a negative cooldown, an id ending in a segment of digits (FR-219, below), an id
ending in a dot or a space (FR-222, below) or an id holding an underscore (FR-230, below).
`config.LoadCueTable` also refuses a duplicate id, a key it does not hold and a cue whose purpose is missing or blank (FR-231). It accepts a path to a table on disk, which would
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

## Resolving a cue

An event resolves to a cue through the table. The reaction service asks the catalogue for the cast
voice's takes for that cue; the domain `Picker` chooses one, avoiding the take it chose for the same cue
last time; the scheduler hands that single clip to the player. One cue plays one file.

There is no fallback chain. A voice that recorded nothing for a cue answers it with silence and the
reaction list records why, because a wrong line delivered confidently is worse than silence. A voice is
expected to be complete, so a gap is something to record rather than something to paper over.

**The acknowledgement.** Casting a voice plays one take of the single cue whose source is the application
rather than the game. The catalogue finds that cue by its source rather than by its id, so the id is
written in the cue table and nowhere else. A voice that recorded no acknowledgement is silent when cast
rather than broken. It goes straight to the player rather than through the scheduler, so it cannot queue
behind the ship; unlike an audition it respects the mute.

**Coverage.** The breakdown dialog behind each cast row lists the cues a voice serves and the cues it does
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

Each event goes through one decision in `ReactionService.Handle`, in this order: resolve the cue; drop a
repeat of the same cue inside the 900 millisecond dedupe window; drop a cue still inside its cooldown;
ask the catalogue for takes and record the cue as unserved where there are none; drop it while muted;
pick one take; submit it to the scheduler. Every step after the first that ends in silence is reported
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

Pre-emption is a cut, not a fade. The player stops the current clip outright and the next starts; there
is no mixer, no ducking and no crossfade. What the player does carry is a single gain applied per audio
buffer, which is what makes the volume slider audible mid-clip.

An audition and the acknowledgement go to the player directly. The acknowledgement starts through `Play`,
so it stops whatever the player was doing, a scheduled clip included; the cut request is not resumed. An
audition starts through `PlayIfIdle`, so it never stops anything.

**Threading.** The audio library runs its own output thread and nothing on it touches the UI. The player
reports completion over a channel. The facade's poll loop selects over that channel, the tray's command
channel and the poll ticker; on completion it tells the scheduler and announces the playback state to the
page, so nothing on the audio path reaches Wails directly.

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
and `-unbound` exit with an error where the recordings directory holds no voice.

## Ambient chatter: not built

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
+-------------------------------------------------------------------------------------------+
| File   Audio   Settings   Help                                                            |
+-------------------------------------------------------------------------------------------+
| [Cast] [Audition] [Status] [Missing takes] [Settings]   [Volume] [Mute] [Theme] [Guide]   |
+-------------------------------------------------------------------------------------------+
|                                                                                           |
|   main pane: a switched view, not a stack of modal dialogs                                |
|                                                                                           |
+-------------------------------------------------------------------------------------------+
```

The nav band is one flat row with the two groups separated by a stretch, so layout order is reading
order. The main pane switches between Cast, Audition, Status, Missing takes, Settings and Guide; it opens
on Cast. The menu bar repeats the ways in: File holds Quit; Audio holds Cast, Audition, Missing takes and
Mute; Settings holds the
pane and the theme; Help holds the guide, the licence and About.

**Machine voices on the Cast pane.** `frontend/src/machineVoices.tsx` lists them under their own heading
after the recorded voices, one cast control a row, in the row a recorded voice uses. The cast one reads
how far making has got: it asks `Making` once, then follows the `making` event, which the poll loop
sends only when the answer has moved, so an idle tick says nothing. Failed lines, a stopped making,
undeleted lines and a refused cast each get a callout beneath the list. `StateDTO.MachineVoice` says
which kind of voice is cast, since a recordings folder may carry a machine voice's id. The words both
lists share, the part a cast voice plays and the counted figures, live in `frontend/src/castWords.ts`;
where making stands before anything is made lives in `frontend/src/making.ts`, apart from `api.ts`,
because a test replaces that module whole.

Four surfaces are modal, all built on one dialog shell so none arrives with rules of its own: About, the
licence, the close choice and the breakdown of what one voice speaks for. Each opens focused on its first
control; the close choice lists Minimise first, since Enter straight after pressing the cross must not
mean stop. The licence dialog shows the `LICENSE` file itself, embedded at build time, so the terms shown
and the terms the source carries cannot differ; About names the licence in a sentence.

**Status.** The cast voice, how many cues it serves out of the table, the journal directory, the status
file, whether playback is live or muted, a line where no audio device was found and a line where the
device has run dry. Beneath them, the reaction list: each decision with its time, its outcome and its cue,
beside the clip's file name, left blank where there was none. The facade keeps the last 200. That list is
how a cue that never speaks gets diagnosed.

**Casting, not selecting.** Choosing a voice is casting a part, so each row names the act it offers:
"Cast Iris"; "Grace is cast as your ship's voice" for the one already in the role. Beneath the name
the row gives the number of recordings the voice holds that answer a cue. Every voice listed can be cast,
because a directory that resolved no recording never becomes a voice. A button at the end of the row
opens the breakdown dialog. Beneath the rows, Make a voice holds a name box with Make folders and Refresh.

**Missing takes.** The recordings directory row with its Browse, then a chooser offering every voice
folder still missing a recording, empty folders included, since a voice made with Make folders holds
nothing until its first take (FR-316). It opens on the voice already chosen, else the cast voice, else the
first. For the voice chosen it lists each missing moment by title, its purpose beneath and the folder its
take belongs in, with Open folder beside it.

**Audition.** The cast pane says what a voice covers; the audition pane lets it be heard. Groups come
from the cue vocabulary's own first segment, the moment in the game's own words, so no second taxonomy is
kept in step with the cue table; a group with no takes is not offered. Each group button plays one clip
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
list shut. A scrolling region joins the ring only while it overflows. The reaction list
is a single stop whose rows are walked with Up and Down. A menu title opens on Down, Enter or Space;
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
pushes the mute state and the cast voice back through two atomics that the menu builder and the tooltip
read. Each push also posts a message to the tray's own window, so the hover text is sent again from the
tray thread once the state has changed. A menu choice never sends it itself: the choice has not been
acted on when the menu returns. A callback invoked from the tray thread would be running on the wrong thread for everything it
wanted to touch, which is a failure mode this design removes rather than manages.

The left button asks for the window on a single click and on a double; the right button opens the menu:
a Voice submenu, Open, Mute and Quit. The tooltip names the cast voice and says when it is muted. The
Voice submenu lists the voices found at startup; it is not rebuilt when a new recordings directory is
chosen or Refresh finds more. The machine voices follow them under a separator. A choice carries whether
it is a machine voice, since a recordings folder may carry a machine voice's id: the check mark and the
tooltip match a voice by name and kind, the kind pushed beside the name through a third atomic. A
machine voice chosen there reaches `CastMachineVoice` rather than `SelectVoice`.

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

**It is five files.** `index.html` carries the markup, `setup.css` the palette and layout, `setup-ring.js`
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

Everything is per user, so no step needs administrator rights: the files under
`%LOCALAPPDATA%\Programs\BridgeTalk`, the Start Menu shortcut under `%APPDATA%`, the Desktop
shortcut in the user's own Desktop directory and the install record and login entry under
`HKEY_CURRENT_USER`. The shortcut boxes apply in both directions, so unticking one removes a shortcut
rather than leaving a stale one behind. Setup copies itself into the install directory as
`uninstall.exe` and registers that copy as the uninstaller and as the Modify target, with `NoModify` and
`NoRepair` both zero.

Uninstall removes the shortcuts, the login entry and the install record, then the folder the made lines
are kept in whatever is ticked (FR-525), leaving the product's data folder around it, which can hold the
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
| Settings | `settings.json` in `BridgeTalk` under Go's user configuration directory (`%APPDATA%` on Windows): both directories and the cast voice, kept as a recorded voice's name or a machine voice's id with the other forgotten. Choosing either directory writes both, as does Make folders adopting the default recordings directory; casting writes the voice |
| Cue table | embedded in the binary |
| Script | `script.toml`, embedded in the binary beside the cue table |
| Saved speech sounds | `sounds.toml`, embedded in the binary beside the script; written by `go run ./tools/sounds`, never by hand |
| Model files on the build machine | `models/` at the repository root, found by walking up to `go.mod` and filled by `go run ./tools/models` from the list embedded in `internal/infrastructure/modelfiles`; not committed |
| Model files when running | `models` beside the executable, the one folder the application reads them from; `voicefiles.Folder` names it for the build machine's folder too |
| Made lines | `Made lines` in the product's local data folder (`%LOCALAPPDATA%\BridgeTalk` on Windows), one folder for the cast machine voice; never under the recordings directory |
| Theme and volume | the page's own storage, inside the web view's folder `%APPDATA%\BridgeTalk.exe` |
| Installed files | `%LOCALAPPDATA%\Programs\BridgeTalk`, per user |
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

- **Fatal, before any window:** a cue table that fails to load.
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
  fails to decode is skipped after being logged as played.
- **Recorded in the reaction list:** a repeat inside the dedupe window, a cue in cooldown, a cue the voice
  has no takes for and a request dropped by the mute or by the priority policy, beside what was queued and
  what played.
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
`Check` is the one statement of the rule, which the facade, setup and config tests all hold their
refusals to. It sits under `internal` beside `product` for the same reason: the library, config, journal,
status and setup packages and the facade all read it, so it belongs to no layer. A refusal the window
shows writes its path with `%s` rather than `%q`, which doubles every Windows separator.

## Quality enforcement

- Structural tests enforce the layer direction, domain purity, the module-size limit and its danger
  band, the composition-root whitelist, the declared bound surface, the wire contract and the rules that
  keep a value in one home: colours in the theme tokens, the product name in `internal/product`, the
  game's own words in the cue table. They also hold the shape of every cue id, the purpose line's
  contrast, the setup page's boxes and header and the speech sound table in `internal/domain/speech`
  against the model's tokenizer file in `models/`. Another lets an address handed to a DLL become a
  uintptr only where the call into it is made. The invariant table above lists every one of them with
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
  to a combined 100% coverage, then holds each other measured package to a floor of its own: the root
  package 75%, `audio` 80%, `audiotest` 86%, `madelines` 98%, `modelfiles` 99%, `runlog` 51%, `setup` 61%, `speechmodel` 91%, `taskbar`
  67%, `tools/sounds` 38%, `tools/models` 48%, `tools/payload` 53%, with `appdata`, `config`, `journal`, `library`, `reporoot`,
  `status`, `tomlfile`, `voicefiles`, `wholefile` and `internal/refusal` at 100%. `internal/infrastructure/window`,
  `installer` and `modelfilestest` carry no floor: the first two have nothing a test can reach without
  the platform behind them; the last is test support exercised by the `modelfiles` tests.
  TESTING.md names what each shortfall is.
- `build.ps1` runs `test.ps1 -Benchmarks` before it builds and offers no switch to skip it, so every
  build also measures making a complete script against NFR-P-203 and NFR-C-502. `wails build` runs the
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
| One cue plays one file | Every recording answers exactly one moment | Several files played in turn for one moment |
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
| Model files found by Go in `models/` beside `go.mod`, filled from a pinned list | Every machine finds them the same way with nothing to set; the rule that a file must match its published SHA-256 lives once, in Go (Oliver, 2026-09-14) | An environment variable naming a folder, with the checksums checked a second time in PowerShell |
