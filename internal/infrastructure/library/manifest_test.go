package library

// FR-210 and FR-211 over a real disk: what a voice's manifest adds, what one entry that cannot be
// used costs and what becomes of a voice whose manifest cannot be used at all.

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
)

// writeManifest puts a manifest holding the given lines in a voice directory.
func writeManifest(t *testing.T, dir string, lines ...string) {
	t.Helper()
	audiotest.WriteFile(t, filepath.Join(dir, ManifestFile), []byte(strings.Join(lines, "\n")+"\n"))
}

// FR-210: the manifest's name is what the voice is shown by and its credit travels with it, while
// the directory name stays the identity. A blank name shows the directory, as no manifest does.
func TestAManifestNamesAndCreditsItsVoice(t *testing.T) {
	root := t.TempDir()
	audiotest.WriteTake(t, filepath.Join(root, "Alice", "StartJump.wav"))
	writeManifest(t, filepath.Join(root, "Alice"), `name = "  Alice Hart "`, `credit = "Recorded by Alice, 2026"`)
	audiotest.WriteTake(t, filepath.Join(root, "Bob", "StartJump.wav"))
	audiotest.WriteTake(t, filepath.Join(root, "Cleo", "StartJump.wav"))
	writeManifest(t, filepath.Join(root, "Cleo"), `name = "   "`)

	voices, report := scanned(t, root, journalTable(t, "StartJump"))
	if report.Any() {
		t.Fatalf("report = %+v, want nothing passed over", report)
	}
	got := make([][3]string, 0, len(voices))
	for _, voice := range voices {
		got = append(got, [3]string{voice.Name, voice.Display(), voice.Credit})
	}
	want := [][3]string{
		{"Alice", "Alice Hart", "Recorded by Alice, 2026"},
		{"Bob", "Bob", ""},
		{"Cleo", "Cleo", ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("name, display, credit = %v, want %v", got, want)
	}
}

// FR-210: declared takes add to those the names found, from any depth inside the voice's
// directory; a manifest alone can make a directory a voice. A take declared where the names
// already found it counts once; nothing the manifest reaches is reported as unmatched.
func TestAManifestAddsTheTakesItDeclares(t *testing.T) {
	root := t.TempDir()
	alice := filepath.Join(root, "Alice")
	audiotest.WriteTake(t, filepath.Join(alice, "StartJump.wav"))
	audiotest.WriteTake(t, filepath.Join(alice, "odd.wav"))
	audiotest.WriteTake(t, filepath.Join(alice, "alternates", "another.mp3"))
	writeManifest(t, alice,
		"[takes]",
		`"isindanger.set" = ["odd.wav", "./alternates/another.mp3"]`,
		`"StartJump" = ["StartJump.wav"]`,
	)
	dana := filepath.Join(root, "Dana")
	audiotest.WriteTake(t, filepath.Join(dana, "recording one.wav"))
	writeManifest(t, dana, "[takes]", `"StartJump" = ["recording one.wav"]`)

	voices, report := scanned(t, root, journalTable(t, "StartJump", "IsInDanger.Set"))
	if report.Any() {
		t.Fatalf("report = %+v, want nothing passed over", report)
	}
	if len(voices) != 2 {
		t.Fatalf("found %d voices, want Alice and Dana", len(voices))
	}
	found := voices[0]
	want := map[string][]string{
		"StartJump":      {filepath.Join(alice, "StartJump.wav")},
		"IsInDanger.Set": {filepath.Join(alice, "alternates", "another.mp3"), filepath.Join(alice, "odd.wav")},
	}
	for id, clips := range want {
		if got, _ := found.Lookup(cue.ID(id)); !reflect.DeepEqual(got, takesOf(clips...)) {
			t.Errorf("%s takes = %v, want %v", id, got, clips)
		}
	}
	if found.Takes != 3 {
		t.Errorf("Alice has %d takes, want 3 with the one declared twice counted once", found.Takes)
	}
	if voices[1].Name != "Dana" || voices[1].Takes != 1 {
		t.Errorf("got %s with %d takes, want Dana made a voice by her manifest alone", voices[1].Name, voices[1].Takes)
	}
}

// FR-210: an entry that cannot be used is passed over and named, costing nothing but itself.
func TestAManifestEntryThatCannotBeUsedIsPassedOverAlone(t *testing.T) {
	root := t.TempDir()
	alice := filepath.Join(root, "Alice")
	elsewhere := filepath.ToSlash(filepath.Join(root, "outside.wav"))
	audiotest.WriteTake(t, filepath.Join(alice, "odd.wav"))
	audiotest.WriteTake(t, filepath.Join(root, "outside.wav"))
	audiotest.WriteFile(t, filepath.Join(alice, "notes.txt"), audiotest.NotARecording)
	audiotest.WriteFile(t, filepath.Join(alice, "broken.wav"), audiotest.NotARecording)
	writeManifest(t, alice,
		"[takes]",
		`"NoSuchCue" = ["odd.wav"]`,
		`"StartJump" = ["odd.wav", "../outside.wav", "`+elsewhere+`", "notes.txt", "broken.wav", "gone.wav"]`,
	)

	voices, report := scanned(t, root, journalTable(t, "StartJump"))

	alone := only(t, voices)
	if got, _ := alone.Lookup("StartJump"); !reflect.DeepEqual(got, takesOf(filepath.Join(alice, "odd.wav"))) {
		t.Errorf("takes = %v, want odd.wav alone", got)
	}
	mentions := []string{`"NoSuchCue" is no cue's id`, `"../outside.wav" is not inside`, `"` + elsewhere + `" is not inside`, `"notes.txt" is not a recording`}
	if len(report.Manifest) != len(mentions) {
		t.Fatalf("manifest reasons = %+v, want %d", report.Manifest, len(mentions))
	}
	for index, mention := range mentions {
		reason := report.Manifest[index]
		if reason.Path != filepath.Join("Alice", ManifestFile) || !strings.Contains(reason.Why, mention) {
			t.Errorf("reason %d = %+v, want Alice's manifest saying %s", index, reason, mention)
		}
	}
	wantUndecodable := []string{filepath.Join("Alice", "broken.wav"), filepath.Join("Alice", "gone.wav")}
	if got := paths(report.Undecodable); !reflect.DeepEqual(got, wantUndecodable) {
		t.Errorf("undecodable = %v, want %v", got, wantUndecodable)
	}
	// broken.wav is still named for no cue, since nothing reached it.
	if got := paths(report.Unmatched); !reflect.DeepEqual(got, []string{filepath.Join("Alice", "broken.wav")}) {
		t.Errorf("unmatched = %v, want broken.wav alone", got)
	}
}

// FR-211: a manifest that cannot be read or used is set aside whole with its takes; the voice is
// found by its names alone with the reason in the report.
func TestAManifestThatCannotBeUsedFallsBackToTheConvention(t *testing.T) {
	cases := []struct {
		name    string
		write   func(t *testing.T, dir string)
		mention string
	}{
		{"an unknown key", func(t *testing.T, dir string) {
			writeManifest(t, dir, `nmae = "Alice Hart"`, "[takes]", `"Docked" = ["odd.wav"]`)
		}, "nmae"},
		{"text that is not TOML", func(t *testing.T, dir string) {
			writeManifest(t, dir, `name = "Alice Hart"`, "[takes", `"Docked" = ["odd.wav"]`)
		}, "cannot be used"},
		{"a value of the wrong type", func(t *testing.T, dir string) {
			writeManifest(t, dir, "name = 3", "[takes]", `"Docked" = ["odd.wav"]`)
		}, "cannot be used"},
		{"a manifest that cannot be read", func(t *testing.T, dir string) {
			if err := os.MkdirAll(filepath.Join(dir, ManifestFile), 0o755); err != nil {
				t.Fatal(err)
			}
		}, "cannot be read"},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			root := t.TempDir()
			alice := filepath.Join(root, "Alice")
			audiotest.WriteTake(t, filepath.Join(alice, "StartJump.wav"))
			audiotest.WriteTake(t, filepath.Join(alice, "odd.wav"))
			each.write(t, alice)

			voices, report := scanned(t, root, journalTable(t, "StartJump", "Docked"))

			found := only(t, voices)
			if found.Display() != "Alice" || found.Takes != 1 || found.Cues() != 1 {
				t.Errorf("got %s with %d takes over %d cues, want Alice by her names alone",
					found.Display(), found.Takes, found.Cues())
			}
			if len(report.Manifest) != 1 || report.Manifest[0].Path != filepath.Join("Alice", ManifestFile) ||
				!strings.Contains(report.Manifest[0].Why, each.mention) {
				t.Errorf("manifest reasons = %+v, want Alice's manifest saying %s", report.Manifest, each.mention)
			}
		})
	}
}

// FR-210: voices are listed by the name they are shown by, ignoring case, so the list a reader
// sees is in order; two shown alike keep their directories' order.
func TestVoicesAreListedByTheNameTheyAreShownBy(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"a-zed", "b-amy", "c-twin", "d-twin"} {
		audiotest.WriteTake(t, filepath.Join(root, dir, "StartJump.wav"))
	}
	writeManifest(t, filepath.Join(root, "a-zed"), `name = "Zed"`)
	writeManifest(t, filepath.Join(root, "b-amy"), `name = "amy"`)
	writeManifest(t, filepath.Join(root, "c-twin"), `name = "Twin"`)
	writeManifest(t, filepath.Join(root, "d-twin"), `name = "Twin"`)

	voices, _ := scanned(t, root, journalTable(t, "StartJump"))

	got := make([]string, 0, len(voices))
	for _, voice := range voices {
		got = append(got, voice.Name)
	}
	if want := []string{"b-amy", "c-twin", "d-twin", "a-zed"}; !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}
