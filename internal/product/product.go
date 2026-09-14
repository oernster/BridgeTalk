// Package product is the one place this application's identity is written down.
//
// The name reaches a reader in the window title, the tray tooltip, the About dialog,
// the setup program, the Start Menu and the Apps list. The same identity also names
// the install directory, the executable, the uninstall key and a window class. Those
// were nine files and three spellings, kept in step by whoever remembered.
//
// Two forms, because two are genuinely needed. Name is what a reader sees and may hold
// punctuation. Slug is what a file system and a registry hold, so it carries no space
// and no character a path cannot. A display name went into a shortcut file name once
// with a colon in it; every shortcut the installer wrote then arrived labelled with
// the first word alone, with no error anywhere.
//
// It sits under internal rather than in a layer because it belongs to none of them: it
// is a leaf that the composition root, the installer and the tray may all read without
// any of them depending on each other.
package product

const (
	// Name is the product as a reader meets it.
	Name = "Bridge Talk"

	// Slug is the same identity where only a file name will do. It must stay free of
	// the characters a path refuses, which a test in tests/structural enforces.
	Slug = "BridgeTalk"

	// DonateURL is the donation page the donate button hands to the desktop (FR-718), the
	// one the site's donate section links. The application never fetches it; the browser
	// does the asking.
	DonateURL = "https://www.paypal.com/ncp/payment/DVP73MPL9JPSU"
)
