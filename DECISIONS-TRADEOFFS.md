# Decisions and trade-offs

The deliberate choices Bridge Talk rests on: what was chosen, what was given
up for it and why. Each entry is the decision as the product makes it today.
The detail behind each one, with the tests that hold it, lives in
[ARCHITECTURE.md](ARCHITECTURE.md) and the specification
([REQUIREMENTS.md](REQUIREMENTS.md)); [PLUGINS-GUIDE.md](PLUGINS-GUIDE.md) is
the plugin contract and [TESTING.md](TESTING.md) names what is deliberately
left untested.

## The product as a whole

### It talks and never listens

Bridge Talk watches the game's journal and status file and speaks in answer.
There is no microphone, no speech recognition and no way to control the game.

- **Rather than:** voice commands, which other tools offer.
- **Gains:** nothing to capture, nothing to send and no input path into the
  game.
- **Costs:** commanders who want spoken commands need another tool beside it.

### It speaks only when the game or the player gives it a reason

Two sources feed it: the journal and the status file. It never makes idle
remarks of its own. The seam for a third source exists; the source does not.

- **Rather than:** a ship that chatters to fill silence.
- **Gains:** every line has a cause that can be named; the voice does not
  turn from characterful to grating.
- **Costs:** long quiet stretches stay quiet.

### Go with a web front end, on Wails

The application is Go with a React and TypeScript page, hosted by Wails in
WebView2 on Windows and webkit2gtk on Linux.

- **Rather than:** a Python and Qt desktop stack.
- **Gains:** a single binary with no runtime to ship; the same web view
  serves the setup program.
- **Costs:** two languages, with the wire between them written twice and
  compared by test.

### Specification before code

Every behaviour is written down as a numbered requirement before it is built,
each naming the test that proves it, beside a table of what is out of scope
and why.

- **Rather than:** building first and describing afterwards.
- **Gains:** a ruling stays ruled; a claim in the docs is one a test holds.
- **Costs:** the specification is a large document that must be kept true.

### Windows and Linux, not macOS

Windows gets a setup program; Linux gets a flatpak. macOS is not built.

- **Rather than:** every desktop platform.
- **Gains:** two platforms held to the same bar rather than three held to a
  lower one.
- **Costs:** commanders on macOS are not served.

### It ships no recordings

What ships is the machine voice model and the files it is made from. A
recorded voice is always the user's own; a plugin's audio is the plugin's.

- **Rather than:** bundling recorded voices.
- **Gains:** no recordings to license; the recordings belong to the person
  who made or holds them.
- **Costs:** a recorded voice has to be supplied before it can be cast.

## Privacy and the network

### One network request, held by test

The update check is the only request the application makes. A structural test
refuses any network package in the code the application links, with one
exemption for the update check; a second test fails once that exemption is no
longer needed. A third holds the page to making no request and naming no
address.

- **Rather than:** a promise that the network is used sparingly.
- **Gains:** "one request" is a test result rather than a sentence; a new way
  out is an edit somebody has to make and defend.
- **Costs:** the tests cannot see a request Wails or its web view makes on its
  own account.

### Update checks: daily, quiet unless there is news

The page asks three seconds after it loads, then every 24 hours; unasked, it
shows only an offer. Help, then Check for updates shows every outcome. The
request names the project's latest release and nothing about the user, the
machine or the game. A running version that is not a release's is never told
it is out of date.

- **Rather than:** no check at all; one that reports every outcome.
- **Gains:** updates are found without nagging; a build from source is never
  offered a release it cannot be compared with.
- **Costs:** one unprompted request a day; the flatpak has to be granted the
  network for it.

### The browser does the asking

The donate button and an update's Download hand an address to the default
browser and stop there. An address that does not begin `https://` is refused
before it is handed over; the page never names an address at all.

- **Rather than:** fetching anything from inside the application.
- **Gains:** no connection of the application's own for either.
- **Costs:** Bridge Talk never learns whether the browser got there.

