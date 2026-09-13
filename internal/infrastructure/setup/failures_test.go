package setup

// The failures extraction, copying and removal report, each forced from a real input
// rather than assumed from a read of the code. A zip entry can name a compression
// method nothing can read or carry a checksum that does not match its bytes; a
// directory opens but cannot be read as a file; a path ending in a dot is one that
// RemoveAll refuses. Each test also pins the message, so it proves the branch it is
// named for rather than any failure at all.

import (
	"archive/zip"
	"bytes"
	"errors"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unknownMethod is a compression method archive/zip has no reader registered for.
const unknownMethod uint16 = 99

// rawZipOf builds an archive holding one entry written exactly as given: nothing is
// compressed and no checksum is computed, so a test can hand extraction an entry the
// writer would never produce on its own.
func rawZipOf(t *testing.T, header *zip.FileHeader, content []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.CreateRaw(header)
	if err != nil {
		t.Fatalf("creating raw entry %q: %v", header.Name, err)
	}
	if _, err := file.Write(content); err != nil {
		t.Fatalf("writing raw entry %q: %v", header.Name, err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing archive: %v", err)
	}
	return buffer.Bytes()
}

// A payload can carry a directory as an entry of its own, not only as the parent of a
// file. It is made as a directory rather than written out as an empty file.
func TestADirectoryEntryInThePayloadIsMadeAsADirectory(t *testing.T) {
	t.Parallel()
	dest := t.TempDir()

	if err := ExtractZip(zipOf(t, map[string]string{"assets/": ""}), dest); err != nil {
		t.Fatalf("extracting: %v", err)
	}
	info, err := os.Stat(filepath.Join(dest, "assets"))
	if err != nil || !info.IsDir() {
		t.Fatalf("assets = %v, %v; want a directory", info, err)
	}
}

// An entry compressed by a method nothing can read stops the extraction with the
// reason, rather than leaving a file of the wrong bytes in the install.
func TestAnEntryInAnUnreadableMethodStopsTheExtraction(t *testing.T) {
	t.Parallel()
	payload := rawZipOf(t, &zip.FileHeader{Name: ExeName, Method: unknownMethod}, []byte("x"))

	err := ExtractZip(payload, t.TempDir())

	if !errors.Is(err, zip.ErrAlgorithm) || !strings.HasPrefix(err.Error(), "open entry ") {
		t.Fatalf("err = %v, want the entry refused as unreadable", err)
	}
}

// An entry whose bytes do not match its checksum is a damaged payload. The damage is
// only known once the bytes have been read, so it is the write that reports it.
func TestAnEntryWhoseChecksumDoesNotMatchStopsTheExtraction(t *testing.T) {
	t.Parallel()
	content := []byte("payload")
	payload := rawZipOf(t, &zip.FileHeader{
		Name:               ExeName,
		Method:             zip.Store,
		CRC32:              crc32.ChecksumIEEE(content) + 1,
		CompressedSize64:   uint64(len(content)),
		UncompressedSize64: uint64(len(content)),
	}, content)

	err := ExtractZip(payload, t.TempDir())

	if !errors.Is(err, zip.ErrChecksum) || !strings.HasPrefix(err.Error(), "write ") {
		t.Fatalf("err = %v, want the damaged entry reported as it was written", err)
	}
}

// A copy whose source opens but cannot be read as a file, as a directory does, reports
// the failure rather than leaving an empty uninstaller behind a success.
func TestACopyWhoseSourceCannotBeReadIsReported(t *testing.T) {
	t.Parallel()
	source := t.TempDir()
	target := filepath.Join(t.TempDir(), "uninstall.exe")

	err := CopyFile(source, target)

	if err == nil || !strings.HasPrefix(err.Error(), "write ") {
		t.Fatalf("err = %v, want the unreadable source reported at the write", err)
	}
}

// A tree that cannot be removed is reported. RemoveAll refuses a path ending in a dot,
// so the failure is forced without depending on permissions a test cannot rely on.
func TestATreeThatCannotBeRemovedIsReported(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	err := RemoveTree(dir + string(os.PathSeparator) + ".")

	if err == nil || !strings.HasPrefix(err.Error(), "remove ") {
		t.Fatalf("err = %v, want the refusal reported", err)
	}
	if _, statErr := os.Stat(dir); statErr != nil {
		t.Errorf("the directory went although the removal failed: %v", statErr)
	}
}
