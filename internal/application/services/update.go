package services

// The update check (FR-756 to FR-759): the latest published release compared with the running one.

import (
	"strconv"
	"strings"

	"github.com/oernster/bridge-talk/internal/application/ports"
)

// UpdateOutcome says what one update check found. It is named rather than left as a pair of flags
// because the window words each one differently and shows only one of them unasked (FR-756).
type UpdateOutcome string

const (
	// UpdateUnreachable means the release could not be read.
	UpdateUnreachable UpdateOutcome = "unreachable"
	// UpdateUncomparable means the running version is not dotted whole numbers, as a build from
	// source is not, so no release can be said to be newer or not (FR-759).
	UpdateUncomparable UpdateOutcome = "uncomparable"
	// UpdateCurrent means the release is not newer than the running version.
	UpdateCurrent UpdateOutcome = "current"
	// UpdateSkipped means the release is newer and is the one the reader chose to skip (FR-758).
	UpdateSkipped UpdateOutcome = "skipped"
	// UpdateAvailable means the release is newer and is offered (FR-757).
	UpdateAvailable UpdateOutcome = "available"
)

// UpdateStatus is the answer to one update check. Latest is the release's version with any leading
// tag prefix removed, empty where it could not be read; Address is what Download hands the browser,
// empty unless the release was read.
type UpdateStatus struct {
	Outcome UpdateOutcome
	Running string
	Latest  string
	Address string
}

// tagPrefix is the letter a release's tag may carry ahead of its version, as in v1.5.0.
const tagPrefix = "v"

// versionSeparator divides a version's whole numbers.
const versionSeparator = "."

// platformFiles names, for each platform the application ships on, the ending of the file a
// release carries for it (FR-757). A platform not named has no file and is sent to the release's
// page.
var platformFiles = map[string]string{
	"windows": ".exe",
	"linux":   ".flatpak",
}

// UpdateService compares the running version with the latest published release.
type UpdateService struct {
	source  ports.ReleaseSource
	running string
	ending  string
}

// NewUpdateService builds the check over its source, the running version and the platform it runs
// on, named as Go names it.
func NewUpdateService(source ports.ReleaseSource, running, platform string) *UpdateService {
	return &UpdateService{source: source, running: running, ending: platformFiles[platform]}
}

// Check reads the latest release and says what it found. A release equal to skipped is reported as
// skipped rather than offered; an empty skipped offers every newer release, which is what a check
// asked for from Help wants (FR-759).
func (s *UpdateService) Check(skipped string) UpdateStatus {
	status := UpdateStatus{Outcome: UpdateUnreachable, Running: s.running}
	release, err := s.source.Latest()
	if err != nil {
		return status
	}
	status.Latest = strings.TrimPrefix(strings.TrimSpace(release.Version), tagPrefix)
	status.Address = s.download(release)

	running, ok := versionParts(s.running)
	if !ok {
		status.Outcome = UpdateUncomparable
		return status
	}
	latest, ok := versionParts(status.Latest)
	switch {
	case !ok || !newer(latest, running):
		status.Outcome = UpdateCurrent
	case status.Latest == skipped:
		status.Outcome = UpdateSkipped
	default:
		status.Outcome = UpdateAvailable
	}
	return status
}

// download answers the release's file for this platform, matched by its ending without regard to
// case; the release's page where it carries none.
func (s *UpdateService) download(release ports.Release) string {
	if s.ending != "" {
		for _, asset := range release.Assets {
			if strings.HasSuffix(strings.ToLower(asset.Name), s.ending) {
				return asset.Address
			}
		}
	}
	return release.Page
}

// versionParts reads a version as dotted whole numbers; ok is false for anything else, a sign or a
// pre-release suffix included, so a version that cannot be read can never be offered.
func versionParts(version string) ([]uint64, bool) {
	fields := strings.Split(version, versionSeparator)
	parts := make([]uint64, 0, len(fields))
	for _, field := range fields {
		number, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return nil, false
		}
		parts = append(parts, number)
	}
	return parts, true
}

// newer reports whether latest is a later version than running. A number one of them lacks counts
// as zero, so 1.5.0.1 is newer than 1.5.0 while 1.5.0 is not newer than 1.5.
func newer(latest, running []uint64) bool {
	for i := range max(len(latest), len(running)) {
		if l, r := partAt(latest, i), partAt(running, i); l != r {
			return l > r
		}
	}
	return false
}

// partAt answers the version's number at index i; zero past its end.
func partAt(parts []uint64, i int) uint64 {
	if i < len(parts) {
		return parts[i]
	}
	return 0
}
