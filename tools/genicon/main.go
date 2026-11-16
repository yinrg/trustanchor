package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

var sizes = []int{16, 24, 32, 48, 64, 128, 256, 512}

const canvas = 512

func main() {
	assetsDir := flag.String("assets", "assets", "资源目录")
	flag.Parse()

	full, err := os.ReadFile(filepath.Join(*assetsDir, "logo.svg"))
	if err != nil {
		panic(err)
	}
	compact, err := os.ReadFile(filepath.Join(*assetsDir, "logo-small.svg"))
	if err != nil {
		panic(err)
	}

	fullMark, err := decode(full)
	if err != nil {
		panic(fmt.Errorf("渲染 logo.svg 失败: %w", err))
	}
	compactMark, err := decode(compact)
	if err != nil {
		panic(fmt.Errorf("渲染 logo-small.svg 失败: %w", err))
	}

	iconDir := filepath.Join(*assetsDir, "icons")
	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		panic(err)
	}

	for _, size := range sizes {
		source := fullMark
		if size < 48 {
			source = compactMark
		}
		img := resizeArea(source, size)
		if err := writePNG(filepath.Join(iconDir, fmt.Sprintf("%dx%d.png", size, size)), img); err != nil {
			panic(err)
		}
		if size == canvas {
			if err := writePNG(filepath.Join(*assetsDir, "icon.png"), img); err != nil {
				panic(err)
			}
		}
	}
}

func decode(svg []byte) (*image.RGBA, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svg), oksvg.WarnErrorMode)
	if err != nil {
		return nil, err
	}

	icon.SetTarget(0, 0, canvas, canvas)
	img := image.NewRGBA(image.Rect(0, 0, canvas, canvas))
	scanner := rasterx.NewScannerGV(canvas, canvas, img, img.Bounds())
	icon.Draw(rasterx.NewDasher(canvas, canvas, scanner), 1.0)
	return img, nil
}

func resizeArea(src *image.RGBA, size int) *image.RGBA {
	b := src.Bounds()
	scale := float64(b.Dx()) / float64(size)
	dst := image.NewRGBA(image.Rect(0, 0, size, size))

	for y := 0; y < size; y++ {
		y0, y1 := float64(y)*scale, float64(y+1)*scale
		for x := 0; x < size; x++ {
			x0, x1 := float64(x)*scale, float64(x+1)*scale

			var sr, sg, sb, sa, weight float64
			for sy := math.Floor(y0); sy < y1; sy++ {
				wy := math.Min(y1, sy+1) - math.Max(y0, sy)
				if wy <= 0 {
					continue
				}
				for sx := math.Floor(x0); sx < x1; sx++ {
					wx := math.Min(x1, sx+1) - math.Max(x0, sx)
					if wx <= 0 {
						continue
					}
					w := wx * wy
					c := src.RGBAAt(int(sx), int(sy))
					sr += float64(c.R) * w
					sg += float64(c.G) * w
					sb += float64(c.B) * w
					sa += float64(c.A) * w
					weight += w
				}
			}
			if weight == 0 {
				continue
			}
			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(sr/weight + 0.5),
				G: uint8(sg/weight + 0.5),
				B: uint8(sb/weight + 0.5),
				A: uint8(sa/weight + 0.5),
			})
		}
	}
	return dst
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
