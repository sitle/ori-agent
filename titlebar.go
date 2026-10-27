package oriagent

import (
	"image"
	"image/color"
	"io"
	"strings"

	"gioui.org/font"
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/sitle/ori-agent/ui"
)

// The window is created undecorated and this bar replaces the native one, as
// in the Claude Code desktop client. Dragging the bar moves the window
// (system.ActionMove); the caption buttons perform the window actions.

func (a *App) handleTitleBar(gtx C) {
	if a.winBtns[ui.WinMinimize].Clicked(gtx) {
		a.w.Perform(system.ActionMinimize)
	}
	if a.winBtns[ui.WinMaximize].Clicked(gtx) {
		if a.maximized {
			a.w.Perform(system.ActionUnmaximize)
		} else {
			a.w.Perform(system.ActionMaximize)
		}
	}
	if a.winBtns[ui.WinClose].Clicked(gtx) {
		a.w.Perform(system.ActionClose)
	}
	if a.backBtn.Clicked(gtx) && a.view != viewChat {
		a.view, a.focus = viewChat, true
	}
	if a.themeBtn.Clicked(gtx) {
		a.setDark(!a.th.Dark)
	}
	if a.sideBtn.Clicked(gtx) {
		a.sideOpen, a.focus = !a.sideOpen, true
	}
	a.handleMenu(gtx)
	a.handlePomodoro(gtx)
	a.handleVoice(gtx)
}

func (a *App) titleBar(gtx C) D {
	th := a.th
	w, h := gtx.Constraints.Max.X, gtx.Dp(40)
	gtx.Constraints = layout.Exact(image.Pt(w, h))
	paint.FillShape(gtx.Ops, th.Sidebar, clip.Rect{Max: image.Pt(w, h)}.Op())
	// The whole bar drags the window; buttons laid out on top take precedence.
	area := clip.Rect{Max: image.Pt(w, h)}.Push(gtx.Ops)
	system.ActionInputOp(system.ActionMove).Add(gtx.Ops)
	area.Pop()
	paint.FillShape(gtx.Ops, th.Border, clip.Rect{Min: image.Pt(0, h-gtx.Dp(1)), Max: image.Pt(w, h)}.Op())
	// Center relaxes the exact height so items keep their natural size.
	return layout.Center.Layout(gtx, a.titleBarItems)
}

func (a *App) titleBarItems(gtx C) D {
	th := a.th
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		ui.HSpace(8),
		layout.Rigid(func(gtx C) D {
			return th.IconButton(gtx, &a.sideBtn, th.Sidebar, true, func(gtx C, c color.NRGBA) D { return ui.Sidebar(gtx, c, a.sideOpen) })
		}),
		layout.Rigid(func(gtx C) D {
			return th.IconButton(gtx, &a.backBtn, th.Sidebar, a.view != viewChat, func(gtx C, c color.NRGBA) D { return ui.Chevron(gtx, c, -1) })
		}),
		layout.Rigid(func(gtx C) D {
			return th.IconButton(gtx, &a.fwdBtn, th.Sidebar, false, func(gtx C, c color.NRGBA) D { return ui.Chevron(gtx, c, 1) })
		}),
		ui.HSpace(14),
		layout.Rigid(func(gtx C) D { return a.logo.Layout(gtx, 20, th.Accent) }),
		ui.HSpace(8),
		layout.Rigid(th.Label(13, a.cfg.AppName, th.Muted, font.Medium).Layout),
		ui.HSpace(14),
		layout.Rigid(func(gtx C) D {
			s := image.Pt(gtx.Dp(1), gtx.Dp(16))
			paint.FillShape(gtx.Ops, th.BorderHi, clip.Rect{Max: s}.Op())
			return D{Size: s}
		}),
		ui.HSpace(14),
		layout.Flexed(1, a.titleArea),
		layout.Rigid(a.voicePill),
		layout.Rigid(a.panelButtons),
		layout.Rigid(func(gtx C) D {
			if a.cfg.Disable.Pomodoro {
				return D{}
			}
			return layout.Inset{Right: 6}.Layout(gtx, a.pomodoroPill)
		}),
		layout.Rigid(func(gtx C) D {
			if a.cfg.Disable.ThemeToggle {
				return D{}
			}
			return layout.Inset{Right: 6}.Layout(gtx, func(gtx C) D {
				return a.themeBtn.Layout(gtx, func(gtx C) D {
					txt, c := a.t("theme.light"), th.Muted
					if !th.Dark {
						txt = a.t("theme.dark")
					}
					if a.themeBtn.Hovered() {
						c = th.Text
					}
					return th.Chip(gtx, txt, c)
				})
			})
		}),
		layout.Rigid(func(gtx C) D { return th.Chip(gtx, a.model().Name, th.Muted) }),
		ui.HSpace(10),
		layout.Rigid(func(gtx C) D { return a.winButton(gtx, ui.WinMinimize) }),
		layout.Rigid(func(gtx C) D { return a.winButton(gtx, ui.WinMaximize) }),
		layout.Rigid(func(gtx C) D { return a.winButton(gtx, ui.WinClose) }),
	)
}

