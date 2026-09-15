package ending_test

// FR-557: endings checked against the shipped script, the voices and the digests the list gives, each
// stale one named.

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/ending"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/script/scripttest"
)

// styleDigests are the digests the list gives each voice's style file.
var styleDigests = map[string]string{"bf_emma": "emma style digest", "am_michael": "michael style digest"}

// nasalScript is Docked and Undocked as the sounds tool saves them: Docked's lines 1 and 3 end on a
// nasal in each accent, as does Undocked's line 3.
func nasalScript(t *testing.T) script.Voiced {
	t.Helper()
	voiced, err := scripttest.Build(map[string]script.Saved{
		"Docked": {
			Lines:    []string{"Down.", "Docked.", "Moving on."},
			British:  []string{"dˈWn.", "dˈɒkt.", "mˈuːvɪŋ ˈɒn."},
			American: []string{"dˈWn.", "dˈɑkt.", "mˈuvɪŋ ˈɔn."},
		},
		"Undocked": {
			Lines:    []string{"Undocked.", "Clear.", "Back at the helm."},
			British:  []string{"ʌndˈɒkt.", "klˈɪə.", "bˈak at ðə hˈɛlm."},
			American: []string{"ʌndˈɑkt.", "klˈɪɹ.", "bˈæk æt ðə hˈɛlm."},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return voiced
}

// cast is the voices checked: a British voice then an American one.
func cast(t *testing.T) []machinevoice.Voice {
	t.Helper()
	var voices []machinevoice.Voice
	for _, id := range []string{"bf_emma", "am_michael"} {
		voice, err := machinevoice.Parse(id)
		if err != nil {
			t.Fatalf("Parse(%q): %v", id, err)
		}
		voices = append(voices, voice)
	}
	return voices
}

// freshVoices is what the pauses tool writes for the cast over a script: an ending for every line
// ending on a nasal in each voice's accent, found with the listed style file.
func freshVoices(t *testing.T, voiced script.Voiced) map[string]ending.Voice {
	t.Helper()
	voices := make(map[string]ending.Voice)
	for _, voice := range cast(t) {
		var entries []ending.Entry
		for place, line := range voiced.EndingOnNasal(voice.Accent()) {
			entries = append(entries, ending.Entry{Cue: line.Cue, Index: line.Index, Sounds: line.Sounds, Digest: "0123456789abcdef", Sample: place})
		}
		voices[voice.ID()] = ending.Voice{Style: styleDigests[voice.ID()], Entries: entries}
	}
	return voices
}

// staleWords checks the endings given against the script, the cast and the digests given, returning
// what each problem says; every problem must be ErrStale and cite FR-557.
func staleWords(t *testing.T, voices map[string]ending.Voice, voiced script.Voiced, model string, styles map[string]string) []string {
	t.Helper()
	book, err := ending.NewBook(fade, modelDigest, voices)
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	var words []string
	for _, problem := range ending.Check(book, cast(t), voiced, model, styles) {
		if !errors.Is(problem, ending.ErrStale) || !strings.Contains(problem.Error(), "(FR-557)") {
			t.Errorf("%v is not ErrStale citing FR-557", problem)
		}
		words = append(words, problem.Error())
	}
	return words
}

// saysEach asserts one problem for each list of fragments, in order, each problem naming its own.
func saysEach(t *testing.T, words []string, fragments ...[]string) {
	t.Helper()
	if len(words) != len(fragments) {
		t.Fatalf("got %d problems %q, want %d", len(words), words, len(fragments))
	}
	for index, wanted := range fragments {
		for _, fragment := range wanted {
			if !strings.Contains(words[index], fragment) {
				t.Errorf("problem %d %q does not name %q", index+1, words[index], fragment)
			}
		}
	}
}

// withEmma answers the voices with bf_emma's entries changed as given, the rest left alone.
func withEmma(voices map[string]ending.Voice, change func([]ending.Entry) []ending.Entry) map[string]ending.Voice {
	emma := voices["bf_emma"]
	emma.Entries = change(slices.Clone(emma.Entries))
	voices["bf_emma"] = emma
	return voices
}

// Endings found for every line ending on a nasal with the listed model and style files are not stale.
func TestEndingsFoundForTheScriptWithTheListedFilesAreNotStale(t *testing.T) {
	t.Parallel()
	voiced := nasalScript(t)
	if words := staleWords(t, freshVoices(t, voiced), voiced, modelDigest, styleDigests); len(words) != 0 {
		t.Errorf("fresh endings are stale: %q", words)
	}
}

// A voice with no endings at all is stale, named once rather than line by line.
func TestAVoiceWithNoEndingsIsStaleNamingIt(t *testing.T) {
	t.Parallel()
	voiced := nasalScript(t)
	voices := freshVoices(t, voiced)
	delete(voices, "am_michael")
	saysEach(t, staleWords(t, voices, voiced, modelDigest, styleDigests), []string{"am_michael", "no endings"})
}

// A line ending on a nasal with no ending for a voice is stale, naming the voice, the cue, the line as
// the script counts it and the line's text.
func TestALineEndingOnANasalWithNoEndingIsStaleNamingItAndItsText(t *testing.T) {
	t.Parallel()
	voiced := nasalScript(t)
	voices := withEmma(freshVoices(t, voiced), func(entries []ending.Entry) []ending.Entry {
		return slices.DeleteFunc(entries, func(entry ending.Entry) bool { return entry.Cue == "Docked" && entry.Index == 2 })
	})
	saysEach(t, staleWords(t, voices, voiced, modelDigest, styleDigests),
		[]string{`bf_emma "Docked" line 3 "Moving on."`, "no ending"})
}

// An ending for a line that no longer ends on a nasal is stale, naming the line's text where the script
// still has the line; where it does not, the cue and the line alone.
func TestAnEndingForALineThatNoLongerEndsOnANasalIsStaleNamingIt(t *testing.T) {
	t.Parallel()
	voiced := nasalScript(t)
	voices := withEmma(freshVoices(t, voiced), func(entries []ending.Entry) []ending.Entry {
		return append(entries,
			ending.Entry{Cue: "Docked", Index: 1, Sounds: "dˈɒkt.", Digest: "0123456789abcdef"},
			ending.Entry{Cue: "Launched", Index: 0, Sounds: "lˈɔːntʃt.", Digest: "0123456789abcdef"},
			ending.Entry{Cue: "Undocked", Index: 3, Sounds: "ʌndˈɒkt.", Digest: "0123456789abcdef"},
		)
	})
	words := staleWords(t, voices, voiced, modelDigest, styleDigests)
	saysEach(t, words,
		[]string{`bf_emma "Docked" line 2 "Docked."`, "no longer ends on a nasal"},
		[]string{`bf_emma "Launched" line 1`, "no longer ends on a nasal"},
		[]string{`bf_emma "Undocked" line 4`, "no longer ends on a nasal"},
	)
	for _, word := range words[1:] {
		if strings.Contains(word, `line 1 "`) || strings.Contains(word, `line 4 "`) {
			t.Errorf("%q names text the script does not have", word)
		}
	}
}

// FR-557's acceptance in the domain: an ending found in a line made from speech sounds other than the
// line's now is stale, naming the voice, the cue, the line and both sounds.
func TestAnEndingFoundInSoundsOtherThanTheLinesNowIsStale(t *testing.T) {
	t.Parallel()
	voiced := nasalScript(t)
	voices := withEmma(freshVoices(t, voiced), func(entries []ending.Entry) []ending.Entry {
		entries[2].Sounds = "bˈak at ðə kəntɹˈQlz."
		return entries
	})
	saysEach(t, staleWords(t, voices, voiced, modelDigest, styleDigests),
		[]string{`bf_emma "Undocked" line 3 "Back at the helm."`, `"bˈak at ðə kəntɹˈQlz."`, `"bˈak at ðə hˈɛlm."`})
}

// Endings found with a model other than the listed one are stale, naming both digests; so is a voice
// found with a style file other than the listed one.
func TestEndingsFoundWithOtherFilesAreStale(t *testing.T) {
	t.Parallel()
	voiced := nasalScript(t)
	saysEach(t, staleWords(t, freshVoices(t, voiced), voiced, "new model digest", styleDigests),
		[]string{"model", `"model digest"`, `"new model digest"`})
	styles := maps.Clone(styleDigests)
	styles["am_michael"] = "new michael style digest"
	saysEach(t, staleWords(t, freshVoices(t, voiced), voiced, modelDigest, styles),
		[]string{"am_michael", "style", `"michael style digest"`, `"new michael style digest"`})
}
