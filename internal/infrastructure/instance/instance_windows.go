//go:build windows

package instance

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"golang.org/x/sys/windows"
)

// sessionNamespace keeps both names to the signed-in user's own session.
const sessionNamespace = `Local\`

// The suffixes naming the mutex that is the claim and the event a later start sets to summon.
const (
	claimSuffix  = ".instance"
	summonSuffix = ".summon"
)

// autoReset and notSignalled are CreateEvent's arguments for an event that resets itself once a
// waiter has seen it, created unset.
const (
	autoReset    = 0
	notSignalled = 0
)

// take claims the one copy through a named mutex: the first start creates it and holds it open for
// the run; a later start finds it already there. Nothing ever waits on the mutex: its existence is
// the claim. Windows removes it once the holder's handle is closed, however the holder ended.
func take(name, _ string, summon bool) (*Claim, error) {
	mutex, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(sessionNamespace+name+claimSuffix))
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		_ = windows.CloseHandle(mutex)
		if summon {
			return nil, summonHolder(name)
		}
		return nil, ErrRunning
	}
	if err != nil {
		return inert(), fmt.Errorf("claiming the one copy: %w", err)
	}
	summons, err := windows.CreateEvent(nil, autoReset, notSignalled, windows.StringToUTF16Ptr(sessionNamespace+name+summonSuffix))
	if err != nil {
		_ = windows.CloseHandle(mutex)
		return inert(), fmt.Errorf("making the summons: %w", err)
	}
	stop, err := windows.CreateEvent(nil, autoReset, notSignalled, nil)
	if err != nil {
		_ = windows.CloseHandle(summons)
		_ = windows.CloseHandle(mutex)
		return inert(), fmt.Errorf("making the summons: %w", err)
	}

	done := make(chan struct{})
	claim := newClaim(func() {
		_ = windows.SetEvent(stop)
		<-done
		for _, handle := range []windows.Handle{stop, summons, mutex} {
			_ = windows.CloseHandle(handle)
		}
	})
	go listen(claim, summons, stop, done)
	return claim, nil
}

// listen passes each summons to the claim until stop is set. A fault here ends the listening and is
// said in the run log, never the application: the window stays; only the summons go unanswered.
func listen(claim *Claim, summons, stop windows.Handle, done chan<- struct{}) {
	defer close(done)
	defer func() {
		if fault := recover(); fault != nil {
			fmt.Fprintf(os.Stderr, "listening for a second start stopped: %v\n%s\n", fault, debug.Stack())
		}
	}()
	for {
		which, err := windows.WaitForMultipleObjects([]windows.Handle{summons, stop}, false, windows.INFINITE)
		if err != nil || which != windows.WAIT_OBJECT_0 {
			return
		}
		claim.summon()
	}
}

// summonHolder sets the running copy's summons, then answers ErrRunning. A summons that cannot be set
// is said alongside it, since the running copy then stays as it is.
func summonHolder(name string) error {
	event, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, windows.StringToUTF16Ptr(sessionNamespace+name+summonSuffix))
	if err != nil {
		return fmt.Errorf("%w; asking it for its window: %v", ErrRunning, err)
	}
	defer windows.CloseHandle(event)
	if err := windows.SetEvent(event); err != nil {
		return fmt.Errorf("%w; asking it for its window: %v", ErrRunning, err)
	}
	return ErrRunning
}
