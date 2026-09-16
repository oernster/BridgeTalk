# Writing a Bridge Talk plugin

This is the contract between Bridge Talk and a plugin. It is the authority on the binary interface;
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

A plugin is a single library file in the `plugins` folder inside the folder Bridge Talk was installed
into, which the setup program creates (FR-576). On a default install that is:

```
%LOCALAPPDATA%\Programs\BridgeTalk\plugins
```

The install folder can be chosen at install time, so read the location from the Apps list entry
rather than assuming the default. Bridge Talk loads every file in that folder at startup and loads a
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
that offered it (FR-569), so keep it stable across releases of your plugin. `voiceName` may change
freely.

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

## When Bridge Talk refuses a plugin

Every refusal names what was refused and why, in the run log at `%LOCALAPPDATA%\BridgeTalk\Log.txt`
(FR-567). A plugin is passed over when the file will not load, when a required function is missing,
when the ABI version does not match, when it offers no voice or when it offers a voice with no name.
One plugin being passed over never stops another loading (FR-561).

If two plugins offer voices under the same name, both are kept and each is shown with the name of the
plugin offering it (FR-568).

## Linux

Linux is in scope for Bridge Talk after everything else. The contract is unchanged there: the same
three functions with the same rules, in a shared object built from your own repository. Nothing in
this document is Windows specific except the paths it gives as examples.

## Checking your layouts against ours

`internal/infrastructure/plugin/plugintest` in this repository writes these layouts in Go and is
held to the same rules your plugin is: the size before the answer, a refusal that is never a size,
then nothing written into a buffer too small to hold the whole answer. It is test support rather
than a library to depend on, so read it as a worked example and write your own.

## A checklist before you publish

- `BridgeTalkPluginABIVersion` returns 1.
- All three functions are exported undecorated, with C linkage.
- Every buffer function answers a size when asked with `size` 0 and a `NULL` buffer.
- Every failure path returns a negative number rather than writing a partial answer.
- Your voice ids are stable across your releases.
- Your paths are absolute, UTF-8 and point at files that exist on the user's machine.
- Nothing your plugin does writes to the audio it reads.
