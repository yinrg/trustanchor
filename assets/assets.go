package assets

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
var iconPNG []byte

//go:embed icons/64x64.png
var windowIconPNG []byte

func Icon() fyne.Resource {
	return fyne.NewStaticResource("icon.png", iconPNG)
}

func WindowIcon() fyne.Resource {
	return fyne.NewStaticResource("window-icon.png", windowIconPNG)
}
