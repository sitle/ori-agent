package oriagent

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/i18n"
	"github.com/sitle/ori-agent/ui"
)

const (
	pomoFocus = iota
	pomoShort
	pomoLong
)

var pomoNames = [3]string{"pomo.focus", "pomo.short", "pomo.long"} // keys

// Pomodoro is a countdown driven by the wall clock: focus periods with a
// long pause every four.
type Pomodoro struct {
	tr        i18n.Translator
	dur       [3]time.Duration
	mode      int
	running   bool
	remaining time.Duration // while paused
	endAt     time.Time     // while running
	cycles    int           // completed focus periods

	open                bool
	pill, scrim         widget.Clickable
	toggle, reset, skip widget.Clickable
	modes               [3]widget.Clickable
}

func (p *Pomodoro) init(c PomodoroConfig, tr i18n.Translator) {
	p.tr = tr
	p.dur = [3]time.Duration{c.Focus, c.Short, c.Long}
	p.remaining = p.dur[pomoFocus]
}

func (p *Pomodoro) Running() bool { return p.running }
func (p *Pomodoro) Mode() string  { return p.tr.T(pomoNames[p.mode]) }
func (p *Pomodoro) Cycles() int   { return p.cycles }

// Left is the time remaining in the current period.
func (p *Pomodoro) Left(now time.Time) time.Duration {
	if p.running {
		return max(0, p.endAt.Sub(now))
	}
	return p.remaining
}

func (p *Pomodoro) setMode(m int) { p.mode, p.running, p.remaining = m, false, p.dur[m] }

func (p *Pomodoro) Start(now time.Time) {
	if !p.running {
		p.running, p.endAt = true, now.Add(p.remaining)
	}
}

func (p *Pomodoro) Pause(now time.Time) {
	if p.running {
		p.running, p.remaining = false, p.Left(now)
	}
}

func (p *Pomodoro) Reset() { p.setMode(p.mode) }

// Next moves to the following period and returns a message announcing it.
func (p *Pomodoro) Next() string {
	if p.mode == pomoFocus {
		p.cycles++
		if p.cycles%4 == 0 {
			p.setMode(pomoLong)
			return p.tr.T("pomo.toLong")
		}
		p.setMode(pomoShort)
		return p.tr.T("pomo.toShort")
	}
	p.setMode(pomoFocus)
	return p.tr.T("pomo.toFocus")
}

func (a *App) handlePomodoro(gtx C) {
	p := &a.pomo
	if p.running && !gtx.Now.Before(p.endAt) {
		a.notify(p.Next())
	}
	if p.pill.Clicked(gtx) {
		p.open = !p.open
	}
	if p.scrim.Clicked(gtx) {
		p.open = false
	}
	if p.toggle.Clicked(gtx) {
		if p.running {
			p.Pause(gtx.Now)
		} else {
			p.Start(gtx.Now)
		}
	}
	if p.reset.Clicked(gtx) {
		p.Reset()
	}
	if p.skip.Clicked(gtx) {
		a.notify(p.Next())
	}
	for i := range p.modes {
		if p.modes[i].Clicked(gtx) {
			p.setMode(i)
		}
	}
}

func clock(d time.Duration) string {
	s := int(math.Ceil(d.Seconds()))
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

func (a *App) pomoFg() color.NRGBA {
	if a.pomo.mode == pomoFocus {
		return a.th.Accent
	}
	return a.th.Success
}

// pomodoroPill sits in the title bar: a small ring and the time left.
func (a *App) pomodoroPill(gtx C) D {
	th, p := a.th, &a.pomo
	left := p.Left(gtx.Now)
	if p.running {
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(250 * time.Millisecond)})
	}
	d := p.pill.Layout(gtx, func(gtx C) D {
		bg := th.Bg
		if p.pill.Hovered() || p.open {
			bg = th.Hover
		}
		c := th.Muted
		if p.running {
			c = th.Text
		}
		return ui.Bordered(gtx, bg, th.Border, 10, func(gtx C) D {
			return layout.Inset{Top: 3, Bottom: 3, Left: 7, Right: 9}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return ui.ProgressRing(gtx, 14, 2, float32(left)/float32(p.dur[p.mode]), th.BorderHi, a.pomoFg())
					}),
					ui.HSpace(6),
					layout.Rigid(th.Mono(12, clock(left), c).Layout),
				)
			})
		})
	})
	if p.open {
		w := gtx.Dp(340)
		ui.Popover(gtx, &p.scrim, image.Pt(d.Size.X-w, d.Size.Y+gtx.Dp(8)), image.Pt(w, gtx.Dp(400)), func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return a.layoutPomodoro(gtx)
		})
	}
	return d
}

