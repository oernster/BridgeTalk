package update

// The release read from GitHub (FR-756), over a client that answers from memory.

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/product"
)

// endpoint is the address FR-756 names, written out here rather than read from the package's own
// constant, so a character changed there fails this test instead of passing along with it. The
// repository's name is the product's, which is written down once (TestTheProductIsNamedOnce).
const endpoint = "https://api.github.com/repos/oernster/" + product.Slug + "/releases/latest"

// answering is a client that records the request, then answers with status and body; with err
// instead where it has one.
type answering struct {
	status  int
	body    string
	err     error
	request *http.Request
}

func (a *answering) Do(request *http.Request) (*http.Response, error) {
	a.request = request
	if a.err != nil {
		return nil, a.err
	}
	return &http.Response{
		StatusCode: a.status,
		Status:     http.StatusText(a.status),
		Body:       io.NopCloser(strings.NewReader(a.body)),
	}, nil
}

// released is an answer carrying one release with two files and one entry naming nothing.
const released = `{
  "tag_name": "v1.5.0",
  "html_url": "https://example.test/releases/tag/v1.5.0",
  "draft": false,
  "assets": [
    {"name": "TheSetup.exe", "browser_download_url": "https://example.test/TheSetup.exe"},
    {"name": "", "browser_download_url": "https://example.test/nameless"},
    {"name": "TheApp.flatpak", "browser_download_url": ""},
    {"name": "TheApp.flatpak", "browser_download_url": "https://example.test/TheApp.flatpak"}
  ]
}`

// FR-756: the request asks the project's own endpoint for its JSON, with GET and no credentials.
func TestTheRequestAsksForTheLatestPublishedRelease(t *testing.T) {
	t.Parallel()
	client := &answering{status: http.StatusOK, body: released}
	if _, err := NewWith(LatestReleaseURL, client).Latest(); err != nil {
		t.Fatalf("Latest: %v", err)
	}
	request := client.request
	if request.Method != http.MethodGet || request.URL.String() != endpoint {
		t.Errorf("asked %s %s, want GET %s", request.Method, request.URL, endpoint)
	}
	if got := request.Header.Get("Accept"); got != "application/vnd.github+json" {
		t.Errorf("Accept %q, want GitHub's JSON", got)
	}
	if len(request.Header) != 1 || request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" {
		t.Errorf("headers %v, want Accept alone", request.Header)
	}
}

// FR-756: the release is read from the answer, passing over any file naming no file or address.
func TestAReleaseIsReadFromTheAnswer(t *testing.T) {
	t.Parallel()
	got, err := NewWith(endpoint, &answering{status: http.StatusOK, body: released}).Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	want := ports.Release{
		Version: "v1.5.0",
		Page:    "https://example.test/releases/tag/v1.5.0",
		Assets: []ports.Asset{
			{Name: "TheSetup.exe", Address: "https://example.test/TheSetup.exe"},
			{Name: "TheApp.flatpak", Address: "https://example.test/TheApp.flatpak"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// FR-756: anything that is not a release is refused rather than read as one.
func TestAnAnswerThatIsNotAReleaseIsRefused(t *testing.T) {
	t.Parallel()
	for name, client := range map[string]*answering{
		"no route":       {err: errors.New("no route to host")},
		"not found":      {status: http.StatusNotFound, body: `{"message":"Not Found"}`},
		"rate limited":   {status: http.StatusForbidden, body: released},
		"not JSON":       {status: http.StatusOK, body: "<html>"},
		"a list":         {status: http.StatusOK, body: `[]`},
		"no tag":         {status: http.StatusOK, body: `{"html_url":"https://example.test/page"}`},
		"no page":        {status: http.StatusOK, body: `{"tag_name":"v1.5.0"}`},
		"a numbered tag": {status: http.StatusOK, body: `{"tag_name":150,"html_url":"https://example.test/page"}`},
		"odd assets":     {status: http.StatusOK, body: `{"tag_name":"v1","html_url":"https://example.test/p","assets":{}}`},
	} {
		if got, err := NewWith(endpoint, client).Latest(); err == nil {
			t.Errorf("%s: read %+v, want a refusal", name, got)
		}
	}
	if _, err := NewWith("://no scheme", &answering{}).Latest(); err == nil {
		t.Error("an address that cannot be asked was not refused")
	}
}

// FR-756: an answer longer than the cap is refused having read no more than the cap and one byte.
func TestAnAnswerLargerThanTheCapIsRefused(t *testing.T) {
	t.Parallel()
	padded := strings.Replace(released, `"draft": false`, `"notes": "`+strings.Repeat("x", answerCap)+`"`, 1)
	_, err := NewWith(endpoint, &answering{status: http.StatusOK, body: padded}).Latest()
	if !errors.Is(err, errTooLarge) {
		t.Fatalf("an answer of %d bytes = %v, want the size refusal", len(padded), err)
	}
	if _, err := NewWith(endpoint, &answering{status: http.StatusOK, body: released}).Latest(); err != nil {
		t.Fatalf("an answer under the cap was refused: %v", err)
	}
}

// failingBody fails partway through, as a connection dropped mid-answer does.
type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("connection reset") }
func (failingBody) Close() error             { return nil }

// dropping answers with a body that fails to read.
type dropping struct{}

func (dropping) Do(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: failingBody{}}, nil
}

// FR-756: an answer that breaks off is refused rather than read as far as it got.
func TestAnAnswerThatBreaksOffIsRefused(t *testing.T) {
	t.Parallel()
	if _, err := NewWith(endpoint, dropping{}).Latest(); err == nil {
		t.Fatal("an answer that broke off was read as a release")
	}
}

// FR-756: the application's own source gives up after five seconds and asks the project's endpoint.
func TestTheCheckGivesUpAfterFiveSeconds(t *testing.T) {
	t.Parallel()
	source := New()
	client, ok := source.client.(*http.Client)
	if !ok || client.Timeout != 5*time.Second {
		t.Fatalf("client %#v, want an http.Client giving up after 5s", source.client)
	}
	if source.address != endpoint {
		t.Fatalf("asks %q, want %q", source.address, endpoint)
	}
}
