package ui

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/yinrg/trustanchor/assets"
	"github.com/yinrg/trustanchor/internal/truststore"
	"github.com/yinrg/trustanchor/internal/updater"
	"github.com/yinrg/trustanchor/internal/version"
)

const (
	AppName          = "Trust Anchor"
	AppID            = "io.github.yinrg.trustanchor"
	prefCheckUpdates = "check_updates_on_start"
)

type appUI struct {
	fyneApp fyne.App
	win     fyne.Window

	hero       *widget.RichText
	heroSub    *widget.Label
	validity   *widget.Label
	primaryBtn *widget.Button
	detailsBtn *widget.Button
	refreshBtn *widget.Button

	updateBtn    *widget.Button
	footerStatus *widget.Label

	checking  atomic.Bool
	certBusy  atomic.Bool
	lastState truststore.State
	hasState  bool
}

func Run(a fyne.App) {
	u := &appUI{fyneApp: a}
	u.win = a.NewWindow(AppName)
	u.win.SetIcon(assets.WindowIcon())
	u.win.Resize(fyne.NewSize(480, 430))
	u.win.CenterOnScreen()
	u.win.SetContent(u.buildContent())
	u.win.SetMainMenu(u.buildMenu())
	u.setupTray()
	u.win.SetCloseIntercept(u.fyneApp.Quit)
	u.win.Show()

	time.AfterFunc(300*time.Millisecond, u.refreshCA)

	time.AfterFunc(1500*time.Millisecond, func() {
		if u.fyneApp.Preferences().BoolWithFallback(prefCheckUpdates, true) {
			u.checkForUpdate(false)
		}
	})

	a.Run()
}

