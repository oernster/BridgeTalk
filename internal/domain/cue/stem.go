package cue

import (
	"fmt"
	"strings"

	"github.com/oernster/bridge-talk/internal/domain/event"
)

// Comms moments (FR-617 to FR-619, FR-637, FR-638).
//
// A message the game sends a commander carries a key beside the words it generates for it, such
// as $Pirate_OnDeclarePiracyAttack07; for a pirate declaring an attack. One moment arrives under
// many keys that differ only in their variant number, some with values after it, so a comms moment
// names the key stem those keys share rather than any one key. Station traffic arrives under many
// stems sharing a few beginnings, so its moment names those beginnings instead.

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

// How narrowly a cue names a message key, from the broadest up: no key at all answers any message,
// beginnings answer a family of stems and a whole key stem answers one (FR-617, FR-638).
const (
	namesNoKey = iota
	namesBeginnings
	namesOneStem
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

// keyBeginnings validates the beginnings of key stems a definition names (FR-638).
//
// Such a moment names one field and the beginnings a key stem in it may have, with nothing beside
// them: a whole key stem or a match field would narrow it a second way. No one stem names the
// family, so its id is not spelled from the beginnings (FR-637); every other rule an id keeps still
// holds. Each beginning is one a key stem could start with, so a stray variant digit is refused.
func keyBeginnings(definition Definition) (map[string][]string, error) {
	if len(definition.Begins) == 0 {
		return nil, nil
	}
	if len(definition.Begins) > 1 {
		return nil, fmt.Errorf("%w: %s names key beginnings for more than one field", ErrInvalidCue, definition.ID)
	}
	if len(definition.Stem) > 0 || len(definition.Match) > 0 {
		return nil, fmt.Errorf(
			"%w: %s names key beginnings beside a key stem or match fields; they narrow it alone",
			ErrInvalidCue, definition.ID,
		)
	}
	beginnings := make(map[string][]string, len(definition.Begins))
	for field, listed := range definition.Begins {
		if len(listed) == 0 {
			return nil, fmt.Errorf("%w: %s names no key beginning", ErrInvalidCue, definition.ID)
		}
		for _, beginning := range listed {
			if read, ok := KeyStem(keyPrefix + beginning + keyEnd); !ok || read != beginning {
				return nil, fmt.Errorf(
					"%w: %s names %q, which no key stem can begin with", ErrInvalidCue, definition.ID, beginning,
				)
			}
		}
		beginnings[field] = append([]string(nil), listed...)
	}
	return beginnings, nil
}

// matchesStems reports whether every field a comms moment names holds a message key with its stem.
func (c Cue) matchesStems(candidate event.Event) bool {
	for field, want := range c.stems {
		stem, keyed := stemIn(candidate, field)
		if !keyed || stem != want {
			return false
		}
	}
	return true
}

// matchesBeginnings reports whether every field a family moment names holds a message key whose stem
// begins with one of its beginnings.
func (c Cue) matchesBeginnings(candidate event.Event) bool {
	for field, listed := range c.beginnings {
		stem, keyed := stemIn(candidate, field)
		if !keyed || !beginsWithAny(stem, listed) {
			return false
		}
	}
	return true
}

// stemIn reads the key stem from one field of an event; false where the field holds no key.
func stemIn(candidate event.Event, field string) (string, bool) {
	value, _ := candidate.Field(field)
	key, isText := value.(string)
	if !isText {
		return "", false
	}
	return KeyStem(key)
}

// beginsWithAny reports whether a stem starts with one of the beginnings.
func beginsWithAny(stem string, beginnings []string) bool {
	for _, beginning := range beginnings {
		if strings.HasPrefix(stem, beginning) {
			return true
		}
	}
	return false
}

// keyNarrowness says how narrowly the cue names a message key.
func (c Cue) keyNarrowness() int {
	switch {
	case len(c.stems) > 0:
		return namesOneStem
	case len(c.beginnings) > 0:
		return namesBeginnings
	}
	return namesNoKey
}

// narrower reports whether c describes a narrower situation than other. A comms moment naming a key
// stem names one message, so it is narrower than a family moment, which is narrower than any cue
// naming no key (FR-617, FR-638); otherwise the cue constraining more fields is (FR-606).
func (c Cue) narrower(other Cue) bool {
	if c.keyNarrowness() != other.keyNarrowness() {
		return c.keyNarrowness() > other.keyNarrowness()
	}
	return c.Specificity() > other.Specificity()
}
