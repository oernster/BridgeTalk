package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/cue"
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

// The real player stops whatever is running before it starts a sequence, so a second
// Play handed to it while a clip sounds cuts that clip off. A press while something
// plays must therefore never reach the device (FR-236). It is ignored rather than
// refused, so it names no clip and raises no error for the pane to show.
func TestAPressWhileAClipPlaysLeavesThatClipPlaying(t *testing.T) {
	app, player, _ := fixtureApp(t)

	if _, err := app.Audition("Alpha", "ShieldState"); err != nil {
		t.Fatalf("auditioning: %v", err)
	}
	ignored, err := app.Audition("Alpha", "Docked")
	if err != nil {
		t.Fatalf("a press while a clip plays raised %v, want it ignored", err)
	}
	if ignored.Clip != "" {
		t.Fatalf("an ignored press named %q, want no clip", ignored.Clip)
	}

	player.mu.Lock()
	defer player.mu.Unlock()
	if len(player.played) != 1 {
		t.Fatalf("the device was given %d sequences, want the first alone", len(player.played))
	}
	if player.stops != 0 {
		t.Fatalf("the device was stopped %d times, want the clip left playing", player.stops)
	}
}

// announcedPlaying reports whether the facade told the page that something is playing.
func announcedPlaying(log *recorder) bool {
	log.mu.Lock()
	defer log.mu.Unlock()
	for _, event := range log.events {
		if state, ok := event.payload.(PlaybackDTO); ok && event.name == playbackEvent && state.Playing {
			return true
		}
	}
	return false
}

// The pane holds its buttons from the moment a clip starts, so an audition that starts
// one says so rather than leaving the pane to find out when it ends (FR-236).
func TestAnAuditionAnnouncesThatItStarted(t *testing.T) {
	app, _, log := fixtureApp(t)

	if _, err := app.Audition("Alpha", "ShieldState"); err != nil {
		t.Fatalf("auditioning: %v", err)
	}

	if !announcedPlaying(log) {
		t.Fatal("an audition started a clip without telling the page")
	}
}

// Casting ends what was playing on purpose and plays the confirmation, which holds the
// audition buttons like any other clip.
func TestACastConfirmationAnnouncesThatItStarted(t *testing.T) {
	app, _, log := fixtureApp(t)

	if err := app.SelectVoice("Alpha"); err != nil {
		t.Fatalf("casting Alpha: %v", err)
	}

	if !announcedPlaying(log) {
		t.Fatal("a cast confirmation started without telling the page")
	}
}

// A reaction the game set off is a clip playing too, so a poll that starts one tells
// the page, which holds the audition buttons exactly as it does for an audition.
func TestAPollThatStartsAReactionAnnouncesIt(t *testing.T) {
	player := newFakePlayer()
	app, log := newTestApp(t, player)
	docked, err := cue.New(cue.Definition{
		ID: "Docked", Source: "journal", Event: "Docked", Priority: "notice",
	})
	if err != nil {
		t.Fatalf("building cue: %v", err)
	}
	app.session.scheduler.Submit(services.Request{Cue: docked, Clips: []string{"docked.mp3"}})

	app.pollAndAnnounce()

	if !announcedPlaying(log) {
		t.Fatal("a poll started a reaction without telling the page")
	}
}

// A poll that starts nothing says nothing, so the page is not told on every tick.
func TestAPollThatStartsNothingAnnouncesNothing(t *testing.T) {
	player := newFakePlayer()
	app, log := newTestApp(t, player)

	app.pollAndAnnounce()

	log.mu.Lock()
	defer log.mu.Unlock()
	if len(log.events) != 0 {
		t.Fatalf("an idle poll emitted %v, want nothing", log.events)
	}
}

// A pane opened part way through a clip asks, so the answer follows the device; with
// no device there is nothing that could be playing.
func TestThePaneCanAskWhetherAnythingIsPlaying(t *testing.T) {
	app, player, _ := fixtureApp(t)

	if app.Playing() {
		t.Fatal("an idle device reported playing")
	}
	player.mu.Lock()
	player.playing = true
	player.mu.Unlock()
	if !app.Playing() {
		t.Fatal("a playing device reported idle")
	}
	app.session.player = nil
	if app.Playing() {
		t.Fatal("no device at all reported playing")
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
