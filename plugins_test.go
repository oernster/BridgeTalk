package main

// Loading plugins where the application actually runs, through the real loader rather than
// a stand-in. Nothing here needs a plugin to exist: what is proved is that each platform
// looks in its own folder, that nothing there is silence, then that anything
// there which is no plugin is named in the log rather than passed over quietly.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/take"
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

	got, err := pluginsFolder(windowsOS, finding(filepath.Join(installed, product.Slug+".exe")), notAsked(t))

	want := filepath.Join(installed, product.PluginsFolder)
	if err != nil || got != want {
		t.Errorf("looked in %q, %v; want %q", got, err, want)
	}
}

// FR-818: off Windows the folder is inside the product's data folder, whatever the executable is.
func TestOffWindowsPluginsAreLookedForInTheDataFolder(t *testing.T) {
	t.Parallel()

	data := filepath.Join("home", "commander", ".local", "share", product.Slug)

	got, err := pluginsFolder("linux", notAsked(t), finding(data))

	want := filepath.Join(data, product.PluginsFolder)
	if err != nil || got != want {
		t.Errorf("looked in %q, %v; want %q", got, err, want)
	}
}

// A folder that cannot be found is answered as the reason, on either rule.
func TestAPluginsFolderThatCannotBeFoundAnswersWhy(t *testing.T) {
	t.Parallel()

	failing := func() (string, error) { return "", os.ErrNotExist }
	for _, goos := range []string{windowsOS, "linux"} {
		if got, err := pluginsFolder(goos, failing, failing); got != "" || err != os.ErrNotExist {
			t.Errorf("%s: answered %q, %v; want the reason", goos, got, err)
		}
	}
}

// finding is a lookup that finds path.
func finding(path string) func() (string, error) {
	return func() (string, error) { return path, nil }
}

// notAsked is a lookup the rule under test must not ask.
func notAsked(t *testing.T) func() (string, error) {
	return func() (string, error) {
		t.Error("the other platform's folder was asked for")
		return "", os.ErrInvalid
	}
}

func TestNoPluginsFolderIsSilence(t *testing.T) {
	t.Parallel()

	log := &written{}
	folder := filepath.Join(t.TempDir(), product.PluginsFolder)
	set := loadPlugins(folder, nil, false, runlog.NewLines(log))
	defer set.Close()

	if len(set.Voices()) != 0 {
		t.Errorf("found %d voices where there is no folder", len(set.Voices()))
	}
	if log.String() != "" {
		t.Errorf("the log says %q about a machine with no plugins, want nothing", log.String())
	}
	if _, err := os.Stat(folder); !os.IsNotExist(err) {
		t.Errorf("a plugins folder was made where the application does not make one: %v", err)
	}
}

// Only Windows has a setup program to make the plugins folder, so the application makes it
// everywhere else (FR-576, FR-818).
func TestTheApplicationMakesThePluginsFolderOffWindowsAlone(t *testing.T) {
	t.Parallel()

	if makesPluginsFolder(windowsOS) {
		t.Error("the application makes the plugins folder on Windows, where setup makes it")
	}
	if !makesPluginsFolder("linux") {
		t.Error("the application does not make the plugins folder on Linux, where nothing else does")
	}
}

// FR-819: where the application makes the folder it is there after loading with nothing said
// about it; a folder already there is used as it is.
func TestAPluginsFolderTheApplicationMakesIsThereAfterLoading(t *testing.T) {
	t.Parallel()

	folder := filepath.Join(t.TempDir(), product.Slug, product.PluginsFolder)
	log := &written{}

	for range 2 {
		set := loadPlugins(folder, nil, true, runlog.NewLines(log))
		set.Close()
	}

	if info, err := os.Stat(folder); err != nil || !info.IsDir() {
		t.Errorf("no plugins folder at %s after loading: %v", folder, err)
	}
	if log.String() != "" {
		t.Errorf("the log says %q about making an empty folder, want nothing", log.String())
	}
}

// A plugins folder that cannot be made stops nothing: the reason is in the log and no plugin is
// loaded (FR-237).
func TestAPluginsFolderThatCannotBeMadeIsNamedInTheLog(t *testing.T) {
	t.Parallel()

	blocked := filepath.Join(t.TempDir(), "not a folder")
	if err := os.WriteFile(blocked, []byte("in the way"), 0o600); err != nil {
		t.Fatalf("writing the file: %v", err)
	}
	folder := filepath.Join(blocked, product.PluginsFolder)
	log := &written{}

	set := loadPlugins(folder, nil, true, runlog.NewLines(log))
	defer set.Close()

	if len(set.Voices()) != 0 {
		t.Errorf("found %d voices with no folder", len(set.Voices()))
	}
	if logged := log.String(); !strings.Contains(logged, "could not be made") || !strings.Contains(logged, folder) {
		t.Errorf("the log says %q, want the folder named with why it was not made", logged)
	}
}

// A file in the plugins folder that is no plugin is named in the log with what was wrong
// with it (FR-567). This is the real loader and the real library loading, so it is also the
// proof that the composition root reaches them.
func TestSomethingInTheFolderThatIsNoPluginIsNamedInTheLog(t *testing.T) {
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
	set := loadPlugins(folder, nil, false, runlog.NewLines(log))
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
	set := loadPlugins("", os.ErrNotExist, false, runlog.NewLines(log))
	defer set.Close()

	if len(set.Voices()) != 0 {
		t.Errorf("found %d voices with nowhere to look", len(set.Voices()))
	}
	if !strings.Contains(log.String(), "cannot tell where its plugins folder is") {
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
				Answers: map[string][]take.Take{"Docked": {take.Of(`C:\audio\docked.mp3`)}},
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
