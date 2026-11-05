package main

import (
	"embed"
	"io/fs"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	app := NewApp()
	frontend, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}
	app.wailsApp = application.New(application.Options{
		Name:        "DailyFlow",
		Description: "Offline portable daily task tracker",
		Services:    []application.Service{application.NewService(app)},
		Assets:      application.AssetOptions{Handler: application.BundledAssetFileServer(frontend)},
	})
	app.window = app.wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "main", Title: "DailyFlow", Width: 1150, Height: 780, MinWidth: 320, MinHeight: 240, Hidden: true,
		URL: "/", BackgroundColour: application.NewRGB(246, 248, 252),
	})
	app.widgetWindow = app.wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "widget", Title: "DailyFlow Tasks", Width: widgetWidth, Height: widgetHeight,
		MinWidth: widgetWidth, MinHeight: widgetHeight, MaxWidth: widgetWidth, MaxHeight: widgetHeight,
		URL: "/?widget=1", InitialPosition: application.WindowCentered, AlwaysOnTop: false, Frameless: true,
		Windows: application.WindowsWindow{HiddenOnTaskbar: true},
		BackgroundType: application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(255, 255, 255, 0),
	})
	app.wailsApp.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(_ *application.ApplicationEvent) {
		if app.trayIconEnabled {
			setupTray(app)
		}
		app.enterWidget()
	})
	app.window.RegisterHook(events.Common.WindowClosing, app.beforeClose)
	app.widgetWindow.RegisterHook(events.Common.WindowClosing, app.beforeWidgetClose)
	if err := app.wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

func setupTray(app *App) {
	if app.wailsApp == nil {
		return
	}
	app.trayMu.Lock()
	defer app.trayMu.Unlock()
	if app.tray != nil {
		return
	}
	trayMenu := app.wailsApp.NewMenu()
	trayMenu.Add("Open DailyFlow").OnClick(func(*application.Context) { app.RestoreFull() })
	trayMenu.Add("Show task widget").OnClick(func(*application.Context) { app.enterWidget() })
	trayMenu.AddSeparator()
	trayMenu.Add("Exit DailyFlow").OnClick(func(*application.Context) { app.ExitApp() })
	tray := app.wailsApp.SystemTray.New()
	tray.SetIcon(appIcon).SetMenu(trayMenu)
	tray.SetTooltip("DailyFlow")
	tray.OnClick(func() { app.RestoreFull() })
	tray.OnRightClick(func() { tray.OpenMenu() })
	app.tray = tray
}
