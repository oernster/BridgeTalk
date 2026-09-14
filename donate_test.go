package main

import (
	"errors"
	"testing"
)

// donationPage is the address FR-718 names. It is written out here rather than read from
// the product package, so a character changed there fails this test instead of passing
// along with it.
const donationPage = "https://www.paypal.com/ncp/payment/DVP73MPL9JPSU"

// handOver records every address the facade hands to the desktop, answering each with
// fails, so a test sees what would have been opened without a browser opening.
type handOver struct {
	addresses []string
	fails     error
}

// open is the seam the facade calls in place of the desktop's browser.
func (h *handOver) open(address string) error {
	h.addresses = append(h.addresses, address)
	return h.fails
}

// FR-718: pressing the donate button hands the desktop exactly the donation page, once,
// and nothing else.
func TestTheDonateButtonHandsOverTheOneDonationPage(t *testing.T) {
	t.Parallel()
	desktop := &handOver{}
	app := &App{browse: desktop.open}

	if err := app.OpenDonation(); err != nil {
		t.Fatalf("OpenDonation: %v", err)
	}

	if len(desktop.addresses) != 1 || desktop.addresses[0] != donationPage {
		t.Fatalf("handed over %q, want exactly %q", desktop.addresses, donationPage)
	}
}

// FR-718: the facade refuses an address that does not begin https://, handing nothing
// to the desktop.
func TestAnAddressThatIsNotHTTPSIsRefusedHandingNothingOver(t *testing.T) {
	t.Parallel()
	desktop := &handOver{}
	app := &App{browse: desktop.open}

	for _, address := range []string{
		"http://www.paypal.com/ncp/payment/DVP73MPL9JPSU",
		"https:/www.paypal.com",
		"javascript:alert(1)",
		"file:///C:/Windows",
		" https://www.paypal.com",
		"",
	} {
		if err := app.openInBrowser(address); !errors.Is(err, errNotSecure) {
			t.Errorf("openInBrowser(%q) = %v, want the https refusal", address, err)
		}
	}

	if len(desktop.addresses) != 0 {
		t.Fatalf("handed over %q, want nothing", desktop.addresses)
	}
}

// FR-718 and FR-719: a hand-over that fails reaches the page as a rejection, which is what
// the live indicator says.
func TestAHandOverThatFailsIsReportedToThePage(t *testing.T) {
	t.Parallel()
	refused := errors.New("no browser answered")
	app := &App{browse: (&handOver{fails: refused}).open}

	if err := app.OpenDonation(); !errors.Is(err, refused) {
		t.Fatalf("OpenDonation = %v, want the hand-over's own failure", err)
	}
}

// Before the window exists there is nothing to open the browser through, which is said
// rather than dropped, so the page does not read a press that did nothing as a success.
func TestAHandOverBeforeTheWindowExistsIsRefused(t *testing.T) {
	t.Parallel()
	app := &App{}

	if err := app.OpenDonation(); !errors.Is(err, errNoWindow) {
		t.Fatalf("OpenDonation = %v, want the refusal for a window not yet there", err)
	}
}
