package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/yinrg/trustanchor/assets"
)

func (u *appUI) setupTray() {
	desk, ok := u.fyneApp.(desktop.App)
	if !ok {
		return
	}
	desk.SetSystemTrayIcon(assets.Icon())
	desk.SetSystemTrayWindow(u.win)
	desk.SetSystemTrayMenu(fyne.NewMenu("",
		fyne.NewMenuItem("显示主窗口", func() {
			u.win.Show()
			u.win.RequestFocus()
		}),
		fyne.NewMenuItem("检查更新", func() { go u.checkForUpdate(true) }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("退出", u.fyneApp.Quit),
	))
}
