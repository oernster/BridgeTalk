// Package update reads the project's latest published release from GitHub (FR-756).
//
// It is the one package of the application permitted a network package; it imports `net/http`
// alone (NFR-S-1). The endpoint answers only a release that is published, neither a draft nor a
// pre-release, so that guard is the endpoint's own contract rather than a check here. The request
// carries no credentials and nothing about the user, the machine or the game.
package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/product"
)

// LatestReleaseURL is GitHub's latest-release endpoint for this project, whose repository is named
// for the product.
const LatestReleaseURL = "https://api.github.com/repos/oernster/" + product.Slug + "/releases/latest"

// acceptHeader asks the API for its own JSON.
const acceptHeader = "application/vnd.github+json"

// requestTimeout is how long the check waits before giving up (FR-756).
const requestTimeout = 5 * time.Second

// answerCap is the most of an answer read (FR-756). A release's answer is a few kilobytes; a larger
// one is not a release, so it is refused rather than read to its end.
const answerCap = 1 << 20

// errTooLarge refuses an answer longer than answerCap. errNotARelease refuses one without a tag or
// a page.
var (
	errTooLarge    = errors.New("the answer is larger than a release")
	errNotARelease = errors.New("the answer names no release tag or page")
)

// Doer sends one request; *http.Client is one and a test supplies another, so no test reaches the
// network.
type Doer interface {
	Do(request *http.Request) (*http.Response, error)
}

// answer is the part of GitHub's answer the check reads.
type answer struct {
	Tag    string      `json:"tag_name"`
	Page   string      `json:"html_url"`
	Assets []answerSet `json:"assets"`
}

// answerSet is one file attached to the release.
type answerSet struct {
	Name     string `json:"name"`
	Download string `json:"browser_download_url"`
}

// Source is a ports.ReleaseSource over GitHub.
type Source struct {
	address string
	client  Doer
}

// New builds the source the application uses, over a client that gives up after requestTimeout.
func New() *Source {
	return NewWith(LatestReleaseURL, &http.Client{Timeout: requestTimeout})
}

// NewWith builds a source over another endpoint and client, for a test.
func NewWith(address string, client Doer) *Source {
	return &Source{address: address, client: client}
}

// Latest reads the latest published release. Every failure is an error; the check reads them all
// alike, as unreachable.
func (s *Source) Latest() (ports.Release, error) {
	request, err := http.NewRequest(http.MethodGet, s.address, nil)
	if err != nil {
		return ports.Release{}, fmt.Errorf("asking for the latest release: %w", err)
	}
	request.Header.Set("Accept", acceptHeader)
	response, err := s.client.Do(request)
	if err != nil {
		return ports.Release{}, fmt.Errorf("asking for the latest release: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ports.Release{}, fmt.Errorf("asking for the latest release: GitHub answered %s", response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, answerCap+1))
	if err != nil {
		return ports.Release{}, fmt.Errorf("reading the latest release: %w", err)
	}
	if len(body) > answerCap {
		return ports.Release{}, errTooLarge
	}
	return parse(body)
}

// parse reads a release out of an answer, passing over any file that names no file or address.
func parse(body []byte) (ports.Release, error) {
	var read answer
	if err := json.Unmarshal(body, &read); err != nil {
		return ports.Release{}, fmt.Errorf("reading the latest release: %w", err)
	}
	if read.Tag == "" || read.Page == "" {
		return ports.Release{}, errNotARelease
	}
	release := ports.Release{Version: read.Tag, Page: read.Page}
	for _, each := range read.Assets {
		if each.Name != "" && each.Download != "" {
			release.Assets = append(release.Assets, ports.Asset{Name: each.Name, Address: each.Download})
		}
	}
	return release, nil
}
