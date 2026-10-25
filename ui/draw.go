package ui

import (
	"image"
	"image/color"
	"math"
	"strings"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// Fill paints a rounded rectangle covering the minimum constraints; it is the
// background half of a layout.Background.
func Fill(gtx C, c color.NRGBA, radius unit.Dp) D {
	r := image.Rectangle{Max: gtx.Constraints.Min}
	paint.FillShape(gtx.Ops, c, clip.UniformRRect(r, gtx.Dp(radius)).Op(gtx.Ops))
	return D{Size: r.Max}
}

// Card draws w with padding on a rounded background.
func Card(gtx C, bg color.NRGBA, radius unit.Dp, in layout.Inset, w layout.Widget) D {
	return layout.Background{}.Layout(gtx,
		func(gtx C) D { return Fill(gtx, bg, radius) },
		func(gtx C) D { return in.Layout(gtx, w) },
	)
}

// Bordered draws w on a rounded background with a 1dp border.
func Bordered(gtx C, bg, border color.NRGBA, radius unit.Dp, w layout.Widget) D {
	return layout.Background{}.Layout(gtx,
		func(gtx C) D { return Fill(gtx, border, radius) },
		func(gtx C) D {
			return layout.UniformInset(1).Layout(gtx, func(gtx C) D {
				return layout.Background{}.Layout(gtx,
					func(gtx C) D { return Fill(gtx, bg, radius-1) },
					w,
				)
			})
		},
	)
}

func Dot(gtx C, c color.NRGBA, size unit.Dp) D {
	s := gtx.Dp(size)
	r := image.Rect(0, 0, s, s)
	paint.FillShape(gtx.Ops, c, clip.Ellipse(r).Op(gtx.Ops))
	return D{Size: r.Max}
}

func Ring(gtx C, c color.NRGBA, size unit.Dp) D {
	s := gtx.Dp(size)
	r := image.Rect(1, 1, s-1, s-1)
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: clip.Ellipse(r).Path(gtx.Ops), Width: float32(gtx.Dp(1.5))}.Op())
	return D{Size: image.Pt(s, s)}
}

// Pulse returns c with an alpha oscillating over time, and schedules a redraw.
// It is the "something is running" animation used everywhere.
func Pulse(gtx C, c color.NRGBA) color.NRGBA {
	phase := float64(gtx.Now.UnixMilli()%1200) / 1200
	c.A = uint8(90 + 165*0.5*(1+math.Sin(phase*2*math.Pi)))
	gtx.Execute(op.InvalidateCmd{})
	return c
}

// HLine is a full-width 1dp separator.
func (t *Theme) HLine(gtx C) D {
	r := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Dp(1))
	paint.FillShape(gtx.Ops, t.Border, clip.Rect(r).Op())
	return D{Size: r.Max}
}

func HSpace(w unit.Dp) layout.FlexChild { return layout.Rigid(layout.Spacer{Width: w}.Layout) }
func VSpace(h unit.Dp) layout.FlexChild { return layout.Rigid(layout.Spacer{Height: h}.Layout) }

// Expand takes the minimum width it is given; as a Flexed child it pushes
// the following children to the end of the row.
func Expand(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }

// Centered lays w out in a column of at most max width, centered horizontally,
// and reports the full width.
func Centered(gtx C, max unit.Dp, w layout.Widget) D {
	m := gtx.Dp(max)
	full := gtx.Constraints.Max.X
	if full <= m {
		return w(gtx)
	}
	off := (full - m) / 2
	gtx.Constraints.Max.X = m
	if gtx.Constraints.Min.X > m {
		gtx.Constraints.Min.X = m
	}
	t := op.Offset(image.Pt(off, 0)).Push(gtx.Ops)
	d := w(gtx)
	t.Pop()
	d.Size.X = full
	return d
}

// OneLine lays l out on a single line, cut with a trailing ellipsis when too
// wide. (Gio's own truncator can draw the ellipsis at the start of the line.)
func OneLine(gtx C, l material.LabelStyle) D {
	l.MaxLines = 1
	width := func(s string) int {
		g := gtx
		g.Constraints = layout.Constraints{Max: image.Pt(1<<20, gtx.Constraints.Max.Y)}
		m := op.Record(g.Ops)
		t := l
		t.Text = s
		d := t.Layout(g)
		m.Stop() // measured only, never drawn
		return d.Size.X
	}
	if width(l.Text) > gtx.Constraints.Max.X {
		r := []rune(l.Text)
		lo, hi := 0, len(r)
		for lo < hi {
			mid := (lo + hi + 1) / 2
			if width(string(r[:mid])+"…") <= gtx.Constraints.Max.X {
				lo = mid
			} else {
				hi = mid - 1
			}
		}
		l.Text = strings.TrimRight(string(r[:lo]), " ") + "…"
	}
	return l.Layout(gtx)
}

// Flow lays widgets out left to right, wrapping to new rows.
func Flow(gtx C, gap unit.Dp, items []layout.Widget) D {
	g := gtx.Dp(gap)
	maxW := gtx.Constraints.Max.X
	cgtx := gtx
	cgtx.Constraints.Min = image.Point{}
	x, y, rowH, w := 0, 0, 0, 0
	for _, it := range items {
		m := op.Record(gtx.Ops)
		d := it(cgtx)
		c := m.Stop()
		if x > 0 && x+d.Size.X > maxW {
			x, y, rowH = 0, y+rowH+g, 0
		}
		t := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
		c.Add(gtx.Ops)
		t.Pop()
		x += d.Size.X + g
		rowH = max(rowH, d.Size.Y)
		w = max(w, x-g)
	}
	return D{Size: image.Pt(w, y+rowH)}
}

// Ruled draws w indented, with a vertical rule on its left spanning its
// height: used for reasoning and command output.
func Ruled(gtx C, c color.NRGBA, w layout.Widget) D {
	macro := op.Record(gtx.Ops)
	d := layout.Inset{Left: 16}.Layout(gtx, w)
	call := macro.Stop()
	paint.FillShape(gtx.Ops, c, clip.Rect{Max: image.Pt(gtx.Dp(2), d.Size.Y)}.Op())
	call.Add(gtx.Ops)
	return d
}
