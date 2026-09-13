package main

// What is kept between runs: the voice that was cast and the directories that were chosen,
// each written only by the choice that made it (FR-201, FR-701, FR-711).

import (
	"errors"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
)

// The whole point of remembering a voice: the next run opens speaking with the voice
// that was cast rather than whichever one sorts first.
func TestTheCastVoiceIsRemembered(t *testing.T) {
	app, _, _ := fixtureApp(t)
	store := &fakeSettings{}
	app.settings = store

	if err := app.SelectVoice("Beta"); err != nil {
		t.Fatalf("casting Beta: %v", err)
	}
	if store.held.Voice != "Beta" {
		t.Errorf("remembered voice is %q, want Beta", store.held.Voice)
	}

	// The full name is stored rather than whatever was typed, so a prefix cast from
	// the command line is not written back as a prefix and left to match a different
	// voice once another one is installed beside it.
	if err := app.SelectVoice("Al"); err != nil {
		t.Fatalf("casting by prefix: %v", err)
	}
	if store.held.Voice != "Alpha" {
		t.Errorf("remembered voice is %q, want the full name Alpha", store.held.Voice)
	}
}

// A cast that is refused changes nothing, so it must not overwrite the voice that is
// still speaking with the name of one that never spoke.
func TestARefusedCastIsNotRemembered(t *testing.T) {
	app, _, _ := fixtureApp(t)
	store := &fakeSettings{}
	app.settings = store
	if err := app.SelectVoice("Beta"); err != nil {
		t.Fatalf("casting Beta: %v", err)
	}

	for _, name := range []string{"Bystander", "a voice nobody owns"} {
		if err := app.SelectVoice(name); err == nil {
			t.Fatalf("casting %q reported success", name)
		}
		if store.held.Voice != "Beta" {
			t.Errorf("after refusing %q the remembered voice is %q, want Beta", name, store.held.Voice)
		}
	}
}

// Only a deliberate cast names a voice. A directory chosen in Settings re-casts by
// name and falls back to whatever voice is there when the stored name is not, so
// writing the voice from there would freeze a voice nobody picked and go on honouring
// it every run afterwards.
func TestChoosingADirectoryDoesNotWriteAVoiceNobodyCast(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.settings = &fakeSettings{}
	answering(app, journalDirFixture(t), nil)

	if _, err := app.ChooseJournalDir(); err != nil {
		t.Fatalf("choosing a journal directory: %v", err)
	}
	if held := app.settings.Load(); held.Voice != "" {
		t.Errorf("remembered voice is %q, want nothing written", held.Voice)
	}

	answering(app, libraryRootFixture(t), nil)
	if _, err := app.ChooseLibraryRoot(); err != nil {
		t.Fatalf("choosing a library root: %v", err)
	}
	if held := app.settings.Load(); held.Voice != "" {
		t.Errorf("remembered voice is %q, want nothing written", held.Voice)
	}
}

// A machine with nowhere to keep settings is a working application that forgets. The
// cast still happens and is still announced; only the memory of it fails.
func TestAVoiceThatCannotBeRememberedStillSpeaks(t *testing.T) {
	app, _, log := fixtureApp(t)
	app.settings = &fakeSettings{failure: errors.New("the disk is full")}

	if err := app.SelectVoice("Beta"); err == nil {
		t.Fatal("a cast that could not be remembered reported success")
	}
	if app.session.active.Name != "Beta" {
		t.Errorf("active voice is %q, want Beta cast regardless", app.session.active.Name)
	}
	if log.countEmitted(stateEvent) == 0 {
		t.Error("the window was never told about a cast that happened")
	}
}

// FR-701 and FR-711: choosing one directory keeps that directory and nothing else. The fixture's
// journal directory was never chosen and does not exist; a recordings directory given for
// one run was never chosen either. Neither may be frozen into Settings by the other
// choice, while what Settings already holds for the other directory stays as it was.
func TestChoosingOneDirectoryKeepsThatDirectoryAlone(t *testing.T) {
	app, _, _ := fixtureApp(t)
	store := &fakeSettings{}
	app.settings = store
	recordings := libraryRootFixture(t)
	answering(app, recordings, nil)

	if _, err := app.ChooseLibraryRoot(); err != nil {
		t.Fatalf("choosing a library root: %v", err)
	}
	if store.held.LibraryRoot != recordings || store.held.JournalDir != "" {
		t.Fatalf("kept %+v, want the recordings directory alone", store.held)
	}

	app.libraryRoot = "given for one run"
	store.held = ports.Settings{LibraryRoot: "chosen earlier"}
	journalDir := journalDirFixture(t)
	answering(app, journalDir, nil)
	if _, err := app.ChooseJournalDir(); err != nil {
		t.Fatalf("choosing a journal directory: %v", err)
	}
	if store.held.JournalDir != journalDir || store.held.LibraryRoot != "chosen earlier" {
		t.Fatalf("kept %+v, want the journal directory beside what was already held", store.held)
	}
}

func TestAChoiceThatCannotBeRememberedIsReported(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.settings = &fakeSettings{failure: errors.New("the disk is full")}
	answering(app, libraryRootFixture(t), nil)

	if _, err := app.ChooseLibraryRoot(); err == nil {
		t.Fatal("a root that could not be remembered reported success")
	}

	app.settings = &fakeSettings{failure: errors.New("the disk is full")}
	answering(app, journalDirFixture(t), nil)
	if _, err := app.ChooseJournalDir(); err == nil {
		t.Fatal("a journal directory that could not be remembered reported success")
	}
}