### A plugin is trusted because the user put it there

No signature, publisher or hash is checked. A plugin is opened only by its
full path inside the plugins folder; the README and the plugin guide both say
it runs with the user's own rights.

- **Rather than:** refusing unsigned plugins; asking about each one.
- **Gains:** no promise about other people's work that the application could
  not keep.
- **Costs:** the user is the only check on what goes in the folder.

### A plain-text run log

Each run adds to a log in the product's own data folder; a windowed start
sends all of its error output there, so a crash leaves its report. Past 1 MB
the log starts afresh. Uninstall removes it.

- **Rather than:** keeping no record; sending reports anywhere.
- **Gains:** a fault that left the screen still has an account of itself.
- **Costs:** anyone with access to the account can read it.

## Moments and the game

### The names on disk are the mapping

Every moment is named in the game's own words: the journal event or the
status flag followed by Set or Cleared. A recording reaches a moment by being
named for it. There is no mapping file.

- **Rather than:** a mapping file per voice, kept in step by hand.
- **Gains:** pointing the application at a folder simply shows what it holds;
  the game's facts and a person's recordings change independently.
- **Costs:** recordings have to be named exactly; Make folders exists to do
  it for the user.

### Exact names, apart from case

A name matches a moment only when the two are equal ignoring case. Nothing
else is normalised: spaces, hyphens or dots where they do not belong resolve
nothing.

- **Rather than:** fuzzy or partial matching.
- **Gains:** a file never answers a moment it was not meant for.
- **Costs:** a near miss is silent until the Missing takes pane shows the gap.

### The cue table is built in

The moments, their priorities, cooldowns and categories live in one table
embedded in the executable. It is edited as a file in the source, not from
the window.

- **Rather than:** a table the user edits.
- **Gains:** every copy agrees about what each moment means; a broken table
  fails the build rather than a run.
- **Costs:** a new moment needs a release.

### Start at the end of the journal

On launch the newest journal's current size is where reading starts. There
is no path that reads a session from its beginning.

- **Rather than:** reading from the start; a cut-off by time.
- **Gains:** opening mid-session does not fire hundreds of clips at once.
- **Costs:** anything that happened before launch is never spoken for.

### The status file is a source of its own

Changes in the status file's flags, screen focus, power distribution and fire
group are moments alongside the journal's. The first reading only sets the
baseline; a reading with every flag clear means no session and is not
announced.

- **Rather than:** the journal alone.
- **Gains:** a large block of moments that never appear in the journal can be
  spoken for; leaving the game is not read as every flag falling.
- **Costs:** a second reader with rules of its own.

### Comms moments matched by key, never read aloud

A pirate's message is matched by the key the game sends with it, whatever
variant of the words was chosen. The ship's voice speaks its own line for the
moment; the message's words are never spoken. A message a player typed
carries no key and reaches no moment. Station traffic is one moment of its
own.

- **Rather than:** reading the message out.
- **Gains:** the words change with every message, so a line would have to be
  made as the event fired; matching the key keeps the response to one
  recording per moment.
- **Costs:** only the comms the table names are answered; any other kind
  waits for a decision of its own.

### Titles generated, purposes written by hand

A moment's title is read from its id. One sentence saying when it is heard is
written beside it in the table; a moment without one is refused.

- **Rather than:** a title and a description both kept by hand.
- **Gains:** the title can never drift from the id; somebody recording a
  take knows when it will be heard.
- **Costs:** every new moment needs its sentence written.

### No fallback

A voice with nothing for a moment answers with silence; the Status pane
records why.

- **Rather than:** substituting another moment's take.
- **Gains:** a wrong line delivered confidently never happens.
- **Costs:** an incomplete voice is quiet in places.

## Speaking and staying quiet

### Priority, cooldown and dedupe

