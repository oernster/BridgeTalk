package main

// FR-572: a plugin's audio is played where it stands and never copied, moved, rewritten or deleted.
// The audio is more likely than a recording to belong to somebody else, so the promise made to
// recordings (CON-7, NFR-S-2) is made again here and held by watching the audio's own folder.

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/domain/take"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin/plugintest"
)

// held is one entry of a folder as it stands: what it holds and when it was last written. A folder is
// held by being there alone. Its own time moved with nothing run against it in one of five tries on
// 2026-09-16, half a millisecond after the test's own last write inside it, so that time says
// nothing about the application.
type held struct {
	folder  bool
	content string
	written time.Time
}

// standing reads every entry under dir, so two readings can be compared whole.
func standing(t *testing.T, dir string) map[string]held {
	t.Helper()
	found := map[string]held{}
	err := filepath.WalkDir(dir, func(at string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		name, _ := filepath.Rel(dir, at)
		if entry.IsDir() {
			found[name] = held{folder: true}
			return nil
		}
		body, err := os.ReadFile(at)
		found[name] = held{content: string(body), written: info.ModTime()}
		return err
	})
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	return found
}

// A plugin voice is cast, listed, checked and heard, which is everything the application does with
// one; the folder its audio lives in is then exactly as it was, file for file and folder for folder,
// with nothing added, moved or written. The cue is proved to have reached that audio first, since a
// folder nobody touched proves nothing.
func TestAPluginVoicesAudioIsLeftAsItWasFound(t *testing.T) {
	audio := t.TempDir()
	docked := filepath.Join(audio, "docked.wav")
	second := filepath.Join(audio, "more", "docked, second part.wav")
	for path, body := range map[string]string{
		docked:                            "the first part",
		second:                            "the second part",
		filepath.Join(audio, "notes.txt"): "the owner's own notes",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("making %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	before := standing(t, audio)

	app, player, _ := fixtureApp(t)
	app.settings = &fakeSettings{}
	source := &fakeSource{name: "journal"}
	app.sources = []ports.EventSource{source}
	app.session.reporter = reporter{app: app}
	app.session.plugins = offering(t, "Bridge Crew", plugintest.Voice{
		ID: "one", Name: "The First Officer", Ready: true,
		Answers: map[string][]take.Take{"Docked": {take.Of(docked, second)}},
	})

	if err := app.CastPluginVoice("Bridge Crew", "one"); err != nil {
		t.Fatalf("casting: %v", err)
	}
	app.PluginVoices()
	app.PluginChecklist()
	app.State()
	source.events = []event.Event{event.New(
		event.SourceJournal, "Docked", event.EdgeNone, map[string]any{}, time.Date(2026, 9, 16, 11, 0, 0, 0, time.UTC),
	)}
	app.pollAndAnnounce()

	if !slices.ContainsFunc(playedSoFar(player), func(played take.Take) bool {
		return slices.Equal(played, take.Of(docked, second))
	}) {
		t.Fatalf("played %v, want the plugin's take played where it stands", playedSoFar(player))
	}
	after := standing(t, audio)
	for name, was := range before {
		if now, ok := after[name]; !ok || now != was {
			t.Errorf("%s was %+v and is now %+v", name, was, now)
		}
	}
	for name := range after {
		if _, ok := before[name]; !ok {
			t.Errorf("%s was added beside the plugin's audio", name)
		}
	}
}
