package oriagent

import (
	"context"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/ui"
)

// Deleting a session: with Config.OnDeleteSession, the session menu and a
// button on the session's row in the sidebar ask to delete it; a dialog
// says what will be lost (Config.DeleteSessionInfo) and asks for
// confirmation, then the application deletes it, and it leaves the sidebar.

type deleteState struct {
	s           *Session // asked to delete; nil: no dialog
	info        string
	running     bool
	ok, cancel  widget.Clickable
	scrim, card widget.Clickable
}

// askDelete opens the dialog for s. A session whose turn runs cannot be
// deleted: the user stops it first.
func (a *App) askDelete(s *Session) {
	if a.cfg.OnDeleteSession == nil || s == nil || s.parent != nil {
		return
	}
	if s.Busy {
		a.notify(a.t("delete.busy"))
		return
	}
	a.del = deleteState{s: s}
	if a.cfg.DeleteSessionInfo != nil {
		a.del.info = a.cfg.DeleteSessionInfo(s)
	}
}

func (a *App) handleDelete(gtx C) {
	d := &a.del
	if d.s == nil {
		return
	}
	if d.cancel.Clicked(gtx) || d.scrim.Clicked(gtx) && !d.running {
		a.del = deleteState{}
		return
	}
	if d.ok.Clicked(gtx) && !d.running {
		a.confirmDelete()
	}
}

// locked reports whether s is being rewound or deleted: no turn may start
// in it meanwhile.
func (a *App) locked(s *Session) bool {
	return s.rewinding || a.del.running && a.del.s == s
}

// confirmDelete has the application delete the session of the dialog.
// Called with the lock held.
func (a *App) confirmDelete() {
	d := &a.del
	if d.s.Busy {
		// A turn started behind the dialog (Enter in the composer).
		a.del = deleteState{}
		a.notify(a.t("delete.busy"))
		return
	}
	d.running = true
	s, fn := d.s, a.cfg.OnDeleteSession
	go func() {
		err := fn(context.Background(), s)
		a.update(func() {
			a.del = deleteState{}
			if err != nil {
				a.notify(a.t("delete.failed", err.Error()))
				return
			}
			a.removeSession(s)
			a.notify(a.t("delete.done", s.Title))
		})
	}()
}

// removeSession takes s out of the sidebar, and shows another session.
// Called with the lock held.
func (a *App) removeSession(s *Session) {
	if p := s.project; p != nil {
		for i, x := range p.Sessions {
			if x == s {
				p.Sessions = append(p.Sessions[:i:i], p.Sessions[i+1:]...)
				break
			}
		}
	}
	if a.cur != s && !a.isChildOf(a.cur, s) {
		return
	}
	if all := a.allSessions(); len(all) > 0 {
		a.cur = all[0]
		return
	}
	n := &Session{Title: a.t("session.new")}
	if s.project != nil {
		s.project.Add(n)
	}
	a.cur = n
}

func (a *App) isChildOf(s, root *Session) bool {
	for ; s != nil; s = s.parent {
		if s == root {
			return true
		}
	}
	return false
}

// layoutDelete draws the confirmation dialog over the window.
func (a *App) layoutDelete(gtx C) D {
	d, th := &a.del, a.th
	if d.s == nil {
		return D{}
	}
	size := gtx.Constraints.Max
	scrim := color.NRGBA{A: 0x80}
	d.scrim.Layout(gtx, func(gtx C) D {
		paint.FillShape(gtx.Ops, scrim, clip.Rect{Max: size}.Op())
		return D{Size: size}
	})
	gtx.Constraints = layout.Exact(size)
	return layout.Center.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X-gtx.Dp(40), gtx.Dp(480))
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		// The card takes the clicks, so that they do not close the dialog.
		return d.card.Layout(gtx, func(gtx C) D {
			return ui.Bordered(gtx, th.Bg, th.BorderHi, 12, func(gtx C) D {
				return layout.UniformInset(20).Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(th.Label(16, a.t("delete.title", d.s.Title), th.Text, font.Medium).Layout),
						ui.VSpace(10),
						layout.Rigid(func(gtx C) D {
							if d.info == "" {
								return D{}
							}
							return layout.Inset{Bottom: 10}.Layout(gtx, th.Body(14, d.info))
						}),
						layout.Rigid(th.Label(13, a.t("delete.warning"), th.Danger, font.Normal).Layout),
						ui.VSpace(18),
						layout.Rigid(func(gtx C) D {
							return layout.E.Layout(gtx, func(gtx C) D {
								if d.running {
									return th.Label(13, a.t("delete.running"), ui.Pulse(gtx, th.Muted), font.Normal).Layout(gtx)
								}
								return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx C) D { return th.Button(gtx, &d.cancel, a.t("delete.cancel"), ui.Secondary) }),
									ui.HSpace(8),
									layout.Rigid(func(gtx C) D { return th.Button(gtx, &d.ok, a.t("delete.ok"), ui.Danger) }),
								)
							})
						}),
					)
				})
			})
		})
	})
}
