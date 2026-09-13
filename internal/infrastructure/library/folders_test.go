package library

// Making a voice's folders, over real temporary directories. Each test names the
// requirement in REQUIREMENTS.md section 3.4 it holds.

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// FR-223: one folder per cue id, each of them the folder form, so a take dropped into
// one resolves for its cue on the next scan with no cue id ever typed.
func TestMakingAVoicesFoldersMakesOneForEveryCue(t *testing.T) {
	root := t.TempDir()
	table := journalTable(t, "DockingGranted", "StartJump.JumpType.Hyperspace", "Synthesis.Name.Repair Basic")

	dir, made, err := MakeVoiceFolders(root, "Oliver", table)
	if err != nil {
		t.Fatalf("making folders: %v", err)
	}
	if dir != filepath.Join(root, "Oliver") {
		t.Fatalf("made %q, want the voice under the root", dir)
	}
	if made != table.Len() {
		t.Fatalf("made %d folders, want one for each of %d cues", made, table.Len())
	}
	for _, item := range table.All() {
		info, err := os.Stat(filepath.Join(dir, string(item.ID())))
		if err != nil || !info.IsDir() {
			t.Errorf("no folder for %s: %v", item.ID(), err)
		}
	}

	// Empty folders are no voice yet (FR-209); one take makes it one.
	if voices, _ := scanned(t, root, table); len(voices) != 0 {
		t.Fatalf("a voice of empty folders was offered: %v", voices)
	}
	writeTake(t, filepath.Join(dir, "DockingGranted", "any name at all.wav"))
	voices, _ := scanned(t, root, table)
	if len(voices) != 1 || voices[0].Takes != 1 {
		t.Fatalf("got %v, want Oliver with the one take dropped in", voices)
	}
}

// FR-224: it adds and never replaces. A recording already in a folder and a file
// standing where a folder would go both come through untouched.
func TestMakingFoldersAgainAddsOnlyWhatIsMissing(t *testing.T) {
	root := t.TempDir()
	table := journalTable(t, "Docked", "Undocked", "Liftoff")
	take := filepath.Join(root, "Oliver", "Docked", "a.wav")
	writeTake(t, take)
	standing := filepath.Join(root, "Oliver", "Undocked")
	if err := os.WriteFile(standing, []byte("mine"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", standing, err)
	}

	_, made, err := MakeVoiceFolders(root, "Oliver", table)
	if err != nil {
		t.Fatalf("making folders: %v", err)
	}
	if made != 1 {
		t.Fatalf("made %d folders, want only Liftoff", made)
	}
	if held, err := os.ReadFile(standing); err != nil || string(held) != "mine" {
		t.Fatalf("the file standing where a folder would go was changed: %q, %v", held, err)
	}
	if held, err := os.ReadFile(take); err != nil || len(held) != 1 {
		t.Fatalf("the recording already there was changed: %v", err)
	}

	if _, made, _ = MakeVoiceFolders(root, "Oliver", table); made != 0 {
		t.Fatalf("a second run made %d folders, want none", made)
	}
}

// FR-225: a name that cannot be a folder on every platform is refused with nothing made.
func TestANameThatCannotBeAFolderIsRefused(t *testing.T) {
	cases := []struct {
		name string
		want error
	}{
		{"", ErrNoVoiceName},
		{"   ", ErrNoVoiceName},
		{" Oliver", ErrVoiceName},
		{"Oliver ", ErrVoiceName},
		{"Oliver.", ErrVoiceName},
		{"..", ErrVoiceName},
		{"a/b", ErrVoiceName},
		{`a\b`, ErrVoiceName},
		{"C:voice", ErrVoiceName},
		{"what?", ErrVoiceName},
		{"tab\there", ErrVoiceName},
		{"con", ErrVoiceName},
		{"COM1.voice", ErrVoiceName},
		{"lpt9", ErrVoiceName},
	}
	table := journalTable(t, "Docked")
	for _, each := range cases {
		root := t.TempDir()
		_, _, err := MakeVoiceFolders(root, each.name, table)
		if !errors.Is(err, each.want) {
			t.Errorf("%q: got %v, want %v", each.name, err, each.want)
		}
		if entries, _ := os.ReadDir(root); len(entries) != 0 {
			t.Errorf("%q: made %d entries in the root after refusing", each.name, len(entries))
		}
	}
}

// Names that look unusual but are folders everywhere are taken as they are.
func TestAnOrdinaryNameIsAccepted(t *testing.T) {
	for _, name := range []string{"Oliver", "Grace Hopper", "Zoë", "voice.two", "console"} {
		if err := CheckVoiceName(name); err != nil {
			t.Errorf("%q was refused: %v", name, err)
		}
	}
}

// There has to be somewhere to make them; it has to be a directory.
func TestFoldersNeedARootThatIsADirectory(t *testing.T) {
	table := journalTable(t, "Docked")

	if _, _, err := MakeVoiceFolders("", "Oliver", table); !errors.Is(err, ErrNoRoot) {
		t.Errorf("no root: got %v, want ErrNoRoot", err)
	}
	if _, _, err := MakeVoiceFolders(filepath.Join(t.TempDir(), "gone"), "Oliver", table); err == nil {
		t.Error("a root that is not there was used")
	}
	file := filepath.Join(t.TempDir(), "plain")
	writeTake(t, file)
	if _, _, err := MakeVoiceFolders(file, "Oliver", table); err == nil {
		t.Error("a file was used as the root")
	}
}

// A file already holding the voice's name leaves nowhere to put its folders.
func TestAFileInTheVoicesPlaceIsReported(t *testing.T) {
	root := t.TempDir()
	writeTake(t, filepath.Join(root, "Oliver"))

	if _, _, err := MakeVoiceFolders(root, "Oliver", journalTable(t, "Docked")); err == nil {
		t.Fatal("folders were reported made inside a file")
	}
}

// A refusal partway through says which folder it was and how far it got.
func TestAFolderTheSystemRefusesIsReported(t *testing.T) {
	root := t.TempDir()
	refused := errors.New("the disk said no")
	calls := 0
	failSecond := func(path string, perm fs.FileMode) error {
		calls++
		if calls == 2 {
			return refused
		}
		return os.Mkdir(path, perm)
	}

	_, made, err := makeVoiceFolders(root, "Oliver", journalTable(t, "Docked", "Undocked"), failSecond)
	if !errors.Is(err, refused) {
		t.Fatalf("got %v, want the refusal carried", err)
	}
	if made != 1 {
		t.Fatalf("made %d, want the one made before the refusal", made)
	}
}
