package config

import (
	_ "embed"
	"fmt"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/ending"
	"github.com/oernster/bridge-talk/internal/infrastructure/tomlfile"
)

//go:embed endings.toml
var embeddedEndings []byte

// endingsHeader opens endings.toml, saying whose file it is and how it is made again.
const endingsHeader = "# The fade after a final nasal in each machine voice's lines, found by the pauses tool (FR-555).\n" +
	madeAgain

// endingsFile is the shape of endings.toml: how many samples a fade lasts, the digest of the model
// the endings were found with and each voice's endings under [voices.<id>]. A key is left out when it
// is zero or empty, so the empty book is written as the header alone.
type endingsFile struct {
	Fade   int                     `toml:"fade,omitzero"`
	Model  string                  `toml:"model,omitempty"`
	Voices map[string]endingsVoice `toml:"voices,omitempty"`
}

// endingsVoice is one voice's endings: the digest of its style file, then one [[voices.<id>.lines]]
// table for each line in the book's order.
type endingsVoice struct {
	Style string        `toml:"style"`
	Lines []endingsLine `toml:"lines"`
}

// endingsLine is one line's ending. A line ending on no burst is written with no sample (FR-555).
type endingsLine struct {
	Cue    string `toml:"cue"`
	Index  int    `toml:"index"`
	Sounds string `toml:"sounds"`
	Digest string `toml:"digest"`
	Sample int    `toml:"sample,omitzero"`
}

// LoadEndings reads the shipped endings, checked against the rules for a book (FR-555).
func LoadEndings() (ending.Book, error) {
	return parseEndings(embeddedEndings)
}

// parseEndings reads endings from their bytes, refusing a key their shape does not hold and a book
// the rules refuse.
func parseEndings(raw []byte) (ending.Book, error) {
	var parsed endingsFile
	if err := tomlfile.Decode(raw, &parsed); err != nil {
		return ending.Book{}, fmt.Errorf("parsing endings: %w", err)
	}
	voices := make(map[string]ending.Voice, len(parsed.Voices))
	for id, voice := range parsed.Voices {
		entries := make([]ending.Entry, 0, len(voice.Lines))
		for _, line := range voice.Lines {
			entries = append(entries, ending.Entry{
				Cue: cue.ID(line.Cue), Index: line.Index, Sounds: line.Sounds, Digest: line.Digest, Sample: line.Sample,
			})
		}
		voices[id] = ending.Voice{Style: voice.Style, Entries: entries}
	}
	return ending.NewBook(parsed.Fade, parsed.Model, voices)
}

// EncodeEndings writes a book in the shape endings.toml holds, under its header: voices sorted, each
// voice's lines in the book's order, so writing the same book again writes the same bytes (FR-555).
// The bytes mean nothing when the error is not nil.
func EncodeEndings(book ending.Book) ([]byte, error) {
	file := endingsFile{Fade: book.Fade(), Model: book.Model(), Voices: make(map[string]endingsVoice)}
	for _, id := range book.Voices() {
		voice, _ := book.Voice(id)
		lines := make([]endingsLine, 0, len(voice.Entries))
		for _, entry := range voice.Entries {
			lines = append(lines, endingsLine{
				Cue: string(entry.Cue), Index: entry.Index, Sounds: entry.Sounds, Digest: entry.Digest, Sample: entry.Sample,
			})
		}
		file.Voices[id] = endingsVoice{Style: voice.Style, Lines: lines}
	}
	written, err := tomlfile.Encode(file)
	return append([]byte(endingsHeader), written...), err
}
