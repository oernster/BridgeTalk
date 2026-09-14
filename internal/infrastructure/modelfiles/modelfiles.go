// Package modelfiles keeps the files a machine voice is made from on the machine that builds the
// setup program (FR-535 to FR-538). One embedded list names each file with the address it is
// downloaded from, its size and its SHA-256. Fetch fills a folder from the list; Check says what in
// the folder is missing or different, downloading nothing.
//
// The folder is models at the repository root, found by walking up to go.mod rather than named by an
// environment variable (Oliver, 2026-09-14). It is not committed.
//
// The SHA-256 here is not the digest voicefiles keys made lines by, though both are SHA-256 today.
// This one is whatever Hugging Face and GitHub publish for a file; that one is the application's own
// choice, free to change without touching the list.
package modelfiles

import (
	"errors"
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/infrastructure/reporoot"
)

const (
	// Folder names the folder at the repository root the files are kept in.
	Folder = "models"
	// TokenizerFile is the model's tokenizer file. Setup does not install it; the test that holds the
	// speech sound symbols to the model reads it.
	TokenizerFile = "tokenizer.json"
)

// File is one listed file.
type File struct {
	// Name is what the file is kept under in the folder, which is the name the application reads.
	Name string
	// Address is where the file is downloaded from.
	Address string
	// Inside is the file's path in the archive Address answers with; empty where Address answers
	// with the file itself.
	Inside string
	// Size is the file's length in bytes.
	Size int64
	// SHA256 is the file's published SHA-256, written as lowercase hexadecimal.
	SHA256 string
}

var (
	// ErrMisshapenEntry means a list entry could fetch the wrong thing or put it in the wrong place.
	ErrMisshapenEntry = errors.New("the entry cannot be trusted")
	// ErrDiffers means a file's size or SHA-256 is not the listed one.
	ErrDiffers = errors.New("differs from the list")
	// ErrDownload means an address did not answer with the file.
	ErrDownload = errors.New("the download failed")
	// ErrNotInArchive means the archive an address answered with does not hold the listed path.
	ErrNotInArchive = errors.New("not in the archive")
)

// Dir answers with the folder at the root of the repository holding from.
func Dir(from string) (string, error) {
	root, err := reporoot.Find(from)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, Folder), nil
}
