package ui

import (
	"bytes"
	"image"
	"image/color"

	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// Logo rasterizes an SVG at the exact pixel size requested, so it stays sharp
// on any screen density; the last raster is cached. Rendering is pure Go
// (oksvg); unsupported SVG features such as filters are skipped.
type Logo struct {
	icon *oksvg.SvgIcon
	size int
	img  paint.ImageOp
}

// NewLogo parses svg; a nil or invalid SVG yields a logo drawn as a dot.
func NewLogo(svg []byte) *Logo {
	if len(svg) == 0 {
		return &Logo{}
	}
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svg), oksvg.IgnoreErrorMode)
	if err != nil {
		return &Logo{}
	}
	return &Logo{icon: icon}
}

// Layout draws the logo in a size×size square; fallback colors the dot drawn
// when there is no SVG.
func (l *Logo) Layout(gtx C, size unit.Dp, fallback color.NRGBA) D {
	if l == nil || l.icon == nil {
		return Dot(gtx, fallback, size)
	}
	s := gtx.Dp(size)
	if s != l.size {
		rgba := image.NewRGBA(image.Rect(0, 0, s, s))
		l.icon.SetTarget(0, 0, float64(s), float64(s))
		l.icon.Draw(rasterx.NewDasher(s, s, rasterx.NewScannerGV(s, s, rgba, rgba.Bounds())), 1)
		l.img, l.size = paint.NewImageOp(rgba), s
	}
	defer clip.Rect{Max: image.Pt(s, s)}.Push(gtx.Ops).Pop()
	l.img.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return D{Size: image.Pt(s, s)}
}
