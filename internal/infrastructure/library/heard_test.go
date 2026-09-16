package library

// FR-745 to FR-747: the audition pane draws on and counts the moments switched on in Chatter alone, a
// group whose moments are all switched off being marked rather than dropped.

import (
	"reflect"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// noSpace and distance are two moments of one group, DockingDenied, each with a take of its own.
const (
	noSpace  cue.ID = "DockingDenied.Reason.NoSpace"
	distance cue.ID = "DockingDenied.Reason.Distance"
)

// deniedCatalogue is a catalogue over a voice with a take for each DockingDenied moment and one for
// HullDamage, answering the chooser's pick at index at.
func deniedCatalogue(t *testing.T, at int) *Catalogue {
	t.Helper()
	voice := voiceOf("Ivy", map[cue.ID][]string{
		noSpace:      {"yes.wav"},
		distance:     {"no.wav"},
		"HullDamage": {"ouch.wav"},
	})
	return catalogueOver(voice, journalTable(t, string(noSpace), string(distance), "HullDamage"), fixedChooser{at: at})
}

// hearingAllBut hears every moment except the ones named.
func hearingAllBut(off ...cue.ID) cue.Heard {
	return func(id cue.ID) bool {
		for _, each := range off {
			if id == each {
				return false
			}
		}
		return true
	}
}

// FR-745: whichever take the chooser picks, it is one of a moment switched on.
func TestAnAuditionDrawsOnlyOnMomentsSwitchedOn(t *testing.T) {
	for _, at := range []int{0, 1} {
		clip, ok := deniedCatalogue(t, at).Audition("DockingDenied", hearingAllBut(noSpace))
		if !ok || clip.Key() != "no.wav" {
			t.Errorf("pick %d: audition = %v, %v; want the take of %s alone", at, clip, ok, distance)
		}
	}
}

// FR-746: a group holds the takes of its moments switched on alone.
func TestAGroupCountsOnlyItsMomentsSwitchedOn(t *testing.T) {
	got := deniedCatalogue(t, 0).Groups(hearingAllBut(noSpace))

	want := []Group{
		{Key: "DockingDenied", Takes: takesOf("no.wav")},
		{Key: "HullDamage", Takes: takesOf("ouch.wav")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("groups = %+v, want %+v", got, want)
	}
}

// FR-747: a group whose moments are all switched off holds nothing, is marked switched off and cannot
// be auditioned; the other groups are as they were.
func TestAGroupWithEveryMomentSwitchedOffIsMarkedSwitchedOff(t *testing.T) {
	catalogue := deniedCatalogue(t, 0)
	heard := hearingAllBut(noSpace, distance)

	groups := catalogue.Groups(heard)

	if len(groups) != 2 || groups[0].Key != "DockingDenied" || !groups[0].SwitchedOff || len(groups[0].Takes) != 0 {
		t.Fatalf("groups = %+v, want DockingDenied marked switched off holding nothing", groups)
	}
	if groups[1].SwitchedOff || len(groups[1].Takes) != 1 {
		t.Errorf("HullDamage = %+v, want it untouched", groups[1])
	}
	if clip, ok := catalogue.Audition("DockingDenied", heard); ok {
		t.Errorf("a group with every moment switched off played %v", clip)
	}
}
