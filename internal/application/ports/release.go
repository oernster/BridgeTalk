package ports

// Asset is one file attached to a published release: its name and the address it is downloaded
// from.
type Asset struct {
	Name    string
	Address string
}

// Release is the project's latest published release (FR-756): its version as tagged, the address
// of its page and the files attached to it.
type Release struct {
	Version string
	Page    string
	Assets  []Asset
}

// ReleaseSource answers the project's latest published release.
//
// Only a release that is published, neither a draft nor a pre-release, is ever answered, so a tag
// pushed while work is under way can never be offered. An error means the release could not be
// read, for whatever reason; the update check treats every one alike, as unreachable.
type ReleaseSource interface {
	Latest() (Release, error)
}
