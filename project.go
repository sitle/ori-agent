package oriagent

import (
	"context"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/ui"
)

const viewNewProject = "newproject"

// newProjectState is the form of Config.OnNewProject.
type newProjectState struct {
	name, desc     widget.Editor
	create, cancel widget.Clickable
	list           widget.List
	busy           bool
	err            string
}

// openNewProject shows the form, empty.
func (a *App) openNewProject() {
	f := &a.newProj
	f.name.SingleLine = true
	f.name.SetText("")
	f.desc.SetText("")
	f.err = ""
	a.view = viewNewProject
}

// createProject runs Config.OnNewProject on a goroutine of its own; the new
// project is added to the sidebar with a new session selected in it.
func (a *App) createProject() {
	f := &a.newProj
	name, desc := strings.TrimSpace(f.name.Text()), strings.TrimSpace(f.desc.Text())
	if name == "" {
		f.err = a.t("project.nameNeeded")
		return
	}
	f.busy, f.err = true, ""
	go func() {
		p, err := a.cfg.OnNewProject(context.Background(), name, desc)
		a.update(func() {
			f.busy = false
			if err != nil {
				f.err = err.Error()
				return
			}
			p.Open = true
			a.projects = append(a.projects, p)
			a.newSessionIn(p)
			a.notify(a.t("project.created", p.Name))
		})
	}()
}

func (a *App) layoutNewProject(gtx C) D {
	th, f := a.th, &a.newProj
	if f.create.Clicked(gtx) && !f.busy {
		a.createProject()
	}
	if f.cancel.Clicked(gtx) && !f.busy {
		a.view = viewChat
	}
	button := func(btn *widget.Clickable, txt string, kind ui.ButtonKind) layout.Widget {
		return func(gtx C) D {
			if f.busy {
				gtx = gtx.Disabled()
			}
			return th.Button(gtx, btn, txt, kind)
		}
	}
	create := a.t("project.create")
	if f.busy {
		create = a.t("project.creating")
	}
	return th.Page(gtx, &f.list, a.t("project.new"), func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return th.Section(gtx, a.t("project.form.title"), a.t("project.form.desc"), func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx C) D { return th.Field(gtx, a.t("project.name"), &f.name, a.t("project.name.hint")) }),
						ui.VSpace(14),
						layout.Rigid(func(gtx C) D {
							gtx.Constraints.Min.Y = gtx.Dp(110)
							return th.Field(gtx, a.t("project.desc"), &f.desc, a.t("project.desc.hint"))
						}),
					)
				})
			}),
			layout.Rigid(func(gtx C) D {
				if f.err == "" {
					return D{}
				}
				return layout.Inset{Bottom: 16}.Layout(gtx, th.Label(13, f.err, th.DiffDel, font.Normal).Layout)
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, ui.Expand),
					layout.Rigid(button(&f.cancel, a.t("project.cancel"), ui.Secondary)),
					ui.HSpace(10),
					layout.Rigid(button(&f.create, create, ui.Primary)),
				)
			}),
		)
	})
}

// newSessionIn selects an empty session of p, or adds one at its top.
func (a *App) newSessionIn(p *Project) {
	var s *Session
	for _, x := range p.Sessions {
		if len(x.Messages) == 0 && !x.Busy {
			s = x
			break
		}
	}
	if s == nil {
		s = &Session{Title: a.t("session.new")}
		p.Add(s)
	}
	p.Open, a.cur = true, s
	a.focus, a.view = true, viewChat
}

// sessionWorkDir is the folder given to the agent for s: its project's,
// or the configured one.
func (a *App) sessionWorkDir(s *Session) string {
	for s != nil && s.parent != nil {
		s = s.parent
	}
	if s != nil && s.project != nil && s.project.WorkDir != "" {
		return s.project.WorkDir
	}
	return a.workDir
}
