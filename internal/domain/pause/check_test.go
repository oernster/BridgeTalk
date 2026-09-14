package pause_test

// FR-554: pauses checked against the shipped script, the voices and the digests the list gives, each
// stale one named.

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/pause"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/script/scripttest"
)

// modelDigest is the digest the list gives the model file.
const modelDigest = "model digest"

// styleDigests are the digests the list gives each voice's style file.
var styleDigests = map[string]string{"bf_emma": "emma style digest", "am_michael": "michael style digest"}

// joinedScript is Docked and Undocked as the sounds tool saves them with commander joined: Docked's
// lines 1 and 3 join, as does Undocked's line 1.
func joinedScript(t *testing.T) script.Voiced {
	t.Helper()
	voiced, err := scripttest.BuildJoining(map[string]script.Saved{
		"Docked": {
			Lines:    []string{"Docked, commander.", "We're down safely.", "Down safe, commander!"},
			British:  []string{"dˈɒktkəmˈɑndə.", "wɪə dˈWn", "dˈWn sˈAfkəmˈɑndə!"},
			American: []string{"dˈɑktkəmˈændəɹ.", "wɪɹ dˈWn", "dˈWn sˈAfkəmˈændəɹ!"},
		},
		"Undocked": {
			Lines:    []string{"Clear, commander?", "Undocked.", "Leaving, commander. Out."},
			British:  []string{"klˈɪəkəmˈɑndə?", "ʌndˈɒkt.", "lˈiːvɪŋ, kəmˈɑndə. ˈWt."},
			American: []string{"klˈɪɹkəmˈændəɹ?", "ʌndˈɑkt.", "lˈivɪŋ, kəmˈændəɹ. ˈWt."},
		},
	}, map[string][]string{"commander": {"kəmˈɑndə", "kəmˈændəɹ"}}, []string{"commander"})
	if err != nil {
		t.Fatalf("BuildJoining: %v", err)
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

// freshVoices is what the pauses tool writes for the cast over a script: a pause for every line the
// script joins in each voice's accent, found with the listed style file.
func freshVoices(t *testing.T, voiced script.Voiced) map[string]pause.Voice {
	t.Helper()
	voices := make(map[string]pause.Voice)
	for _, voice := range cast(t) {
		var entries []pause.Entry
		for place, line := range voiced.Joined(voice.Accent()) {
			entries = append(entries, pause.Entry{
				Cue: line.Cue, Index: line.Index, Sounds: line.Sounds, Digest: "0123456789abcdef", Sample: place + 1,
			})
		}
		voices[voice.ID()] = pause.Voice{Style: styleDigests[voice.ID()], Entries: entries}
	}
	return voices
}

// staleWords checks the pauses given against the script, the cast and the digests given, returning
// what each problem says; every problem must be ErrStale.
func staleWords(t *testing.T, voices map[string]pause.Voice, voiced script.Voiced, model string, styles map[string]string) []string {
	t.Helper()
	book, err := pause.NewBook(silence, modelDigest, voices)
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	var words []string
	for _, problem := range pause.Check(book, cast(t), voiced, model, styles) {
		if !errors.Is(problem, pause.ErrStale) {
			t.Errorf("%v is not ErrStale", problem)
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
func withEmma(voices map[string]pause.Voice, change func([]pause.Entry) []pause.Entry) map[string]pause.Voice {
	emma := voices["bf_emma"]
	emma.Entries = change(slices.Clone(emma.Entries))
	voices["bf_emma"] = emma
	return voices
}

// Pauses found for every joined line with the listed model and style files are not stale.
func TestPausesFoundForTheScriptWithTheListedFilesAreNotStale(t *testing.T) {
	t.Parallel()
	voiced := joinedScript(t)
	if words := staleWords(t, freshVoices(t, voiced), voiced, modelDigest, styleDigests); len(words) != 0 {
		t.Errorf("fresh pauses are stale: %q", words)
	}
}

// A voice with no pauses at all is stale, named once rather than line by line.
func TestAVoiceWithNoPausesIsStaleNamingIt(t *testing.T) {
	t.Parallel()
	voiced := joinedScript(t)
	voices := freshVoices(t, voiced)
	delete(voices, "am_michael")
	saysEach(t, staleWords(t, voices, voiced, modelDigest, styleDigests), []string{"am_michael", "no pauses"})
}

// A line the script joins with no pause for a voice is stale, naming the voice, the cue, the line as
// the script counts it and the line's text.
func TestAJoinedLineWithNoPauseIsStaleNamingTheVoiceTheCueTheLineAndItsText(t *testing.T) {
	t.Parallel()
	voiced := joinedScript(t)
	voices := withEmma(freshVoices(t, voiced), func(entries []pause.Entry) []pause.Entry {
		return slices.DeleteFunc(entries, func(entry pause.Entry) bool { return entry.Cue == "Docked" && entry.Index == 2 })
	})
	saysEach(t, staleWords(t, voices, voiced, modelDigest, styleDigests),
		[]string{`bf_emma "Docked" line 3 "Down safe, commander!"`, "no pause"})
}

// A pause for a line that no longer joins is stale, naming the line's text where the script still
// has the line; where it does not, the cue and the line alone.
func TestAPauseForALineThatNoLongerJoinsIsStaleNamingIt(t *testing.T) {
	t.Parallel()
	voiced := joinedScript(t)
	voices := withEmma(freshVoices(t, voiced), func(entries []pause.Entry) []pause.Entry {
		return append(entries,
			pause.Entry{Cue: "Docked", Index: 1, Sounds: "wɪə dˈWn", Digest: "0123456789abcdef", Sample: 1},
			pause.Entry{Cue: "Launched", Index: 0, Sounds: "lˈɔːntʃt", Digest: "0123456789abcdef", Sample: 1},
			pause.Entry{Cue: "Undocked", Index: 3, Sounds: "ʌndˈɒkt", Digest: "0123456789abcdef", Sample: 1},
		)
	})
	words := staleWords(t, voices, voiced, modelDigest, styleDigests)
	saysEach(t, words,
		[]string{`bf_emma "Docked" line 2 "We're down safely."`, "no longer joins"},
		[]string{`bf_emma "Launched" line 1 `, "no longer joins"},
		[]string{`bf_emma "Undocked" line 4 `, "no longer joins"},
	)
	for _, word := range words[1:] {
		if strings.Contains(word, `line 1 "`) || strings.Contains(word, `line 4 "`) {
			t.Errorf("%q names text the script does not have", word)
		}
	}
}

// FR-554's acceptance in the domain: a pause found in a line made from speech sounds other than the
// line's now is stale, naming the voice, the cue, the line and both sounds.
func TestAPauseFoundInSoundsOtherThanTheLinesNowIsStale(t *testing.T) {
	t.Parallel()
	voiced := joinedScript(t)
	voices := withEmma(freshVoices(t, voiced), func(entries []pause.Entry) []pause.Entry {
		entries[0].Sounds = "dˈɒkt nˈWkəmˈɑndə."
		return entries
	})
	saysEach(t, staleWords(t, voices, voiced, modelDigest, styleDigests),
		[]string{`bf_emma "Docked" line 1 "Docked, commander."`, `"dˈɒkt nˈWkəmˈɑndə."`, `"dˈɒktkəmˈɑndə."`})
}

// Pauses found with a model other than the listed one are stale, naming both digests.
func TestPausesFoundWithAnotherModelAreStale(t *testing.T) {
	t.Parallel()
	voiced := joinedScript(t)
	saysEach(t, staleWords(t, freshVoices(t, voiced), voiced, "new model digest", styleDigests),
		[]string{"model", `"model digest"`, `"new model digest"`})
}

// A voice whose pauses were found with a style file other than the listed one is stale, naming the
// voice and both digests.
func TestAVoiceFoundWithAnotherStyleFileIsStale(t *testing.T) {
	t.Parallel()
	voiced := joinedScript(t)
	styles := maps.Clone(styleDigests)
	styles["am_michael"] = "new michael style digest"
	saysEach(t, staleWords(t, freshVoices(t, voiced), voiced, modelDigest, styles),
		[]string{"am_michael", "style", `"michael style digest"`, `"new michael style digest"`})
}

// Problems come in one order however often the check runs: the model first, then each voice in the
// order given with its style file, its joined lines in the script's order then its pauses for lines
// that no longer join.
func TestStaleProblemsComeInOneOrder(t *testing.T) {
	t.Parallel()
	voiced := joinedScript(t)
	voices := withEmma(freshVoices(t, voiced), func(entries []pause.Entry) []pause.Entry {
		gone := pause.Entry{Cue: "Docked", Index: 1, Sounds: "wɪə dˈWn", Digest: "0123456789abcdef", Sample: 1}
		changed := entries[2]
		changed.Sounds = "klˈɪə nˈWkəmˈɑndə?"
		return []pause.Entry{gone, changed, entries[0]}
	})
	delete(voices, "am_michael")
	styles := maps.Clone(styleDigests)
	styles["bf_emma"] = "new emma style digest"
	first := staleWords(t, voices, voiced, "new model digest", styles)
	saysEach(t, first,
		[]string{"model"},
		[]string{"bf_emma", "style"},
		[]string{`bf_emma "Docked" line 3`, "no pause"},
		[]string{`bf_emma "Undocked" line 1`, `"klˈɪə nˈWkəmˈɑndə?"`},
		[]string{`bf_emma "Docked" line 2`, "no longer joins"},
		[]string{"am_michael", "no pauses"},
	)
	for range 10 {
		if again := staleWords(t, voices, voiced, "new model digest", styles); !slices.Equal(again, first) {
			t.Fatalf("a second check said %q, the first %q", again, first)
		}
	}
}
