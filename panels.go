package oriagent

import (
	"image/color"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/i18n"
	"github.com/sitle/ori-agent/ui"
)

// Panel is a pane of the application at the right of the chat, about the
// session shown: a supervision of the agents, a summary of the work… Its
// button in the title bar opens and closes it; it takes the place of the
// preview while open. Layout draws the content of a scrollable column; it
// is called at each frame, and twice a second while a turn runs in the
// session, so that it can show live state.
type Panel struct {
	ID, Title string                                                    // Title: text or translation key
	Icon      func(gtx layout.Context, c color.NRGBA) layout.Dimensions // optional, before the title
	Layout    func(gtx layout.Context, th *ui.Theme, tr i18n.Translator, s *Session) layout.Dimensions
}

// panelRefresh is how often an open panel is drawn again during a turn.
const panelRefresh = 500 * time.Millisecond

type panelState struct {
	open  string // ID of the open panel; empty: none
	btns  []widget.Clickable
	close widget.Clickable
	list  widget.List
}

func (a *App) handlePanels(gtx C) {
	if len(a.panels.btns) != len(a.cfg.Panels) {
		a.panels.btns = make([]widget.Clickable, len(a.cfg.Panels))
	}
	for i, p := range a.cfg.Panels {
		if a.panels.btns[i].Clicked(gtx) {
			a.togglePanel(p.ID)
		}
	}
	if a.panels.close.Clicked(gtx) {
		a.panels.open = ""
	}
}

// togglePanel opens the panel id, or closes it when it is open; an id not
// in Config.Panels does nothing.
func (a *App) togglePanel(id string) {
	if a.panels.open == id {
		a.panels.open = ""
		return
	}
	for _, p := range a.cfg.Panels {
		if p.ID == id && p.Layout != nil {
			a.panels.open = id
			a.view = viewChat
			return
		}
	}
}

// openPanel is the open panel, nil when none is.
func (a *App) openPanel() *Panel {
	for i := range a.cfg.Panels {
		if a.cfg.Panels[i].ID == a.panels.open {
			return &a.cfg.Panels[i]
		}
	}
	return nil
}

// panelButtons are the buttons of the panels in the title bar.
func (a *App) panelButtons(gtx C) D {
	th := a.th
	var children []layout.FlexChild
	for i := range a.cfg.Panels {
		p, btn := &a.cfg.Panels[i], &a.panels.btns[i]
		children = append(children, layout.Rigid(func(gtx C) D {
			return layout.Inset{Right: 6}.Layout(gtx, func(gtx C) D {
				return btn.Layout(gtx, func(gtx C) D {
					c := th.Muted
					if btn.Hovered() || a.panels.open == p.ID {
						c = th.Text
					}
					if a.panels.open == p.ID {
						c = th.Accent
					}
					return th.Chip(gtx, a.t(p.Title), c)
				})
			})
		}))
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
}

// layoutPanel draws the open panel: its title, a close button and the
// application's content, scrollable.
func (a *App) layoutPanel(gtx C, p *Panel) D {
	th := a.th
	if root := a.cur; root != nil {
		for root.parent != nil {
			root = root.parent
		}
		if root.Busy {
			gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(panelRefresh)})
		}
	}
	ui.Fill(gtx, th.Bg, 0)
	a.panels.list.Axis = layout.Vertical
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Inset{Top: 12, Bottom: 8, Left: 18, Right: 12}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						if p.Icon == nil {
							return D{}
						}
						return layout.Inset{Right: 8}.Layout(gtx, func(gtx C) D { return p.Icon(gtx, th.Text) })
					}),
					layout.Flexed(1, th.Label(14, a.t(p.Title), th.Text, font.Medium).Layout),
					layout.Rigid(func(gtx C) D {
						return a.panels.close.Layout(gtx, func(gtx C) D {
							c, bg := th.Muted, color.NRGBA{}
							if a.panels.close.Hovered() {
								c, bg = th.Text, th.Hover
							}
							return ui.Card(gtx, bg, 6, layout.UniformInset(9), func(gtx C) D { return ui.Close(gtx, c) })
						})
					}),
				)
			})
		}),
		layout.Rigid(th.HLine),
		layout.Flexed(1, func(gtx C) D {
			return a.panels.list.List.Layout(gtx, 1, func(gtx C, _ int) D {
				return layout.Inset{Top: 14, Bottom: 24, Left: 18, Right: 18}.Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return p.Layout(gtx, th, a.tr, a.cur)
				})
			})
		}),
	)
}
