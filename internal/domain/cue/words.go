package cue

import (
	"strings"
	"unicode"
)

// Title reads the id as words, for a reader rather than for the code.
//
// The words are generated rather than written. An id is the game's own spelling, so the
// reading is too: every segment is broken where its capitals begin a new word, a run of
// capitals stays as the initialism it is and everything else is lower case. The first
// segment names the moment; whatever narrows it follows a colon, so
// "StartJump.JumpType.Hyperspace" reads "Start jump: jump type hyperspace".
func (i ID) Title() string {
	segments := strings.Split(string(i), string(groupSeparator))
	head := phrase(segments[0])
	rest := phrase(strings.Join(segments[1:], " "))
	switch {
	case head == "":
		return capitalised(rest)
	case rest == "":
		return capitalised(head)
	default:
		return capitalised(head) + ": " + rest
	}
}

// Heading reads the id's group as words, which is what a list of cues is headed by.
func (i ID) Heading() string { return ID(i.Group()).Title() }

// phrase renders a run of segments as lower-case words, keeping initialisms whole.
func phrase(text string) string {
	found := words(text)
	for index, word := range found {
		if !initialism(word) {
			found[index] = strings.ToLower(word)
		}
	}
	return strings.Join(found, " ")
}

// words breaks text where a space or an underscore stands, where a capital follows a
// lower-case letter or a digit and where a run of capitals gives way to a capitalised
// word, so "FSDJump" is "FSD" then "Jump".
func words(text string) []string {
	runes := []rune(text)
	var out []string
	start := 0
	for index, letter := range runes {
		if letter == ' ' || letter == '_' {
			out = appendWord(out, runes[start:index])
			start = index + 1
			continue
		}
		if index == start || !unicode.IsUpper(letter) {
			continue
		}
		previous := runes[index-1]
		nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
		if unicode.IsLower(previous) || unicode.IsDigit(previous) || nextIsLower {
			out = appendWord(out, runes[start:index])
			start = index
		}
	}
	return appendWord(out, runes[start:])
}

// appendWord adds a word unless it is empty.
func appendWord(out []string, word []rune) []string {
	if len(word) == 0 {
		return out
	}
	return append(out, string(word))
}

// initialism reports whether a word is a run of two or more capitals.
func initialism(word string) bool {
	letters := 0
	for _, letter := range word {
		if unicode.IsLower(letter) {
			return false
		}
		if unicode.IsUpper(letter) {
			letters++
		}
	}
	return letters > 1
}

// capitalised raises the first letter of a phrase.
func capitalised(text string) string {
	for index, letter := range text {
		return string(unicode.ToUpper(letter)) + text[index+len(string(letter)):]
	}
	return text
}
