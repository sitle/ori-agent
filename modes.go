package oriagent

import (
	"context"
	"image"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/ui"
)

// Mode is a working mode of the agent the composer chip offers, such as
// "plan", "edition" or "auto": the application defines them and what they
// mean (Config.Modes). The mode belongs to the project of the session, or
// to the application for the sessions out of projects.
type Mode struct {
	ID    string
	Label string // or a translation key
	Desc  string // one line under the label in the menu; or a translation key
}

// modeState is the chip's menu.
type modeState struct {
	chip, scrim widget.Clickable
	open        bool
	items       []widget.Clickable
	current     string // the application-wide mode
}

// sessionMode is the mode of s: its project's, else the application's.
func (a *App) sessionMode(s *Session) string {
	if s != nil && s.project != nil && s.project.Mode != "" {
		return s.project.Mode
	}
	if a.modes.current == "" && len(a.cfg.Modes) > 0 {
		return a.cfg.Modes[0].ID
	}
	return a.modes.current
}

// setSessionMode records the mode of s's project, or the application's.
func (a *App) setSessionMode(s *Session, id string) {
	if s != nil && s.project != nil {
		s.project.Mode = id
		return
	}
	a.modes.current = id
}

func (a *App) modeByID(id string) (Mode, bool) {
	for _, m := range a.cfg.Modes {
		if m.ID == id {
			return m, true
		}
	}
	return Mode{}, false
}

// chooseMode is the user's choice in the menu: recorded at once, then
// given to Config.OnMode on its own goroutine.
func (a *App) chooseMode(id string) {
	s := a.cur
	prev := a.sessionMode(s)
	a.setSessionMode(s, id)
	if a.cfg.OnMode == nil || prev == id {
		return
	}
	var p *Project
	if s != nil {
		p = s.project
	}
	go func() {
		if err := a.cfg.OnMode(context.Background(), p, id); err != nil {
			a.update(func() {
				// Unless the user chose yet another mode meanwhile.
				if a.sessionMode(s) == id {
					a.setSessionMode(s, prev)
				}
				a.notify(a.t("mode.failed", err.Error()))
			})
		}
	}()
}

// modeChip is the chip of the composer: a menu of Config.Modes, or the
// static ModeLabel when the application gives no modes.
func (a *App) modeChip(gtx C) D {
	th := a.th
	if len(a.cfg.Modes) == 0 {
		if a.cfg.ModeLabel == "-" {
			return D{}
		}
		return layout.Inset{Right: 10}.Layout(gtx, func(gtx C) D { return th.Chip(gtx, a.t(a.cfg.ModeLabel), th.Accent) })
	}
	st := &a.modes
	if len(st.items) != len(a.cfg.Modes) {
		st.items = make([]widget.Clickable, len(a.cfg.Modes))
	}
	if st.chip.Clicked(gtx) {
		st.open = !st.open
	}
	if st.scrim.Clicked(gtx) {
		st.open = false
	}
	for i := range st.items {
		if st.items[i].Clicked(gtx) {
			a.chooseMode(a.cfg.Modes[i].ID)
			st.open = false
		}
	}
	cur := a.sessionMode(a.cur)
	label := cur
	if m, ok := a.modeByID(cur); ok {
		label = a.t(m.Label)
	}
	d := layout.Inset{Right: 10}.Layout(gtx, func(gtx C) D {
		return st.chip.Layout(gtx, func(gtx C) D {
			bg := th.Bg
			if st.chip.Hovered() || st.open {
				bg = th.Hover
			}
			return ui.Bordered(gtx, bg, th.Border, 10, func(gtx C) D {
				return layout.Inset{Top: 3, Bottom: 3, Left: 9, Right: 7}.Layout(gtx, func(gtx C) D {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(th.Label(12, label, th.Accent, font.Medium).Layout),
						ui.HSpace(4),
						layout.Rigid(func(gtx C) D { return ui.Chevron(gtx, th.Accent, 0) }),
					)
				})
			})
		})
	})
	if st.open {
		w := gtx.Dp(380)
		rows := len(a.cfg.Modes)
		h := gtx.Dp(14) + rows*gtx.Dp(54)
		ui.Popover(gtx, &st.scrim, image.Pt(0, -h-gtx.Dp(8)), image.Pt(w, h+gtx.Dp(40)), func(gtx C) D {
			return a.modeMenu(gtx, cur)
		})
	}
	return d
}

func (a *App) modeMenu(gtx C, cur string) D {
	th, st := a.th, &a.modes
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return ui.Bordered(gtx, th.Surface, th.BorderHi, 10, func(gtx C) D {
		return layout.UniformInset(6).Layout(gtx, func(gtx C) D {
			children := make([]layout.FlexChild, 0, len(a.cfg.Modes))
			for i, m := range a.cfg.Modes {
				children = append(children, layout.Rigid(func(gtx C) D {
					btn := &st.items[i]
					return btn.Layout(gtx, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						bg := th.Surface
						switch {
						case m.ID == cur:
							bg = th.Selected
						case btn.Hovered():
							bg = th.Hover
						}
						return ui.Card(gtx, bg, 6, layout.Inset{Top: 7, Bottom: 7, Left: 10, Right: 10}, func(gtx C) D {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(th.Label(13, a.t(m.Label), th.Text, font.SemiBold).Layout),
								layout.Rigid(func(gtx C) D {
									if m.Desc == "" {
										return D{}
									}
									return ui.OneLine(gtx, th.Label(12, a.t(m.Desc), th.Muted, font.Normal))
								}),
							)
						})
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

// SetMode changes the mode of the turn's project (or the application's),
// for the chip, when the agent changes it itself, for instance when the
// user approves a plan. Config.OnMode is not called: the agent keeps it.
func (t *Turn) SetMode(id string) {
	t.app.update(func() {
		s := t.s
		for s.parent != nil {
			s = s.parent
		}
		t.app.setSessionMode(s, id)
	})
}

// Mode is the mode of the session shown.
func (c *Ctl) Mode() string { return c.a.sessionMode(c.a.cur) }

// SetMode changes the mode of the session shown, as the chip does, without
// calling Config.OnMode (a command already applied it).
func (c *Ctl) SetMode(id string) { c.a.setSessionMode(c.a.cur, id) }

// SetContext shows how full the model's context is in the composer: used
// and window in tokens; a window of 0 hides the gauge.
func (t *Turn) SetContext(used, window int) {
	t.app.update(func() {
		s := t.s
		for s.parent != nil {
			s = s.parent // a subagent's context is its own: the gauge is the session's
		}
		if s != t.s {
			return
		}
		s.ContextUsed, s.ContextWindow = used, window
	})
}

// contextGauge is the ring and the percentage of the context used.
func (a *App) contextGauge(gtx C) D {
	th, s := a.th, a.cur
	if s == nil || s.ContextWindow <= 0 {
		return D{}
	}
	frac := float32(s.ContextUsed) / float32(s.ContextWindow)
	if frac > 1 {
		frac = 1
	}
	c := th.Muted
	switch {
	case frac >= 0.9:
		c = th.Danger
	case frac >= 0.7:
		c = th.Warning
	}
	return layout.Inset{Right: 10}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D { return ui.ProgressRing(gtx, 16, 2.5, frac, th.Border, c) }),
			ui.HSpace(5),
			layout.Rigid(th.Label(12, a.t("context.percent", int(frac*100+0.5)), c, font.Medium).Layout),
		)
	})
}
