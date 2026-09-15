package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/config"
)

// overrideFile writes a replacement table and returns its path.
func overrideFile(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("writing %q: %v", path, err)
	}
	return path
}

// absent returns a path in a temporary directory that was never created.
func absent(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), name)
}

func TestTheShippedCueTableLoadsWithNoOverride(t *testing.T) {
	t.Parallel()
	table, err := config.LoadCueTable("")
	if err != nil {
		t.Fatalf("loading the shipped table: %v", err)
	}
	if table.Len() == 0 {
		t.Fatal("the shipped table is empty")
	}
}

// A cue can be retuned without a rebuild. That is the whole reason the loader takes
// an override at all, so it needs proof the file on disk actually wins.
func TestAnOverrideFileReplacesTheShippedCueTable(t *testing.T) {
	t.Parallel()
	path := overrideFile(t, "cues.toml", `
[[category]]
name = "Tests"

[[cue]]
id = "test.only"
source = "journal"
event = "FSDJump"
priority = "notice"
purpose = "When a test says so."
category = "Tests"
`)
	table, err := config.LoadCueTable(path)
	if err != nil {
		t.Fatalf("loading the override: %v", err)
	}
	if table.Len() != 1 {
		t.Fatalf("got %d cues, want only the one in the override", table.Len())
	}
	if string(table.All()[0].ID()) != "test.only" {
		t.Fatalf("id: got %q", table.All()[0].ID())
	}
}

// An override the user named but that is not there is a mistake worth reporting.
// Falling back to the shipped table silently would leave them retuning a file the
// application never reads.
func TestAnOverrideThatIsNotThereIsReported(t *testing.T) {
	t.Parallel()
	if _, err := config.LoadCueTable(absent(t, "cues.toml")); err == nil {
		t.Fatal("a missing cue override was accepted")
	}
}

func TestAnOverrideThatIsNotValidTomlIsReported(t *testing.T) {
	t.Parallel()
	if _, err := config.LoadCueTable(overrideFile(t, "cues.toml", "[[cue")); err == nil {
		t.Fatal("an unparseable cue override was accepted")
	}
}

// FR-219 reaches a user's own table. An id ending in a segment of digits stops the load
// with the id and the reason in the message, rather than loading a table whose flat
// form takes would be misread.
func TestAnOverrideHoldingAnIdEndingInDigitsFailsToLoad(t *testing.T) {
	t.Parallel()
	path := overrideFile(t, "cues.toml", `
[[cue]]
id = "DockingGranted.2"
source = "journal"
event = "DockingGranted"
purpose = "When a test says so."
`)

	_, err := config.LoadCueTable(path)

	if err == nil || !strings.Contains(err.Error(), "DockingGranted.2") ||
		!strings.Contains(err.Error(), "digits") {
		t.Fatalf("err = %v, want the load refused naming the id and the reason", err)
	}
}

// FR-222 reaches a user's own table. An id ending in a space would name a folder Windows
// renames as it creates it, so the load stops with the id and the reason.
func TestAnOverrideHoldingAnIdEndingInASpaceFailsToLoad(t *testing.T) {
	t.Parallel()
	path := overrideFile(t, "cues.toml", `
[[cue]]
id = "DockingGranted "
source = "journal"
event = "DockingGranted"
purpose = "When a test says so."
`)

	_, err := config.LoadCueTable(path)

	if !errorMentions(err, `"DockingGranted "`) || !errorMentions(err, "dot or a space") {
		t.Fatalf("err = %v, want the load refused naming the id and the reason", err)
	}
}

// Two cues sharing an id is not a table with a duplicate; it is a table where one of
// the two is unreachable and nothing says which. That has to be refused rather than
// resolved by whichever happened to be parsed last.
func TestTwoCuesSharingAnIdAreRefused(t *testing.T) {
	t.Parallel()
	path := overrideFile(t, "cues.toml", `
[[cue]]
id = "same.id"
source = "journal"
event = "FSDJump"
priority = "notice"
purpose = "When a test says so."

[[cue]]
id = "same.id"
source = "journal"
event = "Docked"
priority = "notice"
purpose = "When a test says so."
`)
	_, err := config.LoadCueTable(path)
	if err == nil {
		t.Fatal("a table with a duplicate id was accepted")
	}
	if !errorMentions(err, "duplicate") {
		t.Fatalf("the error does not say what is wrong: %v", err)
	}
}

func TestACueTheDomainRefusesIsReported(t *testing.T) {
	t.Parallel()
	path := overrideFile(t, "cues.toml", `
[[cue]]
id = "bad.priority"
source = "journal"
event = "FSDJump"
priority = "screaming"
purpose = "When a test says so."
`)
	if _, err := config.LoadCueTable(path); !errorMentions(err, "screaming") {
		t.Fatalf("err = %v, want the unknown priority refused by name", err)
	}
}

// A title is generated from the id, so a table that writes one is saying something the
// application would never show. The key is refused by name rather than dropped, which
// is also what catches a misspelled key such as a cooldown that would silently not apply.
func TestAKeyTheTableDoesNotHoldIsRefusedByName(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"title", "cooldwn"} {
		path := overrideFile(t, "cues.toml", `
[[cue]]
id = "FSDJump"
source = "journal"
event = "FSDJump"
purpose = "When a test says so."
`+key+` = "5"
`)
		if _, err := config.LoadCueTable(path); !errorMentions(err, key) {
			t.Errorf("a table writing %q gave %v, want it refused naming the key", key, err)
		}
	}
}

// FR-231: every cue says when it is heard. A purpose left out, left empty or made of spaces
// alone stops the load naming the cue, since a take recorded for it would be recorded
// without anyone knowing when it plays.
func TestACueWithNoPurposeIsRefusedByName(t *testing.T) {
	t.Parallel()
	for name, line := range map[string]string{
		"missing": "",
		"empty":   `purpose = ""`,
		"spaces":  `purpose = "   "`,
	} {
		path := overrideFile(t, "cues.toml", `
[[cue]]
id = "Docked"
source = "journal"
event = "Docked"
`+line+`
`)
		if _, err := config.LoadCueTable(path); !errorMentions(err, "Docked") || !errorMentions(err, "purpose") {
			t.Errorf("a %s purpose gave %v, want the load refused naming the cue", name, err)
		}
	}
}

// FR-231: the shipped table is held to the same rule, so every cue it ships says when it
// is heard.
func TestEveryShippedCueHasAPurpose(t *testing.T) {
	t.Parallel()
	table, err := config.LoadCueTable("")
	if err != nil {
		t.Fatalf("loading the shipped table: %v", err)
	}
	for _, item := range table.All() {
		if strings.TrimSpace(item.Purpose()) == "" {
			t.Errorf("%s ships with no purpose", item.ID())
		}
	}
}

// errorMentions reports whether an error's text contains a fragment, so a test can
// assert an error says which thing is wrong rather than merely that it failed.
func errorMentions(err error, fragment string) bool {
	return err != nil && strings.Contains(err.Error(), fragment)
}
