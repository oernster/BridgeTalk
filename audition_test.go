package main

import (
	"errors"
	"strings"
	"testing"
)

func TestTheAuditionPaneListsWhatAVoiceCanBeHeardOn(t *testing.T) {
	app, _, _ := fixtureApp(t)

	groups := app.AuditionGroups("Alpha")
	if len(groups) != 3 {
		t.Fatalf("got %v, want a group for each of the three recorded areas", groups)
	}
	if groups[0].Key != "Cast" || groups[1].Key != "Docked" || groups[2].Key != "ShieldState" {
		t.Fatalf("keys are %q, %q and %q, want them sorted",
			groups[0].Key, groups[1].Key, groups[2].Key)
	}
	if groups[1].Clips != 1 {
		t.Fatalf("the Docked group offers %d clips, want 1", groups[1].Clips)
	}
	if groups[2].Label != "Shield state" {
		t.Fatalf("label = %q, want the key in words", groups[2].Label)
	}
}

// The pane asks about voices the reader has not cast, so a name matching nothing is an
// empty list rather than an error. It returns an allocated slice on purpose: a nil one
// marshals as null and the pane then renders nothing at all instead of an empty pane.
func TestTheAuditionPaneAnswersEmptyForAVoiceThatIsNotThere(t *testing.T) {
	app, _, _ := fixtureApp(t)

	groups := app.AuditionGroups("a voice nobody owns")
	if groups == nil {
		t.Fatal("got nil, want an allocated empty list")
	}
	if len(groups) != 0 {
		t.Fatalf("got %v, want nothing", groups)
	}
}

// Pressing an audition button is an explicit request to hear something, so it plays
// even while the application's reactions to the game are muted. Answering it with
// silence would read as a fault rather than as the mute doing its job.
func TestAnAuditionPlaysEvenWhileMuted(t *testing.T) {
	app, player, _ := fixtureApp(t)
	app.SetMuted(true)

	played, err := app.Audition("Alpha", "ShieldState")
	if err != nil {
		t.Fatalf("auditioning: %v", err)
	}
	if played.Group != "ShieldState" {
		t.Fatalf("group = %q, want ShieldState", played.Group)
	}
	if played.Clip != "a.mp3" {
		t.Fatalf("clip = %q, want the file name alone", played.Clip)
	}

	player.mu.Lock()
	defer player.mu.Unlock()
	if len(player.played) != 1 {
		t.Fatalf("the device was given %d sequences, want one", len(player.played))
	}
}

func TestAnAuditionOfAVoiceThatIsNotThereIsRefused(t *testing.T) {
	app, _, _ := fixtureApp(t)

	if _, err := app.Audition("a voice nobody owns", "ShieldState"); err == nil {
		t.Fatal("a voice that is not installed auditioned")
	}
}

// A group the voice has nothing for is named in the refusal, in the words the pane
// shows rather than the key, so the message matches the button that was pressed.
func TestAnAuditionOfAGroupTheVoiceHasNothingForIsRefusedByName(t *testing.T) {
	app, _, _ := fixtureApp(t)

	_, err := app.Audition("Alpha", "UnderAttack")
	if err == nil {
		t.Fatal("a group the voice has nothing for auditioned")
	}
	if !strings.Contains(err.Error(), "Under attack") {
		t.Errorf("refusal = %q, want it to name the group as the pane does", err)
	}
}

func TestAnAuditionWithNoAudioDeviceSaysSo(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.session.player = nil

	_, err := app.Audition("Alpha", "ShieldState")
	if err == nil {
		t.Fatal("an audition succeeded with no device to play through")
	}
	if !strings.Contains(err.Error(), "audio device") {
		t.Errorf("refusal = %q, want it to name the missing device", err)
	}
}

// A device that refuses the clip has to reach the pane, since the pane's whole job is
// to report what happened; a silent failure reads as the button doing nothing.
func TestADeviceThatRefusesTheClipIsReported(t *testing.T) {
	app, player, _ := fixtureApp(t)
	player.failWith = errors.New("the device is busy")

	_, err := app.Audition("Alpha", "ShieldState")
	if err == nil {
		t.Fatal("a refused clip reported success")
	}
	if !strings.Contains(err.Error(), "the device is busy") {
		t.Errorf("refusal = %q, want the device's own words carried through", err)
	}
}

func TestStoppingAnAuditionStopsTheDevice(t *testing.T) {
	app, player, _ := fixtureApp(t)

	app.StopAudition()

	player.mu.Lock()
	defer player.mu.Unlock()
	if player.stops != 1 {
		t.Fatalf("the device was stopped %d times, want once", player.stops)
	}
}

// A group key is the game's own spelling, so the pane reads it the way the breakdown
// dialog heads it: initialisms kept whole, every other word lower case.
func TestAGroupKeyIsTurnedIntoTheWordsThePaneShows(t *testing.T) {
	cases := map[string]string{
		"FSDJump":         "FSD jump",
		"SRVDestroyed":    "SRV destroyed",
		"DockingGranted":  "Docking granted",
		"LandingGearDown": "Landing gear down",
		"":                "",
	}
	for key, want := range cases {
		if got := label(key); got != want {
			t.Errorf("label(%q) = %q, want %q", key, got, want)
		}
	}
}

// The full path is the reader's own directory and says nothing they do not know, so
// only the file name reaches the pane. Both separators appear on Windows.
func TestAClipIsNamedByItsFileAlone(t *testing.T) {
	cases := map[string]string{
		`C:\Recordings\Alpha\Docked\b.mp3`:     "b.mp3",
		"/home/someone/recordings/alpha/b.mp3": "b.mp3",
		"b.mp3":                                "b.mp3",
		"":                                     "",
	}
	for path, want := range cases {
		if got := clipName(path); got != want {
			t.Errorf("clipName(%q) = %q, want %q", path, got, want)
		}
	}
}
