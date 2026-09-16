package modelfiles_test

// Each platform's own ONNX Runtime: the list answers a platform its files alone; Linux's runtime
// is taken out of a gzip over tar release archive.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles/modelfilestest"
)

// names lists the names of files in order.
func names(files []modelfiles.File) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		out = append(out, file.Name)
	}
	return out
}

// Windows is made from its own runtime and Linux from its own. Neither fetches the other's; both
// fetch every file that names no platform.
func TestEachPlatformIsListedItsOwnRuntime(t *testing.T) {
	files, err := modelfiles.Parse([]byte(`
[sources]
here = "https://example.test/"

[[files]]
name = "model.onnx"
source = "here"
path = "model.onnx"
size = 1
sha256 = "0000000000000000000000000000000000000000000000000000000000000000"

[[files]]
name = "onnxruntime.dll"
platform = "windows"
source = "here"
path = "win.zip"
size = 1
sha256 = "0000000000000000000000000000000000000000000000000000000000000000"

[[files]]
name = "libonnxruntime.so"
platform = "linux"
source = "here"
path = "linux.tgz"
size = 1
sha256 = "0000000000000000000000000000000000000000000000000000000000000000"
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for platform, want := range map[string][]string{
		"windows": {"model.onnx", "onnxruntime.dll"},
		"linux":   {"model.onnx", "libonnxruntime.so"},
	} {
		if got := names(modelfiles.For(files, platform)); !slices.Equal(got, want) {
			t.Errorf("%s is listed %v, want %v", platform, got, want)
		}
	}
}

// The shipped list gives Linux a runtime of its own, so a Linux build has something to load.
func TestTheShippedListGivesLinuxItsRuntime(t *testing.T) {
	files, err := modelfiles.Parse(modelfiles.EmbeddedList())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := names(modelfiles.For(files, "linux")); !slices.Contains(got, "libonnxruntime.so") || slices.Contains(got, "onnxruntime.dll") {
		t.Errorf("Linux is listed %v, want its own runtime and not Windows's", got)
	}
}

// tarball packs members into a gzip over tar archive, each a regular file, with a link beside them
// named for the first so a link can never stand in for the file.
func tarball(t *testing.T, members map[string][]byte, link string) []byte {
	t.Helper()
	var packed bytes.Buffer
	zipped := gzip.NewWriter(&packed)
	writer := tar.NewWriter(zipped)
	if link != "" {
		if err := writer.WriteHeader(&tar.Header{Name: link, Typeflag: tar.TypeSymlink, Linkname: "elsewhere"}); err != nil {
			t.Fatalf("adding the link: %v", err)
		}
	}
	for path, body := range members {
		if err := writer.WriteHeader(&tar.Header{Name: path, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatalf("adding %s: %v", path, err)
		}
		if _, err := writer.Write(body); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing the tar: %v", err)
	}
	if err := zipped.Close(); err != nil {
		t.Fatalf("closing the gzip: %v", err)
	}
	return packed.Bytes()
}

// Linux's runtime is taken from its listed path inside a .tgz, a link of that name passed over; a
// path the archive lacks and a download that is no archive are each refused.
func TestAFileIsTakenFromInsideAGzippedTar(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	dir := t.TempDir()
	runtime := []byte("a linux runtime")
	inside := "release/lib/libonnxruntime.so.1"
	address := server.Offer("/release.tgz", tarball(t, map[string][]byte{inside: runtime}, inside))
	wanted := modelfiles.File{
		Name: "libonnxruntime.so", Address: address, Inside: inside,
		Size: int64(len(runtime)), SHA256: modelfilestest.Digest(runtime),
	}
	absent := wanted
	absent.Name, absent.Inside = "other.so", "release/lib/other.so"
	broken := wanted
	broken.Name, broken.Address = "broken.so", server.Offer("/broken.tgz", []byte("not an archive"))
	cut := wanted
	cut.Name, cut.Address = "cut.so", server.Offer("/cut.tgz", gzipped(t, []byte("too short to be a tar header")))

	err := modelfiles.Fetch(context.Background(), server.Client(), dir,
		[]modelfiles.File{absent, broken, cut, wanted}, io.Discard)

	if !errors.Is(err, modelfiles.ErrNotInArchive) || !errors.Is(err, modelfiles.ErrDownload) {
		t.Fatalf("Fetch = %v, want both %v and %v", err, modelfiles.ErrNotInArchive, modelfiles.ErrDownload)
	}
	holds(t, dir, map[string][]byte{"libonnxruntime.so": runtime})
}

// gzipped packs bytes as gzip alone, so a tar reader finds no header in it.
func gzipped(t *testing.T, body []byte) []byte {
	t.Helper()
	var packed bytes.Buffer
	zipped := gzip.NewWriter(&packed)
	if _, err := zipped.Write(body); err != nil {
		t.Fatalf("writing: %v", err)
	}
	if err := zipped.Close(); err != nil {
		t.Fatalf("closing: %v", err)
	}
	return packed.Bytes()
}
