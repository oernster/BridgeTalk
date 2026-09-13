package main

// FR-237 through the facade: every refusal a pane can show over a file or folder names it
// once, written as the reader would type it, with the reason in plain words and no system
// call. The cases are the ones a probe of every pane found repeating a path.

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// namedOnce fails the test for each way a refusal breaks FR-237.
func namedOnce(t *testing.T, surface string, refused error, path string) {
	t.Helper()
	for _, problem := range refusal.Check(refused, path) {
		t.Errorf("%s %s", surface, problem)
	}
}

// wrongFolders builds the folders a reader can wrongly choose: an empty one, one that does
// not exist, one holding a journal but no status file and a plain file.
func wrongFolders(t *testing.T) (empty, missing, journalOnly, plainFile string) {
	t.Helper()
	base := t.TempDir()
	empty = filepath.Join(base, "Empty Folder")
	journalOnly = filepath.Join(base, "Journal Only")
	for _, dir := range []string{empty, journalOnly} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("making %s: %v", dir, err)
		}
	}
	writeJournal(t, journalOnly)
	plainFile = filepath.Join(base, "plain.txt")
	audiotest.WriteFile(t, plainFile, audiotest.NotARecording)
	return empty, filepath.Join(base, "Missing Folder"), journalOnly, plainFile
}

// Browse on the Journal directory row, answered with each wrong folder.
func TestAJournalDirectoryRefusalNamesItsFolderOnce(t *testing.T) {
	empty, missing, journalOnly, plainFile := wrongFolders(t)
	for _, dir := range []string{empty, missing, journalOnly, plainFile} {
		app, _, _ := fixtureApp(t)
		answering(app, dir, nil)
		_, err := app.ChooseJournalDir()
		namedOnce(t, "Browse on the Journal directory row given "+filepath.Base(dir), err, dir)
	}
}

// Browse on the Recordings row, then Refresh, the Missing takes pane and Make folders over
// a recordings directory that has gone or was never a folder.
func TestARecordingsRefusalNamesItsFolderOnce(t *testing.T) {
	empty, missing, _, plainFile := wrongFolders(t)
	for _, dir := range []string{empty, missing, plainFile} {
		app, _, _ := fixtureApp(t)
		answering(app, dir, nil)
		_, err := app.ChooseLibraryRoot()
		namedOnce(t, "Browse on the Recordings row given "+filepath.Base(dir), err, dir)
	}
	for _, root := range []string{missing, plainFile} {
		app, _, _ := fixtureApp(t)
		app.libraryRoot = root
		voice := filepath.Join(root, "Oliver")
		over := " over " + filepath.Base(root)

		_, err := app.Rescan()
		namedOnce(t, "Refresh"+over, err, root)
		_, err = app.VoiceDirectories()
		namedOnce(t, "the Missing takes voices"+over, err, root)
		_, err = app.Checklist("Oliver")
		namedOnce(t, "the Missing takes list"+over, err, voice)
		err = app.OpenMomentFolder("Oliver", "Docked")
		namedOnce(t, "Open folder"+over, err, voice)
		_, err = app.MakeVoiceFolders("Oliver")
		namedOnce(t, "Make folders"+over, err, root)
	}
}

// A folder the file manager will not open is named once, with the shell's reason alone.
func TestAFolderTheFileManagerWillNotOpenIsNamedOnce(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.reveal = func(dir string) error {
		return &fs.PathError{Op: "CreateProcess", Path: dir, Err: fs.ErrPermission}
	}

	err := app.OpenMomentFolder("Alpha", "Docked")

	namedOnce(t, "Open folder", err, filepath.Join(app.libraryRoot, "Alpha", "Docked"))
}
