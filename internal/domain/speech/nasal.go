package speech

import "slices"

// finalNasals are the speech sounds a line ending on a nasal ends on: n, m and ŋ (FR-555).
var finalNasals = []rune{'n', 'm', 'ŋ'}

// notSounds are the symbols the model reads that are no speech sound of their own: the punctuation,
// the space, the stress and length marks and the marks that change or intone the sound before them. A
// line's last speech sound is read past them (FR-555). A combining mark or a dash is written as an
// escape, as in the symbols table.
var notSounds = []rune{
	';', ':', ',', '.', '!', '?', '\u2014', '…', '"', '(', ')', '“', '”', ' ',
	'ˈ', 'ˌ', 'ː', '\u0303', 'ʰ', 'ʲ', '↓', '→', '↗', '↘',
}

// EndsOnNasal reports whether the last speech sound of a line's speech sounds is n, m or ŋ, reading
// past the symbols after it that are no speech sound (FR-555).
func EndsOnNasal(sounds string) bool {
	symbols := []rune(sounds)
	for at := len(symbols) - 1; at >= 0; at-- {
		if !slices.Contains(notSounds, symbols[at]) {
			return slices.Contains(finalNasals, symbols[at])
		}
	}
	return false
}
