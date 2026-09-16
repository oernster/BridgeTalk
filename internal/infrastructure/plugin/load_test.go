package plugin_test

// Loading a folder of plugins. Every file is tried, one failing never stops another, then
// every plugin passed over is named with the reason it was passed over for. A plugin is
// somebody else's code, so most of what is below is a way of getting it wrong.

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/take"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin/plugintest"
	"github.com/oernster/bridge-talk/internal/product"
)

// folder writes a file per name into a new directory, since the loader reads a directory
// before it opens anything. What the files hold means nothing: the opener decides what a
// path answers.
func folder(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("not really a library"), 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return dir
}

// opening answers each file with the library or the error given for its name.
func opening(answers map[string]plugin.Library, failures map[string]error) plugin.Opener {
	return func(path string) (plugin.Library, error) {
		name := filepath.Base(path)
		if err, ok := failures[name]; ok {
			return nil, err
		}
		if library, ok := answers[name]; ok {
			return library, nil
		}
		return nil, errors.New("nothing answers for this file")
	}
}

// crew is a plugin offering one ready voice that answers two cues, one of them with a take
// recorded in three pieces.
func crew(name string) *plugintest.Plugin {
	return &plugintest.Plugin{
		Name: name,
		Voices: []plugintest.Voice{{
			ID: "one", Name: "The First Officer", Ready: true,
			Answers: map[string][]take.Take{
				"DockingGranted": {take.Of(`C:\audio\granted.mp3`)},
				"StartJump":      {take.Of(`C:\audio\a.mp3`, `C:\audio\b.mp3`, `C:\audio\c.mp3`)},
			},
		}},
	}
}

func TestEveryPluginInTheFolderIsLoadedInNameOrder(t *testing.T) {
	t.Parallel()

	dir := folder(t, "second.dll", "first.dll")
	set := plugin.Load(dir, opening(map[string]plugin.Library{
		"first.dll":  crew("First Crew"),
		"second.dll": crew("Second Crew"),
	}, nil))
	defer set.Close()

	if len(set.Refusals) != 0 {
		t.Fatalf("refused %+v", set.Refusals)
	}
	if len(set.Plugins) != 2 {
		t.Fatalf("loaded %d plugins, want 2", len(set.Plugins))
	}
	if set.Plugins[0].File != "first.dll" || set.Plugins[1].File != "second.dll" {
		t.Errorf("loaded %s then %s, want name order", set.Plugins[0].File, set.Plugins[1].File)
	}
	if set.Plugins[0].Name != "First Crew" {
		t.Errorf("plugin name = %q, want the name it gave rather than its file", set.Plugins[0].Name)
	}
	if len(set.Voices()) != 2 {
		t.Errorf("%d voices across both plugins, want 2", len(set.Voices()))
	}
}

// Every plugin is opened by its whole path inside the folder it was found in, never by its name
// alone, which Windows would look for along its search path and could find somewhere the user never
// put it. Only what the user put in the folder is loaded (FR-560, NFR-S-3).
func TestEveryPluginIsOpenedByItsWholePathInTheFolder(t *testing.T) {
	t.Parallel()

	dir := folder(t, "crew.dll", "deck.dll")
	var opened []string
	set := plugin.Load(dir, func(path string) (plugin.Library, error) {
		opened = append(opened, path)
		return crew(filepath.Base(path)), nil
	})
	defer set.Close()

	want := []string{filepath.Join(dir, "crew.dll"), filepath.Join(dir, "deck.dll")}
	if len(opened) != len(want) || opened[0] != want[0] || opened[1] != want[1] {
		t.Errorf("opened %v, want %v", opened, want)
	}
}

// A voice that cannot speak and gives no reason is said to have given none, so nothing reading the
// reason trails off after a colon (FR-570). A voice that can speak carries no reason at all.
func TestAVoiceThatGivesNoReasonIsSaidToHaveGivenNone(t *testing.T) {
	t.Parallel()

	dir := folder(t, "crew.dll")
	set := plugin.Load(dir, opening(map[string]plugin.Library{
		"crew.dll": &plugintest.Plugin{Name: "Crew", Voices: []plugintest.Voice{
			{ID: "one", Name: "The First Officer", Ready: true},
			{ID: "two", Name: "The Pilot"},
		}},
	}, nil))
	defer set.Close()

	voices := set.Voices()
	if len(voices) != 2 {
		t.Fatalf("loaded %d voices, want 2", len(voices))
	}
	if voices[0].Reason != "" {
		t.Errorf("a voice that can speak carries the reason %q, want none", voices[0].Reason)
	}
	if voices[1].Reason != "it gave no reason" {
		t.Errorf("a voice that cannot speak and gave no reason carries %q", voices[1].Reason)
	}
}

func TestAnAbsentFolderIsNotAFault(t *testing.T) {
	t.Parallel()

	set := plugin.Load(filepath.Join(t.TempDir(), "no-such-folder"), opening(nil, nil))
	defer set.Close()

	if len(set.Plugins) != 0 || len(set.Refusals) != 0 {
		t.Errorf("loaded %+v and refused %+v, want silence", set.Plugins, set.Refusals)
	}
}

func TestAFolderHoldingNothingLoadsNothing(t *testing.T) {
	t.Parallel()

	set := plugin.Load(folder(t), opening(nil, nil))
	defer set.Close()

	if len(set.Plugins) != 0 || len(set.Refusals) != 0 {
		t.Errorf("loaded %+v and refused %+v, want silence", set.Plugins, set.Refusals)
	}
}