func (u *appUI) buildMenu() *fyne.MainMenu {
	file := fyne.NewMenu("文件",
		fyne.NewMenuItem("检查更新…", func() { go u.checkForUpdate(true) }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("安装根证书…", func() { go u.installCA() }),
		fyne.NewMenuItem("移除根证书…", u.confirmRemoveCA),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("退出", u.fyneApp.Quit),
	)
	help := fyne.NewMenu("帮助",
		fyne.NewMenuItem("关于 "+AppName, u.showAbout),
	)
	return fyne.NewMainMenu(file, help)
}

func (u *appUI) checkForUpdate(manual bool) {
	if !u.checking.CompareAndSwap(false, true) {
		return
	}
	defer u.checking.Store(false)

	if !updater.Enabled() {
		if manual {
			u.onUI(func() {
				dialog.ShowInformation("检查更新",
					"更新功能尚未启用。\n请先在 internal/updater 中设置 RepositorySlug。", u.win)
			})
		}
		return
	}

	if manual {
		u.onUI(func() {
			u.updateBtn.SetText("检查中…")
			u.updateBtn.Disable()
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	upd, err := updater.Check(ctx)
	switch {
	case err != nil:
		u.onUI(func() {
			u.resetUpdateButton()
			if manual {
				dialog.ShowError(fmt.Errorf("检查更新失败：%w", err), u.win)
			}
		})
	case upd == nil:
		u.onUI(func() {
			u.resetUpdateButton()
			u.footerStatus.SetText("已是最新版本")
			if manual {
				dialog.ShowInformation("检查更新", "当前已是最新版本。", u.win)
			}
		})
	default:
		u.onUI(func() {
			u.updateBtn.Enable()
			u.updateBtn.SetIcon(theme.DownloadIcon())
			u.updateBtn.SetText("发现新版本 " + upd.Version)
			u.updateBtn.Importance = widget.HighImportance
			u.updateBtn.OnTapped = func() { u.showUpdate(upd) }
			u.updateBtn.Refresh()
			u.footerStatus.SetText("")
			if manual {
				u.showUpdate(upd)
			}
		})
	}
}

func (u *appUI) resetUpdateButton() {
	u.updateBtn.SetIcon(theme.ViewRefreshIcon())
	u.updateBtn.SetText("检查更新")
	u.updateBtn.Importance = widget.LowImportance
	u.updateBtn.OnTapped = func() { go u.checkForUpdate(true) }
	u.updateBtn.Enable()
	u.updateBtn.Refresh()
}

func (u *appUI) showUpdate(upd *updater.Update) {
	u.onUI(func() {
		notes := upd.Notes
		if notes == "" {
			notes = "该版本未提供更新说明。"
		}
		label := widget.NewLabel(notes)
		label.Wrapping = fyne.TextWrapWord
		notesView := container.NewVScroll(label)
		notesView.SetMinSize(fyne.NewSize(460, 220))

		content := container.NewVBox(
			widget.NewLabel(fmt.Sprintf("最新版本 %s（当前 %s）", upd.Version, version.Version)),
			notesView,
			widget.NewButtonWithIcon("前往下载", theme.DownloadIcon(), func() {
				u.openURL(upd.URL)
			}),
		)

		if !upd.CanSelfUpdate() {
			content.Add(widget.NewLabel("macOS 不支持原地替换，请下载安装包后手动覆盖。"))
			dialog.ShowCustom("发现新版本", "关闭", content, u.win)
			return
		}

		dialog.ShowCustomConfirm("发现新版本", "立即更新", "稍后", content, func(ok bool) {
			if ok {
				go u.install(upd)
			}
		}, u.win)
	})
}

func (u *appUI) install(upd *updater.Update) {
	progress := dialog.NewCustomWithoutButtons("正在更新", container.NewVBox(
		widget.NewLabel("正在下载并安装新版本，请稍候…"),
		widget.NewProgressBarInfinite(),
	), u.win)
	u.onUI(progress.Show)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	err := upd.Install(ctx)

	u.onUI(func() {
		progress.Hide()
		if err != nil {
			u.footerStatus.SetText("更新失败")
			dialog.ShowError(fmt.Errorf("更新失败：%w", err), u.win)
			return
		}
		u.footerStatus.SetText("更新完成，重启后生效")
		dialog.ShowCustomConfirm("更新完成", "立即退出", "稍后",
			widget.NewLabel("新版本已安装，重新启动应用后生效。"), func(ok bool) {
				if ok {
					u.fyneApp.Quit()
				}
			}, u.win)
	})
}

func (u *appUI) onUI(fn func()) {
	fyne.Do(fn)
}

func (u *appUI) openURL(raw string) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		dialog.ShowError(fmt.Errorf("无效的链接：%s", raw), u.win)
		return
	}
	if err := u.fyneApp.OpenURL(parsed); err != nil {
		dialog.ShowError(fmt.Errorf("打开链接失败：%w", err), u.win)
	}
}

func (u *appUI) refreshCA() {
	st, err := truststore.Status()
	u.onUI(func() { u.applyCAState(st, err) })
}

func (u *appUI) installCA() {
	if !u.certBusy.CompareAndSwap(false, true) {
		return
	}
	defer u.certBusy.Store(false)

	u.onUI(func() { u.setBusyState("正在请求管理员授权…") })

	st, err := truststore.EnsureInstalled()
	u.onUI(func() {
		if err != nil {
			u.showCAError("安装根证书", err)
			go u.refreshCA()
			return
		}
		u.applyCAState(st, nil)
		u.footerStatus.SetText("已安装到系统受信任根证书存储")
	})
}

func (u *appUI) removeCA() {
	if !u.certBusy.CompareAndSwap(false, true) {
		return
	}
	defer u.certBusy.Store(false)

	u.onUI(func() { u.setBusyState("正在请求管理员授权…") })

	st, err := truststore.EnsureUninstalled()
	u.onUI(func() {
		if err != nil {
			u.showCAError("移除根证书", err)
			go u.refreshCA()
			return
		}
		u.applyCAState(st, nil)
		u.footerStatus.SetText("已从系统信任库移除")
	})
}

func (u *appUI) confirmRemoveCA() {
	go func() {
		st, err := truststore.Status()
		u.onUI(func() {
			if err != nil {
				u.showCAError("移除根证书", err)
				return
			}
			if !st.Installed {
				dialog.ShowInformation("根证书", "当前未安装根证书。", u.win)
				return
			}
			dialog.ShowConfirm("移除根证书",
				"移除后依赖该证书的 HTTPS 连接将失败，确定继续？", func(ok bool) {
					if ok {
						go u.removeCA()
					}
				}, u.win)
		})
	}()
}

func (u *appUI) showCAError(title string, err error) {
	var manual *truststore.ManualCommandError
	if errors.As(err, &manual) {
		entry := widget.NewMultiLineEntry()
		entry.SetText(manual.Command)
		entry.Wrapping = fyne.TextWrapBreak
		dialog.ShowCustom(title, "关闭", container.NewVBox(
			widget.NewLabel("无法自动提权，请在终端执行以下命令："),
			entry,
		), u.win)
		return
	}
	if errors.Is(err, truststore.ErrCancelled) {
		dialog.ShowInformation(title, "已取消管理员授权。", u.win)
		return
	}
	dialog.ShowError(fmt.Errorf("%s失败：%w", title, err), u.win)
}
