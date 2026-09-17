# <img width="128" height="128" alt="application-icon" src="https://github.com/user-attachments/assets/482dbfad-c806-46a7-afec-cfcb0ebf4e87" /> Bridge Talk

A ship's voice for Elite Dangerous, speaking with a machine voice of its own or with recordings you
supply.

Bridge Talk watches the game's journal and status file while you play. When something worth
saying happens, the cast voice speaks for that moment. Start with one of its 28 machine voices, whose
lines it makes on your own machine; record a voice of your own whenever you like. It ships no recordings.

## Who it is for

- Commanders playing Elite Dangerous on Windows or Linux who want their ship to talk straight away:
  cast a machine voice and it speaks, with nothing to record first.
- Commanders who hold recordings they may use and want them spoken in answer to the game.
- Anyone recording a voice for a ship who wants to see exactly which moments a set of recordings
  covers, then hear it before choosing it.

## Who it is not for

- Anyone wanting voice commands. It talks and never listens: there is no microphone, no speech
  recognition and no command and control.
- Anyone wanting a machine voice in another language. The 28 machine voices speak British and
  American English only.
- Anyone on macOS. It is built for Windows as a setup program and for Linux as a flatpak.
- Anyone wanting a separate output device for it. There is no device choice: it speaks through
  whichever output device the system is set to use.

## What it does

- **Speaks with a machine voice, with nothing to record.** 28 voices speak British and American
  English, female and male. On the Cast pane they sit in a panel for each accent and sex; cast one
  and it becomes the ship's voice, with a card above the panels saying how far its lines are made.
  Each line is made on your own machine the first time its moment happens, then kept, so nothing is
  downloaded and nothing is sent anywhere. The Audition pane plays any of them before you cast it.
- **Casts a recorded voice.** Every folder in your recordings directory that holds at least one recording
  for a moment, named for it or in its folder, is a voice. Cast one from the Cast pane or from the Voice menu on the
  notification-area icon and it becomes the ship's voice. Where it holds a recording for
  `Cast.Confirmed` and mute is off, it plays that as it takes the part. The choice is remembered
  for the next run.
- **Takes voices from plugins.** A plugin is a library file placed in the plugins folder, offering
  voices whose audio already sits on your machine. On the Cast pane each plugin's voices stand in a
  section of their own above the machine voices, in the groups the plugin names. Its voices are
  auditioned on the Audition pane, cast from the Cast pane or the notification-area menu like any
  other and remembered for the next run; a voice whose audio is missing is listed with the reason its
  plugin gave. The Missing takes pane lists what the cast plugin
  voice has no take for. A plugin that will not load is named in the run log with the reason and
  never stops the application starting.
- **Plays one recording per moment.** Where a voice holds several takes for a moment, one is chosen
  at random, never the one that moment played last time.
- **Answers pirates.** When a pirate makes itself known, scans your cargo, loses interest, finds
  your hold empty, starts an interdiction or declares its attack, the ship's voice speaks for that
  moment, whichever version of the message the game sent. It never reads the message's own words
  aloud.
- **Knows when to stay quiet.** An alert interrupts anything less urgent; a notice waits its turn;
  an ambient line is dropped when something is already waiting; a flavour line is dropped when
  anything is playing or waiting. Many moments hold a minimum interval between two firings. The
  same moment raised twice within 900 milliseconds is heard once. Mute silences every answer to the game.
- **Speaks only for the moments you choose.** The Chatter pane lists the 262 moments the game raises
  under twelve categories, each with a switch. A moment switched off stays quiet whichever voice is
  cast: it starts no cooldown and makes no line. One switched off while it waits its turn is let go;
  a take already playing is not cut short. Switch all on, Switch all off and a switch for each
  category sit above the list and change many moments at once, asking first. Station traffic
  (docking answers, welcomes and the no fire zone) has a switch of its own, apart from the other
  messages non-player characters send. Each category's name above the list carries a down arrow;
  pressing it moves the list to that category, opening it where it was collapsed. Each category's
  heading in the list carries a disclosure mark that turns while the category is collapsed; pressing
  the heading collapses the category or opens it again. The switches are kept between runs; every
  category is open again each time the pane opens.
- **Explains every decision.** For every moment the game raises that has a cue, the Status pane
  logs what became of it (played, queued, making, dropped, cooldown, duplicate, off or unbound), including the
  ones that produced no sound. It keeps the latest 200. The same pane names the journal directory
  and status file being watched and says when no audio device was found or the device ran dry.
- **Shows what a recorded voice covers.** For any recorded voice, cast or not, the mark beside it
  on the Cast pane opens a list of the moments it has a recording for and the moments it has none
  for. A machine voice or a plugin voice has no such mark.
- **Lists the takes still missing.** The Missing takes pane offers every voice folder still missing
  a recording. For the one chosen it lists each missing moment with a line saying when it is heard,
  then the folder its take belongs in.
- **Auditions a voice** before it is cast. Each button stands for the moments that share one journal
  event or status flag (Start jump is one) and plays one take drawn at random from those switched
  on in Chatter; the buttons stand under Chatter's categories. The buttons are held while anything is playing or
  being made, the ship's own reactions included, so a press never cuts a clip short; Stop ends what
  is playing. An audition ignores the mute.
- **Stays out of the way.** Closing the window asks whether to put it away or quit. Put away, it
  keeps listening from the notification area, whose menu opens the window, mutes, switches voice
  or quits. It can start when you sign in, waiting in the notification area; turn that on in
  Settings; on Windows the setup program offers it too. On a Linux desktop that offers no notification
  area, closing the window quits.