Every moment carries a priority. An alert cuts anything less urgent; a notice
waits its turn; an ambient line is dropped when something is waiting; a
flavour line is dropped when anything is waiting or playing. Many moments
hold a minimum interval; the same moment twice within 900 milliseconds is
heard once.

- **Rather than:** a plain queue.
- **Gains:** a voice worth listening to rather than a slot machine.
- **Costs:** some moments are never heard on a busy evening; the Status pane
  says which and why.

### A cut, not a fade

An alert stops what is playing outright and the next take starts. There is no
mixer, no ducking and no crossfade.

- **Rather than:** mixing or ducking.
- **Gains:** a simple player whose one job is timing.
- **Costs:** an interrupted take ends abruptly.

### The same take is not heard twice running

Where a voice holds several takes for a moment, one is chosen at random,
never the one that moment played last.

- **Rather than:** a fixed take or a plain random draw.
- **Gains:** a moment heard often does not sound canned.
- **Costs:** none recorded.

### The player owns its speaker

Bridge Talk drives the audio output itself rather than through the audio
library's shared speaker, asking Windows for a 100 ms buffer. Stopping or
cutting in drops whatever was queued.

- **Rather than:** the library's own speaker, which kept 240 ms of audio
  queued ahead of every take.
- **Gains:** a take begins within the 150 ms budget, measured at 100.5 ms at
  the 95th percentile on a real device (349.7 ms with the old queue put back).
- **Costs:** the output path is Bridge Talk's own to maintain.

### Pure Go decoding

WAV, MP3, FLAC and Ogg Vorbis are decoded in Go. No system codec, media
framework or outside process is used; the Windows build pins cgo off.

- **Rather than:** a system media framework; a bundled transcoder.
- **Gains:** the same decoding everywhere; a machine with a C toolchain
  cannot quietly produce a different Windows binary.
- **Costs:** formats beyond those four are not played.

### A take may be several parts

A take is one or more parts played in order with no added gap; a part is a
whole file or a span of one.

- **Rather than:** one file per take, which was the rule until plugins needed
  otherwise.
- **Gains:** a line recorded in pieces is never spoken from the middle.
- **Costs:** a part that will not open is passed over and logged, so a take
  can be heard short of a piece.

### An audition never cuts a clip and ignores the mute

An audition plays only when nothing else is, so a press while a clip sounds
is ignored; its buttons are held until it ends. Mute silences the answers to
the game, never a deliberate press.

- **Rather than:** each press starting over what is playing; a mute that
  silences everything.
- **Gains:** hammering a button never chops a clip; a press is never answered
  with silence that reads as a fault.
- **Costs:** a press while the ship is speaking does nothing.

### A switch for every moment, shared by every voice

Chatter gives each moment a switch, kept in the settings file. A moment
switched off starts no cooldown and makes no line. One switched off while it
waits is let go when its turn comes; a take already playing is never cut.

- **Rather than:** switches per voice; editing the queue at the press; keeping
  the switches in the page beside the theme.
- **Gains:** which moments are spoken for is a question about the game, so a
  new voice inherits the answer; the engine has the switches before any page
  loads.
- **Costs:** a voice cannot have moments of its own switched off.

### The switches are one value swapped whole

The loop that watches the game reads the switches while the window changes
them from another thread. Each change replaces the whole set at once.

- **Rather than:** a map edited in place under a lock.
- **Gains:** a reader always holds a complete set with no lock on the path
  every firing takes.
- **Costs:** none recorded.

### A volume curve that sounds even

The slider has twenty steps. Its position picks a point up to six halvings
below full, read once per audio buffer.

- **Rather than:** mapping the slider straight onto gain, where most of the
  travel sounds the same.
- **Gains:** the whole slider is useful; a move is heard mid-clip.
- **Costs:** none recorded.

### No choice of output device

It speaks through whichever device the system is set to use.

- **Rather than:** a device chooser.
- **Gains:** nothing to configure; the system's own setting decides.
- **Costs:** the ship's voice cannot be sent to a different device from the
  game.

