package status

// FlagBit answers with the bit the game assigns to a name in the Flags word.
//
// It lets a test build a Flags value out of the vocabulary the cue table already uses
// rather than out of numbers nobody can read back. It lives in a test file, so it
// widens nothing that ships; the alternative was writing the game's bit assignments
// into the tests a second time, where they could drift from the table above them.
//
// Only the first Flags word is searched. The two words number their bits from one
// each, so a name from the second would answer with a value that means something else
// entirely in the first; a name that is not in this word is a mistake in the test and
// says so at once rather than quietly building the wrong state.
func FlagBit(name string) uint32 {
	for bit, known := range flagBits {
		if known == name {
			return bit
		}
	}
	panic("no bit in the Flags word is named " + name)
}
