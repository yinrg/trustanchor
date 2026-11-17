package ui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/yinrg/trustanchor/assets"
	"github.com/yinrg/trustanchor/internal/truststore"
	"github.com/yinrg/trustanchor/internal/version"
)

const expiryWarningDays = 30

func (u *appUI) buildContent() fyne.CanvasObject {
	return container.NewBorder(nil, u.buildFooter(), nil, nil,
		container.NewPadded(u.buildHero()))
}

func (u *appUI) buildHero() fyne.CanvasObject {
	icon := canvas.NewImageFromResource(assets.WindowIcon())
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(64, 64))

	u.hero = widget.NewRichText(heroSegment("正在检测…", ""))
	u.heroSub = widget.NewLabel("正在读取系统信任库…")
	u.heroSub.Alignment = fyne.TextAlignCenter
	u.heroSub.Wrapping = fyne.TextWrapWord

	u.validity = widget.NewLabel("")
	u.validity.Alignment = fyne.TextAlignCenter
	u.validity.Importance = widget.LowImportance

	u.primaryBtn = widget.NewButtonWithIcon("安装根证书", theme.ConfirmIcon(), func() { go u.installCA() })
	u.primaryBtn.Importance = widget.HighImportance
	u.primaryBtn.Disable()

	u.detailsBtn = widget.NewButton("证书详情 ›", u.showCertDetails)
	u.detailsBtn.Importance = widget.LowImportance
	u.detailsBtn.Disable()

	u.refreshBtn = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() { go u.refreshCA() })
	u.refreshBtn.Importance = widget.LowImportance

	body := container.NewVBox(
		container.NewCenter(icon),
		u.hero,
		u.heroSub,
		u.validity,
		container.NewCenter(u.primaryBtn),
		container.NewCenter(container.NewHBox(u.detailsBtn, u.refreshBtn)),
	)
	return container.NewVBox(layout.NewSpacer(), body, layout.NewSpacer())
}

func (u *appUI) buildFooter() fyne.CanvasObject {
	versionLabel := widget.NewLabel(appVersion())
	versionLabel.Importance = widget.LowImportance

	u.footerStatus = widget.NewLabel("")
	u.footerStatus.Importance = widget.LowImportance
	u.footerStatus.Truncation = fyne.TextTruncateEllipsis

	u.updateBtn = widget.NewButtonWithIcon("检查更新", theme.ViewRefreshIcon(),
		func() { go u.checkForUpdate(true) })
	u.updateBtn.Importance = widget.LowImportance

	row := container.NewBorder(nil, nil, versionLabel, u.updateBtn, u.footerStatus)
	return container.NewVBox(widget.NewSeparator(), container.NewPadded(row))
}

func (u *appUI) applyCAState(st truststore.State, err error) {
	if err != nil {
		u.lastState = truststore.State{}
		u.hasState = false
		u.setHero("状态检测失败", widget.DangerImportance)
		u.heroSub.SetText(err.Error())
		u.validity.SetText("")
		u.primaryBtn.Disable()
		u.detailsBtn.Disable()
		u.refreshBtn.Enable()
		return
	}

	u.lastState = st
	u.hasState = true
	u.refreshBtn.Enable()

	if st.Installed {
		u.setHero("根证书已安装", widget.SuccessImportance)
		u.heroSub.SetText("该证书已受系统信任，可正常访问内部服务")
		u.primaryBtn.SetText("从系统移除")
		u.primaryBtn.SetIcon(theme.DeleteIcon())
		u.primaryBtn.Importance = widget.DangerImportance
		u.primaryBtn.OnTapped = u.confirmRemoveCA
		u.primaryBtn.Enable()
	} else {
		u.setHero("根证书未安装", widget.WarningImportance)
		u.heroSub.SetText("安装后系统将信任由该根证书签发的证书")
		u.primaryBtn.SetText("安装根证书")
		u.primaryBtn.SetIcon(theme.ConfirmIcon())
		u.primaryBtn.Importance = widget.HighImportance
		u.primaryBtn.OnTapped = func() { go u.installCA() }
		u.primaryBtn.Enable()
	}
	u.primaryBtn.Refresh()
	u.detailsBtn.Enable()

	text, importance := validityText(st, time.Now())
	u.validity.SetText(text)
	u.validity.Importance = importance
	u.validity.Refresh()
}

func (u *appUI) setBusyState(text string) {
	u.setHero(text, widget.MediumImportance)
	u.heroSub.SetText("请在系统授权窗口中确认")
	u.primaryBtn.Disable()
	u.detailsBtn.Disable()
	u.refreshBtn.Disable()
}

func (u *appUI) setHero(text string, importance widget.Importance) {
	seg, ok := u.hero.Segments[0].(*widget.TextSegment)
	if !ok {
		return
	}
	seg.Text = text
	seg.Style.ColorName = colorForImportance(importance)
	u.hero.Refresh()
}

