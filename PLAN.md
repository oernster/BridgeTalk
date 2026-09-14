# Plan: machine voices

Open work only. Each milestone ends with the gate green (`test.ps1`: gofmt, go vet, the whole suite
and the coverage floor) plus staticcheck, then a commit block. Built inside out, as REQUIREMENTS.md
section 10 says: domain, then application, then infrastructure, then user interface.

## What exists today

- `internal/domain/machinevoice` holds the 28 voices offered, the accent each speaks with and the name
  each is shown by (FR-508, FR-510, FR-528).
- `internal/domain/speech` holds the 114 speech sound symbols the model reads with their numbers,
  copied from its tokenizer file by a script. It turns speech sounds into those numbers, refusing
  more than 510 (FR-506). It reads the spellings a line gives, refusing every broken form (FR-529,
  FR-531).
- `internal/domain/script` holds the script checked against the cue table (FR-503 to FR-505,
  FR-531). `config` embeds `script.toml`, which holds the `Docked` example alone. A structural test
  reads it through those rules; FR-507's test reports progress until `scriptComplete` is switched on.
- `tools/sounds` makes every line's speech sounds in each accent with misaki in its own venv and
  saves them to `sounds.toml`, which `config` embeds beside the script. The structural test checks
  them (FR-506, FR-532 to FR-534). Run `go run ./tools/sounds` from the repository root after
  changing `script.toml`.
- `internal/domain/making` gives each line's key from its speech sounds, the style file and the
  model (FR-513). Set against the keys on disk, it lists the lines still to make and counts how many
  are current and how many cues are served (FR-511, FR-512, FR-515, FR-522).
- Nothing else of machine voices is built.
- `library.Catalogue` is built straight over a scanned `library.Voice`; `session.useVoice` in
  `main.go` rebuilds it with the reaction service on every cast. There is no audio source port yet
  (FR-501).
- `tomlfile.Decode` is the one strict TOML reader; `config` embeds `cues.toml`. The script follows both.
- Playback already decodes `.flac`. `mewkiz/flac` is an indirect dependency; writing FLAC makes it
  direct.
- Line counts that decide placement: `app.go` 372, `library/voice.go` 370, `main.go` 318. New code
  goes in new files.
- The probes from 2026-09-13 to 14 survive in an old session scratchpad: the ONNX Runtime caller and the
  FLAC writer. They are the
  starting point for the infrastructure, rewritten to the house standard rather than copied.

## M6 Application: the port and the making service

1. FR-501 first, changing no behaviour. `ports.AudioSource` answers the takes for a cue id.
   `library.Voice` already has `Lookup(id)` and becomes the first implementation; the catalogue
   depends on the port. The suite stays green with only wiring edited.
2. Ports for what a machine voice needs: the speech maker (numbers plus style in, samples out), the
   made-line store (keys on disk, write whole, delete a voice's lines) and
   the voice's files (one missing or unreadable is refused, FR-519).
3. `MakingService` over those ports, tested with hand-written fakes. Making starts on cast and on
   start (FR-511, FR-512). It stops when another voice is cast, keeping what is made (FR-516). It
   carries on past a line that fails (FR-518) and stops on a write failure (FR-520). It deletes the
   previous voice's lines (FR-527) and says so when it cannot (FR-530). A machine voice's audio
   source answers only with current made lines (FR-514). The confirmation plays when a current line
   exists (FR-521).

## M7 Infrastructure

Each part behind its port. Windows code sits behind a build tag with stubs beside it, so every
package still builds and vets on any platform.

- Made-line store: FLAC with fixed order 2, the Rice parameter from the mean residual and blocks of
  4096 (FR-526); written to a temporary file then renamed (FR-517); under the per user data
  directory, never the library root (FR-523). The data directory resolution in `library/root.go` is
  extracted and shared rather than repeated.
- ONNX Runtime caller through the OrtApi v23 table, no cgo.
- Voice files: the model, the 28 style files and ONNX Runtime. A missing
  one is refused by name (FR-519) through `internal/refusal`.
- A test compares the symbol table in `internal/domain/speech` with the model's tokenizer file where
  that file is present, so a new model cannot leave the table behind.
- **Reproduce before fixing:** the FLAC library's line per frame at 24 kHz, in an audio player
  test, before silencing it (FR-526 note).
- Tests needing the model files skip where they are absent: NFR-P-203 (768 lines within 10 minutes)
  and NFR-C-502 (60 MB).

## M8 Composition root and user interface

- `session.useVoice` builds the catalogue over either kind of audio source. New facade methods go in
  a new root file, since `app.go` is at 372 lines.
- Cast pane: machine voices listed apart (FR-508) by name (FR-528), how far making has got (FR-515),
  line and write failures (FR-518, FR-520, FR-530) and completeness (FR-522). Tray Voice menu
  (FR-509).
- Wire shapes stated in Go and TypeScript, compared by the wire structural test.
- ARCHITECTURE.md gains the layers, ports, data location and design decisions as each part lands.

## M9 Setup

- The setup program carries every file a machine voice is made from, downloading nothing (FR-524).
- Uninstall deletes the made lines (FR-525).
- **Measure before M7 starts:** the setup program's size, build time and memory while it runs, today
  against the same with about 339 MB of model files embedded. The answer may change how those files
  travel.

## Content track, alongside M3 onwards

`script.toml`: three lines for each of the 256 cues, 768 in all (FR-505, FR-507). Claude drafts a
group of cues at a time from each cue's purpose; Oliver reviews each group, then the sounds tool runs over it. A line is reworded where
its saved speech sounds show a misread word; a spelling is given only where no rewording serves
(FR-529). When the last group lands, `scriptComplete` in `tests/structural/script_test.go` is
switched on, so FR-507 fails the build from then on.

## Waiting on Oliver

- Where the model files live on the build machine, since the model alone is 310.5 MB. A
  recommendation comes before M7.
- Approval before downloading any file not already on this machine, each named with its source and
  size. Which files those are is measured before M7.
