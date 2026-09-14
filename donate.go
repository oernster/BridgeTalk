// The donate button's one act: handing the donation page to the desktop's browser (FR-718).
//
// It sits beside app.go for the reason window_life.go does: app.go is at the edge of the size
// cap; this is a slice that comes out whole.

package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/oernster/bridge-talk/internal/product"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// secureScheme is how every address handed to the browser must begin (FR-718).
const secureScheme = "https://"

// errNotSecure refuses an address that does not begin with secureScheme. errNoWindow refuses
// a hand-over asked for before the window exists, since the browser is opened through it.
var (
	errNotSecure = errors.New("an address handed to the browser must begin " + secureScheme)
	errNoWindow  = errors.New("there is no window to open the browser from")
)

// OpenDonation hands the donation page to the desktop to open in the browser (FR-718). The
// application fetches nothing itself: the browser does the asking. A hand-over that fails is
// returned, so the page can say so (FR-719).
func (a *App) OpenDonation() error { return a.openInBrowser(product.DonateURL) }

// openInBrowser hands an address to the desktop's browser, refusing any that does not begin
// https:// before anything is handed over.
func (a *App) openInBrowser(address string) error {
	if !strings.HasPrefix(address, secureScheme) {
		return fmt.Errorf("refusing %q: %w", address, errNotSecure)
	}
	browse := a.browse
	if browse == nil {
		browse = a.browseInWails
	}
	return browse(address)
}

// browseInWails is the production hand-over. Wails reports nothing back from it, so the one
// failure that can be seen is having no window to open from; that is returned rather than
// dropped.
func (a *App) browseInWails(address string) error {
	if a.ctx == nil {
		return errNoWindow
	}
	runtime.BrowserOpenURL(a.ctx, address)
	return nil
}
