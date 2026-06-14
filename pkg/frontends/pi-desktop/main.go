package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "pi - AI Coding Agent",
		Width:            1200,
		Height:           800,
		MinWidth:         600,
		MinHeight:        400,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 26, G: 26, B: 46, A: 255},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "pi-desktop-0.97.0",
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
