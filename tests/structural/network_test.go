package structural

// NFR-S-1: the application makes one network request, the update check of FR-756; no other. A
// request needs a network package in Go or a request call on the page, so neither may appear in
// anything the application is built from except the one exemption: no package of this module the
// application links may import a network package, apart from the update check's package importing
// net/http; the front end's own source may neither make a request nor name an address to make one
// to. The exemption is held to its reason, so it fails once the package no longer needs it.
//
// What this cannot see: a request made by Wails or its web view on its own account. Wails links
// net/http into the binary, which is why the rule is held over this module's source rather than
// over the binary's whole import graph.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// networkPackages are the standard packages a request is made through. A package whose path starts
// with one of these is one of them.
var networkPackages = []string{"net", "crypto/tls", "golang.org/x/net"}

// pageRequests are what a page makes a request with or makes one to.
var pageRequests = regexp.MustCompile(`\bfetch\s*\(|XMLHttpRequest|\bWebSocket\b|\bEventSource\b|sendBeacon|https?://`)

// frontendPage is the page the front end is served from, which can name an address of its own.
var frontendPage = filepath.Join("frontend", "index.html")

// isNetworkPackage reports whether imported is a network package or one beneath it.
func isNetworkPackage(imported string) bool {
	for _, network := range networkPackages {
		if imported == network || strings.HasPrefix(imported, network+"/") {
			return true
		}
	}
	return false
}

// updateCheckPackage is the one package the application links that may import a network package;
// updateCheckImport is the one it may import (FR-756).
const (
	updateCheckPackage = "internal/infrastructure/update"
	updateCheckImport  = "net/http"
)

// exempt reports whether dir importing imported is the update check's one permitted import.
func exempt(dir, imported string) bool {
	return filepath.ToSlash(dir) == updateCheckPackage && imported == updateCheckImport
}

// No package the application links imports a network package except the update check.
func TestTheApplicationImportsNoNetworkPackage(t *testing.T) {
	root := repoRoot(t)
	for _, dir := range linkedPackages(t) {
		for _, file := range sourceIn(t, filepath.Join(root, dir)) {
			for _, imported := range importsOf(t, file) {
				if isNetworkPackage(imported) && !exempt(dir, imported) {
					t.Errorf("%s imports %s; the application's one network request is the update check (NFR-S-1)", file, imported)
				}
			}
		}
	}
}

// The exemption cannot outlive its reason: the application links the update check's package and
// that package still imports the one network package it is permitted.
func TestTheUpdateCheckStillNeedsItsExemption(t *testing.T) {
	root := repoRoot(t)
	linked := false
	for _, dir := range linkedPackages(t) {
		linked = linked || filepath.ToSlash(dir) == updateCheckPackage
	}
	if !linked {
		t.Fatalf("the application no longer links %s; remove its exemption from NFR-S-1", updateCheckPackage)
	}
	for _, file := range sourceIn(t, filepath.Join(root, updateCheckPackage)) {
		for _, imported := range importsOf(t, file) {
			if imported == updateCheckImport {
				return
			}
		}
	}
	t.Fatalf("%s no longer imports %s; remove its exemption from NFR-S-1", updateCheckPackage, updateCheckImport)
}

// The front end neither makes a request nor names an address to make one to. Its tests are left
// out, since they are not part of the page.
func TestTheFrontEndMakesNoRequest(t *testing.T) {
	files := append(frontendFiles(t), filepath.Join(repoRoot(t), frontendPage))
	for _, file := range files {
		if strings.Contains(filepath.Base(file), ".test.") {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		for number, line := range strings.Split(string(body), "\n") {
			if found := pageRequests.FindString(line); found != "" {
				t.Errorf("%s:%d has %q; the application makes no network request (NFR-S-1)", file, number+1, found)
			}
		}
	}
}

// The pattern above is what the page test rests on, so it is held to the forms it has to catch and
// to the ordinary words it must not.
func TestTheRequestPatternCatchesEachWayARequestIsMade(t *testing.T) {
	for _, request := range []string{
		`fetch("/voices")`, `await fetch (url)`, `new XMLHttpRequest()`, `new WebSocket(address)`,
		`new EventSource(feed)`, `navigator.sendBeacon(url, body)`, `const donate = "https://example.org"`,
		`@import url(http://fonts.example/a.css);`,
	} {
		if !pageRequests.MatchString(request) {
			t.Errorf("%q makes a request and was not caught", request)
		}
	}
	for _, ordinary := range []string{`refetchVoices()`, `const fetched = true`, `// the socket is closed`} {
		if pageRequests.MatchString(ordinary) {
			t.Errorf("%q makes no request and was caught", ordinary)
		}
	}
}
