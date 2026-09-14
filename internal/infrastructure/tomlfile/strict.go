// Package tomlfile reads the TOML files the application is given: the cue table, the script, the
// saved speech sounds and a voice's manifest. It also writes the saved speech sounds for the
// sounds tool.
//
// Every file is read strictly. A key the shape does not hold is refused rather than dropped in
// silence, because a file edited by hand with a misspelled key would otherwise appear to say
// something it does not. That rule is written once, here, so the readers cannot come to disagree
// about it.
package tomlfile

import (
	"errors"
	"fmt"

	"github.com/BurntSushi/toml"
)

// ErrUnknownKey means the file holds a key the shape it was read into does not.
var ErrUnknownKey = errors.New("unknown key")

// Decode parses raw into shape, refusing a key the shape does not hold. A file that is not
// TOML answers with the parser's own error; so does one holding a value of the wrong type.
func Decode(raw []byte, shape any) error {
	meta, err := toml.Decode(string(raw), shape)
	if err != nil {
		return err
	}
	if unknown := meta.Undecoded(); len(unknown) > 0 {
		return fmt.Errorf("%w %s", ErrUnknownKey, unknown[0])
	}
	return nil
}
