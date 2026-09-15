# <img width="128" height="128" alt="application-icon" src="https://github.com/user-attachments/assets/482dbfad-c806-46a7-afec-cfcb0ebf4e87" /> Bridge Talk

A ship's voice for Elite Dangerous, speaking with a machine voice of its own or with recordings you
supply.

Bridge Talk watches the game's journal and status file while you play. When something worth
saying happens, the cast voice speaks for that moment. Start with one of its 28 machine voices, whose
lines it makes on your own machine; record a voice of your own whenever you like. It ships no recordings.

## Who it is for

- Commanders playing Elite Dangerous on Windows who want their ship to talk straight away: cast a
  machine voice and it speaks, with nothing to record first.
- Commanders who hold recordings they may use and want them spoken in answer to the game.
- Anyone recording a voice for a ship who wants to see exactly which moments a set of recordings
  covers, then hear it before choosing it.

## Who it is not for

- Anyone wanting voice commands. It talks and never listens: there is no microphone, no speech
  recognition and no command and control.
- Anyone wanting a machine voice in another language. The 28 machine voices speak British and
  American English only.
- Anyone not on Windows. The only build and the only setup program are for Windows.
- Anyone wanting a separate output device for it. There is no device choice: it speaks through
  whichever device Windows is set to use.

## What it does

- **Speaks with a machine voice, with nothing to record.** 28 voices speak British and American
  English, female and male. On the Cast pane they sit in a panel for each accent and sex; cast one
  and it becomes the ship's voice, standing apart above the panels with how far its lines are made.
  Each line is made on your own machine the first time its moment happens, then kept, so nothing is
  downloaded and nothing is sent anywhere. The Audition pane plays any of them before you cast it.
- **Casts a recorded voice.** Every folder in your recordings directory that holds at least one recording
  for a moment, named for it or in its folder, is a voice. Cast one from the Cast pane or from the Voice menu on the
  notification-area icon and it becomes the ship's voice. Where it holds a recording for
  `Cast.Confirmed` and mute is off, it plays that as it takes the part. The choice is remembered
  for the next run.
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
- **Speaks only for the moments you choose.** The Chatter pane lists every moment the game raises
  under its subject, each with a switch; a moment switched off stays quiet whichever voice is cast.
  A category's switch, Switch all on and Switch all off change many at once, asking first. Station
  traffic has a switch of its own, apart from every other message.
- **Explains every decision.** For every moment the game raises that has a cue, the Status pane
  logs what became of it (played, queued, making, dropped, cooldown, duplicate, off or unbound), including the
  ones that produced no sound. It keeps the latest 200. The same pane names the journal directory
  and status file being watched and says when no audio device was found or the device ran dry.
- **Shows what a voice covers.** For any voice, cast or not, the mark beside it on the Cast pane
  lists the moments it has a recording for and the moments it has none for.
- **Lists the takes still missing.** The Missing takes pane offers every voice folder still missing
  a recording. For the one chosen it lists each missing moment with a line saying when it is heard,
  then the folder its take belongs in.
- **Auditions a voice** before it is cast. Each button plays one take drawn at random from a part of
  the game, such as docking or combat. The
  buttons are held while anything is playing, the ship's own reactions included, so a press never
  cuts a clip short; Stop ends what is playing. An audition ignores the mute.
- **Stays out of the way.** Closing the window asks whether to put it away or quit. Put away, it
  keeps listening from the notification area, whose menu opens the window, mutes, switches voice
  or quits. It can start when you sign in to Windows, waiting in the notification area; turn that
  on in Settings or in the setup program.
- **Remembers how you like it.** The volume slider and the light or dark theme on the band are kept
  between runs.
- **Leaves your recordings alone.** It never changes or removes a file in your recordings
  directory. The only things it writes there are empty folders: a voice's folders when you press
  Make folders and a moment's folder when you press Open folder beside it. Uninstalling does not
  touch it. It makes no network requests of its own.

## Naming recordings

The quickest way is to let it name them. On the Cast pane, type a voice's name and press Make
folders: that voice gets a folder for every moment, each already named. Put each recording in the
folder for its moment (any file name will do), then press Refresh. Where no recordings directory
is chosen yet, it makes them in its own `Recordings` folder (`%LOCALAPPDATA%\BridgeTalk\Recordings`
on Windows) and uses that folder from then on; uninstalling never removes it.

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

## Getting it

Download the setup program from [ernster.dev/BridgeTalk](https://ernster.dev/BridgeTalk/) and run
it. Everything it writes is for your own Windows account, so it never asks for administrator rights.
It installs under your own application data unless you choose another folder on its first screen.
Once the window opens, cast a machine voice from the Cast pane. To use recordings of your own,
choose your recordings directory on the Missing takes pane, then cast that voice from the Cast pane.

The journal directory is `Saved Games\Frontier Developments\Elite Dangerous` under your own profile
until you choose another on the Settings pane. When the journal directory in use cannot be read,
the window opens anyway: the Status pane and the Settings pane say why until you choose one that
can be.

## Building and testing

- [DEVELOPMENT.md](DEVELOPMENT.md) sets up a machine, builds the application with its setup
  program and lists the command-line options.
- [TESTING.md](TESTING.md) covers running the tests, what each coverage figure means and what is
  deliberately left untested.
- [ARCHITECTURE.md](ARCHITECTURE.md) explains the layering and the reasoning behind each decision.

## Supporting the project

Bridge Talk is free and stays free. There is no paid tier, no licence key and no feature held back
behind a donation. If it has been useful, a donation supports its maintenance and continued
development.

<a href="https://www.paypal.com/ncp/payment/DVP73MPL9JPSU"><img src="docs/images/donate.png" alt="Donate to Bridge Talk" width="120"></a>

## Licence

GNU General Public License, version 3: see `LICENSE`. The application shows the same text under
Help, then Licence.
