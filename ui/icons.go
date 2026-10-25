package ui

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// Icons are drawn with vector paths so they stay sharp at any scale and
// need no icon font. Each takes the color to draw with.

// Chevron points left (-1), right (1) or down (0).
func Chevron(gtx C, c color.NRGBA, dir int) D {
	s := float32(gtx.Dp(10))
	var p clip.Path
	p.Begin(gtx.Ops)
	switch dir {
	case -1:
		p.MoveTo(f32.Pt(s*0.7, 0))
		p.LineTo(f32.Pt(s*0.25, s/2))
		p.LineTo(f32.Pt(s*0.7, s))
	case 1:
		p.MoveTo(f32.Pt(s*0.3, 0))
		p.LineTo(f32.Pt(s*0.75, s/2))
		p.LineTo(f32.Pt(s*0.3, s))
	default:
		p.MoveTo(f32.Pt(0, s*0.3))
		p.LineTo(f32.Pt(s/2, s*0.75))
		p.LineTo(f32.Pt(s, s*0.3))
	}
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: float32(gtx.Dp(1.5))}.Op())
	return D{Size: image.Pt(int(s), int(s))}
}

func SmallChevron(gtx C, c color.NRGBA, dir int) D {
	gtx.Metric.PxPerDp *= 0.8
	return Chevron(gtx, c, dir)
}

func Close(gtx C, c color.NRGBA) D {
	s := float32(gtx.Dp(10))
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(0, 0))
	p.LineTo(f32.Pt(s, s))
	p.MoveTo(f32.Pt(s, 0))
	p.LineTo(f32.Pt(0, s))
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: float32(gtx.Dp(1.5))}.Op())
	return D{Size: image.Pt(int(s), int(s))}
}

// Gear is a settings cog: eight teeth around a ring.
func Gear(gtx C, c color.NRGBA, size unit.Dp) D {
	s := gtx.Dp(size)
	fs := float32(s)
	ctr := f32.Pt(fs/2, fs/2)
	tw, th := fs*0.2, fs*0.24
	for i := 0; i < 8; i++ {
		t := op.Affine(f32.Affine2D{}.Rotate(ctr, float32(i)*math.Pi/4)).Push(gtx.Ops)
		r := image.Rect(int(fs/2-tw/2), 0, int(fs/2+tw/2), int(th))
		paint.FillShape(gtx.Ops, c, clip.UniformRRect(r, gtx.Dp(1)).Op(gtx.Ops))
		t.Pop()
	}
	in := int(fs * 0.17)
	paint.FillShape(gtx.Ops, c, clip.Stroke{
		Path:  clip.Ellipse(image.Rect(in, in, s-in, s-in)).Path(gtx.Ops),
		Width: fs * 0.16,
	}.Op())
	return D{Size: image.Pt(s, s)}
}

// Sidebar is a window outline with a left panel, filled when open.
func Sidebar(gtx C, c color.NRGBA, open bool) D {
	w, h := gtx.Dp(18), gtx.Dp(16)
	sw := gtx.Dp(1.5)
	r := image.Rect(0, 0, w, h)
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: clip.UniformRRect(r.Inset(sw/2+1), gtx.Dp(3)).Path(gtx.Ops), Width: float32(sw)}.Op())
	x := gtx.Dp(6)
	paint.FillShape(gtx.Ops, c, clip.Rect{Min: image.Pt(x, 1), Max: image.Pt(x+sw, h-1)}.Op())
	if open {
		paint.FillShape(gtx.Ops, c, clip.Rect{Min: image.Pt(sw+1, sw+1), Max: image.Pt(x, h-sw-1)}.Op())
	}
	return D{Size: r.Max}
}

func Mic(gtx C, c color.NRGBA) D {
	w, h := gtx.Dp(14), gtx.Dp(16)
	fw := float32(w)
	cw := gtx.Dp(6)
	cx := (w - cw) / 2
	paint.FillShape(gtx.Ops, c, clip.UniformRRect(image.Rect(cx, 0, cx+cw, gtx.Dp(10)), cw/2).Op(gtx.Ops))
	sw := float32(gtx.Dp(1.5))
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(sw, float32(gtx.Dp(6))))
	p.QuadTo(f32.Pt(fw/2, float32(gtx.Dp(17))), f32.Pt(fw-sw, float32(gtx.Dp(6))))
	p.MoveTo(f32.Pt(fw/2, float32(gtx.Dp(11.5))))
	p.LineTo(f32.Pt(fw/2, float32(h)))
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: sw}.Op())
	return D{Size: image.Pt(w, h)}
}

// Stop is the square shown on a recording button.
func Stop(gtx C, c color.NRGBA) D {
	q := gtx.Dp(10)
	paint.FillShape(gtx.Ops, c, clip.UniformRRect(image.Rect(0, 0, q, q), gtx.Dp(2)).Op(gtx.Ops))
	return D{Size: image.Pt(q, q)}
}

func Paperclip(gtx C, c color.NRGBA) D {
	w, h := gtx.Dp(14), gtx.Dp(16)
	u := func(v float32) float32 { return float32(gtx.Dp(1)) * v }
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(u(10), u(5)))
	p.LineTo(f32.Pt(u(10), u(11.5)))
	p.QuadTo(f32.Pt(u(10), u(15)), f32.Pt(u(7), u(15)))
	p.QuadTo(f32.Pt(u(4), u(15)), f32.Pt(u(4), u(11.5)))
	p.LineTo(f32.Pt(u(4), u(3.5)))
	p.QuadTo(f32.Pt(u(4), u(1)), f32.Pt(u(6), u(1)))
	p.QuadTo(f32.Pt(u(8), u(1)), f32.Pt(u(8), u(3.5)))
	p.LineTo(f32.Pt(u(8), u(11)))
	p.QuadTo(f32.Pt(u(8), u(12.5)), f32.Pt(u(7), u(12.5)))
	p.QuadTo(f32.Pt(u(6), u(12.5)), f32.Pt(u(6), u(11)))
	p.LineTo(f32.Pt(u(6), u(5)))
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: u(1.4)}.Op())
	return D{Size: image.Pt(w, h)}
}

// Window caption icons for a custom title bar.
const (
	WinMinimize = iota
	WinMaximize
	WinClose
)

// WindowIcon draws a caption button glyph; bg is the bar color, used to mask
// the back square of the "restore" glyph when maximized.
func WindowIcon(gtx C, kind int, c, bg color.NRGBA, maximized bool) D {
	s := gtx.Dp(10)
	sw := float32(gtx.Dp(1))
	outline := func(r image.Rectangle) {
		paint.FillShape(gtx.Ops, c, clip.Stroke{Path: clip.UniformRRect(r, gtx.Dp(1)).Path(gtx.Ops), Width: sw}.Op())
	}
	switch kind {
	case WinMinimize:
		paint.FillShape(gtx.Ops, c, clip.Rect{Min: image.Pt(0, s/2), Max: image.Pt(s, s/2+gtx.Dp(1))}.Op())
	case WinMaximize:
		if maximized {
			o := gtx.Dp(2)
			outline(image.Rect(o, 0, s, s-o))
			paint.FillShape(gtx.Ops, bg, clip.Rect{Min: image.Pt(0, o), Max: image.Pt(s-o, s)}.Op())
			outline(image.Rect(0, o, s-o, s))
		} else {
			outline(image.Rect(0, 0, s, s))
		}
	case WinClose:
		return Close(gtx, c)
	}
	return D{Size: image.Pt(s, s)}
}
