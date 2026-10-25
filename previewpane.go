package oriagent

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/sitle/ori-agent/preview"
)

// showPreview opens d in the right pane, rendering PDF pages in background.
func (a *App) showPreview(d *preview.Doc) {
	if d == nil || a.cfg.Disable.Preview {
		return
	}
	a.doc = d
	d.T = a.tr
	d.Load(a.w.Invalidate)
}

// splitDrag processes the divider's pointer events. Only the last drag event
// of a frame is applied: all of them are relative to the previous layout.
func (a *App) splitDrag(gtx C) {
	var dragX float32
	dragged := false
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &a.split, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Enter | pointer.Leave})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			a.dragging, a.pressX = true, e.Position.X
		case pointer.Drag:
			dragX, dragged = e.Position.X, true
		case pointer.Release, pointer.Cancel:
			a.dragging = false
		case pointer.Enter:
			a.splitHover = true
		case pointer.Leave:
			a.splitHover = false
		}
	}
	if dragged {
		a.previewW -= int(dragX - a.pressX)
	}
}

func (a *App) splitHandle(gtx C) D {
	th := a.th
	w, h := gtx.Dp(7), gtx.Constraints.Max.Y
	c, lw := th.Border, gtx.Dp(1)
	if a.dragging || a.splitHover {
		c, lw = th.Accent, gtx.Dp(2)
	}
	paint.FillShape(gtx.Ops, th.Bg, clip.Rect{Max: image.Pt(w, h)}.Op())
	paint.FillShape(gtx.Ops, c, clip.Rect{Min: image.Pt((w-lw)/2, 0), Max: image.Pt((w+lw)/2, h)}.Op())
	area := clip.Rect{Max: image.Pt(w, h)}.Push(gtx.Ops)
	event.Op(gtx.Ops, &a.split)
	pointer.CursorColResize.Add(gtx.Ops)
	area.Pop()
	return D{Size: image.Pt(w, h)}
}

// clampPreview keeps the pane between 300dp and what leaves the chat 380dp.
func (a *App) clampPreview(gtx C, avail int) int {
	if a.previewW == 0 {
		a.previewW = gtx.Dp(520)
	}
	lo, hi := gtx.Dp(300), avail-gtx.Dp(380)
	a.previewW = max(lo, min(a.previewW, hi))
	return a.previewW
}
