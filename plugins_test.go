package main

// Loading plugins where the application actually runs, through the real loader rather than
// a stand-in. Nothing here needs a plugin to exist: what is proved is that the folder beside
// the application is the one looked in, that nothing there is silence, then that anything
// there which is no plugin is named in the log rather than passed over quietly.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin/plugintest"
	"github.com/oernster/bridge-talk/internal/infrastructure/runlog"
	"github.com/oernster/bridge-talk/internal/product"
)

// written collects what was logged, so a test can read it.
type written struct{ lines strings.Builder }

func (w *written) Write(raw []byte) (int, error) { return w.lines.Write(raw) }

func (w *written) String() string { return w.lines.String() }

func TestPluginsAreLookedForBesideTheApplication(t *testing.T) {
	t.Parallel()

	// The executable named here is the one an install writes, so the folder beside it is the
	// install directory (FR-560).
	installed := filepath.Join("C:", "Programs", product.Slug)

	got := pluginsBeside(filepath.Join(installed, product.Slug+".exe"))

	want := filepath.Join(installed, pluginsFolder)
	if got != want {
		t.Errorf("looked in %q, want %q", got, want)
	}
}

func TestNoPluginsFolderIsSilence(t *testing.T) {
	t.Parallel()

	log := &written{}
	set := loadPlugins(filepath.Join(t.TempDir(), product.Slug+".exe"), nil, runlog.NewLines(log))
	defer set.Close()

	if len(set.Voices()) != 0 {
		t.Errorf("found %d voices where there is no folder", len(set.Voices()))
	}
	if log.String() != "" {
		t.Errorf("the log says %q about a machine with no plugins, want nothing", log.String())
	}
}

// A file in the plugins folder that is no plugin is named in the log with what was wrong
// with it (FR-567). This is the real loader and the real library loading, so it is also the
// proof that the composition root reaches them.
func TestSomethingInTheFolderThatIsNoPluginIsNamedInTheLog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	folder := filepath.Join(dir, pluginsFolder)
	if err := os.Mkdir(folder, 0o700); err != nil {
		t.Fatalf("making the plugins folder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(folder, "notes.dll"), []byte("this is text"), 0o600); err != nil {
		t.Fatalf("writing the file: %v", err)
	}

	log := &written{}
	set := loadPlugins(filepath.Join(dir, product.Slug+".exe"), nil, runlog.NewLines(log))
	defer set.Close()

	if len(set.Voices()) != 0 {
		t.Errorf("a text file offered %d voices", len(set.Voices()))
	}
	logged := log.String()
	if !strings.Contains(logged, "notes.dll") {
		t.Errorf("the log says %q, want the file named", logged)
	}
	if !strings.Contains(logged, "passed over") {
		t.Errorf("the log says %q, want it to say the plugin was passed over", logged)
	}
}

// A voice inside a plugin that loaded perfectly well is passed over when its audio is not on
// this machine (FR-570); that reaches the log too (FR-567). The plugin itself says nothing,
// since it loaded: only what was passed over is written.
func TestAVoicePassedOverInsideALoadedPluginIsNamedInTheLog(t *testing.T) {
	t.Parallel()

	set := offeringAll(t, loadedPlugin{file: "crew.dll", name: "Bridge Crew", voices: []plugintest.Voice{
		{ID: "one", Name: "The First Officer", Ready: true},
		{ID: "two", Name: "The Engineer", Reason: "its recordings are gone"},
		{ID: "three", Name: "The Pilot"},
	}})

	log := &written{}
	reportPlugins(set, runlog.NewLines(log))

	logged := log.String()
	if !strings.Contains(logged, "the voice The Engineer in the plugin crew.dll was passed over: its recordings are gone") {
		t.Errorf("the log says %q, want the voice named with the reason it gave", logged)
	}
	if !strings.Contains(logged, "the voice The Pilot in the plugin crew.dll was passed over: it gave no reason") {
		t.Errorf("the log says %q, want a voice that gave no reason said so", logged)
	}
	if strings.Contains(logged, "The First Officer") {
		t.Errorf("the log says %q about a voice that can speak, want nothing", logged)
	}
	if strings.Contains(logged, "note: the plugin crew.dll was passed over") {
		t.Errorf("the log says %q about a plugin that loaded, want nothing", logged)
	}
}

func TestAnApplicationThatCannotTellWhereItIsLoadsNoPlugin(t *testing.T) {
	t.Parallel()

	log := &written{}
	set := loadPlugins("", os.ErrNotExist, runlog.NewLines(log))
	defer set.Close()

	if len(set.Voices()) != 0 {
		t.Errorf("found %d voices with nowhere to look", len(set.Voices()))
	}
	if !strings.Contains(log.String(), "cannot tell where it is") {
		t.Errorf("the log says %q, want it to say why nothing was loaded", log.String())
	}
}

// The plugin thread ends when the application does, with everything else the session holds.
// A voice that answered before the session was released answers nothing after it.
func TestReleasingTheSessionClosesThePluginThread(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one.dll"), []byte("stand-in"), 0o600); err != nil {
		t.Fatalf("writing the file: %v", err)
	}
	set := plugin.Load(dir, func(string) (plugin.Library, error) {
		return &plugintest.Plugin{
			Name: "Crew",
			Voices: []plugintest.Voice{{
				ID: "one", Name: "The First Officer", Ready: true,
				Answers: map[string][][]string{"Docked": {{`C:\audio\docked.mp3`}}},
			}},
		}, nil
	})

	current, _ := fixtureSession(t, newFakePlayer())
	current.plugins = set
	voice := set.Voices()[0]

	if _, ok := voice.Lookup("Docked"); !ok {
		t.Fatal("the voice answered nothing before the session was released")
	}

	current.release()

	if _, ok := voice.Lookup("Docked"); ok {
		t.Error("the voice still answers after the session was released, so the thread is still running")
	}
}
