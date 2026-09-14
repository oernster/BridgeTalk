# Plan: machine voices

Open work only. Each milestone ends with the gate green (`test.ps1`: gofmt, go vet, the whole suite
and the coverage floor) plus staticcheck, then a commit block. Built inside out, as REQUIREMENTS.md
section 10 says: domain, then application, then infrastructure, then user interface.

## What exists today

- `internal/domain/machinevoice` holds the 28 voices offered, the accent each speaks with and the name
  each is shown by (FR-508, FR-510, FR-528).
- `internal/domain/speech` holds the 114 speech sound symbols the model reads with their numbers,
  copied from its tokenizer file by a script; `TestTheSymbolTableIsTheModelsOwn` in `tests/structural`
  holds the table to `models/tokenizer.json`. It turns speech sounds into those numbers, refusing
  more than 510 (FR-506). It reads the spellings a line gives, refusing every broken form (FR-529,
  FR-531).
- `internal/domain/script` holds the script checked against the cue table (FR-503 to FR-505,
  FR-531). `config` embeds `script.toml`, which holds three lines for each of the 256 cues. A
  structural test reads it through those rules and fails the build on a cue without lines (FR-507).
- `tools/sounds` makes every line's speech sounds in each accent with misaki in its own venv and
  saves them to `sounds.toml`, which `config` embeds beside the script. The structural test checks
  them (FR-506, FR-532 to FR-534). Run `go run ./tools/sounds` from the repository root after
  changing `script.toml`.
- `internal/domain/making` gives each line's key from its speech sounds, the style file and the
  model (FR-513). Set against the keys on disk, it lists the lines still to make and counts how many
  are current and how many cues are served (FR-511, FR-512, FR-515, FR-522).
- `services.MakingService` makes a cast machine voice's missing lines over the ports in
  `ports/making.go`, tested with hand-written fakes (FR-511, FR-512, FR-514, FR-516, FR-518 to
  FR-520, FR-527, FR-530). The audio source a cast answers with plays current made lines alone.
  `script/scripttest` builds a voiced script for every suite that needs one.
- `library.Catalogue` answers from `ports.AudioSource` under the name it is given (FR-501); a
  scanned `library.Voice` is the one implementation. `catalogueOf` in `voices.go` builds it for a
  recorded voice; `session.speakWith` in `main.go` rebuilds it over either kind of voice on every cast.
  FR-215's files figure is `Voice.Files`, since only a voice on disk has files.
- `ports/making.go` declares what a machine voice is made through: `SpeechMaker`, `VoiceFiles`
  (answering a `Material` of style and digests) and `MadeLines`. `services/makingtest` holds the
  hand-written fakes every suite makes lines over.
  `speech.Style` holds a voice's style file and chooses the row for a line's symbol count.
- `internal/infrastructure/voicefiles` reads the model, ONNX Runtime and a voice's style file from
  one folder laid out flat as `model.onnx`, `onnxruntime.dll` and `<id>.bin`. It refuses a file that
  is missing or cannot be read, naming it once (FR-519). It gives the digests made lines are keyed by
  (FR-513). `voicefiles.StyleFile` is the one home of a style file's name.
- `internal/infrastructure/modelfiles` embeds `models.toml`, the list of those files plus the
  tokenizer file with pinned addresses, sizes and published SHA-256s (FR-535): Hugging Face revision
  `1939ad2a`, ONNX Runtime release v1.23.2. `go run ./tools/models` fills `models/` at the repository
  root (FR-536, FR-537); `-check` downloads nothing (FR-538). The folder is found by walking up to
  `go.mod` in `internal/infrastructure/reporoot`, which the structural tests use too. On 2026-09-14
  the tool downloaded the 26 style files this machine lacked; `models/` now holds all 31 and `-check`
  passes. `modelfilestest.Require` hands a test the folder, skipping where a file is missing and
  failing where one differs (FR-538).
- `internal/infrastructure/speechmodel` implements `ports.SpeechMaker` through ONNX Runtime's C API
  with cgo disabled: the OrtApi v23 table read by position, the library loaded by its full path. The
  model is loaded when the first line is made, then kept until `Close`; a load that fails is tried
  again on the next line. Off Windows every line fails with `ErrUnsupported`. On 2026-09-14 it made the
  shipped `Docked` line for `bf_emma` from the real model files.
- `tests/machinevoice` makes the shipped script for `bf_emma` with the real model, store and making
  service, failing over NFR-P-203's ten minutes or NFR-C-502's 60 MB. It carries the `benchmarks`
  build tag: `./test.ps1 -Benchmarks` and every build run it, the everyday gate does not (Oliver,
  2026-09-14). It skipped while the script lacked lines for any cue; it has not yet been run over the
  complete script.
- `internal/infrastructure/madelines` keeps made lines as mono 16-bit FLAC at 24 kHz, one folder a
  voice, written to a part then renamed (FR-517, FR-526). Its frame headers leave the rate to the
  stream info, so the FLAC library logs nothing when the player decodes a made line. It deletes every voice's lines but one
  (or all of them), naming what it cannot delete (FR-527, FR-530). Its folder sits in the product's local
  data folder, which `internal/infrastructure/appdata` finds for it and for `library` (FR-523).
- `tomlfile.Decode` is the one strict TOML reader; `config` embeds `cues.toml`. The script follows both.
- Playback already decodes `.flac`. `mewkiz/flac` is an indirect dependency; writing FLAC makes it
  direct.
- Casting a machine voice is wired behind the facade. `CastMachineVoice` in `machine.go` casts through
  the making service and confirms in a current made line (FR-511, FR-519, FR-521); casting a recorded
  voice deletes every made line (FR-527); either kind is kept apart from the other (FR-540). A run
  opens with the kept machine voice, falling back to a recorded one with a warning (FR-512, FR-541);
  a rescan keeps it (FR-542); closing stops making, then releases the model. `newMaking` in `main.go`
  reads the model files from `models` beside the executable (FR-539), refusing every machine voice
  where that or the made lines' folder cannot be found (FR-523).
- The Cast pane lists the machine voices under their own heading by name (FR-508, FR-528), from
  `frontend/src/machineVoices.tsx`. The cast one reads its lines made and moments spoken, following
  a `making` event the poll loop sends on each change (FR-515, FR-522); failed lines, a stopped making,
  undeleted lines and a refused cast each get a callout (FR-518 to FR-520, FR-530). The state says
  whether the cast voice is a machine voice, so a recordings folder carrying its id is not marked cast.
- The tray's Voice menu lists the machine voices after the recorded voices under a separator, each
  cast by its id; its check mark and hover text match a voice by name and kind (FR-509, FR-540).
- Line counts that decide placement: `app.go` 376, `library/voice.go` 348, `main.go` 358. New code
  goes in new files.
- The probes from 2026-09-13 to 14 survive in an old session scratchpad: the ONNX Runtime caller and the
  FLAC writer. They are the
  starting point for the infrastructure, rewritten to the house standard rather than copied.
