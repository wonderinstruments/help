package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Suppress WebKit/GTK runtime warnings from leaking to terminal
	suppressStderr()

	var initialDoc string
	if len(os.Args) > 1 {
		initialDoc = os.Args[1]
	}
	app := NewApp(initialDoc)

	err := wails.Run(&options.App{
		Title:  "Help",
		Width:  1024,
		Height: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		panic(err)
	}
}
