package main

import (
 "embed"
 "io/fs"
 "github.com/wailsapp/wails/v2"
 "github.com/wailsapp/wails/v2/pkg/options"
 "github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed frontend/dist
var assets embed.FS

func main() {
 app := NewApp()
 frontend, err := fs.Sub(assets, "frontend/dist")
 if err != nil { panic(err) }
 err = wails.Run(&options.App{
  Title: "DailyFlow", Width: 1150, Height: 780, MinWidth: 320, MinHeight: 240,
  AssetServer: &assetserver.Options{Assets: frontend},
  BackgroundColour: &options.RGBA{R: 246, G: 248, B: 252, A: 1},
  OnStartup: app.startup, OnBeforeClose: app.beforeClose,
  AlwaysOnTop: false, Bind: []interface{}{app},
 })
 if err != nil { println("DailyFlow:", err.Error()) }
}
