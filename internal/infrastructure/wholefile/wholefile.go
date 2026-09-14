// Package wholefile puts a file in place whole or not at all. The file is written beside its place,
// then renamed into it, so whoever reads the place finds the old file or the new one and never half of
// either; whatever fails, nothing is left beside it. Made lines (FR-517), the stored settings and the
// model files (FR-537) are written this way, so the rule has one home.
package wholefile

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/oernster/bridge-talk/internal/refusal"
)

// Write fills a file beside path, named path followed by suffix, through write, then renames it into
// path. A file beside path that cannot be made is refused naming it, without calling write. An error
// write answers with comes back as it is; a rename that is refused names path. In every failure the
// file beside path is removed.
func Write(path, suffix string, perm fs.FileMode, write func(io.Writer) error) error {
	part := path + suffix
	file, err := os.OpenFile(part, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("writing %s: %w", part, refusal.Reason(err))
	}
	if err := errors.Join(write(file), file.Close()); err != nil {
		os.Remove(part)
		return err
	}
	if err := os.Rename(part, path); err != nil {
		os.Remove(part)
		return fmt.Errorf("putting %s in place: %w", path, refusal.Reason(err))
	}
	return nil
}
