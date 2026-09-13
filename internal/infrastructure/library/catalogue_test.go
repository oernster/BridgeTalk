package library

// The catalogue over one voice: what it plays for a cue, what it covers and what the
// audition pane can draw on. The voices here are built directly rather than scanned,
// because the scan has tests of its own and a literal says exactly what a voice holds.

import (
	"reflect"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// fixedChooser always answers with the same index, bounded by the range it is given.
type fixedChooser struct{ at int }

func (f fixedChooser) Intn(n int) int { return f.at % n }

// voiceOf builds a voice holding exactly the takes given, counted as a scan counts them.
func voiceOf(name string, takes map[cue.ID][]string) Voice {
	voice := Voice{Name: name, byCue: takes}
	for _, clips := range takes {
		voice.Takes += len(clips)
	}
	return voice
}

// ids lists the ids of cues in order.
func ids(cues []cue.Cue) []cue.ID {
	out := make([]cue.ID, 0, len(cues))
	for _, item := range cues {
		out = append(out, item.ID())
	}
	return out
}

// The acknowledgement is the one cue from the application's own source. It is found by
// that source, so an id nobody wrote into this package still answers.
func TestTheAcknowledgementIsFoundByItsSource(t *testing.T) {
	table := cue.NewTable([]cue.Cue{
		newCue(t, "DockingGranted", "journal"),
		newCue(t, "Cast.Confirmed", "application"),
	})
	voice := voiceOf("Ivy", map[cue.ID][]string{
		"DockingGranted": {"granted.wav"},
		"Cast.Confirmed": {"first.wav", "second.wav"},
	})

	clip, ok := NewCatalogue(voice, table, fixedChooser{at: 1}).Acknowledgement()

	if !ok || clip != "second.wav" {
		t.Fatalf("acknowledgement = %q, %v; want the take the chooser picked", clip, ok)
	}
}

// A voice with no acknowledgement recorded is silent when cast rather than broken, as is
// a table that has no acknowledgement cue at all.
func TestAnAcknowledgementNobodyRecordedIsSilence(t *testing.T) {
	withCue := cue.NewTable([]cue.Cue{newCue(t, "Cast.Confirmed", "application")})
	unrecorded := voiceOf("Ivy", map[cue.ID][]string{})
	if _, ok := NewCatalogue(unrecorded, withCue, fixedChooser{}).Acknowledgement(); ok {
		t.Error("a voice that recorded no acknowledgement produced one")
	}

	withoutCue := journalTable(t, "Cast.Confirmed")
	recorded := voiceOf("Ivy", map[cue.ID][]string{"Cast.Confirmed": {"hello.wav"}})
	if _, ok := NewCatalogue(recorded, withoutCue, fixedChooser{}).Acknowledgement(); ok {
		t.Error("a table with no application cue produced an acknowledgement")
	}
}

// Clips answers only for a cue the voice recorded, with every take it holds for it.
func TestClipsAnswerOnlyForARecordedCue(t *testing.T) {
	voice := voiceOf("Ivy", map[cue.ID][]string{"DockingGranted": {"a.wav", "b.wav"}})
	catalogue := NewCatalogue(voice, journalTable(t, "DockingGranted", "ShieldState.ShieldsUp.false"), fixedChooser{})

	performance, ok := catalogue.Clips("DockingGranted")
	if !ok || !reflect.DeepEqual(performance.Clips, []string{"a.wav", "b.wav"}) {
		t.Fatalf("clips = %v, %v; want both takes", performance.Clips, ok)
	}
	if _, ok := catalogue.Clips("ShieldState.ShieldsUp.false"); ok {
		t.Error("a cue the voice never recorded answered")
	}
	if catalogue.ActiveVoice() != "Ivy" {
		t.Errorf("active voice = %q, want Ivy", catalogue.ActiveVoice())
	}
}

// FR-215, first figure: the cues served out of the whole table.
func TestCoverageCountsTheCuesServedOutOfTheTable(t *testing.T) {
	voice := voiceOf("Ivy", map[cue.ID][]string{
		"DockingGranted":              {"a.wav"},
		"ShieldState.ShieldsUp.false": {"b.wav"},
	})
	table := journalTable(t, "DockingGranted", "ShieldState.ShieldsUp.false", "StartJump")

	served, total := NewCatalogue(voice, table, fixedChooser{}).Coverage()

	if served != 2 || total != 3 {
		t.Errorf("coverage = %d of %d, want 2 of 3", served, total)
	}
}

// FR-215, second figure: distinct files used against the recordings present. One file
// answering two cues is one file used; a recording present that answers nothing is not used.
func TestFilesCountDistinctFilesUsedAgainstRecordingsPresent(t *testing.T) {
	voice := voiceOf("Ivy", map[cue.ID][]string{
		"DockingGranted":              {"shared.wav"},
		"ShieldState.ShieldsUp.false": {"shared.wav", "own.wav"},
	})
	voice.Present = 4

	used, present := NewCatalogue(voice, journalTable(t), fixedChooser{}).Files()

	if used != 2 || present != 4 {
		t.Errorf("files = %d used of %d present, want 2 of 4", used, present)
	}
}

// Served and Unbound split the table between them: every cue in exactly one, both sorted.
func TestServedAndUnboundPartitionTheTable(t *testing.T) {
	voice := voiceOf("Ivy", map[cue.ID][]string{
		"ShieldState.ShieldsUp.false": {"a.wav"},
		"DockingGranted":              {"b.wav"},
	})
	table := journalTable(t, "ShieldState.ShieldsUp.false", "StartJump", "DockingGranted", "HullDamage")
	catalogue := NewCatalogue(voice, table, fixedChooser{})

	if got, want := ids(catalogue.Served()), []cue.ID{"DockingGranted", "ShieldState.ShieldsUp.false"}; !reflect.DeepEqual(got, want) {
		t.Errorf("served = %v, want %v", got, want)
	}
	if got, want := ids(catalogue.Unbound()), []cue.ID{"HullDamage", "StartJump"}; !reflect.DeepEqual(got, want) {
		t.Errorf("unbound = %v, want %v", got, want)
	}
}

// A group is the id's first segment holding the distinct takes of its cues. A group with
// nothing recorded is left out rather than offered as a button that plays silence.
func TestGroupsGatherDistinctTakesUnderTheFirstSegment(t *testing.T) {
	voice := voiceOf("Ivy", map[cue.ID][]string{
		"DockingDenied.Reason.NoSpace":  {"yes.wav"},
		"DockingDenied.Reason.Distance": {"yes.wav", "no.wav"},
		"HullDamage":                    {"ouch.wav"},
	})
	table := journalTable(t,
		"DockingDenied.Reason.NoSpace", "DockingDenied.Reason.Distance", "HullDamage", "zz.silent")

	got := NewCatalogue(voice, table, fixedChooser{}).Groups()

	want := []Group{
		{Key: "DockingDenied", Clips: []string{"no.wav", "yes.wav"}},
		{Key: "HullDamage", Clips: []string{"ouch.wav"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("groups = %+v, want %+v", got, want)
	}
}

// FR-216: an audition draws one take from the named group; a group the voice lacks
// answers nothing.
func TestAnAuditionDrawsFromTheNamedGroup(t *testing.T) {
	voice := voiceOf("Ivy", map[cue.ID][]string{
		"DockingDenied.Reason.NoSpace":  {"yes.wav"},
		"DockingDenied.Reason.Distance": {"no.wav"},
		"HullDamage":                    {"ouch.wav"},
	})
	table := journalTable(t, "DockingDenied.Reason.NoSpace", "DockingDenied.Reason.Distance", "HullDamage")
	catalogue := NewCatalogue(voice, table, fixedChooser{at: 1})

	if clip, ok := catalogue.Audition("DockingDenied"); !ok || clip != "yes.wav" {
		t.Errorf("audition = %q, %v; want the take the chooser picked", clip, ok)
	}
	if _, ok := catalogue.Audition("UnderAttack"); ok {
		t.Error("a group the voice has nothing for was auditioned")
	}
}