## Machine voices

### Kokoro through ONNX Runtime, with no C written

The 28 English voices of the Kokoro model are run through ONNX Runtime's C
interface, called from Go with no binding written and the library loaded by
its full path.

- **Rather than:** a bundled Python helper of about 1 GB; cgo bindings, which
  need a C toolchain and headers on every build machine.
- **Gains:** the Windows build stays pure Go; the full path keeps the older
  copy Windows ships from standing in.
- **Costs:** the call table is read by position, so the runtime's version is
  pinned; British and American English only.

### Pronunciation worked out before the build

Every line's speech sounds are made by a development tool running misaki in
its own Python environment, then saved beside the script and embedded. The
application runs no Python and works out no pronunciation.

- **Rather than:** a Go port of the pronunciation step, which reached 99.4
  percent of the reference lines where misaki matched all 512.
- **Gains:** exact speech sounds; no dictionaries or Python in the package.
- **Costs:** a change to the script means running the tool again; a misread
  word is put right in the script by hand.

### Three lines for every moment

The script gives each moment exactly three lines, shared by every machine
voice. A moment without three distinct lines fails the build.

- **Rather than:** one line each.
- **Gains:** a moment heard often does not sound the same each time.
- **Costs:** three lines to write and keep for every moment.

### Each line made the first time it is needed, then kept

Casting a machine voice makes only its confirmation. Every other line is made
the first time its moment fires, then kept. A moment that fires before its
line exists waits up to two seconds for it.

- **Rather than:** speaking a line as its event fires, which takes 204 to
  348 ms against a 150 ms budget; making every line at the cast, which took
  3 minutes 17 seconds for one voice.
- **Gains:** a cast is heard confirming within a second and a half (measured
  at 1.30 to 1.35 seconds); nothing is made that is never heard.
- **Costs:** the first time a moment fires it may be late or let go.

### Made lines kept for every machine voice

A made line is kept until it is no longer current. Casting another voice of
any kind keeps them; casting a machine voice deletes only that voice's stale
lines. Uninstall removes them all.

- **Rather than:** deleting the lines of every voice not cast, which was the
  first rule.
- **Gains:** switching between voices never makes the same line twice.
- **Costs:** disk space: one voice's complete script measured 51.7 MB.

### The model loaded when it is first wanted

The model loads when a machine voice is cast or a line is first made, without
waiting; it stays loaded until the maker closes and is never unloaded. A load
that fails is tried again on the next line.

- **Rather than:** loading at start whatever is cast; remembering a failed
  load.
- **Gains:** a player who casts only recorded voices never pays for it; the
  first line made on call is spared the 539 ms load.
- **Costs:** loaded, it was measured at a 408.5 MB working set.

### Made lines kept as 16-bit FLAC

Each line is stored as mono 16-bit FLAC at 24 kHz, written beside its place
then renamed into it.

- **Rather than:** keeping the model's floating-point samples.
- **Gains:** lossless storage at a fraction of the size; an interrupted write
  leaves no made line.
- **Costs:** the samples are clamped and rounded to 16 bits.

### Pauses and fades measured before the build

A final "commander" gets 40 ms of silence before it; a line ending on a nasal
is faded over the 30 ms before the hiss the model adds there. Where to cut is
found for every voice by a development tool using Praat and saved with the
script. The application applies a cut only where the samples it made match
those measured; otherwise the line is written as made and logged. A doubtful
break gets no pause.

- **Rather than:** processing the sound while the application runs; leaving
  the model's output alone.
- **Gains:** "commander" no longer restarts the pitch; the hiss after a
  final nasal is gone. Both stay exact, since a mismatch is never cut.
- **Costs:** the files must be found again whenever the script or the model
  changes; a structural test fails the build while they are stale. The full
  run found 999 of 6,720 joined lines doubtful.

## Recorded voices

### A checklist, not a recorder

