package library

// The catalogue over one audio source: what it plays for a cue, what it covers and what the
// audition pane can draw on. The voices here are built directly rather than scanned,
// because the scan has tests of its own and a literal says exactly what a voice holds.

import (
	"reflect"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/take"
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

// catalogueOver builds the catalogue over a recorded voice as the composition root does:
// under the name the voice is shown by.
func catalogueOver(voice Voice, table cue.Table, chooser fixedChooser) *Catalogue {
	return NewCatalogue(voice, voice.Display(), table, chooser)
}

// takesOnly is an audio source that is no scanned voice: takes by cue and nothing more.
// Its takes have one part each, which is what a source of single files answers.
type takesOnly map[cue.ID][]string

func (t takesOnly) Lookup(id cue.ID) ([]take.Take, bool) {
	clips, ok := t[id]
	if !ok || len(clips) == 0 {
		return nil, false
	}
	return takesOf(clips...), true
}

// takesOf builds one take per path, the shape a voice recording one file per take answers.
func takesOf(paths ...string) []take.Take {
	out := make([]take.Take, 0, len(paths))
	for _, path := range paths {
		out = append(out, take.Of(path))
	}
	return out
}

// ids lists the ids of cues in order.
func ids(cues []cue.Cue) []cue.ID {
	out := make([]cue.ID, 0, len(cues))
	for _, item := range cues {
		out = append(out, item.ID())
	}
	return out
}

// FR-501: the catalogue answers from any audio source under the name it is given, knowing
// nothing of where the takes came from.
func TestTheCatalogueAnswersFromAnyAudioSource(t *testing.T) {
	source := takesOnly{"DockingGranted": {"made.flac"}, "StartJump": {}}
	catalogue := NewCatalogue(source, "Emma (British, female)",
		journalTable(t, "DockingGranted", "StartJump"), fixedChooser{})

	performance, ok := catalogue.Clips("DockingGranted")
	if !ok || !reflect.DeepEqual(performance.Takes, takesOf("made.flac")) {
		t.Fatalf("clips = %v, %v; want the take the source holds", performance.Takes, ok)
	}
	if _, ok := catalogue.Clips("StartJump"); ok {
		t.Error("a cue the source holds no take for answered")
	}
	if served, total := catalogue.Coverage(); served != 1 || total != 2 {
		t.Errorf("coverage = %d of %d, want 1 of 2", served, total)
	}
	if catalogue.ActiveVoice() != "Emma (British, female)" {
		t.Errorf("active voice = %q, want the name the catalogue was given", catalogue.ActiveVoice())
	}
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

	clip, ok := catalogueOver(voice, table, fixedChooser{at: 1}).Acknowledgement()

	if !ok || clip.Key() != "second.wav" {
		t.Fatalf("acknowledgement = %q, %v; want the take the chooser picked", clip, ok)
	}
}

// A voice with no acknowledgement recorded is silent when cast rather than broken, as is
// a table that has no acknowledgement cue at all.
func TestAnAcknowledgementNobodyRecordedIsSilence(t *testing.T) {
	withCue := cue.NewTable([]cue.Cue{newCue(t, "Cast.Confirmed", "application")})
	unrecorded := voiceOf("Ivy", map[cue.ID][]string{})
	if _, ok := catalogueOver(unrecorded, withCue, fixedChooser{}).Acknowledgement(); ok {
		t.Error("a voice that recorded no acknowledgement produced one")
	}

	withoutCue := journalTable(t, "Cast.Confirmed")
	recorded := voiceOf("Ivy", map[cue.ID][]string{"Cast.Confirmed": {"hello.wav"}})
	if _, ok := catalogueOver(recorded, withoutCue, fixedChooser{}).Acknowledgement(); ok {
		t.Error("a table with no application cue produced an acknowledgement")
	}
}

// Clips answers only for a cue the voice recorded, with every take it holds for it.
func TestClipsAnswerOnlyForARecordedCue(t *testing.T) {
	voice := voiceOf("Ivy", map[cue.ID][]string{"DockingGranted": {"a.wav", "b.wav"}})
	catalogue := catalogueOver(voice, journalTable(t, "DockingGranted", "ShieldState.ShieldsUp.false"), fixedChooser{})

	performance, ok := catalogue.Clips("DockingGranted")
	if !ok || !reflect.DeepEqual(performance.Takes, takesOf("a.wav", "b.wav")) {
		t.Fatalf("clips = %v, %v; want both takes", performance.Takes, ok)
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

	served, total := catalogueOver(voice, table, fixedChooser{}).Coverage()

	if served != 2 || total != 3 {
		t.Errorf("coverage = %d of %d, want 2 of 3", served, total)
	}
}

// Served and Unbound split the table between them: every cue in exactly one, both sorted.
func TestServedAndUnboundPartitionTheTable(t *testing.T) {
	voice := voiceOf("Ivy", map[cue.ID][]string{
		"ShieldState.ShieldsUp.false": {"a.wav"},
		"DockingGranted":              {"b.wav"},
	})
	table := journalTable(t, "ShieldState.ShieldsUp.false", "StartJump", "DockingGranted", "HullDamage")
	catalogue := catalogueOver(voice, table, fixedChooser{})

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

	got := catalogueOver(voice, table, fixedChooser{}).Groups(cue.HeardAll)

	want := []Group{
		{Key: "DockingDenied", Takes: takesOf("no.wav", "yes.wav")},
		{Key: "HullDamage", Takes: takesOf("ouch.wav")},
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
	catalogue := catalogueOver(voice, table, fixedChooser{at: 1})

	if clip, ok := catalogue.Audition("DockingDenied", cue.HeardAll); !ok || clip.Key() != "yes.wav" {
		t.Errorf("audition = %q, %v; want the take the chooser picked", clip, ok)
	}
	if _, ok := catalogue.Audition("UnderAttack", cue.HeardAll); ok {
		t.Error("a group the voice has nothing for was auditioned")
	}
}
