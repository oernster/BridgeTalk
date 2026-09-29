package services_test

// The update check (FR-756 to FR-759): what a release is compared with and what is offered.

import (
	"errors"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
)

// publishedRelease answers one release; its error instead where it has one.
type publishedRelease struct {
	release ports.Release
	err     error
}

func (p publishedRelease) Latest() (ports.Release, error) { return p.release, p.err }

// Addresses a release carries in these tests.
const (
	releasePage   = "https://example.test/releases/tag/v1.5.0"
	windowsSetup  = "https://example.test/releases/download/v1.5.0/TheSetup.exe"
	linuxFlatpak  = "https://example.test/releases/download/v1.5.0/TheApp.flatpak"
	runningNow    = "1.4.2"
	noSkip        = ""
	windowsSystem = "windows"
)

// tagged is a release carrying both platforms' files under the given tag.
func tagged(tag string) publishedRelease {
	return publishedRelease{release: ports.Release{
		Version: tag,
		Page:    releasePage,
		Assets: []ports.Asset{
			{Name: "TheSetup.exe", Address: windowsSetup},
			{Name: "TheApp.flatpak", Address: linuxFlatpak},
		},
	}}
}

// FR-756: a later release is offered, its tag's leading v removed.
func TestANewerReleaseIsOffered(t *testing.T) {
	t.Parallel()
	for _, tag := range []string{"v1.5.0", "1.5.0", " v1.4.3 ", "v2.0", "v1.4.2.1"} {
		got := services.NewUpdateService(tagged(tag), runningNow, windowsSystem).Check(noSkip)
		if got.Outcome != services.UpdateAvailable {
			t.Errorf("%q against %s = %s, want available", tag, runningNow, got.Outcome)
		}
		if got.Running != runningNow {
			t.Errorf("%q: running %q, want %q", tag, got.Running, runningNow)
		}
	}
	got := services.NewUpdateService(tagged("v1.5.0"), runningNow, windowsSystem).Check(noSkip)
	if got.Latest != "1.5.0" {
		t.Errorf("latest %q, want the tag less its v", got.Latest)
	}
}

// FR-756: the running release, an older one or the same one written longer is not offered.
func TestTheRunningReleaseOrAnOlderOneIsNotOffered(t *testing.T) {
	t.Parallel()
	for _, tag := range []string{"v1.4.2", "v1.4.1", "v0.9.9", "v1.4.2.0", "v1.4"} {
		got := services.NewUpdateService(tagged(tag), runningNow, windowsSystem).Check(noSkip)
		if got.Outcome != services.UpdateCurrent {
			t.Errorf("%q against %s = %s, want current", tag, runningNow, got.Outcome)
		}
	}
}

// FR-756: a tag that is not dotted whole numbers, a pre-release suffix or a sign among them, is
// never newer, so a malformed tag cannot prompt.
func TestAVersionThatIsNotDottedNumbersIsNeverNewer(t *testing.T) {
	t.Parallel()
	for _, tag := range []string{"v1.5.0-rc1", "v9.x", "", "v", "v1..5", "v+2.0.0", "v-2.0.0", "V2.0.0", "v99999999999999999999.0"} {
		got := services.NewUpdateService(tagged(tag), runningNow, windowsSystem).Check(noSkip)
		if got.Outcome != services.UpdateCurrent {
			t.Errorf("%q = %s, want current: a tag that cannot be read is never newer", tag, got.Outcome)
		}
	}
}

// FR-756: a release that cannot be read is unreachable, with nothing to offer and nothing to open.
func TestAnUnreachableReleaseIsSaidToBeUnreachable(t *testing.T) {
	t.Parallel()
	source := publishedRelease{err: errors.New("no route to host")}
	got := services.NewUpdateService(source, runningNow, windowsSystem).Check(noSkip)
	want := services.UpdateStatus{Outcome: services.UpdateUnreachable, Running: runningNow}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// FR-759: a running version that is not dotted whole numbers cannot be compared, which is said
// rather than read as up to date.
func TestABuildFromSourceCannotBeCompared(t *testing.T) {
	t.Parallel()
	for _, running := range []string{"0.0.0-dev", "", "dev"} {
		got := services.NewUpdateService(tagged("v1.5.0"), running, windowsSystem).Check(noSkip)
		if got.Outcome != services.UpdateUncomparable || got.Latest != "1.5.0" {
			t.Errorf("running %q = %+v, want uncomparable naming 1.5.0", running, got)
		}
	}
}

// FR-757: Download is the release's own file for this platform, matched by its ending in any case.
func TestTheDownloadIsTheFileForThisPlatform(t *testing.T) {
	t.Parallel()
	for platform, want := range map[string]string{"windows": windowsSetup, "linux": linuxFlatpak} {
		got := services.NewUpdateService(tagged("v1.5.0"), runningNow, platform).Check(noSkip)
		if got.Address != want {
			t.Errorf("%s downloads %q, want %q", platform, got.Address, want)
		}
	}
	shouting := publishedRelease{release: ports.Release{
		Version: "v1.5.0", Page: releasePage,
		Assets: []ports.Asset{{Name: "THESETUP.EXE", Address: windowsSetup}},
	}}
	if got := services.NewUpdateService(shouting, runningNow, windowsSystem).Check(noSkip); got.Address != windowsSetup {
		t.Errorf("an ending in capitals downloads %q, want %q", got.Address, windowsSetup)
	}
}

// FR-757: a release carrying no file for this platform sends the reader to the release's page; so
// does a platform with no file of its own.
func TestADownloadFallsBackToTheReleasePage(t *testing.T) {
	t.Parallel()
	bare := publishedRelease{release: ports.Release{Version: "v1.5.0", Page: releasePage}}
	if got := services.NewUpdateService(bare, runningNow, windowsSystem).Check(noSkip); got.Address != releasePage {
		t.Errorf("a release with no files downloads %q, want its page", got.Address)
	}
	if got := services.NewUpdateService(tagged("v1.5.0"), runningNow, "darwin").Check(noSkip); got.Address != releasePage {
		t.Errorf("a platform with no file downloads %q, want the page", got.Address)
	}
}

// FR-758: the release the reader skipped is reported as skipped rather than offered.
func TestASkippedReleaseIsNotOfferedAgain(t *testing.T) {
	t.Parallel()
	got := services.NewUpdateService(tagged("v1.5.0"), runningNow, windowsSystem).Check("1.5.0")
	if got.Outcome != services.UpdateSkipped {
		t.Fatalf("a skipped release = %s, want skipped", got.Outcome)
	}
}

// FR-758: a release later than the one skipped is a new question and is offered.
func TestALaterReleaseThanTheSkippedOneIsOffered(t *testing.T) {
	t.Parallel()
	got := services.NewUpdateService(tagged("v1.5.1"), runningNow, windowsSystem).Check("1.5.0")
	if got.Outcome != services.UpdateAvailable {
		t.Fatalf("1.5.1 with 1.5.0 skipped = %s, want available", got.Outcome)
	}
}
