package assets

import (
	"bytes"
	"image"
	_ "image/png"
	"testing"
)

func decode(t *testing.T, icon []byte) image.Image {
	t.Helper()
	img, _, err := image.Decode(bytes.NewReader(icon))
	if err != nil {
		t.Fatalf("图标解码失败: %v", err)
	}
	return img
}

func TestWindowIconFitsX11Limit(t *testing.T) {
	b := decode(t, WindowIcon().Content()).Bounds()
	if b.Dx() > 64 || b.Dy() > 64 {
		t.Fatalf("窗口图标为 %v，超过 64x64：X11 会静默丢弃超大的 _NET_WM_ICON", b)
	}
}

func TestIconIs512(t *testing.T) {
	b := decode(t, Icon().Content()).Bounds()
	if b.Dx() != 512 || b.Dy() != 512 {
		t.Fatalf("应用图标应为 512x512，实际 %v", b)
	}
}