The Missing takes pane lists each moment a voice has no take for, with when it
is heard and the folder its take belongs in. Recording happens in a program
built for it.

- **Rather than:** a recorder built into the window, which was specified and
  then withdrawn.
- **Gains:** no microphone, no capture code; the user records in whatever
  they already know.
- **Costs:** two programs to move between while recording.

### Make folders makes folders

Typing a name and pressing Make folders gives the voice an empty folder for
every moment. With no recordings directory chosen, they go in the product's
own Recordings folder, which is then kept.

- **Rather than:** asking where with a folder picker first.
- **Gains:** one press leaves every recording a folder to drop into.
- **Costs:** a user who wanted them elsewhere has to choose the directory
  first.

### The recordings are never changed

Nothing in the recordings directory is changed or removed. The only writes
are empty folders on request. A structural test lists every write the
application links with where it goes; uninstall never touches the directory.

- **Rather than:** tidying, renaming or converting recordings.
- **Gains:** a collection somebody spent hours recording cannot be damaged.
- **Costs:** a misnamed file stays misnamed until its owner renames it.

### A folder writes each dot as an underscore

A moment's folder is its id with every dot as an underscore; a file named for
a moment keeps the dots. No id may hold an underscore, so a folder maps back
to exactly one moment.

- **Rather than:** dotted folder names.
- **Gains:** folders that every file manager shows plainly; no two moments
  can share one.
- **Costs:** two spellings of one name for the user to learn.

### The folder name is the identity

A voice may carry a `voice.toml` giving a display name, a credit and takes its
names do not reach. The folder name stays what the settings remember. A
manifest that cannot be read is set aside whole.

- **Rather than:** a required manifest; the display name as the identity.
- **Gains:** a voice works with no manifest at all; renaming it on screen
  never loses the cast.
- **Costs:** renaming the folder does lose it.

### Every take decoded at the scan, nothing cached

A scan opens the start of every take, leaving out and reporting one that will
not play. There is no cache: the scan runs at start, when a directory is
chosen and on Refresh.

- **Rather than:** finding a bad file when its moment fires; caching scans.
- **Gains:** a broken take is reported where it was found; nothing can go
  stale. Ten voices of 5,000 recordings scanned in under 900 ms, measured.
- **Costs:** every scan reads every file.

## Plugins

### A C interface, three functions

A plugin is a native library in the plugins folder offering voices whose
audio is already on the machine. It exports three functions: its version, a
description of itself with its voices, then one answer per moment.

- **Rather than:** a Go plugin, which does not run on Windows; a helper
  process over a pipe, which is a second program to install and keep alive;
  a dozen smaller calls.
- **Gains:** a plugin can be written in any language; a small surface.
- **Costs:** every answer is a byte layout both sides must read alike.

### Buffers owned by Bridge Talk

Every answer is asked for its size first, then filled into a buffer Bridge
Talk owns. A negative return is always a refusal, never a size. An answer
over 4 MiB is refused before anything is made to hold it.

- **Rather than:** the plugin allocating with a fourth function to free;
  negative meaning the bytes needed.
- **Gains:** nothing is allocated on one side and freed on the other; a size
  and an error can never be confused; a garbage size cannot end the run.
- **Costs:** two calls for every answer.

### One locked thread for every plugin

Every call into every plugin is made one at a time from a single goroutine
holding one operating system thread.

- **Rather than:** a lock alone, which serialises without keeping the
  thread.
- **Gains:** a plugin author needs no locking; something tied to a thread is
  found on it again.
- **Costs:** a channel round trip per moment. No plugin existed to measure;
  it was taken as a precaution while it was cheap.

### What a plugin sends is foreign input

No length or count a plugin sends is trusted; text must be UTF-8 and bytes
left over are a refusal. A fault raised by a call ends that call alone. A
plugin that will not load is named in the run log and never stops the
application starting.

