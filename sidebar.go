package oriagent

import (
	"image"
	"strconv"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/sitle/ori-agent/ui"
)

const sidebarWidth = 260

func itoa(n int) string { return strconv.Itoa(n) }

func (a *App) layoutSidebar(gtx C) D {
	th := a.th
	w := gtx.Dp(sidebarWidth)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	paint.FillShape(gtx.Ops, th.Sidebar, clip.Rect{Max: gtx.Constraints.Max}.Op())
	paint.FillShape(gtx.Ops, th.Border, clip.Rect{Min: image.Pt(w-gtx.Dp(1), 0), Max: gtx.Constraints.Max}.Op())

	return layout.Inset{Top: 12, Bottom: 10, Left: 12, Right: 13}.Layout(gtx, func(gtx C) D {
		children := []layout.FlexChild{
			layout.Rigid(func(gtx C) D {
				return a.newBtn.Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					bg := th.Sidebar
					if a.newBtn.Hovered() {
						bg = th.Hover
					}
					return ui.Bordered(gtx, bg, th.Border, 8, func(gtx C) D {
						return layout.Inset{Top: 8, Bottom: 8, Left: 10, Right: 10}.Layout(gtx, func(gtx C) D {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(th.Label(16, "+", th.Muted, font.Normal).Layout),
								ui.HSpace(8),
								layout.Rigid(th.Label(14, a.t("session.new"), th.Text, font.Medium).Layout),
							)
						})
					})
				})
			}),
			ui.VSpace(20),
			layout.Flexed(1, a.layoutTree),
		}
		// Custom pages, above the footer.
		for i, p := range a.cfg.Pages {
			children = append(children, layout.Rigid(func(gtx C) D {
				return th.NavRow(gtx, &a.pageBtns[i], th.Sidebar, a.view == p.ID, func(gtx C) D {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							if p.Icon == nil {
								return ui.Ring(gtx, th.Muted, 14)
							}
							return p.Icon(gtx, th.Muted)
						}),
						ui.HSpace(10),
						layout.Rigid(th.Label(14, a.t(p.Title), th.Text, font.Normal).Layout),
					)
				})
			}))
		}
		if !a.cfg.Disable.Profile || !a.cfg.Disable.Setup {
			children = append(children, layout.Rigid(th.HLine), ui.VSpace(8), layout.Rigid(a.sidebarFooter))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

// sidebarFooter is one row, as in the desktop client: identity opens the
// profile, the gear on the right opens the configuration.
func (a *App) sidebarFooter(gtx C) D {
	th := a.th
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D {
			if a.cfg.Disable.Profile {
				return ui.Expand(gtx)
			}
			return th.NavRow(gtx, &a.navProfile, th.Sidebar, a.view == viewProfile, func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D { return th.Avatar(gtx, a.initials(), 32) }),
					ui.HSpace(10),
					layout.Flexed(1, func(gtx C) D {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								name := a.fullName()
								if name == "" {
									name = a.t("profile.self")
								}
								return ui.OneLine(gtx, th.Label(13, name, th.Text, font.Medium))
							}),
							layout.Rigid(func(gtx C) D {
								return ui.OneLine(gtx, th.Label(11, a.profile.fields[pEmail].Text(), th.Muted, font.Normal))
							}),
						)
					}),
				)
			})
		}),
		ui.HSpace(4),
		layout.Rigid(func(gtx C) D {
			if a.cfg.Disable.Setup {
				return D{}
			}
			btn := &a.navSetup
			return btn.Layout(gtx, func(gtx C) D {
				bg, c := th.Sidebar, th.Muted
				switch {
				case a.view == viewSetup:
					bg, c = th.Selected, th.Text
				case btn.Hovered():
					bg, c = th.Hover, th.Text
				}
				return ui.Card(gtx, bg, 8, layout.UniformInset(9), func(gtx C) D { return ui.Gear(gtx, c, 18) })
			})
		}),
	)
}

func (a *App) fullName() string {
	return strings.TrimSpace(a.profile.fields[pFirst].Text() + " " + a.profile.fields[pLast].Text())
}

func (a *App) initials() string {
	var s string
	for _, w := range strings.Fields(a.fullName()) {
		s += strings.ToUpper(string([]rune(w)[:1]))
	}
	if s == "" {
		return "?"
	}
	return s
}

// ---- projects → sessions → subagents tree ----

func (a *App) handleTree(gtx C) {
	for _, p := range a.projects {
		if p.btn.Clicked(gtx) {
			p.Open = !p.Open
		}
	}
	walkSessions(a.projects, func(s *Session) {
		if s.btn.Clicked(gtx) {
			a.cur, a.focus, a.view = s, true, viewChat
		}
		if s.chev.Clicked(gtx) {
			s.Open = !s.Open
		}
	})
}

