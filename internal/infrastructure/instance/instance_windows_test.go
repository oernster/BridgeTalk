//go:build windows

package instance

// One copy at a time (FR-760) on Windows, over real named objects in this session.

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

// summonWait is how long a test waits for a summons that should arrive.
const summonWait = 5 * time.Second

// claimName answers a name no other test or run uses, so tests cannot hold each other's claims.
func claimName(t *testing.T) string {
	return fmt.Sprintf("instance-test-%d-%s", os.Getpid(), t.Name())
}

// taken takes the first claim under the test's own name, releasing it when the test ends.
func taken(t *testing.T) (*Claim, string) {
	t.Helper()
	name := claimName(t)
	claim, err := Take(name, "", true)
	if err != nil {
		t.Fatalf("the first claim: %v", err)
	}
	t.Cleanup(claim.Release)
	return claim, name
}

// FR-760: while one start holds the claim, a later start is refused it.
func TestOnlyOneClaimIsGrantedAtATime(t *testing.T) {
	t.Parallel()
	_, name := taken(t)
	second, err := Take(name, "", false)
	if !errors.Is(err, ErrRunning) || second != nil {
		t.Fatalf("a second claim = %v, %v; want none and ErrRunning", second, err)
	}
}

// FR-760: a later start asks the holder for its window, which the holder's claim passes on.
func TestASecondStartSummonsTheFirst(t *testing.T) {
	t.Parallel()
	first, name := taken(t)
	if _, err := Take(name, "", true); err != ErrRunning {
		t.Fatalf("a second start = %v, want exactly ErrRunning: the summons should have gone through", err)
	}
	select {
	case <-first.Summons():
	case <-time.After(summonWait):
		t.Fatal("the first start was never summoned")
	}
}

// FR-760: a later start made hidden, as the sign-in entry makes one, asks for nothing.
func TestAHiddenSecondStartSummonsNobody(t *testing.T) {
	t.Parallel()
	first, name := taken(t)
	if _, err := Take(name, "", false); !errors.Is(err, ErrRunning) {
		t.Fatalf("a hidden second start = %v, want ErrRunning", err)
	}
	select {
	case <-first.Summons():
		t.Fatal("a hidden second start summoned the window")
	case <-time.After(quietWait):
	}
}

// A released claim is gone with its names, so the next start takes it; releasing twice is harmless.
func TestAReleasedClaimCanBeTakenAgain(t *testing.T) {
	t.Parallel()
	name := claimName(t)
	first, err := Take(name, "", true)
	if err != nil {
		t.Fatalf("the first claim: %v", err)
	}
	first.Release()
	first.Release()
	again, err := Take(name, "", true)
	if err != nil {
		t.Fatalf("a claim after release = %v, want it granted", err)
	}
	again.Release()
}

// FR-760: a claim Windows will not make is said with the reason and answered with a claim that does
// nothing, so the start carries on. A name with a separator in it is one Windows refuses.
func TestAClaimWindowsRefusesIsSaidAndStillStarts(t *testing.T) {
	t.Parallel()
	claim, err := Take(claimName(t)+`\refused`, "", true)
	if err == nil || errors.Is(err, ErrRunning) || claim == nil {
		t.Fatalf("a refused claim = %v, %v; want an inert claim and the reason", claim, err)
	}
	claim.Release()
}
