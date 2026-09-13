package setup

// FR-237 in the setup program: a refusal over a file or folder names it once, written as
// the reader would type it, with the reason in plain words and no system call.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/refusal"
)

func TestSetupRefusalsNameTheirPathOnce(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	plainFile := filepath.Join(base, "plain.txt")
	if err := os.WriteFile(plainFile, []byte("x"), dirPerm); err != nil {
		t.Fatalf("writing %s: %v", plainFile, err)
	}
	missing := filepath.Join(base, "Missing Setup.exe")
	blocked := filepath.Join(plainFile, "Programs")
	blockedCopy := filepath.Join(blocked, "uninstall.exe")
	undeletable := base + string(os.PathSeparator) + "."
	_, measured := DirSizeKB(missing)

	cases := []struct {
		act     string
		refused error
		path    string
	}{
		{"copying from a file that is not there", CopyFile(missing, filepath.Join(base, "copy.exe")), missing},
		{"copying into a folder that cannot exist", CopyFile(plainFile, blockedCopy), blockedCopy},
		{"extracting into a folder that cannot exist", ExtractZip(zipOf(t, map[string]string{ExeName: "x"}), blocked), blocked},
		{"measuring a folder that is not there", measured, missing},
		{"removing a tree that cannot go", RemoveTree(undeletable), undeletable},
	}
	for _, each := range cases {
		for _, problem := range refusal.Check(each.refused, each.path) {
			t.Errorf("%s %s", each.act, problem)
		}
	}
}
