package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	fileMenu := menu.NewMenu()
	fileMenu.AddText("Открыть реплей…", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
		app.ChooseReplay()
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Выход", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		runtime.Quit(app.ctx)
	})

	err := wails.Run(&options.App{
		Title:     "MVP-Test",
		Width:     1280,
		Height:    860,
		MinWidth:  1000,
		MinHeight: 700,
		Menu:      fileMenu,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 23, B: 42, A: 1},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
