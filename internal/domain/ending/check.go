package ending

import (
	"errors"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/measured"
	"github.com/oernster/bridge-talk/internal/domain/script"
)

// ErrStale is returned for each way the endings no longer match the script, the voices or the files
// the lines are made from (FR-557).
var ErrStale = errors.New("stale endings")

// Check returns every way the book is stale against the shipped script, the voices given and the
// digests the list gives the model file and each voice's style file, each naming what is stale
// (FR-557). The model comes first, then each voice in the order given: its style file, its lines
// ending on a nasal in the script's order, then its endings for lines that no longer end on one.
func Check(book Book, voices []machinevoice.Voice, voiced script.Voiced, model string, styles map[string]string) []error {
	return measured.Check(kind, book.Book, voices, voiced, voiced.EndingOnNasal, model, styles)
}
