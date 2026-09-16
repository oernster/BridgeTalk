package modelfiles

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/oernster/bridge-talk/internal/infrastructure/wholefile"
	"github.com/oernster/bridge-talk/internal/refusal"
)

const (
	// partSuffix follows a file's name while it is downloaded and checked, so nothing unchecked ever
	// stands under a listed name.
	partSuffix = ".part"
	// archiveSuffix follows a file's name while the archive it is taken from is on disk.
	archiveSuffix = ".archive"
	// folderPerm is how the folder is made: readable by everyone, writable by the owner.
	folderPerm = 0o755
	// filePerm is how each file is written: readable by everyone, writable by the owner.
	filePerm = 0o644
)

// Fetch leaves dir holding every listed file at its listed size and SHA-256 (FR-536). A file that
// already matches is left alone. Any other is downloaded beside its place, checked, then renamed
// into it. One that fails is refused with nothing left under its name while the rest are still
// fetched (FR-537). A line for each file is written to out.
func Fetch(ctx context.Context, client *http.Client, dir string, files []File, out io.Writer) error {
	if err := os.MkdirAll(dir, folderPerm); err != nil {
		return fmt.Errorf("making %s: %w", dir, refusal.Reason(err))
	}
	var problems []error
	for _, file := range files {
		place := filepath.Join(dir, file.Name)
		if verify(place, file) == nil {
			fmt.Fprintf(out, "%s matches the list\n", file.Name)
			continue
		}
		fmt.Fprintf(out, "downloading %s from %s\n", file.Name, file.Address)
		if err := fetchOne(ctx, client, place, file); err != nil {
			problems = append(problems, err)
		}
	}
	return errors.Join(problems...)
}

// fetchOne downloads a file beside its place, checking it as it arrives. Only a file that matches the
// list is put in place.
func fetchOne(ctx context.Context, client *http.Client, place string, file File) error {
	return wholefile.Write(place, partSuffix, filePerm, func(part io.Writer) error {
		digest := sha256.New()
		size, err := download(ctx, client, place, file, io.MultiWriter(part, digest))
		if err != nil {
			return err
		}
		if err := matches(file.Name, file, size, digest); err != nil {
			return fmt.Errorf("downloaded from %s: %w", file.Address, err)
		}
		return nil
	})
}

// download writes the file into into, answering with how many bytes it wrote: the address's answer
// itself, else the listed member of the archive it answers with. The archive stands beside the
// file's place only while the member is taken out of it.
func download(ctx context.Context, client *http.Client, place string, file File, into io.Writer) (int64, error) {
	if file.Inside == "" {
		return get(ctx, client, file.Address, into)
	}
	archive := place + archiveSuffix
	defer os.Remove(archive)
	err := wholefile.Write(archive, partSuffix, filePerm, func(saved io.Writer) error {
		_, err := get(ctx, client, file.Address, saved)
		return err
	})
	if err != nil {
		return 0, err
	}
	return extract(archive, file, into)
}

// get writes the answer to a request for address into into, answering with how many bytes it wrote.
func get(ctx context.Context, client *http.Client, address string, into io.Writer) (int64, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrDownload, err)
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrDownload, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%w: %s answered %s", ErrDownload, address, response.Status)
	}
	written, err := io.Copy(into, response.Body)
	if err != nil {
		return written, fmt.Errorf("%w from %s: %w", ErrDownload, address, err)
	}
	return written, nil
}

// tarSuffix ends the address of a release archive packed as gzip over tar, as Linux's are.
const tarSuffix = ".tgz"

// extract writes the listed member of the archive at archive into into, answering with how many
// bytes it wrote.
func extract(archive string, file File, into io.Writer) (int64, error) {
	if strings.HasSuffix(file.Address, tarSuffix) {
		return extractTar(archive, file, into)
	}
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return 0, fmt.Errorf("%w: %s answered with no archive for %s: %w", ErrDownload, file.Address, file.Name, err)
	}
	defer reader.Close()
	member, err := reader.Open(file.Inside)
	if err != nil {
		return 0, fmt.Errorf("%s: %s at %s: %w", file.Name, file.Inside, file.Address, ErrNotInArchive)
	}
	defer member.Close()
	return io.Copy(into, member)
}

// extractTar writes the listed member of the gzip over tar archive at archive into into. Only a
// regular file answers: a link of the same name is passed over, so what is written is the bytes
// the list's size and SHA-256 describe.
func extractTar(archive string, file File, into io.Writer) (int64, error) {
	opened, err := os.Open(archive)
	if err != nil {
		return 0, fmt.Errorf("%w: %s: %w", ErrDownload, file.Address, err)
	}
	defer opened.Close()
	unzipped, err := gzip.NewReader(opened)
	if err != nil {
		return 0, fmt.Errorf("%w: %s answered with no archive for %s: %w", ErrDownload, file.Address, file.Name, err)
	}
	defer unzipped.Close()
	reader := tar.NewReader(unzipped)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return 0, fmt.Errorf("%s: %s at %s: %w", file.Name, file.Inside, file.Address, ErrNotInArchive)
		}
		if err != nil {
			return 0, fmt.Errorf("%w: %s answered with no archive for %s: %w", ErrDownload, file.Address, file.Name, err)
		}
		if header.Name == file.Inside && header.Typeflag == tar.TypeReg {
			return io.Copy(into, reader)
		}
	}
}
