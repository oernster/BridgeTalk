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

The page asks shortly after it loads, then once a day; unasked, it shows only
an offer. Help, then Check for updates shows every outcome. The request names
the project's latest release and nothing about the user, the machine or the
game. A running version that is not a release's is never told it is out of
date.

- **Rather than:** no check at all; one that reports every outcome.
- **Gains:** updates are found without nagging; a build from source is never
  offered a release it cannot be compared with.
- **Costs:** one unprompted request a day; the flatpak has to be granted the
  network for it.

### The browser does the asking

The donate button and an update's Download hand an address to the default
browser and stop there. An address that is not secure is refused before it is
handed over; the page never names an address at all.

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

Each run adds to a log in the product's own data folder and a crash leaves
its report there; on Windows a windowed start sends all of its error output
there too. A log grown past a set size starts afresh. Uninstall removes it.

- **Rather than:** keeping no record; sending reports anywhere.
- **Gains:** a fault that left the screen still has an account of itself.
- **Costs:** anyone with access to the account can read it.

## Moments and the game

### The names on disk are the mapping

Every moment is named in the game's own words: the journal event or the
status flag followed by Set or Cleared. A recording reaches a moment by being
named for it, matched exactly apart from case; nothing else is normalised.
There is no mapping file.

- **Rather than:** a mapping file per voice, kept in step by hand; fuzzy or
  partial matching.
- **Gains:** pointing the application at a folder simply shows what it holds;
  a file never answers a moment it was not meant for; the game's facts and a
  person's recordings change independently.
- **Costs:** recordings have to be named exactly, so a near miss is silent
  until the Missing takes pane shows the gap; Make folders exists to do the
  naming for the user.

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
hold a minimum interval; the same moment raised twice in quick succession is
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

### An audio path of its own, in pure Go

WAV, MP3, FLAC and Ogg Vorbis are decoded in Go, with no system codec, media
framework or outside process; the Windows build pins cgo off. Bridge Talk
drives the audio output itself rather than through the audio library's shared
speaker, so stopping or cutting in drops whatever was queued.

- **Rather than:** a system media framework or a bundled transcoder; the
  library's own speaker, which kept audio queued ahead of every take.
- **Gains:** the same decoding everywhere; a machine with a C toolchain cannot
  quietly produce a different Windows binary; a take begins within its
  latency budget, which a test on a real device holds.
- **Costs:** formats beyond those four are not played; the output path is
  Bridge Talk's own to maintain.

### A take may be several parts

A take is one or more parts played in order with no added gap. A part is a
whole file or a span of a larger one, read where it stands.

- **Rather than:** one file per take, which was the rule until plugins needed
  otherwise; a plugin copying each recording out to a file of its own.
- **Gains:** a line recorded in pieces is never spoken from the middle; audio
  held many recordings to a file is played without anything copied to the
  user's disk.
- **Costs:** a part that will not open is passed over and logged, so a take
  can be heard short of a piece; a span outside its file is found only as it
  is played.

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
waits is let go when its turn comes; a take already playing is never cut. A
change replaces the whole set of switches at once.

- **Rather than:** switches per voice; editing the queue at the press; keeping
  the switches in the page beside the theme; a set edited in place under a
  lock.
- **Gains:** which moments are spoken for is a question about the game, so a
  new voice inherits the answer; the engine has the switches before any page
  loads; the loop watching the game always reads a complete set without
  taking a lock.
- **Costs:** a voice cannot have moments of its own switched off.

### A volume curve that sounds even

The slider picks a point on a curve of halvings below full rather than a
straight share of the gain. The level is read once per audio buffer.

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

The English voices of the Kokoro model are run through ONNX Runtime's C
interface, called from Go with no binding written and the library loaded by
its full path. The same call helper serves plugins.

- **Rather than:** a bundled Python helper of about a gigabyte; cgo bindings,
  which need a C toolchain and headers on every build machine.
- **Gains:** the Windows build stays pure Go; the full path keeps the older
  copy Windows ships from standing in; one helper reaches ONNX Runtime and
  plugins on both platforms.
- **Costs:** the call table is read by position, so the runtime's version is
  pinned; passing an address across the boundary follows a rule a structural
  test holds, since the language does not; British and American English only.

### Speech prepared before the build

Every line's speech sounds, the pause before a final "commander" and the fade
over the hiss the model adds after a final nasal are worked out by
development tools, misaki for the sounds and Praat for the cuts, then shipped
with the script. The application runs no Python and works out no
pronunciation. It applies a cut only where the samples it made match those
measured; otherwise the line is written as made and logged. A doubtful break
gets no pause.

- **Rather than:** working out pronunciation or processing the sound while
  the application runs; a Go port of the pronunciation step, which fell short
  of misaki's own; leaving the model's output alone.
- **Gains:** exact speech sounds with no dictionaries or Python in the
  package; "commander" no longer restarts the pitch and the hiss after a
  final nasal is gone; both stay exact, since a mismatch is never cut.
- **Costs:** a change to the script or the model means running the tools
  again; a structural test fails the build while their output is stale; a
  misread word is put right in the script by hand.

### Three lines for every moment

The script gives each moment exactly three lines, shared by every machine
voice. A moment without three distinct lines fails the build.

- **Rather than:** one line each.
- **Gains:** a moment heard often does not sound the same each time.
- **Costs:** three lines to write and keep for every moment.

### Lines made when they are first needed

Casting a machine voice makes only its confirmation; every other line is made
the first time its moment fires, which waits a short while for it. The model
loads when a machine voice is cast or a line is first made, without holding
up the cast, then stays loaded; a load that fails is tried again on the next
line.

