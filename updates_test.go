package main

// The update check's surface on the facade (FR-756 to FR-759).

import (
	"errors"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
)

// latestRelease answers one release, whatever is asked.
type latestRelease struct{ release ports.Release }

func (l latestRelease) Latest() (ports.Release, error) { return l.release, nil }

// Addresses the offered release carries in these tests.
const (
	offeredSetup = "https://example.test/releases/download/v1.5.0/TheSetup.exe"
	offeredPage  = "https://example.test/releases/tag/v1.5.0"
)

// updatingApp is a facade running 1.4.2 on Windows, whose check finds tag, over store, handing
// addresses to desktop.
func updatingApp(tag string, store ports.SettingsStore, desktop *handOver) *App {
	release := ports.Release{
		Version: tag,
		Page:    offeredPage,
		Assets:  []ports.Asset{{Name: "TheSetup.exe", Address: offeredSetup}},
	}
	return &App{
		settings: store,
		browse:   desktop.open,
		updates:  services.NewUpdateService(latestRelease{release}, "1.4.2", "windows"),
	}
}

// FR-757: Download hands the browser the offered release's file for this platform with nothing
// else; the page is told no address.
func TestDownloadHandsTheOfferedReleaseToTheBrowser(t *testing.T) {
	t.Parallel()
	desktop := &handOver{}
	app := updatingApp("v1.5.0", &fakeSettings{}, desktop)

	got := app.CheckForUpdates(false)
	want := UpdateDTO{Outcome: "available", Running: "1.4.2", Latest: "1.5.0"}
	if got != want {
		t.Fatalf("CheckForUpdates = %+v, want %+v", got, want)
	}
	if err := app.DownloadUpdate(); err != nil {
		t.Fatalf("DownloadUpdate: %v", err)
	}
	if len(desktop.addresses) != 1 || desktop.addresses[0] != offeredSetup {
		t.Fatalf("handed over %q, want exactly %q", desktop.addresses, offeredSetup)
	}
}

// FR-757 and FR-758: before any release is offered, Download and Skip are refused; so they are after a
// check that offered none. Nothing is handed over; nothing is kept.
func TestADownloadWithNothingOfferedIsRefused(t *testing.T) {
	t.Parallel()
	desktop := &handOver{}
	store := &fakeSettings{}
	app := updatingApp("v1.4.2", store, desktop)

	for _, when := range []string{"before a check", "after a check offering nothing"} {
		if err := app.DownloadUpdate(); !errors.Is(err, errNothingOffered) {
			t.Errorf("%s: DownloadUpdate = %v, want the refusal", when, err)
		}
		if err := app.SkipUpdate(); !errors.Is(err, errNothingOffered) {
			t.Errorf("%s: SkipUpdate = %v, want the refusal", when, err)
		}
		app.CheckForUpdates(true)
	}
	if len(desktop.addresses) != 0 || store.saves != 0 {
		t.Fatalf("handed over %q and saved %d times, want nothing", desktop.addresses, store.saves)
	}
}

// FR-758: Skip keeps the offered release's version, leaving every other choice as it was.
func TestSkippingKeepsTheOfferedRelease(t *testing.T) {
	t.Parallel()
	store := &fakeSettings{held: ports.Settings{Voice: "Hugo"}}
	app := updatingApp("v1.5.0", store, &handOver{})

	app.CheckForUpdates(false)
	if err := app.SkipUpdate(); err != nil {
		t.Fatalf("SkipUpdate: %v", err)
	}
	want := ports.Settings{Voice: "Hugo", SkippedUpdate: "1.5.0"}
	if store.held.Voice != want.Voice || store.held.SkippedUpdate != want.SkippedUpdate {
		t.Fatalf("kept %+v, want %+v", store.held, want)
	}

	refused := errors.New("the disk is full")
	store.failure = refused
	if err := app.SkipUpdate(); !errors.Is(err, refused) {
		t.Fatalf("a skip that could not be kept = %v, want the store's own failure", err)
	}
}

// FR-758: an automatic check does not offer the skipped release; nothing is then offered to act on.
func TestAnAutomaticCheckHonoursTheSkippedRelease(t *testing.T) {
	t.Parallel()
	app := updatingApp("v1.5.0", &fakeSettings{held: ports.Settings{SkippedUpdate: "1.5.0"}}, &handOver{})

	if got := app.CheckForUpdates(false); got.Outcome != "skipped" {
		t.Fatalf("an automatic check = %+v, want the release reported skipped", got)
	}
	if err := app.DownloadUpdate(); !errors.Is(err, errNothingOffered) {
		t.Fatalf("DownloadUpdate after a skipped release = %v, want the refusal", err)
	}
}

// FR-759: a check asked for from Help offers the skipped release anyway.
func TestAManualCheckOffersASkippedRelease(t *testing.T) {
	t.Parallel()
	desktop := &handOver{}
	app := updatingApp("v1.5.0", &fakeSettings{held: ports.Settings{SkippedUpdate: "1.5.0"}}, desktop)

	if got := app.CheckForUpdates(true); got.Outcome != "available" {
		t.Fatalf("a manual check = %+v, want the skipped release offered", got)
	}
	if err := app.DownloadUpdate(); err != nil || len(desktop.addresses) != 1 {
		t.Fatalf("DownloadUpdate = %v handing over %q, want the offered file", err, desktop.addresses)
	}
}

// With no settings store the automatic check skips nothing: there is nowhere a skip was kept.
func TestAnAutomaticCheckWithNoStoreSkipsNothing(t *testing.T) {
	t.Parallel()
	if got := updatingApp("v1.5.0", nil, &handOver{}).CheckForUpdates(false); got.Outcome != "available" {
		t.Fatalf("with no store = %+v, want the release offered", got)
	}
}
