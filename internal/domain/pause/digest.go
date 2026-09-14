// Package pause holds the pause before a final commander in a machine voice's joined lines: the
// digest tying a pause to the samples it was found in, the silence it inserts, the rule that finds a
// break doubtful and the check that the pauses shipped are not stale (FR-551 to FR-554).
package pause

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
)

// DigestLength is how many hex characters of the SHA-256 a digest keeps (FR-551, FR-553).
const DigestLength = 16

// sampleBytes is how many bytes one sample's IEEE 754 bits take.
var sampleBytes = binary.Size(float32(0))

// Digest is a short digest of a line's samples: the first DigestLength hex characters of the
// SHA-256 over each sample's IEEE 754 bits, little-endian, in order (FR-551, FR-553). The pauses
// tool digests the samples it finds a break in the same way, so a line made with other samples is
// told apart from those measured.
func Digest(samples []float32) string {
	written := make([]byte, 0, len(samples)*sampleBytes)
	for _, sample := range samples {
		written = binary.LittleEndian.AppendUint32(written, math.Float32bits(sample))
	}
	sum := sha256.Sum256(written)
	return hex.EncodeToString(sum[:])[:DigestLength]
}
