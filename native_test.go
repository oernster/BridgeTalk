package main

// Where machine voices and plugins are not available yet (FR-817, FR-818), asked on every platform
// through the facade's field and the parameters the tray and the loader take.

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/runlog"
	"github.com/oernster/bridge-talk/internal/product"
)

// linuxName is the platform the worked examples stand on.
const linuxName = "Linux"

// FR-817: no machine voice is offered and the state names the platform, so the Cast pane can say
// why where the voices would be.
func TestWhereMachineVoicesAreMissingNoneIsOfferedAndThePlatformIsNamed(t *testing.T) {
	app, _ := newTestApp(t, newFakePlayer())
	if offered := app.MachineVoices(); len(offered) == 0 {
		t.Fatal("no machine voice was offered where they are available")
	}
	if got := app.State().NativeMissingOn; got != "" {
		t.Errorf("state names %q where machine voices are available, want nothing", got)
	}

	app.nativeMissingOn = linuxName
	if offered := app.MachineVoices(); offered == nil || len(offered) != 0 {
		t.Errorf("offered %v, want an empty list", offered)
	}
	if got := app.State().NativeMissingOn; got != linuxName {
		t.Errorf("state names %q, want %q", got, linuxName)
	}
}

// FR-817: the tray menu lists the recorded voices alone.
func TestWhereMachineVoicesAreMissingTheTrayOffersRecordedVoicesAlone(t *testing.T) {
	current, _ := fixtureSession(t, newFakePlayer())

	if got, want := trayChoices(current.available, nil, linuxName), playable(current.available); !slices.Equal(got, want) {
		t.Errorf("tray offers %+v, want %+v", got, want)
	}
}

// FR-818: the plugins folder is not looked in, however much is in it; the log says why no plugin was
// loaded.
func TestWherePluginsAreMissingTheFolderIsNotLookedIn(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	folder := filepath.Join(dir, product.PluginsFolder)
	if err := os.Mkdir(folder, 0o700); err != nil {
		t.Fatalf("making the plugins folder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(folder, "notes.dll"), []byte("this is text"), 0o600); err != nil {
		t.Fatalf("writing the file: %v", err)
	}

	log := &written{}
	set := loadPlugins(filepath.Join(dir, product.Slug+".exe"), nil, linuxName, runlog.NewLines(log))
	defer set.Close()

	if len(set.Voices()) != 0 || len(set.Refusals) != 0 {
		t.Errorf("voices %d, refusals %v; want the folder left unread", len(set.Voices()), set.Refusals)
	}
	logged := log.String()
	if strings.Contains(logged, "notes.dll") || !strings.Contains(logged, linuxName) {
		t.Errorf("the log says %q, want it to say plugins are not loaded on %s without naming the file", logged, linuxName)
	}
}
