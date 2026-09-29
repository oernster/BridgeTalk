// The update check's surface (FR-756 to FR-759): the check, the download and the skip.
//
// It sits beside app.go for the reason donate.go does: app.go is near the size cap; this is a
// slice that comes out whole. The page never holds an address (FR-757): the facade keeps the
// release it last offered and Download and Skip act on that one.

package main

import (
	"errors"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
)

// errNothingOffered refuses a Download or a Skip asked for before any check has offered a release.
var errNothingOffered = errors.New("no update has been offered to act on")

// CheckForUpdates runs one update check. An automatic check leaves out the release the reader
// skipped (FR-758); one asked for from Help offers it anyway (FR-759). Wails runs a bound call off
// the window's thread, so the wait of up to five seconds never holds the window. A newer release is
// kept as the offer Download and Skip act on.
func (a *App) CheckForUpdates(manual bool) UpdateDTO {
	skipped := ""
	if !manual && a.settings != nil {
		skipped = a.settings.Load().SkippedUpdate
	}
	status := a.updates.Check(skipped)
	if status.Outcome == services.UpdateAvailable {
		a.mu.Lock()
		a.offered = status
		a.mu.Unlock()
	}
	return UpdateDTO{Outcome: string(status.Outcome), Running: status.Running, Latest: status.Latest}
}

// DownloadUpdate hands the offered release's file for this platform to the browser, its page where
// it carries none (FR-757), refused as any hand-over is unless it begins https:// (FR-718).
func (a *App) DownloadUpdate() error {
	offered, err := a.offer()
	if err != nil {
		return err
	}
	return a.openInBrowser(offered.Address)
}

// SkipUpdate keeps the offered release's version, so an automatic check does not offer it again
// (FR-758).
func (a *App) SkipUpdate() error {
	offered, err := a.offer()
	if err != nil {
		return err
	}
	return a.keep(func(held *ports.Settings) { held.SkippedUpdate = offered.Latest })
}

// offer answers the release last offered, refusing where none has been.
func (a *App) offer() (services.UpdateStatus, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.offered.Outcome != services.UpdateAvailable {
		return services.UpdateStatus{}, errNothingOffered
	}
	return a.offered, nil
}