// treeRow is one visible line of the sidebar tree.
type treeRow struct {
	project *Project
	session *Session
	depth   int
	last    bool // last child of its parent, for the └ guide
	first   bool // first project, no top margin
}

func (a *App) treeRows() []treeRow {
	var rows []treeRow
	var walk func(ss []*Session, depth int)
	walk = func(ss []*Session, depth int) {
		for i, s := range ss {
			rows = append(rows, treeRow{session: s, depth: depth, last: i == len(ss)-1})
			if s.Open {
				walk(s.Children, depth+1)
			}
		}
	}
	for i, p := range a.projects {
		rows = append(rows, treeRow{project: p, first: i == 0})
		if p.Open {
			walk(p.Sessions, 0)
		}
	}
	return rows
}

func (a *App) layoutTree(gtx C) D {
	rows := a.treeRows()
	return material.List(a.th.Material, &a.sideList).Layout(gtx, len(rows), func(gtx C, i int) D {
		if r := rows[i]; r.project != nil {
			return a.projectRow(gtx, r)
		}
		return a.sessionRow(gtx, rows[i])
	})
}

func (a *App) projectRow(gtx C, r treeRow) D {
	th, p := a.th, r.project
	in := layout.Inset{Top: 16, Bottom: 4, Left: 8}
	if r.first {
		in.Top = 0
	}
	return in.Layout(gtx, func(gtx C) D {
		return p.btn.Layout(gtx, func(gtx C) D {
			c := th.Muted
			if p.btn.Hovered() {
				c = th.Text
			}
			dir := 0
			if !p.Open {
				dir = 1
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(th.Label(12, p.Name, c, font.SemiBold).Layout),
				ui.HSpace(6),
				layout.Rigid(func(gtx C) D { return ui.SmallChevron(gtx, c, dir) }),
			)
		})
	})
}

func (a *App) sessionRow(gtx C, r treeRow) D {
	th, s := a.th, r.session
	indentDp := unit.Dp(22 * r.depth)
	indent := gtx.Dp(indentDp)
	// Lay the row out first: the guide needs its height.
	macro := op.Record(gtx.Ops)
	d := layout.Inset{Left: indentDp, Top: 1, Bottom: 1}.Layout(gtx, func(gtx C) D {
		return a.sessionCard(gtx, s)
	})
	call := macro.Stop()
	if r.depth > 0 {
		x := indent - gtx.Dp(12)
		lw, h := gtx.Dp(1), d.Size.Y
		if r.last {
			mid := h / 2
			paint.FillShape(gtx.Ops, th.BorderHi, clip.Rect{Min: image.Pt(x, 0), Max: image.Pt(x+lw, mid)}.Op())
			paint.FillShape(gtx.Ops, th.BorderHi, clip.Rect{Min: image.Pt(x, mid), Max: image.Pt(x+gtx.Dp(8), mid+lw)}.Op())
		} else {
			paint.FillShape(gtx.Ops, th.BorderHi, clip.Rect{Min: image.Pt(x, 0), Max: image.Pt(x+lw, h)}.Op())
		}
	}
	call.Add(gtx.Ops)
	return d
}

func (a *App) sessionCard(gtx C, s *Session) D {
	th := a.th
	return s.btn.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		bg := th.Sidebar
		switch {
		case s == a.cur && a.view == viewChat:
			bg = th.Selected
		case s.btn.Hovered():
			bg = th.Hover
		}
		size := unit.Sp(14)
		if s.parent != nil {
			size = 13
		}
		return ui.Card(gtx, bg, 8, layout.Inset{Top: 7, Bottom: 7, Left: 10, Right: 10}, func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx C) D { return ui.OneLine(gtx, th.Label(size, s.Title, th.Text, font.Normal)) }),
				layout.Rigid(func(gtx C) D {
					if len(s.Children) == 0 {
						return D{}
					}
					// Folds the subagents without selecting the session.
					return s.chev.Layout(gtx, func(gtx C) D {
						c := th.Muted
						if s.chev.Hovered() {
							c = th.Text
						}
						dir := 0
						if !s.Open {
							dir = 1
						}
						return layout.Inset{Left: 6, Right: 2}.Layout(gtx, func(gtx C) D { return ui.SmallChevron(gtx, c, dir) })
					})
				}),
				layout.Rigid(func(gtx C) D {
					if !s.Busy {
						return D{}
					}
					return layout.Inset{Left: 8}.Layout(gtx, func(gtx C) D { return ui.Dot(gtx, ui.Pulse(gtx, th.Muted), 7) })
				}),
			)
		})
	})
}
