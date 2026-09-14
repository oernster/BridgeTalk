package pause_test

// FR-551 and FR-553: the digest that ties a pause to the samples its break was found in.

import (
	"math"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/pause"
)

// A digest is the first DigestLength hex characters of the SHA-256 over each sample's IEEE 754
// bits, little-endian, in order. The digests wanted were worked out apart from this code with
// Python's hashlib over struct.pack("<f", ...), the way the pauses tool packs its samples.
func TestADigestIsTheSha256OfEachSamplesLittleEndianBitsCutShort(t *testing.T) {
	t.Parallel()
	for _, each := range []struct {
		samples []float32
		want    string
	}{
		{nil, "e3b0c44298fc1c14"},
		{[]float32{1, -0.5}, "e5e0ce39fac89cd9"},
		{[]float32{-0.5, 1}, "ef150ed06e83e087"},
	} {
		if got := pause.Digest(each.samples); got != each.want || len(got) != pause.DigestLength {
			t.Errorf("Digest(%v) = %q, want %q", each.samples, got, each.want)
		}
	}
}

// Samples that sound the same while written with other bits have another digest: zero and
// negative zero differ, so a line made with either is told from the other.
func TestSamplesWrittenWithOtherBitsHaveAnotherDigest(t *testing.T) {
	t.Parallel()
	negativeZero := float32(math.Copysign(0, -1))
	if pause.Digest([]float32{0}) == pause.Digest([]float32{negativeZero}) {
		t.Error("zero and negative zero share a digest")
	}
}
