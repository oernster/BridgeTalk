package cue

import (
	"fmt"
	"strings"

	"github.com/oernster/bridge-talk/internal/domain/event"
)

// Comms moments (FR-617 to FR-619).
//
// A message the game sends a commander carries a key beside the words it generates for it, such
// as $Pirate_OnDeclarePiracyAttack07; for a pirate declaring an attack. One moment arrives under
// many keys that differ only in their variant number, some with values after it, so a comms moment
// names the key stem those keys share rather than any one key.

const (
	// keyPrefix begins every message key; a message without it is text a player typed (FR-618).
	keyPrefix = "$"
	// keyValues begins the values a key carries after its name, such as :#units=20.
	keyValues = ":#"
	// keyEnd closes a message key.
	keyEnd = ";"
	// keyVariantDigits are what a key's variant number is written in, straight after its name.
	keyVariantDigits = "0123456789"
)

// KeyStem reads the key stem out of a message key: the key with its leading $, the digits ending
// its name, any values from the first :# onwards and its closing ; removed. It answers false where
// the text is no key at all, which is how a message a player typed reads (FR-618).
func KeyStem(key string) (string, bool) {
	rest, keyed := strings.CutPrefix(key, keyPrefix)
	if !keyed {
		return "", false
	}
	rest, _, _ = strings.Cut(rest, keyValues)
	rest = strings.TrimRight(rest, keyEnd)
	rest = strings.TrimRight(rest, keyVariantDigits)
	return rest, rest != ""
}

// keyStems validates the key stem a definition names, keyed by the field holding the message key.
//
// A comms moment names one stem and no match field beside it, since the stem alone says which
// message it is. Its id is its event's name followed by the stem with every underscore written as
// a dot (FR-619): no id may hold an underscore (FR-230); spelled this way the id's folder is
// the event's name and the stem joined by an underscore (FR-229).
func keyStems(definition Definition) (map[string]string, error) {
	if len(definition.Stem) == 0 {
		return nil, nil
	}
	if len(definition.Stem) > 1 {
		return nil, fmt.Errorf("%w: %s names more than one key stem", ErrInvalidCue, definition.ID)
	}
	if len(definition.Match) > 0 {
		return nil, fmt.Errorf(
			"%w: %s names a key stem beside match fields; a comms moment is narrowed by its stem alone",
			ErrInvalidCue, definition.ID,
		)
	}
	stems := make(map[string]string, len(definition.Stem))
	for field, stem := range definition.Stem {
		if read, ok := KeyStem(keyPrefix + stem + keyEnd); !ok || read != stem {
			return nil, fmt.Errorf("%w: %s names %q, which is no key stem", ErrInvalidCue, definition.ID, stem)
		}
		want := definition.Event + string(groupSeparator) +
			strings.ReplaceAll(stem, string(folderSeparator), string(groupSeparator))
		if definition.ID != want {
			return nil, fmt.Errorf(
				"%w: %s names the key stem %s, so its id is spelled %s", ErrInvalidCue, definition.ID, stem, want,
			)
		}
		stems[field] = stem
	}
	return stems, nil
}

// matchesStems reports whether every field a comms moment names holds a message key with its stem.
func (c Cue) matchesStems(candidate event.Event) bool {
	for field, want := range c.stems {
		value, _ := candidate.Field(field)
		key, isText := value.(string)
		stem, keyed := KeyStem(key)
		if !isText || !keyed || stem != want {
			return false
		}
	}
	return true
}

// narrower reports whether c describes a narrower situation than other. A comms moment names one
// message, so it is narrower than any cue naming no key stem (FR-617); otherwise the cue
// constraining more fields is (FR-606).
func (c Cue) narrower(other Cue) bool {
	if len(c.stems) != len(other.stems) {
		return len(c.stems) > len(other.stems)
	}
	return c.Specificity() > other.Specificity()
}