- **Rather than:** trusting a well-behaved plugin.
- **Gains:** a misbehaving plugin costs only itself.
- **Costs:** a fault inside the plugin's own code still cannot be caught.

### Only version 2 is read

The interface is at version 2, which added a voice's group and a part that is
a span of a file. Any other version is passed over by name.

- **Rather than:** reading both versions side by side.
- **Gains:** a version 1 layout is never read wrongly.
- **Costs:** a version 1 plugin must be rebuilt.

### Spans read where they stand

A part may be a span of a larger file, read in place through a section of the
open file with a 64-bit offset and length. A plugin's audio is never copied,
moved or rewritten; a test watches its folder to hold that.

- **Rather than:** a plugin copying each recording out to a file of its own.
- **Gains:** nothing is copied to the user's disk.
- **Costs:** a span outside its file is found only as it is played.

### Never unloaded

A plugin, like ONNX Runtime, stays loaded until the process ends.

- **Rather than:** freeing a library once it is no longer used.
- **Gains:** no risk of unloading under a thread of its own.
- **Costs:** whether it could be unloaded safely has not been measured.

### The plugins folder belongs to the user

Setup makes the folder and never fills it: the payload never carries one. An
update leaves it as it was. Uninstall keeps it unless Also remove my plugins
is ticked. On Linux it sits in the user's data folder, since the flatpak's
install directory is read only.

- **Rather than:** a folder setup manages like its own files.
- **Gains:** an update or a repair never overwrites a user's plugin.
- **Costs:** two places to look, one per platform.

### Proved without a plugin file

No C toolchain is on the build machine, so a Go stand-in answers in the real
layouts. The native calls are measured against libraries the operating system
ships.

- **Rather than:** putting a C compiler on every build machine.
- **Gains:** the layouts, the protocol and every refusal are tested.
- **Costs:** the call into a real plugin file is not proved by a test; the
  guide's worked example has not been compiled.

## The interface

### Closing asks; the ship keeps listening

On Windows the cross asks whether to put the window away or quit. Put away,
it listens from the notification area. With no tray to hide into, the cross
simply closes.

- **Rather than:** the cross always quitting.
- **Gains:** a resident application can be put out of the way without being
  stopped.
- **Costs:** one more question at every close.

### No tray icon on Linux

On Linux the window's own panel button is the one icon. The cross quits and a
hidden start shows the window.

- **Rather than:** a tray icon beside the panel button; a patched copy of the
  tray library drawing it only while the window is put away, which was built
  and taken out.
- **Gains:** one icon rather than two doing the same thing; no patched
  library to carry.
- **Costs:** on Linux it cannot keep listening with its window away.

### One copy at a time

A second start asks the running copy for its window and ends before building
anything; a start from the sign-in entry asks nothing. The claim is the
application's own, taken before the tray, plugins, model or audio device
exist.

- **Rather than:** Wails' own single-instance lock, taken after the tray and
  the device are up; it skips removing the tray icon and on Linux needs a
  name the flatpak is not granted.
- **Gains:** there is only ever one ship's voice.
- **Costs:** a claim of its own to keep, one per platform.

### The window opens whatever the journal says

A journal directory that cannot be found or read is said on the Status and
Settings panes; nothing is watched until one that can be is chosen. Only a
broken table built into the executable stops a start.

- **Rather than:** refusing to start.
- **Gains:** the user can always reach the setting that fixes it.
- **Costs:** a window that is not yet listening.

### A fault in the loop ends the loop alone

A fault in the loop watching the game ends that loop. Its stack reaches the
run log; the Status pane says the application has stopped reacting. The
window keeps working.

- **Rather than:** the whole application ending, which once left its only
  account in a log nobody had been asked to open.
- **Gains:** the user is told, on screen, in the fault's own words.
- **Costs:** the loop stays ended until the application is started again.

### A refusal cannot be dropped on the page

Every call that can be refused takes a refusal handler; one written without
it does not compile. A call that did not happen answers null rather than an
empty value.