func (a *App) layoutPomodoro(gtx C) D {
	th, p := a.th, &a.pomo
	left := p.Left(gtx.Now)
	return ui.Bordered(gtx, th.Surface, th.BorderHi, 12, func(gtx C) D {
		return layout.UniformInset(16).Layout(gtx, func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return layout.Flex{}.Layout(gtx, a.modeTab(0), ui.HSpace(4), a.modeTab(1), ui.HSpace(4), a.modeTab(2))
				}),
				ui.VSpace(18),
				layout.Rigid(func(gtx C) D {
					return layout.Stack{Alignment: layout.Center}.Layout(gtx,
						layout.Stacked(func(gtx C) D {
							return ui.ProgressRing(gtx, 140, 7, float32(left)/float32(p.dur[p.mode]), th.Border, a.pomoFg())
						}),
						layout.Stacked(func(gtx C) D {
							return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(th.Mono(30, clock(left), th.Text).Layout),
								layout.Rigid(th.Label(12, p.Mode(), th.Muted, font.Medium).Layout),
							)
						}),
					)
				}),
				ui.VSpace(18),
				layout.Rigid(func(gtx C) D {
					label := a.t("pomo.start")
					switch {
					case p.running:
						label = a.t("pomo.pause")
					case left < p.dur[p.mode]:
						label = a.t("pomo.resume")
					}
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D { return th.Button(gtx, &p.toggle, label, ui.Primary) }),
						ui.HSpace(8),
						layout.Rigid(func(gtx C) D { return th.Button(gtx, &p.reset, a.t("pomo.reset"), ui.Secondary) }),
						ui.HSpace(8),
						layout.Rigid(func(gtx C) D { return th.Button(gtx, &p.skip, a.t("pomo.skip"), ui.Secondary) }),
					)
				}),
				ui.VSpace(16),
				layout.Rigid(func(gtx C) D {
					var dots []layout.FlexChild
					for i := 0; i < 4; i++ {
						done := i < p.cycles%4
						dots = append(dots, layout.Rigid(func(gtx C) D {
							return layout.Inset{Right: 6}.Layout(gtx, func(gtx C) D {
								if done {
									return ui.Dot(gtx, th.Accent, 9)
								}
								return ui.Ring(gtx, th.BorderHi, 9)
							})
						}))
					}
					dots = append(dots, ui.HSpace(4), layout.Rigid(th.Label(12, a.t("pomo.cycles", p.cycles), th.Muted, font.Normal).Layout))
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx, dots...)
				}),
			)
		})
	})
}

func (a *App) modeTab(i int) layout.FlexChild {
	return layout.Rigid(func(gtx C) D {
		th, btn := a.th, &a.pomo.modes[i]
		return btn.Layout(gtx, func(gtx C) D {
			bg, c := th.Surface, th.Muted
			switch {
			case a.pomo.mode == i:
				bg, c = th.Selected, th.Text
			case btn.Hovered():
				bg = th.Hover
			}
			return ui.Card(gtx, bg, 6, layout.Inset{Top: 5, Bottom: 5, Left: 9, Right: 9}, th.Label(12, a.t(pomoNames[i]), c, font.Medium).Layout)
		})
	})
}
