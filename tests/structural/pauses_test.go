package structural

// FR-554: the shipped pauses.toml checked against the machine voices, the shipped script and the
// digests the model files list gives (FR-535), every stale pause named.

import (
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/pause"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
)

// freshSilence is the silence the books built below insert: 40 ms at the model's 24 kHz (FR-553).
// The check never reads it.
const freshSilence = 960

// freshDigest is the digest the books built below give every pause.
const freshDigest = "0123456789abcdef"

// stalePauses returns every way a book is stale against the voices, the voiced script and the digests
// the model files list gives the model file and each voice's style file (FR-554). The rules live in
// the domain; this hands them the list's digests.
func stalePauses(book pause.Book, voices []machinevoice.Voice, voiced script.Voiced, listed []modelfiles.File) []error {
	digests := make(map[string]string, len(listed))
	for _, file := range listed {
		digests[file.Name] = file.SHA256
	}
	styles := make(map[string]string, len(voices))
	for _, voice := range voices {
		styles[voice.ID()] = digests[voicefiles.StyleFile(voice)]
	}
	return pause.Check(book, voices, voiced, digests[voicefiles.ModelFile], styles)
}

// shippedPauseInputs loads the shipped script with its saved sounds and the shipped model files list.
func shippedPauseInputs(t *testing.T) (script.Voiced, []modelfiles.File) {
	t.Helper()
	table, err := config.LoadCueTable("")
	if err != nil {
		t.Fatalf("loading the shipped cue table: %v", err)
	}
	voiced, err := config.LoadVoicedScript(table)
	if err != nil {
		t.Fatalf("script.toml with sounds.toml: %v", err)
	}
	listed, err := modelfiles.Listed()
	if err != nil {
		t.Fatalf("the model files list: %v", err)
	}
	return voiced, listed
}

// TestTheShippedPausesAreNotStale holds FR-554 over pauses.toml, naming every stale pause rather than
// the first alone. It fails until the pauses tool has written pauses.toml (FR-551).
func TestTheShippedPausesAreNotStale(t *testing.T) {
	voiced, listed := shippedPauseInputs(t)
	book, err := config.LoadPauses()
	if err != nil {
		t.Fatalf("pauses.toml: %v", err)
	}
	for _, problem := range stalePauses(book, machinevoice.All(), voiced, listed) {
		t.Errorf("pauses.toml: %v", problem)
	}
}

// freshVoices is what the pauses tool would write over the shipped script with the listed files: a
// pause for every line the script joins in each voice's accent.
func freshVoices(voiced script.Voiced, listed []modelfiles.File) map[string]pause.Voice {
	voices := make(map[string]pause.Voice)
	for _, voice := range machinevoice.All() {
		var entries []pause.Entry
		for _, line := range voiced.Joined(voice.Accent()) {
			entries = append(entries, pause.Entry{Cue: line.Cue, Index: line.Index, Sounds: line.Sounds, Digest: freshDigest, Sample: 1})
		}
		at := slices.IndexFunc(listed, func(file modelfiles.File) bool { return file.Name == voicefiles.StyleFile(voice) })
		voices[voice.ID()] = pause.Voice{Style: listed[at].SHA256, Entries: entries}
	}
	return voices
}

// listedDigest returns the SHA-256 the list gives a file.
func listedDigest(listed []modelfiles.File, name string) string {
	return listed[slices.IndexFunc(listed, func(file modelfiles.File) bool { return file.Name == name })].SHA256
}

// staleOver builds a book from the voices given with the model digest given, then answers what
// stalePauses says of it over the shipped script and the list given.
func staleOver(t *testing.T, voices map[string]pause.Voice, model string, voiced script.Voiced, listed []modelfiles.File) []string {
	t.Helper()
	book, err := pause.NewBook(freshSilence, model, voices)
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	var words []string
	for _, problem := range stalePauses(book, machinevoice.All(), voiced, listed) {
		words = append(words, problem.Error())
	}
	return words
}

// namesOne asserts a single problem naming every fragment given.
func namesOne(t *testing.T, words []string, fragments ...string) {
	t.Helper()
	if len(words) != 1 {
		t.Fatalf("got %d problems %q, want one", len(words), words)
	}
	for _, fragment := range fragments {
		if !strings.Contains(words[0], fragment) {
			t.Errorf("%q does not name %q", words[0], fragment)
		}
	}
}

// withBreathable answers the voices with bf_emma's pause for "Breathable atmosphere, commander."
// changed as given.
func withBreathable(voices map[string]pause.Voice, change func(entries []pause.Entry, at int) []pause.Entry) map[string]pause.Voice {
	emma := voices["bf_emma"]
	entries := slices.Clone(emma.Entries)
	at := slices.IndexFunc(entries, func(entry pause.Entry) bool { return entry.Cue == "BreathableAtmosphere.Set" && entry.Index == 1 })
	emma.Entries = change(entries, at)
	voices["bf_emma"] = emma
	return voices
}

// The helper passes pauses found for every joined line of the shipped script with the listed files.
func TestPausesFoundForTheShippedScriptWithTheListedFilesPass(t *testing.T) {
	voiced, listed := shippedPauseInputs(t)
	if words := staleOver(t, freshVoices(voiced, listed), listedDigest(listed, voicefiles.ModelFile), voiced, listed); len(words) != 0 {
		t.Errorf("fresh pauses are stale: %q", words)
	}
}

// A joined line of the shipped script with no pause is named with its voice, its cue and its text.
func TestAShippedJoinedLineWithNoPauseIsNamed(t *testing.T) {
	voiced, listed := shippedPauseInputs(t)
	voices := withBreathable(freshVoices(voiced, listed), func(entries []pause.Entry, at int) []pause.Entry {
		return slices.Delete(entries, at, at+1)
	})
	namesOne(t, staleOver(t, voices, listedDigest(listed, voicefiles.ModelFile), voiced, listed),
		`bf_emma "BreathableAtmosphere.Set" line 2 "Breathable atmosphere, commander."`, "no pause")
}

// FR-554's acceptance: a pause found in sounds other than the line's now, as after the sounds tool ran
// and the pauses tool did not, is named with the cue and the line.
func TestAShippedPauseFoundInOtherSoundsIsNamed(t *testing.T) {
	voiced, listed := shippedPauseInputs(t)
	voices := withBreathable(freshVoices(voiced, listed), func(entries []pause.Entry, at int) []pause.Entry {
		entries[at].Sounds = "ˈɛː ɪz bɹˈiːðəbᵊlkəmˈɑndə."
		return entries
	})
	namesOne(t, staleOver(t, voices, listedDigest(listed, voicefiles.ModelFile), voiced, listed),
		`bf_emma "BreathableAtmosphere.Set" line 2 "Breathable atmosphere, commander."`, "ˈɛː ɪz bɹˈiːðəbᵊlkəmˈɑndə.")
}

// Pauses found with a model other than the listed one are named with both digests; so is a voice
// found with a style file other than the listed one.
func TestShippedPausesFoundWithOtherFilesAreNamed(t *testing.T) {
	voiced, listed := shippedPauseInputs(t)
	model := listedDigest(listed, voicefiles.ModelFile)
	namesOne(t, staleOver(t, freshVoices(voiced, listed), "an old model digest", voiced, listed),
		"model", `"an old model digest"`, model)

	voices := freshVoices(voiced, listed)
	emma := voices["bf_emma"]
	emma.Style = "an old style digest"
	voices["bf_emma"] = emma
	namesOne(t, staleOver(t, voices, model, voiced, listed), "bf_emma", "style", `"an old style digest"`)
}
