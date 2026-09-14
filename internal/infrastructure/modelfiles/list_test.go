package modelfiles_test

// FR-535: the one list of the files a machine voice is made from, with where each comes from.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
	"github.com/oernster/bridge-talk/internal/infrastructure/tomlfile"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
)

// The shipped list names the model, ONNX Runtime, the tokenizer file and one style file for each
// voice offered; nothing else. The names come from where the application reads them, so the
// list cannot drift from them.
func TestTheListNamesEveryFileAVoiceIsMadeFrom(t *testing.T) {
	t.Parallel()
	files, err := modelfiles.Listed()
	if err != nil {
		t.Fatalf("Listed: %v", err)
	}

	want := []string{voicefiles.ModelFile, voicefiles.RuntimeFile, modelfiles.TokenizerFile}
	for _, voice := range machinevoice.All() {
		want = append(want, voicefiles.StyleFile(voice))
	}
	got := make([]string, 0, len(files))
	for _, file := range files {
		got = append(got, file.Name)
	}
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("the list names %v, want %v", got, want)
	}
}

// aDigest is a well formed SHA-256 for the lists written below.
var aDigest = strings.Repeat("0123456789abcdef", 4)

// trusted is a list every refusal below changes in one place.
var trusted = `[sources]
pinned = "https://example.com/pinned/"

[[files]]
name = "bf_alice.bin"
source = "pinned"
path = "voices/bf_alice.bin"
size = 4
sha256 = "` + aDigest + `"
`

// An entry's address is its source followed by its path.
func TestAnEntrysAddressIsItsSourceThenItsPath(t *testing.T) {
	t.Parallel()
	files, err := modelfiles.Parse([]byte(trusted))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := modelfiles.File{
		Name: "bf_alice.bin", Address: "https://example.com/pinned/voices/bf_alice.bin", Size: 4, SHA256: aDigest,
	}
	if len(files) != 1 || files[0] != want {
		t.Errorf("Parse = %+v, want %+v", files, want)
	}
}

// A list that could fetch the wrong thing or put it in the wrong place is refused, saying which
// entry and why.
func TestAListThatCannotBeTrustedIsRefused(t *testing.T) {
	t.Parallel()
	cases := []struct {
		what, from, to string
		want           error
	}{
		{"an unknown source", `source = "pinned"`, `source = "elsewhere"`, modelfiles.ErrMisshapenEntry},
		{"a source reached without https", `"https://example.com/pinned/"`, `"http://example.com/pinned/"`, modelfiles.ErrMisshapenEntry},
		{"no name", `name = "bf_alice.bin"`, `name = ""`, modelfiles.ErrMisshapenEntry},
		{"a name that climbs out of the folder", `name = "bf_alice.bin"`, `name = "../bf_alice.bin"`, modelfiles.ErrMisshapenEntry},
		{"no path", `path = "voices/bf_alice.bin"`, `path = ""`, modelfiles.ErrMisshapenEntry},
		{"no size", `size = 4`, `size = 0`, modelfiles.ErrMisshapenEntry},
		{"a short digest", aDigest, "abc", modelfiles.ErrMisshapenEntry},
		{"a digest that is not hexadecimal", aDigest, strings.Repeat("g", len(aDigest)), modelfiles.ErrMisshapenEntry},
		{"a misspelled key", `sha256 =`, `sha256sum =`, tomlfile.ErrUnknownKey},
	}
	for _, each := range cases {
		_, err := modelfiles.Parse([]byte(strings.Replace(trusted, each.from, each.to, 1)))
		if !errors.Is(err, each.want) {
			t.Errorf("%s: Parse = %v, want %v", each.what, err, each.want)
		}
	}

	_, err := modelfiles.Parse([]byte(trusted + trusted[strings.Index(trusted, "[[files]]"):]))
	if !errors.Is(err, modelfiles.ErrMisshapenEntry) || !strings.Contains(err.Error(), "bf_alice.bin") {
		t.Errorf("a name given twice: Parse = %v, want %v naming bf_alice.bin", err, modelfiles.ErrMisshapenEntry)
	}
}
