// Package journal reads Elite Dangerous journal files incrementally.
//
// The byte-offset plus partial-line algorithm is the one proven in the author's
// o7Debrief and EDColonisationAsst. The game appends journal lines one at a time,
// so reading while it is mid-write can yield a final line with no trailing newline.
// The reader returns that trailing partial so the caller can prepend it next pass,
// which is what guarantees no event is lost or counted twice.
package journal

import (
	"bytes"
	"os"
)

// emptyOffset is where a fresh read starts.
const emptyOffset int64 = 0

// newline terminates each complete journal line.
var newline = []byte{'\n'}

// TailResult is the outcome of one incremental read.
type TailResult struct {
	// Lines holds the newly completed, non-empty lines in file order.
	Lines []string
	// Offset is the byte position to resume from next time.
	Offset int64
	// Partial holds trailing bytes not yet terminated by a newline.
	Partial []byte
}

// ReadNewBytes reads bytes appended to path since offset and splits them to lines.
//
// The carried partial from the previous call is prepended before splitting. A file
// now smaller than the recorded offset was rotated or truncated, so the read starts
// again from the beginning. A missing or unreadable file yields no lines and leaves
// the offset where it was, because a journal directory can legitimately be busy.
func ReadNewBytes(path string, offset int64, partial []byte) TailResult {
	start, carried := normaliseForRotation(path, offset, partial)

	handle, err := os.Open(path)
	if err != nil {
		return TailResult{Offset: start, Partial: carried}
	}
	defer func() { _ = handle.Close() }()

	if _, err := handle.Seek(start, 0); err != nil {
		return TailResult{Offset: start, Partial: carried}
	}
	chunk := readAll(handle)

	buffer := append(append([]byte{}, carried...), chunk...)
	complete, remainder := splitLines(buffer)
	return TailResult{
		Lines:   decodeComplete(complete),
		Offset:  start + int64(len(chunk)),
		Partial: remainder,
	}
}

// FileSize returns the current size of a file; zero when it cannot be read.
// Used to start a watcher at the end of the newest journal rather than replaying it.
func FileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return emptyOffset
	}
	return info.Size()
}

// normaliseForRotation resets the offset and partial when the file has shrunk.
func normaliseForRotation(path string, offset int64, partial []byte) (int64, []byte) {
	if FileSize(path) < offset {
		return emptyOffset, nil
	}
	return offset, partial
}

// readAll drains an open handle from its current position.
//
// It reports no error. A journal directory can legitimately be busy; a read that
// stops early yields whatever arrived before it stopped, which is exactly what the
// caller wants: the bytes are kept, the offset advances by what was read and the next
// pass picks up from there. An error return would only ever have been nil.
func readAll(handle *os.File) []byte {
	var out []byte
	buffer := make([]byte, 64*1024)
	for {
		read, err := handle.Read(buffer)
		if read > 0 {
			out = append(out, buffer[:read]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				return out
			}
			if read == 0 {
				return out
			}
		}
		if read < len(buffer) {
			return out
		}
	}
}

// splitLines separates complete lines from an unterminated remainder.
//
// bytes.Split always yields at least one element, even for an empty buffer, so the
// last element is always there to be taken as the remainder.
func splitLines(buffer []byte) ([][]byte, []byte) {
	parts := bytes.Split(buffer, newline)
	return parts[:len(parts)-1], parts[len(parts)-1]
}

// decodeComplete trims each complete line and drops the empty ones.
func decodeComplete(parts [][]byte) []string {
	var lines []string
	for _, part := range parts {
		text := string(bytes.TrimSpace(part))
		if text != "" {
			lines = append(lines, text)
		}
	}
	return lines
}
