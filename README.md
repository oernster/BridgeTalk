# Bridge Talk

A ship's voice for Elite Dangerous, speaking with recordings you supply.

Bridge Talk watches the game's journal and status file while you play. When something worth
saying happens, it plays one of your own recordings for that moment. It ships no audio of its own.

## Who it is for

- Commanders playing Elite Dangerous on Windows who hold recordings they may use and want them
  spoken in answer to the game.
- Anyone recording a voice for a ship who wants to see exactly which moments a set of recordings
  covers, then hear it before choosing it.

## Who it is not for

- Anyone wanting voice commands. It talks and never listens: there is no microphone, no speech
  recognition and no command and control.
- Anyone wanting sound straight away. Nothing plays until it is pointed at recordings.

## What it does

- **Casts a voice.** Every directory inside your recordings directory is a voice. Casting one makes
  it the ship's voice, which says something as it takes the part.
- **Plays one recording per moment.** Where a voice holds several takes for a moment, one is chosen
  at random, never the one that moment played last time.
- **Knows when to stay quiet.** An alert interrupts anything less urgent; a notice waits its turn;
  an ambient line is dropped when something is already waiting. Many moments hold a minimum
  interval between two firings. Mute silences every answer to the game.
- **Explains every decision.** The Status pane logs each one with its reason, including the ones
  that produced no sound.
- **Shows what a voice covers.** For any voice, cast or not, it lists the moments it has a
  recording for and the moments it has none for.
- **Auditions a voice** before it is cast.
- **Stays out of the way.** Put away, it keeps listening from the notification area and it can
  start when you sign in to Windows.
- **Leaves your recordings alone.** It never changes or removes anything in the directory you
  choose; the one thing it writes there is the empty folders you ask it to make. It makes no
  network requests of its own.

## Naming recordings

The quickest way is to let it name them. On the Cast pane, type a voice's name and press Make
folders: that voice gets a folder for every moment, each already named. Put each recording in the
folder for its moment (any file name will do), then press Look again. Where no recordings directory
is chosen yet, it makes them in its own `Recordings` folder (`%LOCALAPPDATA%\BridgeTalk\Recordings` on
Windows) and uses that folder from then on; setup never removes it.

Record each take in any program you like, such as Audacity (free, from
https://www.audacityteam.org/), saving it as WAV, MP3, FLAC or Ogg.
The Missing takes pane lists every voice still missing a recording. For the one chosen it shows each
missing moment with a line saying when it is heard, then the folder its audio file belongs in; Open
folder beside one opens that folder.

To name them by hand instead: every moment is named in the game's own words, as the journal and
the status file spell it. A
journal moment is the event name, followed by a field and a value where one field narrows it. A
status moment is the flag name followed by `Set` or `Cleared`. A folder for a moment writes each
dot as an underscore; a file named for a moment keeps the dots.

```
<recordings directory>/
  Alice/
    StartJump_JumpType_Hyperspace/   a folder holding any number of takes, dots written as _
      first.wav
      second.wav
  Bob/
    DockingGranted.wav               a file named for the moment
    DockingGranted.2.wav             a further take
    LightsOn.Set.mp3
```

A name has to match exactly apart from case. WAV, MP3, FLAC and Ogg files are played. The window's
guide, under Help, covers the rest.

## Built with

| Part | Choice |
|---|---|
| Backend | Go |
| Desktop shell | Wails v2 over WebView2 |
| Front end | React and TypeScript, built with Vite |
| Audio | beep over oto, decoding WAV, MP3, FLAC and Ogg in pure Go |
| Cue table | TOML, embedded in the executable |

## Running it

Build the setup program as `DEVELOPMENT_README.md` describes, then run it. Once the window opens,
choose your recordings directory on the Missing takes pane and cast a voice from the Cast pane. The journal
directory is found under your own profile until you choose one.

## Testing

```bash
./test.ps1
```

## Building

```bash
./build.ps1
```

`./build.ps1 -SkipInstaller` builds the application alone. The tests run first and the build does
not start unless they pass.

## Licence

GNU General Public License, version 3: see `LICENSE`. The application shows the same text under
Help, then Licence.
