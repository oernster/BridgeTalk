package speech_test

// The style row the model reads beside a line's numbers: one row for each count of symbols a
// line can come to, as Kokoro chooses it.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// numberedStyle builds a style whose every row is filled with its own row number, so the row
// For chose can be read off any value in it.
func numberedStyle(t *testing.T) speech.Style {
	t.Helper()
	values := make([]float32, 0, speech.MaxSymbols*speech.StyleWidth)
	for row := range speech.MaxSymbols {
		for range speech.StyleWidth {
			values = append(values, float32(row))
		}
	}
	style, err := speech.NewStyle(values)
	if err != nil {
		t.Fatalf("NewStyle: %v", err)
	}
	return style
}

// mustTokens turns speech sounds the test expects to be readable into the model's numbers.
func mustTokens(t *testing.T, sounds string) []int64 {
	t.Helper()
	tokens, err := speech.Tokens(sounds)
	if err != nil {
		t.Fatalf("Tokens(%q): %v", sounds, err)
	}
	return tokens
}

// A line's row is chosen by how many symbols it comes to, the boundary at each end not counted:
// one symbol reads the first row; MaxSymbols read the last.
func TestAStyleRowIsChosenByHowManySymbolsALineComesTo(t *testing.T) {
	style := numberedStyle(t)
	for sounds, row := range map[string]float32{
		"k":                                    0,
		"kəm":                                  2,
		strings.Repeat("ə", speech.MaxSymbols): speech.MaxSymbols - 1,
	} {
		got, err := style.For(mustTokens(t, sounds))
		if err != nil {
			t.Fatalf("For(%d symbols): %v", len([]rune(sounds)), err)
		}
		if len(got) != speech.StyleWidth || slices.ContainsFunc(got, func(value float32) bool { return value != row }) {
			t.Errorf("For(%d symbols) did not answer the whole of row %v", len([]rune(sounds)), row)
		}
	}
}

// A style file that is not MaxSymbols rows of StyleWidth numbers is refused, as is a style made
// without NewStyle.
func TestAStyleOfTheWrongSizeIsRefused(t *testing.T) {
	for _, count := range []int{0, speech.MaxSymbols*speech.StyleWidth - 1, speech.MaxSymbols*speech.StyleWidth + 1} {
		if _, err := speech.NewStyle(make([]float32, count)); !errors.Is(err, speech.ErrMisshapenStyle) {
			t.Errorf("NewStyle(%d numbers) = %v, want ErrMisshapenStyle", count, err)
		}
	}
	if _, err := (speech.Style{}).For(mustTokens(t, "k")); !errors.Is(err, speech.ErrMisshapenStyle) {
		t.Errorf("a style made without NewStyle answered %v, want ErrMisshapenStyle", err)
	}
}

// A line of no sounds has no row to read; numbers longer than the model reads have none either.
func TestALineWithNoRowIsRefused(t *testing.T) {
	style := numberedStyle(t)
	if _, err := style.For(mustTokens(t, "")); !errors.Is(err, speech.ErrNoSounds) {
		t.Errorf("no sounds answered %v, want ErrNoSounds", err)
	}
	tooMany := make([]int64, speech.MaxSymbols+len(mustTokens(t, ""))+1)
	if _, err := style.For(tooMany); !errors.Is(err, speech.ErrTooLong) {
		t.Errorf("%d numbers answered %v, want ErrTooLong", len(tooMany), err)
	}
}

// The values handed in and the row handed out are each the holder's own, so changing either
// changes nothing inside the style.
func TestAStylesValuesAreItsOwn(t *testing.T) {
	values := make([]float32, speech.MaxSymbols*speech.StyleWidth)
	style, err := speech.NewStyle(values)
	if err != nil {
		t.Fatalf("NewStyle: %v", err)
	}
	values[0] = 1
	row, _ := style.For(mustTokens(t, "k"))
	row[1] = 1
	again, _ := style.For(mustTokens(t, "k"))
	if again[0] != 0 || again[1] != 0 {
		t.Errorf("the style changed when a caller's slice did: row starts %v", again[:2])
	}
}