// titleArea shows the current location; in chat it is a dropdown of
// session actions. It always reports the full width so the rest of the bar
// stays right-aligned.
func (a *App) titleArea(gtx C) D {
	th := a.th
	full := gtx.Constraints.Max.X
	title := a.viewTitle()
	var d D
	switch {
	case a.renaming:
		d = a.renameField(gtx)
	case a.view != viewChat:
		d = layout.Inset{Left: 8}.Layout(gtx, th.Label(14, title, th.Text, font.SemiBold).Layout)
	default:
		gtx.Constraints.Min.X = 0 // size the button to its title
		d = a.titleBtn.Layout(gtx, func(gtx C) D {
			bg := th.Sidebar
			if a.titleBtn.Hovered() || a.menuOpen {
				bg = th.Hover
			}
			return ui.Card(gtx, bg, 6, layout.Inset{Top: 4, Bottom: 4, Left: 8, Right: 8}, func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						gtx.Constraints.Max.X -= gtx.Dp(24) // room for the chevron
						return ui.OneLine(gtx, th.Label(14, title, th.Text, font.SemiBold))
					}),
					ui.HSpace(8),
					layout.Rigid(func(gtx C) D { return ui.Chevron(gtx, th.Muted, 0) }),
				)
			})
		})
		if a.menuOpen {
			ui.Popover(gtx, &a.menuScrim, image.Pt(0, d.Size.Y+gtx.Dp(6)), image.Pt(gtx.Dp(260), gtx.Dp(400)), a.layoutMenu)
		}
	}
	d.Size.X = full
	return d
}

func (a *App) winButton(gtx C, kind int) D {
	th := a.th
	btn := &a.winBtns[kind]
	return btn.Layout(gtx, func(gtx C) D {
		sz := image.Pt(gtx.Dp(46), gtx.Dp(40))
		c := th.Muted
		if btn.Hovered() {
			bg := th.Hover
			c = th.Text
			if kind == ui.WinClose {
				bg, c = th.Danger, th.OnStrong
			}
			paint.FillShape(gtx.Ops, bg, clip.Rect{Max: sz}.Op())
		}
		gtx.Constraints = layout.Exact(sz)
		return layout.Center.Layout(gtx, func(gtx C) D { return ui.WindowIcon(gtx, kind, c, th.Sidebar, a.maximized) })
	})
}

// ---- session menu ----

const (
	menuRename = iota
	menuNew
	menuCopyPath
	menuClear
	menuArchive
	menuDelete
)

func (a *App) layoutMenu(gtx C) D {
	items := []ui.MenuItem{
		{Btn: &a.menuItems[menuRename], Label: a.t("menu.rename")},
		{Btn: &a.menuItems[menuNew], Label: a.t("menu.new")},
		{Btn: &a.menuItems[menuCopyPath], Label: a.t("menu.copyPath")},
		{Btn: &a.menuItems[menuClear], Label: a.t("menu.clear")},
		{Separator: true},
		{Btn: &a.menuItems[menuArchive], Label: a.t("menu.archive"), Danger: true},
	}
	if a.cfg.OnDeleteSession != nil && a.cur.parent == nil {
		items = append(items, ui.MenuItem{Btn: &a.menuItems[menuDelete], Label: a.t("menu.delete"), Danger: true})
	}
	return a.th.Menu(gtx, items)
}

func (a *App) handleMenu(gtx C) {
	if a.titleBtn.Clicked(gtx) {
		a.menuOpen = !a.menuOpen
	}
	if a.menuScrim.Clicked(gtx) {
		a.menuOpen = false
	}
	for i := range a.menuItems {
		if !a.menuItems[i].Clicked(gtx) {
			continue
		}
		a.menuOpen = false
		s := a.cur
		switch i {
		case menuRename:
			a.renaming, a.renameSeen, a.renameFrames = true, false, 0
			a.renameEd.SetText(s.Title)
			a.renameEd.SetCaret(len([]rune(s.Title)), 0) // select all
		case menuNew:
			a.newSession()
		case menuCopyPath:
			gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(a.sessionWorkDir(s)))})
			a.notify(a.t("toast.pathCopied"))
		case menuClear:
			s.Messages = nil
			a.notify(a.t("toast.cleared"))
		case menuArchive:
			a.archive(s)
		case menuDelete:
			a.askDelete(s)
		}
	}
	if !a.renaming {
		return
	}
	for {
		ev, ok := a.renameEd.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			a.commitRename()
		}
	}
	// Clicking elsewhere also validates the new name, once the field has
	// actually received focus (which takes one frame after it is laid out).
	if a.renameFrames < 2 {
		return
	}
	if gtx.Focused(&a.renameEd) {
		a.renameSeen = true
	} else if a.renameSeen {
		a.commitRename()
	}
}

func (a *App) commitRename() {
	if !a.renaming {
		return
	}
	if t := strings.TrimSpace(a.renameEd.Text()); t != "" {
		a.cur.Title = t
	}
	a.renaming, a.focus = false, true
}

func (a *App) renameField(gtx C) D {
	th := a.th
	if a.renameFrames == 0 {
		gtx.Execute(key.FocusCmd{Tag: &a.renameEd})
	}
	a.renameFrames++
	gtx.Constraints.Min.X = 0
	gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(320))
	return ui.Bordered(gtx, th.Bg, th.Accent, 6, func(gtx C) D {
		return layout.Inset{Top: 4, Bottom: 4, Left: 8, Right: 8}.Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			e := material.Editor(th.Material, &a.renameEd, a.t("rename.hint"))
			e.TextSize = 14
			e.Color, e.HintColor = th.Text, th.Muted
			e.Font.Weight = font.SemiBold
			return e.Layout(gtx)
		})
	})
}
