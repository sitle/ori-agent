package ui

import (
	"image"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
)

// Popover draws w at offset from the current position, above everything laid
// out later in the frame (op.Defer). A window-wide transparent scrim under it
// is clicked when the user clicks outside: check scrim.Clicked to close.
func Popover(gtx C, scrim *widget.Clickable, offset image.Point, maxSize image.Point, w layout.Widget) {
	macro := op.Record(gtx.Ops)
	t := op.Offset(image.Pt(-10000, -10000)).Push(gtx.Ops)
	scrim.Layout(gtx, func(gtx C) D { return D{Size: image.Pt(20000, 20000)} })
	t.Pop()
	t = op.Offset(offset).Push(gtx.Ops)
	gtx.Constraints = layout.Constraints{Max: maxSize}
	w(gtx)
	t.Pop()
	op.Defer(gtx.Ops, macro.Stop())
}

// OverlayAbove draws w right above the current widget, over what was laid out
// before it, without taking layout space (used for the command palette).
func OverlayAbove(gtx C, gap unit.Dp, w layout.Widget) {
	macro := op.Record(gtx.Ops)
	d := w(gtx)
	call := macro.Stop()
	defer op.Offset(image.Pt(0, -d.Size.Y-gtx.Dp(gap))).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}

// MenuItem is one row of a Menu. A Separator item draws a rule and ignores
// the other fields.
type MenuItem struct {
	Btn       *widget.Clickable
	Label     string
	Danger    bool
	Separator bool
}

// Menu lays out a dropdown menu panel filling the max width.
func (t *Theme) Menu(gtx C, items []MenuItem) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return Bordered(gtx, t.Surface, t.BorderHi, 10, func(gtx C) D {
		return layout.UniformInset(6).Layout(gtx, func(gtx C) D {
			children := make([]layout.FlexChild, 0, len(items))
			for _, it := range items {
				if it.Separator {
					children = append(children, layout.Rigid(func(gtx C) D {
						return layout.Inset{Top: 4, Bottom: 4}.Layout(gtx, t.HLine)
					}))
					continue
				}
				c := t.Text
				if it.Danger {
					c = t.DiffDel
				}
				children = append(children, layout.Rigid(func(gtx C) D {
					return it.Btn.Layout(gtx, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						bg := t.Surface
						if it.Btn.Hovered() {
							bg = t.Hover
						}
						return Card(gtx, bg, 6, layout.Inset{Top: 7, Bottom: 7, Left: 10, Right: 10}, t.Label(13, it.Label, c, font.Normal).Layout)
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

// Toast is a transient message shown near the bottom of the window.
type Toast struct {
	msg   string
	until time.Time
}

func (t *Toast) Show(msg string) { t.msg, t.until = msg, time.Now().Add(2500*time.Millisecond) }

// Layout draws the toast over the whole area; call it last in the frame.
func (t *Toast) Layout(gtx C, th *Theme) {
	if t.msg == "" {
		return
	}
	if !gtx.Now.Before(t.until) {
		t.msg = ""
		return
	}
	gtx.Execute(op.InvalidateCmd{At: t.until})
	gtx.Constraints.Min = image.Point{}
	macro := op.Record(gtx.Ops)
	d := Card(gtx, th.Text, 10, layout.Inset{Top: 10, Bottom: 10, Left: 18, Right: 18}, th.Label(13, t.msg, th.Bg, font.Medium).Layout)
	call := macro.Stop()
	x := (gtx.Constraints.Max.X - d.Size.X) / 2
	y := gtx.Constraints.Max.Y - d.Size.Y - gtx.Dp(120)
	defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}