- **Rather than:** speaking a line as its event fires, which is slower than
  the latency budget allows; making every line at the cast, which takes
  minutes; loading the model at start whatever is cast.
- **Gains:** a cast is heard confirming within moments; nothing is made that
  is never heard; a player who casts only recorded voices never pays for the
  model.
- **Costs:** the first time a moment fires it may be late or let go; the
  loaded model holds a large share of memory until the application closes.

### Made lines kept for every machine voice

A made line is kept until it is no longer current, stored losslessly at 16
bits and written whole or not at all. Casting another voice of any kind keeps
them; casting a machine voice deletes only that voice's stale lines.
Uninstall removes them all.

- **Rather than:** deleting the lines of every voice not cast, which was the
  first rule; keeping the model's floating-point samples.
- **Gains:** switching between voices never makes the same line twice; an
  interrupted write leaves no made line.
- **Costs:** disk space for every voice used; the samples are rounded to 16
  bits.

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

A voice may carry an optional manifest giving a display name, a credit and
takes its names do not reach. The folder name stays what the settings
remember. A manifest that cannot be read is set aside whole.

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
  stale; a large library still scans within its time budget.
- **Costs:** every scan reads every file.

## Plugins

### A C interface of three functions

A plugin is a native library in the plugins folder offering voices whose
audio is already on the machine. It exports three functions: its version, a
description of itself with its voices, then one answer per moment. Every
answer is asked for its size first, then filled into a buffer Bridge Talk
owns; a negative return is always a refusal, never a size. Only the current
version of the interface is read.

- **Rather than:** a Go plugin, which does not run on Windows; a helper
  process over a pipe, which is a second program to install and keep alive;
  a dozen smaller calls; the plugin allocating with a fourth function to
  free; reading every version side by side.
- **Gains:** a plugin can be written in any language; a small surface;
  nothing is allocated on one side and freed on the other; a size and an
  error can never be confused; an older layout is never read wrongly.
- **Costs:** every answer is a byte layout both sides must read alike, asked
  for in two calls; a plugin built against an older version must be rebuilt.

### One locked thread; never unloaded

Every call into every plugin is made one at a time from a single goroutine
holding one operating system thread. A plugin, like ONNX Runtime, stays
loaded until the process ends.

- **Rather than:** a lock alone, which serialises without keeping the
  thread; freeing a library once it is no longer used.
- **Gains:** a plugin author needs no locking; something tied to a thread is
  found on it again; no risk of unloading under a thread of the plugin's own.
- **Costs:** a channel round trip per moment. No plugin existed to measure, so
  the thread is a precaution taken while it was cheap; whether a plugin could
  be unloaded safely has not been measured.

### What a plugin sends is foreign input

No length or count a plugin sends is trusted; text must be UTF-8 and bytes
left over are a refusal. An answer larger than a set cap is refused before
anything is made to hold it. A fault raised by a call ends that call alone. A
plugin that will not load is named in the run log and never stops the
application starting.

- **Rather than:** trusting a well-behaved plugin.
- **Gains:** a misbehaving plugin costs only itself; a garbage size cannot
  end the run.
- **Costs:** a fault inside the plugin's own code still cannot be caught.

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
application's look, one file carrying the application and its model files.
Which screen opens is read from the machine. Repair and reinstall are
different acts over one install path. While the application runs, setup
offers to close it, forcing it and waiting a short while. Shortcuts are
written through the shell's own shortcut object, each path handed over as a
value.

- **Rather than:** a generic installer; typing paths into a script, which
  misread some folder names and saved a shortcut under the wrong name or not
  at all.
- **Gains:** one identity throughout; never a half-written install over a
  locked executable; any folder name works for a shortcut.
- **Costs:** the setup program is Bridge Talk's own to maintain; one large
  download; a COM dependency.

### Per user, never asking for administrator rights

Everything setup writes is under the user's own folders and registry. The
install folder can be changed; it goes in a folder of its own inside the one
picked. A folder already holding other files is refused, since uninstall
deletes the install folder whole.

- **Rather than:** a machine-wide install.
- **Gains:** no administrator prompt; deleting the install folder whole
  cannot take files that were there before the install.
- **Costs:** each account installs separately.

### Model files fetched from a pinned list

The model, ONNX Runtime and the voices' style files are not committed. A Go
tool fills a folder from a list of addresses, sizes and published checksums;
the test gate stops while a file is missing or differs.

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
EDDiscovery and EDCoPilot is drawn from their own pages. Its stylesheet is
linked by a hash of its content, stamped with the version from the source.

- **Rather than:** a developer's project page; one long page; relying on a
  browser's cache to expire.
- **Gains:** the people deciding whether to install it find what they need;
  a deployed page is never drawn with an old stylesheet.
- **Costs:** developers go to the repository instead; every change to the
  stylesheet means stamping the pages again.

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

The domain and application layers are held to full coverage. Every other
package is held to a floor set from what it measured, never from a target; a
gap is named in TESTING.md or closed.

- **Rather than:** one figure over everything; floors as aspirations.
- **Gains:** a floor fails only when cover is lost, which is when it is worth
  hearing.
- **Costs:** a floor on a package whose figure varies by machine has to be
  measured on each.

### Small files

No source file may pass a fixed line limit; one that comes close is cut well
below it rather than trimmed to fit.

- **Rather than:** letting files grow.
- **Gains:** files split at real seams.
- **Costs:** many small files.

### Every value has one home

The product's name, the version, each colour and the game's vocabulary each
live in one place; tests refuse a second spelling.

- **Rather than:** copies written where they are needed.
- **Gains:** a change is made once and cannot drift.
- **Costs:** the site has to be stamped from the source.

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
