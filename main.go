package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"wailsproxy/internal/config"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	app := NewApp()

	appMenu := buildAppMenu(app)

	err := wails.Run(&options.App{
		Title:     "WailsProxy - 现代跨平台代理客户端",
		Width:     1180,
		Height:    760,
		MinWidth:  1020,
		MinHeight: 680,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		Menu:             appMenu,
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				HideTitleBar:               true,
				FullSizeContent:            true,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "WailsProxy",
				Message: "基于 Go + Wails v2 + Ant Design v5 的现代化跨平台代理客户端",
				Icon:    appIcon,
			},
		},
	})

	if err != nil {
		log.Fatal("Error:", err.Error())
	}
}

func buildAppMenu(app *App) *menu.Menu {
	rootMenu := menu.NewMenu()
	rootMenu.Append(menu.AppMenu())
	rootMenu.Append(menu.EditMenu())

	proxyMenu := rootMenu.AddSubmenu("代理控制")
	proxyMenu.AddText("显示主窗口", keys.CmdOrCtrl("1"), func(_ *menu.CallbackData) {
		if app.ctx != nil {
			wailsRuntime.WindowShow(app.ctx)
		}
	})
	proxyMenu.AddSeparator()
	proxyMenu.AddText("系统代理模式 (System Proxy)", nil, func(_ *menu.CallbackData) {
		_ = app.SetProxyMode(config.ProxyModeSystem)
	})
	proxyMenu.AddText("纯代理模式 (Manual / Core Only)", nil, func(_ *menu.CallbackData) {
		_ = app.SetProxyMode(config.ProxyModeManual)
	})
	proxyMenu.AddText("TUN 虚拟网卡模式 (TUN Mode)", nil, func(_ *menu.CallbackData) {
		_ = app.SetProxyMode(config.ProxyModeTun)
	})
	proxyMenu.AddSeparator()
	proxyMenu.AddText("规则分流 (Rule)", nil, func(_ *menu.CallbackData) {
		_ = app.SetRoutingMode(config.RoutingModeRule)
	})
	proxyMenu.AddText("全局代理 (Global)", nil, func(_ *menu.CallbackData) {
		_ = app.SetRoutingMode(config.RoutingModeGlobal)
	})
	proxyMenu.AddText("直接连接 (Direct)", nil, func(_ *menu.CallbackData) {
		_ = app.SetRoutingMode(config.RoutingModeDirect)
	})
	proxyMenu.AddSeparator()
	proxyMenu.AddText("复制终端代理命令", keys.CmdOrCtrl("e"), func(_ *menu.CallbackData) {
		if app.ctx != nil {
			cmd := app.GetTerminalProxyCommand("zsh")
			_ = wailsRuntime.ClipboardSetText(app.ctx, cmd)
		}
	})
	proxyMenu.AddSeparator()
	proxyMenu.AddText("彻底退出并还原代理", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		app.mu.Lock()
		app.forceQuit = true
		app.mu.Unlock()
		app.ShutdownCleanup()
		if app.ctx != nil {
			wailsRuntime.Quit(app.ctx)
		}
	})

	return rootMenu
}
