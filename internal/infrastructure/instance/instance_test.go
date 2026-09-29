package instance

// One copy at a time (FR-760): what every platform shares.

import (
	"testing"
	"time"
)

// quietWait is how long a test waits to be sure that no summons arrives.
const quietWait = 200 * time.Millisecond

// FR-760: a start whose claim could not be made runs regardless, so it is handed a claim that is
// never summoned and gives up nothing, rather than none.
func TestAClaimThatCannotBeMadeStillStarts(t *testing.T) {
	t.Parallel()
	claim := inert()
	select {
	case <-claim.Summons():
		t.Fatal("a claim that holds nothing was summoned")
	case <-time.After(quietWait):
	}
	claim.Release()
	claim.Release()
}

// Two asks arriving before the first is read are one: the window comes back either way, so the
// second never waits on a reader.
func TestAsksWaitingTogetherAreOne(t *testing.T) {
	t.Parallel()
	claim := inert()
	claim.summon()
	claim.summon()
	<-claim.Summons()
	select {
	case <-claim.Summons():
		t.Fatal("two waiting asks were passed on as two")
	default:
	}
}
