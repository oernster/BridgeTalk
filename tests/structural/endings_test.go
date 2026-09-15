package structural

// FR-557: the shipped endings.toml checked against the machine voices, the shipped script and the
// digests the model files list gives (FR-535), every stale ending named.

import (
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/ending"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
)

// freshFade is the fade the books built below apply: 30 ms at the model's 24 kHz (FR-556). The check
// never reads it.
const freshFade = 720

// staleEndings returns every way a book is stale against the voices, the voiced script and the digests
// the model files list gives (FR-557). The rules live in the domain; this hands them the list's digests.
func staleEndings(book ending.Book, voices []machinevoice.Voice, voiced script.Voiced, listed []modelfiles.File) []error {
	model, styles := listedDigests(listed, voices)
	return ending.Check(book, voices, voiced, model, styles)
}

// TestTheShippedEndingsAreNotStale holds FR-557 over endings.toml, naming every stale ending rather than
// the first alone. It fails until the pauses tool has written endings.toml (FR-555).
func TestTheShippedEndingsAreNotStale(t *testing.T) {
	voiced, listed := shippedPauseInputs(t)
	book, err := config.LoadEndings()
	if err != nil {
		t.Fatalf("endings.toml: %v", err)
	}
	for _, problem := range staleEndings(book, machinevoice.All(), voiced, listed) {
		t.Errorf("endings.toml: %v", problem)
	}
}

// freshEndings is what the pauses tool would write over the shipped script with the listed files: an
// ending for every line ending on a nasal in each voice's accent.
func freshEndings(voiced script.Voiced, listed []modelfiles.File) map[string]ending.Voice {
	voices := make(map[string]ending.Voice)
	for _, voice := range machinevoice.All() {
		var entries []ending.Entry
		for _, line := range voiced.EndingOnNasal(voice.Accent()) {
			entries = append(entries, ending.Entry{Cue: line.Cue, Index: line.Index, Sounds: line.Sounds, Digest: freshDigest, Sample: 1})
		}
		voices[voice.ID()] = ending.Voice{Style: listedDigest(listed, voicefiles.StyleFile(voice)), Entries: entries}
	}
	return voices
}

// staleEndingsOver builds a book from the voices given with the model digest given, then answers what
// staleEndings says of it over the shipped script and the list given.
func staleEndingsOver(t *testing.T, voices map[string]ending.Voice, model string, voiced script.Voiced, listed []modelfiles.File) []string {
	t.Helper()
	book, err := ending.NewBook(freshFade, model, voices)
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	var words []string
	for _, problem := range staleEndings(book, machinevoice.All(), voiced, listed) {
		words = append(words, problem.Error())
	}
	return words
}

// withHelm answers the voices with bf_emma's ending for "Back at the helm." changed as given.
func withHelm(voices map[string]ending.Voice, change func(entries []ending.Entry, at int) []ending.Entry) map[string]ending.Voice {
	emma := voices["bf_emma"]
	entries := slices.Clone(emma.Entries)
	at := slices.IndexFunc(entries, func(entry ending.Entry) bool { return entry.Cue == "InMainShip.Set" && entry.Index == 2 })
	emma.Entries = change(entries, at)
	voices["bf_emma"] = emma
	return voices
}

// The helper passes endings found for every line of the shipped script ending on a nasal with the
// listed files.
func TestEndingsFoundForTheShippedScriptWithTheListedFilesPass(t *testing.T) {
	voiced, listed := shippedPauseInputs(t)
	if words := staleEndingsOver(t, freshEndings(voiced, listed), listedDigest(listed, voicefiles.ModelFile), voiced, listed); len(words) != 0 {
		t.Errorf("fresh endings are stale: %q", words)
	}
}

// A line of the shipped script ending on a nasal with no ending is named with its voice, its cue and
// its text.
func TestAShippedLineEndingOnANasalWithNoEndingIsNamed(t *testing.T) {
	voiced, listed := shippedPauseInputs(t)
	voices := withHelm(freshEndings(voiced, listed), func(entries []ending.Entry, at int) []ending.Entry {
		return slices.Delete(entries, at, at+1)
	})
	namesOne(t, staleEndingsOver(t, voices, listedDigest(listed, voicefiles.ModelFile), voiced, listed),
		`bf_emma "InMainShip.Set" line 3 "Back at the helm."`, "no ending")
}

// FR-557's acceptance: an ending found in sounds other than the line's now, as after the sounds tool ran
// and the pauses tool did not, is named with the cue and the line.
func TestAShippedEndingFoundInOtherSoundsIsNamed(t *testing.T) {
	voiced, listed := shippedPauseInputs(t)
	voices := withHelm(freshEndings(voiced, listed), func(entries []ending.Entry, at int) []ending.Entry {
		entries[at].Sounds = "bˈak at ðə kəntɹˈQlz."
		return entries
	})
	namesOne(t, staleEndingsOver(t, voices, listedDigest(listed, voicefiles.ModelFile), voiced, listed),
		`bf_emma "InMainShip.Set" line 3 "Back at the helm."`, "bˈak at ðə kəntɹˈQlz.")
}

// Endings found with a model other than the listed one are named with both digests; so is a voice found
// with a style file other than the listed one.
func TestShippedEndingsFoundWithOtherFilesAreNamed(t *testing.T) {
	voiced, listed := shippedPauseInputs(t)
	model := listedDigest(listed, voicefiles.ModelFile)
	namesOne(t, staleEndingsOver(t, freshEndings(voiced, listed), "an old model digest", voiced, listed),
		"model", `"an old model digest"`, model)

	voices := freshEndings(voiced, listed)
	emma := voices["bf_emma"]
	emma.Style = "an old style digest"
	voices["bf_emma"] = emma
	namesOne(t, staleEndingsOver(t, voices, model, voiced, listed), "bf_emma", "style", `"an old style digest"`)
}