- **Rather than:** a structural test, which was written and deleted because
  it could not follow a promise through a helper.
- **Gains:** no pane waits forever on an answer that will never come; an
  empty list always means empty.
- **Costs:** every call site names what it does when refused.

### Nothing shown twice

A path is named once in a refusal; a list of moments shows each under its
title alone; no line repeats what the line beside it says.

- **Rather than:** headings, tooltips and messages that echo their
  neighbours.
- **Gains:** less to read; a reader is never left comparing two copies.
- **Costs:** none recorded.

### One home for every colour

Every colour in the page lives in one token file, with a set for each theme.
Structural tests hold the note lines, Chatter's switches and the heading pills
to stated contrast in both themes.

- **Rather than:** colours written where they are used.
- **Gains:** the themes stay consistent; a pairing that cannot be read fails
  the suite.
- **Costs:** the setup page and the window backgrounds set in Go sit outside
  the rule.

### Everything on the keyboard

One ring holds every control; Tab and the arrows step through it and wrap. A
ring is drawn on controls only, never on a list or a container. The setup
program wears the same ring.

- **Rather than:** the browser's own tab order.
- **Gains:** the whole application and its setup work without a mouse.
- **Costs:** the ring is written twice, since the setup page has no build
  step to share code through.

### Reading surfaces read themselves

The guide and every dialog with text scroll slowly on their own, hold, rewind
and repeat. Any touch suspends the cycle; it resumes where the reader left it.
It is not switched off by the system's reduced-motion setting.

- **Rather than:** static pages; honouring reduced motion.
- **Gains:** long text reads hands free. On Windows that setting follows the
  general animation switch, which people turn off for speed.
- **Costs:** someone who wants a page still has to touch it.

### The guide is drawn from the real controls

The guide's pictures come from the same artwork the controls are drawn from;
a test fails when the band shows a picture no entry does.

- **Rather than:** screenshots or drawings of its own.
- **Gains:** the guide cannot drift from the window.
- **Costs:** the guide is laid out by code.

## Building and installing

### A setup program of its own

Install, update, repair and removal are a second Wails application wearing the
application's look. Which screen opens is read from the machine. Repair and
reinstall are different acts over one install path. While the application
runs, setup offers to close it, forcing it and waiting up to five seconds.

- **Rather than:** a generic installer.
- **Gains:** one identity throughout; never a half-written install over a
  locked executable.
- **Costs:** the setup program is Bridge Talk's own to maintain.

### Per user, never asking for administrator rights

Everything setup writes is under the user's own folders and registry. The
install folder can be changed; it goes in a folder of its own inside the one
picked. A folder already holding other files is refused, since uninstall
deletes the install folder whole.

- **Rather than:** a machine-wide install.
- **Gains:** no administrator prompt; deleting the install folder whole
  cannot take files that were there before the install.
- **Costs:** each account installs separately.

### Shortcuts written through the shell's own object

Shortcuts are made through Windows' shortcut object over COM, each path handed
over as a value.

- **Rather than:** typing paths into a PowerShell script, which read a dollar
  sign as a variable and saved a shortcut under the wrong name or not at all.
- **Gains:** any folder name works; a failure is reported.
- **Costs:** a COM dependency.

### The payload embedded as a string

The setup program carries the application and its model files. They are
embedded as a string rather than a byte slice.

- **Rather than:** a byte slice, which charges the whole payload to memory at
  start.
- **Gains:** measured at 12.8 MB of private memory at start against 323.6 MB.
- **Costs:** the model files take the payload from 6.0 MB to 326.2 MB.

### Model files fetched from a pinned list

The model, ONNX Runtime and the 28 style files are not committed. A Go tool
fills a folder from a list of addresses, sizes and published SHA-256s; the
test gate stops while a file is missing or differs.

- **Rather than:** committing them; an environment variable naming a folder
  with the checksums checked again in PowerShell.