- **Remembers how you like it.** The volume slider and the light or dark theme on the band are kept
  between runs.
- **Leaves your recordings alone.** It never changes or removes a file in your recordings
  directory. The only things it writes there are empty folders: a voice's folders when you press
  Make folders and a moment's folder when you press Open folder beside it. Uninstalling does not
  touch it. It makes no network requests of its own.

## Built with

| Part | Choice |
|---|---|
| Backend | Go |
| Desktop shell | Wails v2 over WebView2 on Windows and webkit2gtk on Linux |
| Front end | React and TypeScript, built with Vite |
| Audio | beep over oto, decoding WAV, MP3, FLAC and Ogg Vorbis in pure Go |
| Machine voices | the Kokoro-82M model, run through ONNX Runtime |
| Cue table | TOML, embedded in the executable |
| Linux package | a flatpak on the GNOME runtime |

## Getting it

Both downloads are on [ernster.dev/BridgeTalk](https://ernster.dev/BridgeTalk/).

On Windows, download the setup program and run it. Everything it writes is for your own Windows
account, so it never asks for administrator rights. It installs under your own application data
unless you choose another folder on its first screen.

On Linux, download `BridgeTalk.flatpak` and install it for your own account:

```bash
flatpak install --user BridgeTalk.flatpak
```

Then start Bridge Talk from your applications menu. The sandbox is granted no network access.

Once the window opens, cast a machine voice from the Cast pane. To use recordings of your own,
choose your recordings directory on the Missing takes pane, then cast that voice from the Cast pane.

The journal directory is `Saved Games\Frontier Developments\Elite Dangerous` under your own profile
on Windows. On Linux it is looked for in that same place inside the game's Proton or Wine prefix,
with every place looked named where none holds it. Either way, choose another on the Settings pane
to override it. When the journal directory in use cannot be read,
the window opens anyway: the Status pane and the Settings pane say why until you choose one that
can be.

## Naming recordings

The quickest way is to let it name them. On the Cast pane, type a voice's name and press Make
folders: that voice gets a folder for every moment, each already named. Put each recording in the
folder for its moment (any file name will do), then press Refresh. Where no recordings directory
is chosen yet, it makes them in its own `Recordings` folder (`%LOCALAPPDATA%\BridgeTalk\Recordings`
on Windows, `~/.var/app/uk.codecrafter.BridgeTalk/data/BridgeTalk/Recordings` in the Linux flatpak)
and uses that folder from then on; uninstalling never removes it.

Record each take in any program you like, such as Audacity, saving it as WAV, MP3, FLAC or Ogg
Vorbis. On the Missing takes pane, Open folder beside a missing moment makes that moment's folder
where it is not there yet, then opens it.

To name them by hand instead: every moment is named in the game's own words, as the journal and
the status file spell it. A journal moment is the event name, followed by a field and a value where
one field narrows it. A status moment is the flag name followed by `Set` or `Cleared`. A folder for
a moment writes each dot as an underscore; a file named for a moment keeps the dots.

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

A name has to match exactly apart from case. Only files directly inside a moment's folder are
read. WAV, MP3, FLAC and Ogg Vorbis files are played. The window's guide, under Help, covers the
rest.

A voice can also hold a `voice.toml`, which is optional. It gives the voice the name it is shown by,
with a credit line beneath its figures on the Cast pane; its `[takes]` table lists recordings the
names above do not reach, as paths inside the voice's folder:

```toml
name = "Alice Hart"
credit = "Recorded by Alice, 2026"

[takes]
"IsInDanger.Set" = ["odd-name.wav", "alternates/another.wav"]
```

The voice's folder name is still what Settings remembers. A `voice.toml` that cannot be read is set
aside and the voice is found by its names alone.

## Testing

The gate checks the model files, formatting, vet and staticcheck, runs the Go suite and the front
end's lint, type check and tests, then holds each package to its coverage floor:

```powershell
./test.ps1
```

The front end's suites run with:

```powershell
npm --prefix frontend test
```

[TESTING.md](TESTING.md) covers what each coverage figure means and what is deliberately left
untested.

## Building

```powershell
./build.ps1
```

It runs the gate, then writes the application to `build/bin/BridgeTalk.exe` and the setup program
to `dist-installer/BridgeTalkSetup.exe`. On Linux, `bash build_flatpak.sh` builds and installs the
flatpak and writes `BridgeTalk.flatpak`. [DEVELOPMENT.md](DEVELOPMENT.md) sets up a machine from
nothing, fetches the model files the build needs and lists the command-line options.
[ARCHITECTURE.md](ARCHITECTURE.md) explains the layering and the reasoning behind each decision.
[PLUGINS-GUIDE.md](PLUGINS-GUIDE.md) is the contract for writing a plugin: a library file offering
voices whose audio already sits on the machine the application runs on. Bridge Talk does not check
who wrote a plugin or whether it has been altered; a plugin runs with your own rights, so put in the
plugins folder only a plugin whose author you trust.

## Supporting

Bridge Talk is free and stays free. There is no paid tier, no licence key and no feature held back
behind a donation. If it has been useful, a donation supports its maintenance and continued
development.

<a href="https://www.paypal.com/ncp/payment/DVP73MPL9JPSU"><img src="docs/images/donate.png" alt="Donate to Bridge Talk" width="120"></a>

## Licence

GNU General Public License, version 3, with a plugin exception: see `LICENSE`. The exception lets a
plugin that speaks only the interface in [PLUGINS-GUIDE.md](PLUGINS-GUIDE.md) carry terms of its own
author's choosing; it grants nothing for a work built from Bridge Talk's own code. The application
shows the same text under Help, then Licence.
