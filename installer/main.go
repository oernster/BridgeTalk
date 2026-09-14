// Command installer is the bespoke Bridge Talk setup program.
//
// It is built as a Wails app so it wears the same WebView and the same palette as
// the application it installs, light theme and dark theme alike. It carries the
// built application as an embedded zip and covers install, update, repair,
// reinstall, downgrade and uninstall, all per user with no administrator rights.
package main

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/infrastructure/setup"
	"github.com/oernster/bridge-talk/internal/product"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	windowsoptions "github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// payload is embedded as a string rather than a byte slice (FR-524). Measured with a
// stand-in program carrying the full payload: as a byte slice it was charged to the
// process as 323.6 MB of private memory from the moment it started; as a string, 12.8 MB.
//
//go:embed payload.zip
var payload string

// appVersion is overridden at build time with -ldflags "-X main.appVersion=x.y.z",
// so the setup program never holds a version literal of its own.
var appVersion = "dev"

const (
	windowTitle = product.Name + " Setup"
	// The window is fixed, so its height has to clear the tallest screen: the
	// install one, which carries four options under the path box. Both figures are
	// measured against that screen at the shipped type sizes rather than chosen; they
	// are rechecked whenever the type changes: text that grows without the
	// window growing with it turns a fixed dialog into a scrolling one.
	windowWidth  = 860
	windowHeight = 780
	// webviewFolder holds the setup window's own WebView2 cache. It is pinned under
	// TEMP rather than left to default into %APPDATA%, so running setup leaves no
	// folder behind next to the application's own.
	webviewFolder = product.Slug + "Setup"
)

// dark and light are the surface colours from the application's palette. One of them
// paints the window before the page loads, so setup never flashes the wrong ground.
var (
	dark  = options.RGBA{R: 0x0c, G: 0x0c, B: 0x0e, A: 1}
	light = options.RGBA{R: 0xf6, G: 0xf4, B: 0xf1, A: 1}
)

func main() {
	prefersDark := setup.SystemPrefersDark()
	background := light
	if prefersDark {
		background = dark
	}
	app := NewApp(payload, appVersion, prefersDark)
	_ = wails.Run(&options.App{
		Title:            windowTitle,
		Width:            windowWidth,
		Height:           windowHeight,
		DisableResize:    true,
		BackgroundColour: &background,
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		Bind:             []interface{}{app},
		Windows: &windowsoptions.Options{
			WebviewUserDataPath: filepath.Join(os.TempDir(), webviewFolder),
		},
	})
}