- **Gains:** every machine finds them the same way; the checksum rule lives
  once, in Go.
- **Costs:** a fresh clone downloads them before the gate will run.

### A flatpak for Linux

Linux gets a flatpak on the GNOME runtime, built by one script with cgo on for
webkit2gtk and the audio output. Its grants are one list a structural test
holds; the network is among them for the update check alone.

- **Rather than:** a package per distribution.
- **Gains:** one build for every distribution; the grants cannot widen
  unnoticed.
- **Costs:** the Linux build differs from the pure Go Windows one.

### The GPL with a plugin exception

Bridge Talk is GPL-3.0 with an added permission: a plugin that speaks only the
published interface may carry terms of its author's choosing.

- **Rather than:** the GPL alone.
- **Gains:** plugin authors know where they stand.
- **Costs:** the exception grants nothing for work built from Bridge Talk's
  own code, which has to be said wherever it is offered.

### A website written for the commander

The site is four pages: home, features, why and download. It explains what
Bridge Talk does, with no build instructions. Its comparison with EDDI,
EDDiscovery and EDCoPilot is drawn from their own pages.

- **Rather than:** a developer's project page.
- **Gains:** the people deciding whether to install it find what they need.
- **Costs:** developers go to the repository instead.

## Engineering

### Layers held by test

The code is domain, application, infrastructure and interface, each depending
only inward, with one composition root wiring them by constructor. Structural
tests hold the boundaries; a guard missing from the architecture's table is
treated as missing.

- **Rather than:** convention alone; a dependency injection framework.
- **Gains:** the rules about moments, priorities and making lines are tested
  with no disk, clock or device.
- **Costs:** more packages and explicit wiring.

### Complete coverage where it means something

The domain and application layers are held to 100 percent. Every other package
is held to a floor set from what it measured, never from a target; a gap is
named in TESTING.md or closed.

- **Rather than:** one figure over everything; floors as aspirations.
- **Gains:** a floor fails only when cover is lost, which is when it is worth
  hearing.
- **Costs:** a floor on a package whose figure varies by machine has to be
  measured on each.

### Small files

No source file may pass 400 lines; one between 381 and 400 is cut to 350 or
fewer.

- **Rather than:** letting files grow.
- **Gains:** files split at real seams.
- **Costs:** many small files.

### Every value has one home

The product's name, the version, each colour and the game's vocabulary each
live in one place; tests refuse a second spelling.

- **Rather than:** copies written where they are needed.
- **Gains:** a change is made once and cannot drift.
- **Costs:** the site has to be stamped from the source.

### Addresses converted inside the call

An address handed to a native library becomes a number only in the argument
list of the one call helper, which is marked so the compiler moves such
values off the stack.

- **Rather than:** converting before the call, which let a moving goroutine
  stack leave ONNX Runtime writing an old copy: 13 of 300 lines broke in the
  reproduction.
- **Gains:** one helper serves ONNX Runtime and plugins on both platforms; a
  structural test refuses the pattern that broke.
- **Costs:** the rule is subtle and lives in a test rather than in the
  language.

### Fakes written by hand; guards proved to bite

No Go test uses a mocking library and no test writes to the machine it runs
on. Every guard is proved by planting a violation and reading the exit code.

- **Rather than:** mocks and assumed guards.
- **Gains:** a passing test means the real thing works; a guard is known to
  bite.
- **Costs:** fakes are written and kept by hand.

### The gate runs everything, every time

One script checks the model files, formatting, vet, a pinned staticcheck, the
Go suite, the Linux half of the one-copy claim, the front end's lint, types
and tests, then every coverage floor. The build runs it with the benchmarks
and offers no way to skip it.

- **Rather than:** checks left to a release build; tools that move under a
  change that touched nothing.
- **Gains:** a failure is found while working; every build times a machine
  voice's cast.
- **Costs:** a build takes minutes.