// A directory inside the plugins folder is not a plugin file, so it is passed over without
// being reported: nothing failed.
func TestADirectoryInsideTheFolderIsIgnored(t *testing.T) {
	t.Parallel()

	dir := folder(t, "good.dll")
	if err := os.Mkdir(filepath.Join(dir, "notes"), 0o700); err != nil {
		t.Fatalf("making a directory: %v", err)
	}

	set := plugin.Load(dir, opening(map[string]plugin.Library{"good.dll": crew("Crew")}, nil))
	defer set.Close()

	if len(set.Plugins) != 1 || len(set.Refusals) != 0 {
		t.Errorf("loaded %d and refused %+v, want the one file alone", len(set.Plugins), set.Refusals)
	}
}

func TestOnePluginFailingDoesNotStopAnother(t *testing.T) {
	t.Parallel()

	dir := folder(t, "broken.dll", "good.dll")
	set := plugin.Load(dir, opening(
		map[string]plugin.Library{"good.dll": crew("Crew")},
		map[string]error{"broken.dll": errors.New("it could not be loaded: the file is not a library")},
	))
	defer set.Close()

	if len(set.Plugins) != 1 || set.Plugins[0].File != "good.dll" {
		t.Fatalf("loaded %+v, want the good one", set.Plugins)
	}
	if len(set.Refusals) != 1 || set.Refusals[0].File != "broken.dll" {
		t.Fatalf("refusals = %+v, want the broken one named", set.Refusals)
	}
	if !strings.Contains(set.Refusals[0].Why, "not a library") {
		t.Errorf("refusal said %q, want what was wrong with it", set.Refusals[0].Why)
	}
}

// Each of these is a plugin that loads and then misbehaves. All are refused by name, with
// the reason carrying enough for the plugin's author to act on.
func TestAPluginThatMisbehavesIsRefusedWithAReason(t *testing.T) {
	t.Parallel()

	noVoices := crew("Crew")
	noVoices.Voices = nil
	unnamed := crew("Crew")
	unnamed.Voices[0].Name = ""
	noID := crew("Crew")
	noID.Voices[0].ID = ""
	corrupt := crew("Crew")
	corrupt.Corrupt = true
	refusing := crew("Crew")
	refusing.RefuseDescribe = true

	for _, each := range []struct {
		name    string
		library plugin.Library
		mention string
	}{
		{"built against another version", &plugintest.Plugin{ABI: 99, Name: "Old"}, "version 99"},
		{"refusing to say what it is", refusing, "would not say what it is"},
		{"describing itself in no layout", corrupt, "will not read"},
		{"offering no voice", noVoices, "offers no voice"},
		{"offering a voice with no name", unnamed, "has no name"},
		{"offering a voice with no id", noID, "has no id"},
	} {
		t.Run(each.name, func(t *testing.T) {
			t.Parallel()

			dir := folder(t, "one.dll")
			set := plugin.Load(dir, opening(map[string]plugin.Library{"one.dll": each.library}, nil))
			defer set.Close()

			if len(set.Plugins) != 0 {
				t.Fatalf("loaded %+v, want it passed over", set.Plugins)
			}
			if len(set.Refusals) != 1 {
				t.Fatalf("refusals = %+v, want one", set.Refusals)
			}
			if !strings.Contains(set.Refusals[0].Why, each.mention) {
				t.Errorf("refusal said %q, want it to mention %q", set.Refusals[0].Why, each.mention)
			}
		})
	}
}

// A version mismatch names both versions, so the reader knows which way it is wrong and
// what to build against (FR-564).
func TestAVersionMismatchNamesBothVersions(t *testing.T) {
	t.Parallel()

	set := plugin.Load(folder(t, "one.dll"), opening(
		map[string]plugin.Library{"one.dll": &plugintest.Plugin{ABI: 99}}, nil))
	defer set.Close()

	why := set.Refusals[0].Why
	if !strings.Contains(why, "99") || !strings.Contains(why, strconv.Itoa(plugin.ABIVersion)) {
		t.Errorf("refusal said %q, want the version it stated and the one implemented", why)
	}
}

// FR-581: version 2 is the only version implemented, so a plugin built against version 1, whose
// layouts carry no group and no span, is refused by name rather than read wrongly.
func TestAPluginBuiltAgainstVersionOneIsRefused(t *testing.T) {
	t.Parallel()
	const first = 1

	set := plugin.Load(folder(t, "old.dll"), opening(
		map[string]plugin.Library{"old.dll": &plugintest.Plugin{ABI: first}}, nil))
	defer set.Close()

	if len(set.Plugins) != 0 || len(set.Refusals) != 1 {
		t.Fatalf("loaded %d and refused %+v, want the one plugin refused", len(set.Plugins), set.Refusals)
	}
	want := "built against interface version 1; this is " + product.Name + " 2"
	if refused := set.Refusals[0]; refused.File != "old.dll" || !strings.Contains(refused.Why, want) {
		t.Errorf("refusal = %+v, want old.dll named with %q", refused, want)
	}
}

func TestClosingTwiceIsSafe(t *testing.T) {
	t.Parallel()

	set := plugin.Load(folder(t, "one.dll"), opening(
		map[string]plugin.Library{"one.dll": crew("Crew")}, nil))

	set.Close()
	set.Close()
}
