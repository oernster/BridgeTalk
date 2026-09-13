package library

// The scan, over real temporary directories wherever a real disk can hold the case.
//
// Every rule here is a requirement in REQUIREMENTS.md section 3.4 and each test names
// the one it holds. The one test that does not touch a disk is the case variant merge:
// Windows refuses to create two directories differing only in case, so that tree is
// held in memory and handed to the scan through its lister.

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// writeTake creates a file of one byte. The scan reads names, never content.
func writeTake(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte{0}, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// newCue builds one valid cue from a given source.
func newCue(t *testing.T, id string, source string) cue.Cue {
	t.Helper()
	built, err := cue.New(cue.Definition{ID: id, Source: source, Event: "Anything"})
	if err != nil {
		t.Fatalf("building cue %q: %v", id, err)
	}
	return built
}

// journalTable builds a table of journal cues over the given ids.
func journalTable(t *testing.T, ids ...string) cue.Table {
	t.Helper()
	built := make([]cue.Cue, 0, len(ids))
	for _, id := range ids {
		built = append(built, newCue(t, id, "journal"))
	}
	return cue.NewTable(built)
}

// scanned runs a scan that is expected to succeed.
func scanned(t *testing.T, root string, table cue.Table) ([]Voice, Report) {
	t.Helper()
	voices, report, err := Scan(root, table)
	if err != nil {
		t.Fatalf("scanning %s: %v", root, err)
	}
	return voices, report
}

// only returns the single voice a scan found, failing on any other count.
func only(t *testing.T, voices []Voice) Voice {
	t.Helper()
	if len(voices) != 1 {
		t.Fatalf("found %d voices %v, want exactly one", len(voices), voices)
	}
	return voices[0]
}

// paths lists the paths a report names, in the order it names them.
func paths(reasons []Reason) []string {
	out := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		out = append(out, reason.Path)
	}
	return out
}

// FR-205: a directory named for a cue holds that cue's takes, one per recognised audio
// file directly inside it. Anything else inside it is neither a take nor reported.
func TestAFolderNamedForACueHoldsThatCuesTakes(t *testing.T) {
	root := t.TempDir()
	cueDir := filepath.Join(root, "Alice", "StartJump")
	writeTake(t, filepath.Join(cueDir, "b.WAV"))
	writeTake(t, filepath.Join(cueDir, "a.wav"))
	writeTake(t, filepath.Join(cueDir, "notes.txt"))
	writeTake(t, filepath.Join(cueDir, "older", "c.wav"))

	voices, report := scanned(t, root, journalTable(t, "StartJump", "DockingGranted"))
	alice := only(t, voices)

	want := []string{filepath.Join(cueDir, "a.wav"), filepath.Join(cueDir, "b.WAV")}
	if got, ok := alice.Lookup("StartJump"); !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("takes = %v, %v; want %v", got, ok, want)
	}
	if alice.Takes != 2 || alice.Cues() != 1 {
		t.Errorf("Takes = %d, Cues = %d; want 2 and 1", alice.Takes, alice.Cues())
	}
	if alice.Name != "Alice" || alice.Root != filepath.Join(root, "Alice") {
		t.Errorf("Name = %q, Root = %q", alice.Name, alice.Root)
	}
	if _, ok := alice.Lookup("DockingGranted"); ok {
		t.Error("a cue nothing was recorded for answered a lookup")
	}
	if report.Any() {
		t.Errorf("report = %+v, want nothing passed over", report)
	}
}

// FR-206: a file named for a cue is a take for it, optionally followed by a dot and
// digits so one take can be told from another.
func TestAFileNamedForACueIsATakeWithDigitsTellingTakesApart(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"DockingGranted.wav", "DockingGranted.2.wav", "DockingGranted.10.MP3"} {
		writeTake(t, filepath.Join(root, "Bob", name))
	}

	voices, report := scanned(t, root, journalTable(t, "DockingGranted"))
	bob := only(t, voices)

	if got, _ := bob.Lookup("DockingGranted"); len(got) != 3 {
		t.Fatalf("takes = %v, want all three files", got)
	}
	if report.Any() {
		t.Errorf("report = %+v, want nothing passed over", report)
	}
}

