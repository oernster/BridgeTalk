package tomlfile

import (
	"bytes"

	"github.com/BurntSushi/toml"
)

// Encode writes shape as TOML: the one place the application writes TOML, beside the one place it
// reads it.
func Encode(shape any) ([]byte, error) {
	var written bytes.Buffer
	if err := toml.NewEncoder(&written).Encode(shape); err != nil {
		return nil, err
	}
	return written.Bytes(), nil
}
