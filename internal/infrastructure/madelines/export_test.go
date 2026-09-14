package madelines

// What the tests need to name without the store handing it to every caller.
const (
	// BlockSize is how many samples one FLAC frame holds.
	BlockSize = blockSize
	// ShortestPredicted is the fewest samples a frame is predicted over; a shorter frame is
	// written verbatim.
	ShortestPredicted = shortestPredicted
	// PartSuffix follows a made line's path while it is being written (FR-517).
	PartSuffix = partSuffix
)