// FR-206 again, from the other side: only a segment made wholly of digits marks another
// take. Letters, a mixture or nothing at all after the dot make a name that is no cue.
func TestOnlyASegmentOfDigitsMarksAnotherTake(t *testing.T) {
	root := t.TempDir()
	unmatched := []string{"DockingGranted..wav", "DockingGranted.2x.wav", "DockingGranted.b.wav"}
	for _, name := range append([]string{"DockingGranted.wav"}, unmatched...) {
		writeTake(t, filepath.Join(root, "Bob", name))
	}

	voices, report := scanned(t, root, journalTable(t, "DockingGranted"))

	if takes := only(t, voices).Takes; takes != 1 {
		t.Errorf("Takes = %d, want the one file named exactly", takes)
	}
	want := make([]string, 0, len(unmatched))
	for _, name := range unmatched {
		want = append(want, filepath.Join("Bob", name))
	}
	if got := paths(report.Unmatched); !reflect.DeepEqual(got, want) {
		t.Errorf("unmatched = %v, want %v", got, want)
	}
}

// FR-207: a name matches a cue id by case insensitive equality and by nothing else. No
// punctuation is normalised; no near miss counts.
func TestMatchingIsExactApartFromCase(t *testing.T) {
	root := t.TempDir()
	carol := filepath.Join(root, "Carol")
	writeTake(t, filepath.Join(carol, "startjump", "a.wav"))
	writeTake(t, filepath.Join(carol, "dockingGRANTED.ogg"))
	nearMisses := []string{
		"Docking-Granted.wav",
		"Start Jump",
		"StartJum",
		"StartJump.extra",
		"Start_Jump",
	}
	for _, name := range nearMisses {
		if filepath.Ext(name) == ".wav" {
			writeTake(t, filepath.Join(carol, name))
			continue
		}
		writeTake(t, filepath.Join(carol, name, "take.wav"))
	}

	voices, report := scanned(t, root, journalTable(t, "StartJump", "DockingGranted"))
	found := only(t, voices)

	if _, ok := found.Lookup("StartJump"); !ok {
		t.Error("a folder differing from its cue id only in case was not matched")
	}
	if _, ok := found.Lookup("DockingGranted"); !ok {
		t.Error("a file differing from its cue id only in case was not matched")
	}
	if found.Takes != 2 {
		t.Errorf("Takes = %d, want only the two exact matches", found.Takes)
	}
	if got := len(report.Unmatched); got != len(nearMisses) {
		t.Errorf("unmatched = %v, want every near miss reported", paths(report.Unmatched))
	}
}

// memoryLister lists a tree held in memory, so a scan can be shown a directory that no
// Windows disk will hold.
func memoryLister(tree fstest.MapFS) lister {
	return func(dir string) ([]os.DirEntry, error) {
		return fs.ReadDir(tree, filepath.ToSlash(dir))
	}
}

// FR-218: two directories naming the same cue in different cases are one set of takes,
// with the duplication reported. Linux holds such a pair; Windows refuses the second.
func TestDirectoriesDifferingOnlyInCaseMergeTheirTakes(t *testing.T) {
	tree := fstest.MapFS{
		"lib/Dana/DockingGranted/a.wav": {},
		"lib/Dana/DOCKINGGRANTED/b.wav": {},
	}

	voices, report, err := scan("lib", journalTable(t, "DockingGranted"), memoryLister(tree))
	if err != nil {
		t.Fatalf("scanning the tree: %v", err)
	}

	dana := only(t, voices)
	if got, _ := dana.Lookup("DockingGranted"); len(got) != 2 {
		t.Fatalf("takes = %v, want the takes of both directories", got)
	}
	want := []string{filepath.Join("Dana", "DockingGranted")}
	if got := paths(report.Duplicated); !reflect.DeepEqual(got, want) {
		t.Fatalf("duplicated = %v, want %v", got, want)
	}
	if !report.Any() {
		t.Error("a report naming a duplication says nothing was passed over")
	}
}

// A cue directory that cannot be listed contributes nothing rather than failing the
// scan, so one unreadable folder never costs a voice everything else it holds.
func TestACueFolderThatCannotBeListedContributesNothing(t *testing.T) {
	tree := fstest.MapFS{
		"lib/Gail/DockingGranted/a.wav": {},
		"lib/Gail/StartJump/b.wav":      {},
	}
	refused := filepath.Join("lib", "Gail", "DockingGranted")
	read := func(dir string) ([]os.DirEntry, error) {
		if dir == refused {
			return nil, errors.New("access is denied")
		}
		return memoryLister(tree)(dir)
	}

	voices, _, err := scan("lib", journalTable(t, "DockingGranted", "StartJump"), read)
	if err != nil {
		t.Fatalf("scanning the tree: %v", err)
	}

	gail := only(t, voices)
	if _, ok := gail.Lookup("DockingGranted"); ok {
		t.Error("a folder that could not be listed produced takes")
	}
	if gail.Takes != 1 {
		t.Errorf("Takes = %d, want the one readable take", gail.Takes)
	}
}

