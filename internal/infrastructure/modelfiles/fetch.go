package modelfiles

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/refusal"
)

const (
	// partSuffix follows a file's name while it is downloaded and checked, so nothing unchecked ever
	// stands under a listed name.
	partSuffix = ".part"
	// archiveSuffix follows a part's name while the archive it is taken from is on disk.
	archiveSuffix = ".archive"
	// folderPerm is how the folder is made: readable by everyone, writable by the owner.
	folderPerm = 0o755
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

// fetchOne downloads a file beside its place, checks it, then renames it into its place. Whatever
// happens, no part is left behind.
func fetchOne(ctx context.Context, client *http.Client, place string, file File) error {
	part := place + partSuffix
	defer os.Remove(part)
	if err := download(ctx, client, file, part); err != nil {
		return err
	}
	if err := verify(part, file); err != nil {
		return fmt.Errorf("downloaded from %s: %w", file.Address, err)
	}
	if err := os.Rename(part, place); err != nil {
		return fmt.Errorf("putting %s in place: %w", place, refusal.Reason(err))
	}
	return nil
}

// download writes the file to part: the address's answer itself, else the listed member of the archive
// it answers with.
func download(ctx context.Context, client *http.Client, file File, part string) error {
	if file.Inside == "" {
		return get(ctx, client, file.Address, part)
	}
	archive := part + archiveSuffix
	defer os.Remove(archive)
	if err := get(ctx, client, file.Address, archive); err != nil {
		return err
	}
	return extract(archive, file, part)
}

// get writes the answer to a request for address to a new file at path.
func get(ctx context.Context, client *http.Client, address, path string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDownload, err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDownload, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %s answered %s", ErrDownload, address, response.Status)
	}
	if err := writeAll(path, response.Body); err != nil {
		return fmt.Errorf("%w from %s: %w", ErrDownload, address, err)
	}
	return nil
}

// extract writes the listed member of the archive at archive to a new file at part.
func extract(archive string, file File, part string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return fmt.Errorf("%w: %s answered with no archive for %s: %w", ErrDownload, file.Address, file.Name, err)
	}
	defer reader.Close()
	member, err := reader.Open(file.Inside)
	if err != nil {
		return fmt.Errorf("%s: %s at %s: %w", file.Name, file.Inside, file.Address, ErrNotInArchive)
	}
	defer member.Close()
	return writeAll(part, member)
}

// writeAll writes everything reader holds to a new file at path.
func writeAll(path string, reader io.Reader) error {
	written, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, refusal.Reason(err))
	}
	_, copyErr := io.Copy(written, reader)
	if err := errors.Join(copyErr, written.Close()); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
