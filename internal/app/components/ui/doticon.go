package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"fyne.io/fyne/v2"
)

// CircleIcon returns a filled circle centered on a transparent square (canvasSize×canvasSize).
// Use a larger canvas than the circle so tab icons look smaller after Fyne scales them.
func CircleIcon(c color.Color, canvasSize, diameter int) fyne.Resource {
	if canvasSize < 4 {
		canvasSize = 4
	}
	if diameter < 2 {
		diameter = 2
	}
	if diameter > canvasSize {
		diameter = canvasSize
	}
	img := image.NewNRGBA(image.Rect(0, 0, canvasSize, canvasSize))
	r, g, b, a := c.RGBA()
	col := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	cx := float64(canvasSize-1) / 2
	rad := float64(diameter) / 2
	rad2 := rad * rad
	for y := 0; y < canvasSize; y++ {
		for x := 0; x < canvasSize; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cx
			if dx*dx+dy*dy <= rad2 {
				img.SetNRGBA(x, y, col)
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return fyne.NewStaticResource("circle-dot.png", buf.Bytes())
}

var (
	// canvas 18, diameter 6 → visually small after Fyne tab icon scaling
	DotGray  = CircleIcon(color.NRGBA{R: 0x9E, G: 0x9E, B: 0x9E, A: 0xFF}, 18, 6)
	DotGreen = CircleIcon(color.NRGBA{R: 0x2E, G: 0xC4, B: 0x5A, A: 0xFF}, 18, 6)
	DotRed   = CircleIcon(color.NRGBA{R: 0xE5, G: 0x39, B: 0x35, A: 0xFF}, 18, 6)
)
