package speech_test

// FR-555: a line ends on a nasal where its last speech sound is n, m or ŋ, read past the stress and
// length marks, the punctuation and the spaces after it.

import (
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// FR-555: the last speech sound decides. The lines of section 6.1 that burst end on n, ŋ and m; a line
// ending on a vowel, on commander or on t does not end on a nasal.
func TestALineEndsOnANasalWhereItsLastSpeechSoundIsNMOrEng(t *testing.T) {
	t.Parallel()
	for sounds, want := range map[string]bool{
		"jˌuːl biː hˈɪəɹɪŋ fɹɒm mˌiː fɹɒm hˈɪə ˈɒn.":        true,
		"tɹˈɛspəs wˈɔːnɪŋ. mˈuːv əlˈɒŋ.":                    true,
		"bˈak at ðə hˈɛlm.":                                 true,
		"ɹɪfjˈuːzd. wˌɪə tˈuː fˈɑː ˈWt tə ɹɪkwˈɛst dˈɒkɪŋ.": true,
		"vˈYs lˈɪŋk ɪstˈablɪʃt. ɹˈɛdi wˌɛn juː ɑː.":         false,
		"plˈanɪt klˈQs əhˈɛdkəmˈɑndə.":                      false,
		"sˈɛtlmᵊnt ɪn ɹˈAnʤ.":                               false,
		"ʌndˈɒkt.":                                          false,
	} {
		if got := speech.EndsOnNasal(sounds); got != want {
			t.Errorf("EndsOnNasal(%q) = %v, want %v", sounds, got, want)
		}
	}
}

// FR-555: stress and length marks, final marks, quotes and spaces after the last speech sound are read
// past, as is a line with no final mark at all.
func TestMarksAndPunctuationAfterTheLastSpeechSoundAreReadPast(t *testing.T) {
	t.Parallel()
	for _, sounds := range []string{"ˈɒn", "dˈʌn!", "ɡˈɒn?", "nˈ", "ŋˌ", "mː.", "ˈɒn…", "ˈɒn”", "ˈɒn. ", "ˈɒn;"} {
		if !speech.EndsOnNasal(sounds) {
			t.Errorf("EndsOnNasal(%q) = false, want true", sounds)
		}
	}
}

// FR-555 names n, m and ŋ alone: the model's other nasal symbols, a line with no speech sound and a
// nasal followed by another sound do not end on a nasal.
func TestOnlyNMAndEngCountAsTheNasalALineEndsOn(t *testing.T) {
	t.Parallel()
	for _, sounds := range []string{"ɳ.", "ɲ.", "ɴ.", "", ". ", "ˈ", "nt.", "ɒnə."} {
		if speech.EndsOnNasal(sounds) {
			t.Errorf("EndsOnNasal(%q) = true, want false", sounds)
		}
	}
}