func heroSegment(text string, color fyne.ThemeColorName) *widget.TextSegment {
	return &widget.TextSegment{
		Text: text,
		Style: widget.RichTextStyle{
			Alignment: fyne.TextAlignCenter,
			ColorName: color,
			SizeName:  theme.SizeNameHeadingText,
			TextStyle: fyne.TextStyle{Bold: true},
		},
	}
}

func colorForImportance(importance widget.Importance) fyne.ThemeColorName {
	switch importance {
	case widget.SuccessImportance:
		return theme.ColorNameSuccess
	case widget.WarningImportance:
		return theme.ColorNameWarning
	case widget.DangerImportance:
		return theme.ColorNameError
	default:
		return ""
	}
}

func (u *appUI) showCertDetails() {
	if !u.hasState {
		return
	}
	st := u.lastState

	fingerprint := widget.NewLabel(st.Fingerprint)
	fingerprint.TextStyle = fyne.TextStyle{Monospace: true}
	fingerprint.Wrapping = fyne.TextWrapBreak

	var copyBtn *widget.Button
	copyBtn = widget.NewButtonWithIcon("复制", theme.ContentCopyIcon(), func() {
		u.fyneApp.Clipboard().SetContent(st.Fingerprint)
		copyBtn.SetText("已复制")
		time.AfterFunc(1500*time.Millisecond, func() {
			fyne.Do(func() { copyBtn.SetText("复制") })
		})
	})
	copyBtn.Importance = widget.LowImportance

	state := "未安装"
	if st.Installed {
		state = "已安装"
	}

	content := container.NewVBox(
		detailRow("状态", widget.NewLabel(state), nil),
		detailRow("主体", wrappedLabel(st.Subject), nil),
		detailRow("SHA-256", fingerprint, copyBtn),
		detailRow("有效期", wrappedLabel(validityRange(st)), nil),
		detailRow("系统位置", wrappedLabel(st.Location), nil),
		widget.NewSeparator(),
		wrappedLabel("Firefox 使用独立证书库，不受系统信任库影响，需单独导入或启用企业根证书策略。"),
	)

	view := container.NewVScroll(container.NewPadded(content))
	view.SetMinSize(fyne.NewSize(460, 320))
	dialog.ShowCustom("证书详情", "关闭", view, u.win)
}

func (u *appUI) showAbout() {
	icon := canvas.NewImageFromResource(assets.WindowIcon())
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(48, 48))

	header := container.NewHBox(icon, container.NewVBox(
		widget.NewLabelWithStyle(AppName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("系统根证书信任库管理"),
	))

	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		widget.NewLabel("版本 "+version.Version),
		widget.NewLabel("构建 "+version.Commit+" · "+version.Date),
		widget.NewLabel("平台 "+version.Platform()),
		widget.NewSeparator(),
		newCheckOnStartItem(u.fyneApp),
	)
	dialog.ShowCustom("关于 "+AppName, "关闭", content, u.win)
}

func newCheckOnStartItem(a fyne.App) fyne.CanvasObject {
	prefs := a.Preferences()
	check := widget.NewCheck("启动时检查更新", func(checked bool) {
		prefs.SetBool(prefCheckUpdates, checked)
	})
	check.SetChecked(prefs.BoolWithFallback(prefCheckUpdates, true))
	return check
}

func detailRow(name string, value, action fyne.CanvasObject) fyne.CanvasObject {
	key := widget.NewLabel(name)
	key.Importance = widget.LowImportance
	keyBox := container.NewGridWrap(fyne.NewSize(100, 26), key)
	return container.NewBorder(nil, nil, keyBox, action, value)
}

func wrappedLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapBreak
	return label
}

func validityText(st truststore.State, now time.Time) (string, widget.Importance) {
	if st.NotAfter.IsZero() {
		return "", widget.LowImportance
	}
	end := st.NotAfter.Format("2006-01-02")
	days := expiryDays(st.NotAfter, now)
	switch {
	case days < 0:
		return fmt.Sprintf("证书已于 %s 过期", end), widget.DangerImportance
	case days <= expiryWarningDays:
		return fmt.Sprintf("证书有效期至 %s（剩余 %d 天，即将过期）", end, days), widget.WarningImportance
	default:
		return "证书有效期至 " + end, widget.LowImportance
	}
}

func validityRange(st truststore.State) string {
	if st.NotAfter.IsZero() {
		return "未知"
	}
	start := "未知"
	if !st.NotBefore.IsZero() {
		start = st.NotBefore.Format("2006-01-02")
	}
	end := st.NotAfter.Format("2006-01-02")
	days := expiryDays(st.NotAfter, time.Now())
	switch {
	case days < 0:
		return fmt.Sprintf("%s → %s（已过期 %d 天）", start, end, -days)
	case days == 0:
		return fmt.Sprintf("%s → %s（今天到期）", start, end)
	default:
		return fmt.Sprintf("%s → %s（剩余 %d 天）", start, end, days)
	}
}

func expiryDays(notAfter, now time.Time) int {
	return int(notAfter.Sub(now).Hours() / 24)
}

func appVersion() string {
	return "v" + strings.TrimPrefix(version.Version, "v")
}
