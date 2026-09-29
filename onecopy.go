// One copy at a time (FR-760): the claim a run takes before it builds anything.

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/oernster/bridge-talk/internal/infrastructure/appdata"
	"github.com/oernster/bridge-talk/internal/infrastructure/instance"
	"github.com/oernster/bridge-talk/internal/product"
)

// claimTheOneCopy takes this run's claim to be the one copy. running is true where another copy
// already holds it; this start then ends, having asked that copy for its window unless it was
// started hidden, as the sign-in entry starts it.
//
// A claim that cannot be made is said in the log and the run carries on over a claim that does
// nothing (FR-760): being unable to tell whether another copy runs is no reason to refuse to start.
func claimTheOneCopy(hidden bool) (claim *instance.Claim, running bool) {
	// The data folder is used on Linux alone; where it cannot be found the claim says so.
	dir, _ := appdata.Dir()
	claim, err := instance.Take(product.AppID, dir, !hidden)
	switch {
	case errors.Is(err, instance.ErrRunning):
		if err != instance.ErrRunning {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		}
		return nil, true
	case err != nil:
		fmt.Fprintf(os.Stderr, "warning: %v (starting regardless)\n", err)
	}
	return claim, false
}