// FR-209: a directory yielding no take is not a voice. That includes one holding a
// folder named for a cue with no audio inside it.
func TestADirectoryResolvingNothingIsReportedRatherThanOffered(t *testing.T) {
	root := t.TempDir()
	writeTake(t, filepath.Join(root, "Alice", "DockingGranted.wav"))
	writeTake(t, filepath.Join(root, "Bystander", "holiday snap.mp3"))
	writeTake(t, filepath.Join(root, "Eve", "DockingGranted", "readme.txt"))

	voices, report := scanned(t, root, journalTable(t, "DockingGranted"))

	if only(t, voices).Name != "Alice" {
		t.Fatalf("voices = %v, want Alice alone", voices)
	}
	if got, want := paths(report.Empty), []string{"Bystander", "Eve"}; !reflect.DeepEqual(got, want) {
		t.Errorf("empty = %v, want %v", got, want)
	}
}

// FR-208 plus FR-203: a subdirectory or audio file matching no cue is reported with the
// path it was found at. A file in no recognised format is ignored without a word.
func TestNamesMatchingNoCueAreReportedWhereTheyWereFound(t *testing.T) {
	root := t.TempDir()
	frank := filepath.Join(root, "Frank")
	writeTake(t, filepath.Join(frank, "DockingGranted.wav"))
	writeTake(t, filepath.Join(frank, "not a cue", "x.wav"))
	writeTake(t, filepath.Join(frank, "typo.wav"))
	writeTake(t, filepath.Join(frank, "cover.jpg"))

	_, report := scanned(t, root, journalTable(t, "DockingGranted"))

	want := []string{filepath.Join("Frank", "not a cue"), filepath.Join("Frank", "typo.wav")}
	if got := paths(report.Unmatched); !reflect.DeepEqual(got, want) {
		t.Errorf("unmatched = %v, want %v", got, want)
	}
}

// FR-213: a tree named in space separated prose resolves nothing, however closely its
// words follow the cue ids, so no directory is offered as a voice by accident.
func TestATreeNamedInProseResolvesNothing(t *testing.T) {
	root := t.TempDir()
	narrator := filepath.Join(root, "Narrator")
	writeTake(t, filepath.Join(narrator, "Carrier Jump Request", "take.wav"))
	writeTake(t, filepath.Join(narrator, "Reservoir Replenished.wav"))
	writeTake(t, filepath.Join(narrator, "Sorted By Hand", "Colonisation Contribution", "one.mp3"))

	voices, report := scanned(t, root, journalTable(t, "CarrierJumpRequest", "ReservoirReplenished", "ColonisationContribution"))

	if len(voices) != 0 {
		t.Fatalf("voices = %v, want none", voices)
	}
	if got, want := paths(report.Empty), []string{"Narrator"}; !reflect.DeepEqual(got, want) {
		t.Errorf("empty = %v, want %v", got, want)
	}
}

// Voices are listed by name ignoring case, so a lower case directory does not sink to
// the bottom of the list.
func TestVoicesAreListedByNameIgnoringCase(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"zed", "Alice", "bob"} {
		writeTake(t, filepath.Join(root, name, "DockingGranted.wav"))
	}
	writeTake(t, filepath.Join(root, "stray.wav"))

	voices, _ := scanned(t, root, journalTable(t, "DockingGranted"))

	got := make([]string, 0, len(voices))
	for _, voice := range voices {
		got = append(got, voice.Name)
	}
	if want := []string{"Alice", "bob", "zed"}; !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

// FR-202 depends on this: a root that cannot be read is an error, never an empty list.
func TestARootThatCannotBeReadIsAnError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")

	voices, _, err := Scan(missing, journalTable(t, "DockingGranted"))

	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want one saying the root does not exist", err)
	}
	if voices != nil {
		t.Errorf("voices = %v, want none alongside the error", voices)
	}
}

// ScanVoice indexes a directory without a library root around it. One that cannot be
// read is a voice with nothing in it, named for the directory asked about.
func TestScanVoiceIndexesOneDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Hana")
	writeTake(t, filepath.Join(dir, "DockingGranted.flac"))
	table := journalTable(t, "DockingGranted")

	if hana, _ := ScanVoice(dir, table); hana.Takes != 1 || hana.Name != "Hana" {
		t.Errorf("Takes = %d, Name = %q; want 1 and Hana", hana.Takes, hana.Name)
	}

	absent, report := ScanVoice(filepath.Join(dir, "absent"), table)
	if absent.Takes != 0 || absent.Name != "absent" || report.Any() {
		t.Errorf("got %+v and %+v, want an empty voice named absent", absent, report)
	}
}
