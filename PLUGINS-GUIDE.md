# Writing a Bridge Talk plugin

This is the contract between Bridge Talk and a plugin, the guide to building one and the guide to
using one once it is built. It is written for a person and for an assistant such as Claude alike; the
section [For an assistant building a plugin](#for-an-assistant-building-a-plugin) gathers what an
assistant must hold to. It is the authority on the binary interface;
`ARCHITECTURE.md` says where a plugin sits inside the application and why the interface has this
shape; `REQUIREMENTS.md` section 6.3 says what the application promises about plugins.

Bridge Talk plays a short piece of audio when something happens in Elite Dangerous. Out of the box
the audio is either recordings the user made or lines made on their machine by a machine voice. A
plugin is a third source: a library file that offers one voice or more whose audio is already on the
user's own computer, arranged however that audio happens to be arranged.

Bridge Talk knows none of that arrangement. It asks one question, in its own vocabulary: for this cue
id, what may this voice play? Everything else is the plugin's business.

## What a plugin may do; what it may not

A plugin supplies audio. It does not supply behaviour (`REQUIREMENTS.md`, FR-502).

| A plugin may | A plugin may not |
|---|---|
| Offer any number of voices | Add a cue, rename one or change the cue table |
| Answer any number of takes for a cue | Change when, whether or how loudly anything plays |
| Answer a take made of several files played in order | Draw anything, open a window or show a message |
| Say that the audio a voice needs is missing, with a reason | Expect Bridge Talk to copy, move or write its audio |
| Read files anywhere the user can read them | Assume Bridge Talk will free anything it allocates |

Bridge Talk plays a plugin's audio where it stands. It never copies it, moves it, rewrites it or
deletes it (FR-572).

## Where a plugin goes

A plugin is a single library file in the `plugins` folder. On Windows the folder is inside the
folder Bridge Talk was installed into, which the setup program creates (FR-576). On a default install
that is:

```
%LOCALAPPDATA%\Programs\BridgeTalk\plugins
```

The install folder can be chosen at install time, so read the location from the Apps list entry
rather than assuming the default.

On Linux a plugin is a shared object and the folder is inside Bridge Talk's own data folder, since
the flatpak's install directory is read only (FR-818). The folder is `BridgeTalk/plugins` under
`$XDG_DATA_HOME`, which for the flatpak is the application's data folder:

```
~/.var/app/uk.codecrafter.BridgeTalk/data/BridgeTalk/plugins
```

Bridge Talk makes that folder itself when it starts on Linux (FR-819). Bridge Talk loads every file in that folder at startup and loads a
plugin from nowhere else (FR-560). The file may be named anything; the name means nothing, because a
plugin states its own name through the interface (FR-565).

Bridge Talk does not check a signature or a publisher. A plugin runs inside Bridge Talk with the
rights of the person who put it there, which is stated plainly as a non claim in `REQUIREMENTS.md`
section 5. Publish your plugin where your users can see who wrote it.

## The interface

Three exported functions. That is the whole surface.

```c
int32_t BridgeTalkPluginABIVersion(void);

int32_t BridgeTalkPluginDescribe(uint8_t *buffer, int32_t size);

int32_t BridgeTalkPluginTakes(int32_t voiceIndex,
                              const uint8_t *cueId, int32_t cueIdLength,
                              uint8_t *buffer, int32_t size);
```

Export them undecorated, with C linkage. From C++ that means `extern "C"`. From Go, build with
`-buildmode=c-shared` and mark each with `//export`. From Rust, `#[no_mangle] pub extern "C"`.

The current ABI version is **1**. `BridgeTalkPluginABIVersion` is the first function Bridge Talk
calls. If it answers a version Bridge Talk does not implement, nothing else is called, the plugin is
passed over and the run log names the file, the version it stated and the version Bridge Talk
implements (FR-564).

### The calling rules, which are the same for every function that fills a buffer

1. **Ask for the size first.** Called with `size` of 0 and `buffer` of `NULL`, the function writes
   nothing and returns the number of bytes the answer needs.
2. **Then ask for the answer.** Called with a buffer at least that large, it fills the buffer and
   returns the number of bytes written.
3. **A negative return is always a refusal**, never a size. `-1` means the plugin cannot answer this
   call. Bridge Talk records it and carries on. Because negatives are refusals and nothing else, a
   size can never be mistaken for an error.
4. **Bridge Talk owns the buffer.** It allocates it, it frees it. Nothing is allocated on one side of
   the boundary and freed on the other, so the two sides need not share a memory allocator.
5. **Never keep the pointer.** A buffer is valid only for the duration of the call.
6. **Text is UTF-8** with no terminating zero. Every string is preceded by its length in bytes, so a
   zero byte inside one is harmless.
7. **Integers are signed, 32 bit, little endian.** No floating point crosses the boundary, no struct
   is passed by value and there are no callbacks.
8. **An answer is at most 4 MiB.** The size asked for is the one number acted on before anything can
   be read, since the buffer is made to fit before a byte arrives; a size field is 32 bits wide, so a
   plugin answering garbage could ask Bridge Talk to set aside two gigabytes. A size above the limit
   is refused by the number it asked for and the second call is never made. No honest answer comes
   near it: the largest is a description of every voice a plugin offers, which is names and reasons.

### Calls arrive one at a time, on one thread

Bridge Talk makes every call into every plugin from a single operating system thread, which it holds
for the life of the process, making one call at a time. A plugin therefore needs no locking of
its own and may keep state that belongs to a thread, such as an initialised COM apartment.

### A plugin is never unloaded

Once loaded, a plugin stays loaded until Bridge Talk exits. There is no shutdown function to
implement and no call telling a plugin to release anything.

## The two answers

Both are plain byte layouts. Read them in order; there is no padding, no alignment requirement and
no header.

A **string** is written as its length in bytes as an `int32`, then that many bytes of UTF-8. An empty
string is a length of 0 with no bytes after it.

### BridgeTalkPluginDescribe

Who the plugin is, plus the voices it offers.

```
int32   voiceCount
string  pluginName
repeated voiceCount times:
    string  voiceId       stable, unique within this plugin, never shown to the user
    string  voiceName     what the user sees in the voice list
    int32   ready         1 when the audio this voice needs is present, else 0
    string  readyReason   empty when ready is 1; when 0, why, in words a user can act on
```

`voiceId` is how Bridge Talk remembers a cast voice between runs, alongside the name of the plugin
that offered it (FR-569), so keep both `voiceId` and `pluginName` stable across releases of your
plugin. A user who renames your file loses nothing, since the file's name means nothing; a user
whose plugin changes its own name loses the voice they had cast. `voiceName` may change freely,
since nothing is remembered by it.

A voice whose `ready` is 0 is shown but cannot be cast, with `readyReason` as the explanation
(FR-570). Use it for the case where the audio a voice needs has been moved or removed.

A plugin offering no voice at all is passed over with the reason recorded, as is one offering a voice
with no name (FR-566).

### BridgeTalkPluginTakes

What a voice may play for one cue. `voiceIndex` is the position of the voice in the `Describe`
answer, counting from 0. `cueId` is a Bridge Talk cue id in UTF-8, such as
`StartJump.JumpType.Hyperspace`, with its length in bytes; it is never zero terminated.

```
int32   takeCount
repeated takeCount times:
    int32   partCount     at least 1
    repeated partCount times:
        string  path      an absolute path to an audio file
```

- **A take is one alternative.** Where a voice has three different recordings for a cue, that is
  three takes. Bridge Talk picks one of them, avoiding the one it picked last time for that cue.
- **A part is one file of a take.** Where a take is a line recorded in pieces, those pieces are its
  parts, in the order they are to be heard. Bridge Talk plays them one after another with no added
  gap. Most takes have one part.
- **A cue this voice cannot serve** answers a `takeCount` of 0. That is an ordinary answer rather
  than a refusal: a cue no voice serves is silence, which Bridge Talk prefers to a wrong line.
- **Paths are absolute** and are opened exactly as given. Bridge Talk decodes `.mp3`, `.wav`,
  `.flac` and `.ogg`.
- **A part that will not open is passed over** and the remaining parts are played (FR-574). If no
  part of the chosen take plays, the cue is silent and the log says what was tried (FR-575).

Bridge Talk may ask for the same cue many times in a session, then asks about every cue when it works
out what a voice covers. Keep the answer cheap; do not walk the disk on every call.

## Cue ids

Cue ids are Bridge Talk's own vocabulary, spelled in the game's words. They live in
`internal/infrastructure/config/cues.toml` in this repository, which is the list to map your audio
against. They are stable: an id is not renamed once it ships.

Mapping your audio onto those ids is your plugin's whole job; it belongs in your plugin's
repository. Nothing about the audio you read, the folders it sits in, how it is arranged or the words
it is described by appears anywhere in this repository (`REQUIREMENTS.md`, CON-9).

## A worked example

A plugin in C offering one voice, The Quartermaster, whose one recording answers `DockingGranted`.
Every other cue is answered with no takes. The same code path answers the size and then the answer,
so the two can never disagree: a cursor with no buffer only counts, a cursor with one writes.

**Not yet built.** No C toolchain was on the development machine when this was written
(2026-09-16), so this file has not been compiled and no plugin has yet been loaded by Bridge Talk. It
follows the contract above line for line; treat it as a starting point and prove it by the steps in
[Using a plugin](#using-a-plugin).

```c
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#define EXPORT __declspec(dllexport)
#define REFUSED (-1)

static const char *PLUGIN_NAME = "Quartermaster Voices";
static const char *VOICE_ID = "quartermaster";
static const char *VOICE_NAME = "The Quartermaster";
static const char *DOCKED_CUE = "DockingGranted";
static const char *DOCKED_PART = "C:\\QuartermasterAudio\\docking-granted.wav";

/* A cursor writes when it holds a buffer and only counts when it holds none. */
typedef struct {
    uint8_t *buffer;
    int32_t size;
    int32_t used;
    int failed;
} cursor;

static void put_bytes(cursor *c, const void *bytes, int32_t length) {
    if (c->buffer != NULL) {
        if (c->used + length > c->size) {
            c->failed = 1;
            return;
        }
        memcpy(c->buffer + c->used, bytes, (size_t)length);
    }
    c->used += length;
}

static void put_int(cursor *c, int32_t value) {
    uint8_t little[4] = {(uint8_t)value, (uint8_t)(value >> 8), (uint8_t)(value >> 16),
                         (uint8_t)(value >> 24)};
    put_bytes(c, little, 4);
}

static void put_text(cursor *c, const char *text) {
    int32_t length = (int32_t)strlen(text);
    put_int(c, length);
    put_bytes(c, text, length);
}

/* A buffer too small for the whole answer is a refusal, never a partial answer. */
static int32_t finish(const cursor *c) { return c->failed ? REFUSED : c->used; }

static int audio_present(void) {
    FILE *file = fopen(DOCKED_PART, "rb");
    if (file == NULL) {
        return 0;
    }
    fclose(file);
    return 1;
}

EXPORT int32_t BridgeTalkPluginABIVersion(void) { return 1; }

EXPORT int32_t BridgeTalkPluginDescribe(uint8_t *buffer, int32_t size) {
    cursor c = {buffer, size, 0, 0};
    int ready = audio_present();
    put_int(&c, 1);
    put_text(&c, PLUGIN_NAME);
    put_text(&c, VOICE_ID);
    put_text(&c, VOICE_NAME);
    put_int(&c, ready);
    put_text(&c, ready ? "" : "its recording is not in C:\\QuartermasterAudio");
    return finish(&c);
}

EXPORT int32_t BridgeTalkPluginTakes(int32_t voiceIndex, const uint8_t *cueId, int32_t cueIdLength,
                                     uint8_t *buffer, int32_t size) {
    cursor c = {buffer, size, 0, 0};
    int32_t cueLength = (int32_t)strlen(DOCKED_CUE);
    if (voiceIndex != 0) {
        return REFUSED;
    }
    if (cueIdLength != cueLength || memcmp(cueId, DOCKED_CUE, (size_t)cueLength) != 0) {
        put_int(&c, 0); /* a cue this voice cannot serve: no takes, which is not a refusal */
        return finish(&c);
    }
    put_int(&c, 1); /* one take */
    put_int(&c, 1); /* of one part */
    put_text(&c, DOCKED_PART);
    return finish(&c);
}
```

Build it as a 64 bit library, since the Bridge Talk the development machine builds is a 64 bit x86
program and Windows will not load a 32 bit library into it. With MinGW-w64:
`x86_64-w64-mingw32-gcc -shared -O2 -o quartermaster.dll quartermaster.c`. With Microsoft's compiler
from an x64 developer prompt: `cl /LD /O2 quartermaster.c`. Neither command has been run here.

The example keeps its answers as constants. A real plugin reads where its audio is once, at its first
call, builds its map from cue id to takes then and answers every later call from that map.

## When Bridge Talk refuses a plugin

Every refusal names what was refused and why, in the run log at `%LOCALAPPDATA%\BridgeTalk\Log.txt`
(FR-567). A plugin is passed over when the file will not load, when a required function is missing,
when the ABI version does not match, when it offers no voice or when it offers a voice with no name
or no id (FR-566). A voice whose `ready` is 0 is not a refusal of the plugin: the voice is shown with
its reason and cannot be cast; the log names it with that reason too. A voice that gives an empty
reason is said to have given none.
A single answer is passed over when it does not read as its layout, when the plugin asks for more
than 4 MiB or when a call into it panics: the call answers nothing and the application carries on.
One plugin being passed over never stops another loading (FR-561).

If two plugins offer voices under the same name, both are kept and each is shown with the name of the
plugin offering it (FR-568).

## Using a plugin

For the person installing one; also for an author proving one works.

1. **Put the file in the plugins folder** given in [Where a plugin goes](#where-a-plugin-goes). Setup
   makes the folder on Windows (FR-576); Bridge Talk makes it on Linux (FR-819). Any file name will do.
2. **Start Bridge Talk; if it is running, quit it and start it again.** Plugins are loaded once, as the application
   starts; one added while it runs is not seen until the next start.
3. **Look on the Cast pane.** Each voice the plugin offers is listed by the name it gave (FR-565). Two
   voices sharing a name are each shown with the plugin offering them (FR-568). A voice whose audio is
   missing is listed with the reason it gave and cannot be cast (FR-570).
4. **Cast the voice** from the Cast pane or from the Voice menu of the icon in the notification
   area, which lists every plugin voice that can speak after the machine voices (FR-509). The choice
   is kept for the next run by the plugin's name and the voice's id (FR-569).
5. **See what it lacks** on the Missing takes pane, which lists the moments the cast voice has no take
   for. It offers no folder to open for a plugin voice, since the plugin decides where its audio is
   (FR-571).
6. **Read the run log** at `%LOCALAPPDATA%\BridgeTalk\Log.txt` when something is not as expected.
   Every plugin passed over, every voice passed over and every part of a take that would not open is
   named there with the reason (FR-567, FR-574).

Updating Bridge Talk leaves the plugins folder as it was; the setup program never carries a plugin
of its own (FR-577). Uninstalling offers **Also remove my plugins**, unticked, so the folder is kept
unless asked otherwise (FR-578).

Bridge Talk does not check who wrote a plugin or whether it has been altered. A plugin runs with
the rights of the person who put it there (NFR-S-3). Install only a plugin whose author you trust.

## Linux

Bridge Talk loads plugins on Linux too. The contract is unchanged there: the same three functions
with the same rules, in a shared object built from your own repository, placed in the folder
[Where a plugin goes](#where-a-plugin-goes) gives for Linux. Nothing else in this document is
Windows specific except the paths it gives as examples. No plugin has yet been loaded on Linux; the
loader's refusals have been tested there, a real plugin's calls have not.

## Checking your layouts against ours

`internal/infrastructure/plugin/plugintest` in this repository writes these layouts in Go and is
held to the same rules your plugin is: the size before the answer, a refusal that is never a size,
then nothing written into a buffer too small to hold the whole answer. It is test support rather
than a library to depend on. Go does not let a package under `internal` be imported from another
module, so read it as a worked example and write your own.

## For an assistant building a plugin

Hold to each of these; they are the contract restated as rules, then the order to build in.

- **Read this whole guide and `internal/infrastructure/config/cues.toml` first.** This guide is the
  authority on the interface. Where anything read elsewhere disagrees with it, the guide wins; say so.
- **Build in the plugin's own repository.** Add nothing to this repository and name nothing about the
  audio, its source or its arrangement here (CON-9).
- **Answer the size and the answer from one code path**, as the worked example does, so they cannot
  disagree. Allocate nothing for Bridge Talk to free, keep no pointer past the call and answer `-1`
  for any failure rather than a partial answer.
- **Keep `pluginName` and every `voiceId` stable** across releases; they are how a cast voice is found
  again.
- **Give absolute UTF-8 paths** to files that exist. Write nothing to the audio the plugin reads.
- **Keep `BridgeTalkPluginTakes` cheap:** build the map from cue id to takes once, not on every call.
- **Build a 64 bit library** with the three functions exported undecorated, with C linkage.

Build in this order, proving each step before the next:

1. `BridgeTalkPluginABIVersion` answering 1.
2. `BridgeTalkPluginDescribe` offering one voice, ready, with a stable id.
3. `BridgeTalkPluginTakes` answering no takes for every cue.
4. The map from cue id to takes, one cue first, then the rest.
5. The ready check and its reason, for audio that has been moved or removed.

**Say what was proved and what was not.** Compiling is not proof that Bridge Talk loads a plugin.
The proof is the steps in [Using a plugin](#using-a-plugin): the voice on the Cast pane, cast, a
moment heard, the run log naming nothing passed over. Where any of that has not been done, the
handover says so in as many words.

## A checklist before you publish

- `BridgeTalkPluginABIVersion` returns 1.
- All three functions are exported undecorated, with C linkage.
- Every buffer function answers a size when asked with `size` 0 and a `NULL` buffer.
- Every failure path returns a negative number rather than writing a partial answer.
- Your voice ids are stable across your releases.
- Your paths are absolute, UTF-8 and point at files that exist on the user's machine.
- Nothing your plugin does writes to the audio it reads.
- The library is 64 bit.
- Bridge Talk has loaded it: its voice is on the Cast pane and the run log names nothing passed over.
