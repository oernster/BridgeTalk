package main

import (
	"embed"
	"fmt"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// Window geometry. The default is wide enough for the reaction log's columns without
// horizontal scrolling and tall enough for the Cast pane to reach its machine voices,
// which sit below the recorded ones; the minimum is where the nav band stops fitting
// on one row.
//
// The minimum is measured rather than chosen; it is re-measured whenever the band
// gains a button. The band needs 1022 pixels for its eight buttons, the volume slider
// at its full width, the gaps and its own padding, so anything under that squeezes
// the slider or pushes the last button off the row; 1045 leaves the two groups
// visibly apart. The default grew with the icons, so a window opened at it has room
// for the band and a useful pane rather than the band and a sliver.
const (
	windowWidth     = 1344
	windowHeight    = 960
	windowMinWidth  = 1045
	windowMinHeight = 600
)

// launch runs the Wails window.
//
// Started hidden, the window exists but is not shown: the tray icon summons it, which
// is what the login entry wants. Anything else opens it as usual.
func launch(app *App, hidden bool) error {
	app.startedHidden = hidden
	err := wails.Run(&options.App{
		StartHidden:      hidden,
		Title:            appTitle,
		Width:            windowWidth,
		Height:           windowHeight,
		MinWidth:         windowMinWidth,
		MinHeight:        windowMinHeight,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 12, G: 12, B: 14, A: 1},
		OnBeforeClose:    app.beforeClose,
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnShutdown:       app.shutdown,
		Bind:             []interface{}{app},
	})
	if err != nil {
		return fmt.Errorf("running the window: %w", err)
	}
	return nil
}
