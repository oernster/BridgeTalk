# Plan: machine voices

Open work only. Each milestone ends with the gate green (`test.ps1`: gofmt, go vet, the whole suite
and the coverage floor) plus staticcheck, then a commit block. Built inside out, as REQUIREMENTS.md
section 10 says: domain, then application, then infrastructure, then user interface.

## What exists today

- No machine voice code. The domain holds `cue`, `event` and `selection`.
- `library.Catalogue` is built straight over a scanned `library.Voice`; `session.useVoice` in
  `main.go` rebuilds it with the reaction service on every cast. There is no audio source port yet
  (FR-501).
- `tomlfile.Decode` is the one strict TOML reader; `config` embeds `cues.toml`. The script follows both.
- Playback already decodes `.flac`. `mewkiz/flac` is an indirect dependency; writing FLAC makes it
  direct.
- Line counts that decide placement: `app.go` 372, `library/voice.go` 370, `main.go` 318. New code
  goes in new files.
- The probes from 2026-09-13 to 14 survive in an old session scratchpad: the ONNX Runtime caller, the
  eSpeak NG caller, the dictionary lookup with misaki's rules and the FLAC writer. They are the
  starting point for the infrastructure, rewritten to the house standard rather than copied.

## M1 Domain: the machine voices

Package `internal/domain/machinevoice`.

- The 28 ids (FR-508); accent and sex read from the id's prefix (FR-510); the name on screen (FR-528).
- Tests: every id is named as FR-528 says; an unknown prefix is refused.

## M2 Domain: speech sounds

Package `internal/domain/speech`.

- The 115 symbols the model reads, with each symbol's number, as a committed table. An
  infrastructure test compares it with the model's tokenizer file where that file is present.
- Speech sounds to the model's numbers, refusing more than 510 (FR-506).
- Reading `[word](/sounds/)` and `[word](/British/American/)` out of a line (FR-529). Every broken
  form FR-531 lists is refused with a reason.

## M3 Domain plus structural tests: the script

Package `internal/domain/script`; `script.toml` embedded by infrastructure through `tomlfile.Decode`.

- A script is built from cue ids to lines against the cue table. An unknown cue (FR-504), a cue
  without three distinct lines (FR-505), a line over the limit (FR-506) or a broken spelling (FR-531)
  is refused, naming the cue and the line.
- Structural tests over the embedded file for FR-504, FR-505 and FR-531, each proved by planting a
  violation.
- FR-507 (every cue has lines) is a test that fails until the script is complete. It is switched on
  when the content track ends, not before.

## M4 Domain: pronunciation

In `speech`. Words to speech sounds in each accent: misaki's gold then silver dictionaries, the -s,
-ed and -ing stem rules, the special cases, the next-vowel flag and the final flap and glottal stop
replacements. A word no dictionary holds goes to an injected fallback, eSpeak NG in infrastructure.
A spelling given in the line (FR-529) bypasses all of it.

- NFR-Q-501 against a committed reference file of misaki's own output for all 256 purposes.
- **Measure first:** how many of those words reach the fallback. If the reference test cannot pass
  without eSpeak NG, it becomes an infrastructure test that skips where eSpeak NG is absent; the
  requirement is amended to say so.

## M5 Domain: what to make

In `machinevoice` or a sibling package.

- A made line's currency key from the line's text, the style file's digest and the model's digest
  (FR-513).
- Given the script, a voice and the keys already on disk: the lines still to make (FR-511, FR-512),
  how many are current (FR-515) and how many cues have at least one (FR-522).

## M6 Application: the port and the making service

1. FR-501 first, changing no behaviour. `ports.AudioSource` answers the takes for a cue id.
   `library.Voice` already has `Lookup(id)` and becomes the first implementation; the catalogue
   depends on the port. The suite stays green with only wiring edited.
2. Ports for what a machine voice needs: the speech maker (numbers plus style in, samples out), the
   fallback pronouncer, the made-line store (keys on disk, write whole, delete a voice's lines) and
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
- Dictionary loader for misaki's four English dictionaries.
- eSpeak NG caller, the fallback.
- ONNX Runtime caller through the OrtApi v23 table, no cgo.
- Voice files: the model, the 28 style files, ONNX Runtime plus eSpeak NG with its data. A missing
  one is refused by name (FR-519) through `internal/refusal`.
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
  against the same with about 370 MB of model files embedded. The answer may change how those files
  travel.

## Content track, alongside M3 onwards

`script.toml`: three lines for each of the 256 cues, 768 in all (FR-505, FR-507). Claude drafts a
group of cues at a time from each cue's purpose; Oliver reviews each group. A line is reworded where
the pronunciation test shows a misread word; a spelling is given only where no rewording serves
(FR-529).

## Waiting on Oliver

- Where the model files live on the build machine, since the model alone is 310.5 MB. A
  recommendation comes before M7.
- Approval before downloading any file not already on this machine, each named with its source and
  size. Which files those are is measured before M7.
