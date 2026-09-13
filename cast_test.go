package main

import (
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/library"
)

// voiceNamed finds one row of the cast pane, failing rather than indexing blindly.
func voiceNamed(t *testing.T, rows []VoiceDTO, name string) VoiceDTO {
	t.Helper()
	for _, row := range rows {
		if row.Name == name {
			return row
		}
	}
	t.Fatalf("no row for %q in %v", name, rows)
	return VoiceDTO{}
}

// The cast pane lists what the scan found, which is every directory that resolved at
// least one take. A directory holding audio under no cue's name is not among them: it
// is reported by the scan rather than offered as a voice that plays nothing.
func TestTheCastPaneListsEveryVoiceTheScanFound(t *testing.T) {
	app, _, _ := fixtureApp(t)

	rows := app.Voices()
	if len(rows) != 2 {
		t.Fatalf("got %d rows %v, want Alpha and Beta", len(rows), rows)
	}

	alpha := voiceNamed(t, rows, "Alpha")
	if alpha.InUse != 3 {
		t.Errorf("Alpha holds %d takes, want the three it recorded", alpha.InUse)
	}
}

func TestCastingAVoiceRebuildsTheSessionAndTellsThePage(t *testing.T) {
	app, _, log := fixtureApp(t)

	if err := app.SelectVoice("Alpha"); err != nil {
		t.Fatalf("casting Alpha: %v", err)
	}
	if app.session.active.Name != "Alpha" {
		t.Fatalf("active voice is %q, want Alpha", app.session.active.Name)
	}
	if !app.session.hasVoice() {
		t.Fatal("a voice was cast but the session reports none")
	}

	payload := log.lastEmitted(t, stateEvent)
	state, ok := payload.(StateDTO)
	if !ok {
		t.Fatalf("state payload was %T, want StateDTO", payload)
	}
	if state.Voice != "Alpha" {
		t.Fatalf("the announced voice is %q, want Alpha", state.Voice)
	}
	if state.Bound != 3 || state.Total != 4 {
		t.Fatalf("announced coverage is %d of %d, want 3 of 4", state.Bound, state.Total)
	}
}

func TestCastingAVoiceThatIsNotInstalledIsRefused(t *testing.T) {
	app, _, _ := fixtureApp(t)

	err := app.SelectVoice("a voice nobody owns")
	if err == nil {
		t.Fatal("a voice that is not installed was cast")
	}
	if !strings.Contains(err.Error(), "Alpha") {
		t.Errorf("refusal = %q, want it to list what is available", err)
	}
}

// The breakdown is asked from a row in the chooser, so it answers for any voice rather
// than only the cast one: the point of asking is to decide whether to cast it.
func TestTheBreakdownAnswersForAVoiceThatIsNotCast(t *testing.T) {
	app, _, _ := fixtureApp(t)

	breakdown := app.CueBreakdown("Alpha")
	if breakdown.Voice != "Alpha" {
		t.Fatalf("voice = %q, want Alpha", breakdown.Voice)
	}
	if len(breakdown.Served) != 3 {
		t.Fatalf("served = %v, want the three cues Alpha recorded", breakdown.Served)
	}
	if len(breakdown.Unserved) != 1 {
		t.Fatalf("unserved = %v, want the one cue nothing records", breakdown.Unserved)
	}
	if breakdown.Unserved[0].ID != "zz.silent" {
		t.Fatalf("unserved cue is %q, want zz.silent", breakdown.Unserved[0].ID)
	}
	// The title is what reaches the screen; an id is a name for the code.
	if breakdown.Unserved[0].Title != "Zz: silent" {
		t.Fatalf("title = %q, want the id read as words", breakdown.Unserved[0].Title)
	}
	shields := breakdown.Served[2]
	if shields.Group != "ShieldState" || shields.Heading != "Shield state" {
		t.Fatalf("group %q headed %q, want the id's first segment and it in words",
			shields.Group, shields.Heading)
	}
}

// The dialog behind the row is a thing to read, so a name matching nothing yields an
// empty breakdown rather than an error nothing can display.
func TestTheBreakdownOfAVoiceThatIsNotThereIsEmptyRatherThanAnError(t *testing.T) {
	app, _, _ := fixtureApp(t)

	breakdown := app.CueBreakdown("a voice nobody owns")
	if breakdown.Voice != "a voice nobody owns" {
		t.Fatalf("voice = %q, want the name that was asked about", breakdown.Voice)
	}
	if len(breakdown.Served) != 0 || len(breakdown.Unserved) != 0 {
		t.Fatalf("got %v and %v, want both empty", breakdown.Served, breakdown.Unserved)
	}
}

