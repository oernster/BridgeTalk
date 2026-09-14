package modelfiles

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/oernster/bridge-talk/internal/refusal"
)

// Check says what in dir differs from the list, downloading nothing (FR-538). It answers with the
// names of the missing files in list order, with an error naming each file that differs or cannot be
// read.
func Check(dir string, files []File) ([]string, error) {
	var missing []string
	var problems []error
	for _, file := range files {
		err := verify(filepath.Join(dir, file.Name), file)
		switch {
		case err == nil:
		case errors.Is(err, fs.ErrNotExist):
			missing = append(missing, file.Name)
		default:
			problems = append(problems, err)
		}
	}
	return missing, errors.Join(problems...)
}

// Verify answers nil where dir holds every listed file as listed, downloading nothing (FR-538).
// Otherwise it answers with one error naming the missing files, then each file that differs or
// cannot be read, so every tool that checks the folder says so in the same words.
func Verify(dir string, files []File) error {
	missing, err := Check(dir, files)
	if len(missing) > 0 {
		err = errors.Join(fmt.Errorf("%s is missing %s", dir, strings.Join(missing, ", ")), err)
	}
	return err
}

// verify answers nil where the file at path has the listed size and SHA-256. A file that is not
// there answers with an error errors.Is reports as fs.ErrNotExist.
func verify(path string, file File) error {
	opened, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, refusal.Reason(err))
	}
	defer opened.Close()
	digest := sha256.New()
	size, err := io.Copy(digest, opened)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, refusal.Reason(err))
	}
	return matches(path, file, size, digest)
}

// matches answers nil where size and digest are the listed ones; otherwise it refuses the file named
// by what, saying what differs.
func matches(what string, file File, size int64, digest hash.Hash) error {
	if size != file.Size {
		return fmt.Errorf("%s is %d bytes where the list gives %d: %w", what, size, file.Size, ErrDiffers)
	}
	if sum := hex.EncodeToString(digest.Sum(nil)); sum != file.SHA256 {
		return fmt.Errorf("%s has SHA-256 %s where the list gives %s: %w", what, sum, file.SHA256, ErrDiffers)
	}
	return nil
}