// The tray offers exactly the voices the scan found. A directory that resolved no take
// never became a voice, so it cannot reach the menu.
func TestOnlyDirectoriesHoldingTakesAreOfferedToTheTray(t *testing.T) {
	current, _ := fixtureSession(t, newFakePlayer())

	names := playable(current.available)
	if len(names) != 2 {
		t.Fatalf("got %v, want the two voices holding takes", names)
	}
	for _, name := range names {
		if name == "Bystander" {
			t.Errorf("a directory holding no take was offered as a voice")
		}
	}
}

// The rule the composition root applies to every setting with more than one source.
// A flag is this run's instruction, so it beats the stored choice; the stored choice
// beats nothing at all, which is what sends the caller off to detect.
func TestThisRunsInstructionBeatsWhatWasStored(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		flag, held string
		want       string
	}{
		{"the flag wins where both are given", "Hugo", "Grace", "Hugo"},
		{"the stored choice is taken where no flag is given", "", "Grace", "Grace"},
		{"the flag is taken where nothing is stored", "Hugo", "", "Hugo"},
		{"nothing chosen answers with nothing", "", "", ""},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			t.Parallel()
			if got := preferred(each.flag, each.held); got != each.want {
				t.Errorf("preferred(%q, %q) = %q, want %q", each.flag, each.held, got, each.want)
			}
		})
	}
}

// Naming a particular voice as the default would mean the application started only for
// somebody who owned that one, so no preference takes the first there is.
func TestPickingWithNoPreferenceTakesTheFirstVoiceFound(t *testing.T) {
	current, _ := fixtureSession(t, newFakePlayer())

	chosen, err := pick(current.available, "")
	if err != nil {
		t.Fatalf("picking with no preference: %v", err)
	}
	if chosen.Name != "Alpha" {
		t.Fatalf("got %q, want the alphabetically first voice", chosen.Name)
	}
}

func TestPickingFromNothingIsRefused(t *testing.T) {
	if _, err := pick(nil, ""); err == nil {
		t.Fatal("a voice was picked from an empty list")
	}
}

// A name typed by hand is rarely typed exactly, so an unambiguous prefix is accepted.
// An ambiguous one is not: guessing which of two the reader meant would cast the
// wrong voice silently.
func TestPickingAcceptsAnUnambiguousPrefixAndRefusesAnAmbiguousOne(t *testing.T) {
	current, _ := fixtureSession(t, newFakePlayer())
	found := current.available

	if chosen, err := pick(found, "alp"); err != nil || chosen.Name != "Alpha" {
		t.Fatalf("picking by prefix gave %q, %v; want Alpha", chosen.Name, err)
	}
	if chosen, err := pick(found, "ALPHA"); err != nil || chosen.Name != "Alpha" {
		t.Fatalf("picking by exact name in another case gave %q, %v", chosen.Name, err)
	}

	// Two voices whose names share a prefix cannot be told apart by it.
	ambiguous := append([]library.Voice{}, found...)
	ambiguous = append(ambiguous, library.Voice{Name: "Alphabet"})
	if _, err := pick(ambiguous, "alph"); err == nil {
		t.Fatal("an ambiguous prefix picked a voice")
	}
}

// Casting a voice answers in that voice.
//
// Choosing from a list of names is a decision taken without hearing anything, so the
// confirmation that settles it is the voice itself agreeing to the part.
func TestCastingAVoiceIsConfirmedInThatVoice(t *testing.T) {
	app, player, _ := fixtureApp(t)

	if err := app.SelectVoice("Alpha"); err != nil {
		t.Fatalf("casting Alpha: %v", err)
	}

	if len(player.played) != 1 {
		t.Fatalf("played %v, want one confirming clip", player.played)
	}
	if len(player.played[0]) != 1 {
		t.Fatalf("played %d clips to confirm a cast, want 1", len(player.played[0]))
	}
}

// A muted application stays quiet while it is being reorganised. An audition is a
// request to hear something; casting is a request to choose something.
func TestCastingAVoiceWhileMutedSaysNothing(t *testing.T) {
	app, player, _ := fixtureApp(t)
	app.session.muted = true

	if err := app.SelectVoice("Alpha"); err != nil {
		t.Fatalf("casting Alpha: %v", err)
	}

	if len(player.played) != 0 {
		t.Fatalf("a muted application played %v while being cast", player.played)
	}
}
