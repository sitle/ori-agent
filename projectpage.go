package oriagent

import (
	"context"
	"io"
	"strings"

	"gioui.org/font"
	"gioui.org/io/clipboard"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/preview"
	"github.com/sitle/ori-agent/ui"
)

const viewProject = "project"

// Permissions are what the agent may do in a project's sessions; they are
// given to it in Turn.AllowCommands and Turn.AskEdits.
type Permissions struct {
	AllowCommands bool
	AskEdits      bool
}

// ProjectEdit is what the user changed on a project's page.
type ProjectEdit struct {
	Name        string
	Description string
	Permissions *Permissions // nil when the project has none of its own
}

// ProjectFile is a row of the Files section of a project's page, such as
// a file the user attached in its sessions.
type ProjectFile struct {
	Name    string
	Detail  string // size, date, version, checksum…
	Path    string // opened in the preview pane on click; empty: not clickable
	Warning string // shown in red when set, such as "changed since added"
}

// projectPageState is the page of one project: name, description,
// permissions, folder, files, and removal.
type projectPageState struct {
	p                    *Project
	name, desc           widget.Editor
	allowCmds, askEdits  widget.Bool
	save, cancel, remove widget.Clickable
	copyDir              widget.Clickable
	list                 widget.List
	busy, confirmRemove  bool
	err                  string

	files        []ProjectFile
	fileBtns     []widget.Clickable
	filesLoading bool
}

// openProject shows the page of p, with its current values.
func (a *App) openProject(p *Project) {
	f := &a.projPage
	f.p, f.err, f.confirmRemove = p, "", false
	f.name.SingleLine = true
	f.name.SetText(p.Name)
	f.desc.SetText(p.Description)
	if p.Permissions != nil {
		f.allowCmds.Value, f.askEdits.Value = p.Permissions.AllowCommands, p.Permissions.AskEdits
	}
	f.files, f.fileBtns = nil, nil
	if a.cfg.ProjectFiles != nil {
		// Listing may read the disk (checksums): on a goroutine of its own.
		f.filesLoading = true
		go func() {
			files := a.cfg.ProjectFiles(p)
			a.update(func() {
				if f.p == p {
					f.files, f.fileBtns, f.filesLoading = files, make([]widget.Clickable, len(files)), false
				}
			})
		}()
	}
	a.view = viewProject
}

// saveProject applies the edits, through Config.OnUpdateProject when set
// (on a goroutine of its own; an error stays on the page).
func (a *App) saveProject() {
	f := &a.projPage
	p := f.p
	edit := ProjectEdit{Name: strings.TrimSpace(f.name.Text()), Description: strings.TrimSpace(f.desc.Text())}
	if edit.Name == "" {
		f.err = a.t("project.nameNeeded")
		return
	}
	if p.Permissions != nil {
		edit.Permissions = &Permissions{AllowCommands: f.allowCmds.Value, AskEdits: f.askEdits.Value}
	}
	apply := func(e ProjectEdit) {
		p.Name, p.Description = e.Name, e.Description
		if e.Permissions != nil {
			perms := *e.Permissions
			p.Permissions = &perms
		}
		a.view = viewChat
		a.notify(a.t("project.saved", p.Name))
	}
	if a.cfg.OnUpdateProject == nil {
		apply(edit)
		return
	}
	f.busy, f.err = true, ""
	go func() {
		got, err := a.cfg.OnUpdateProject(context.Background(), p, edit)
		a.update(func() {
			f.busy = false
			if err != nil {
				f.err = err.Error()
				return
			}
			apply(got)
		})
	}()
}

// removeProject takes p out of the sidebar once Config.OnDeleteProject
// agrees; the first click only asks for confirmation.
func (a *App) removeProject() {
	f := &a.projPage
	if !f.confirmRemove {
		f.confirmRemove = true
		return
	}
	p := f.p
	f.busy, f.err = true, ""
	go func() {
		err := a.cfg.OnDeleteProject(context.Background(), p)
		a.update(func() {
			f.busy, f.confirmRemove = false, false
			if err != nil {
				f.err = err.Error()
				return
			}
			a.dropProject(p)
			a.notify(a.t("project.removed", p.Name))
		})
	}()
}

// dropProject removes p and, if the shown session was in it, shows a
// session of the first project left.
func (a *App) dropProject(p *Project) {
	for i, x := range a.projects {
		if x == p {
			a.projects = append(a.projects[:i:i], a.projects[i+1:]...)
			break
		}
	}
	root := a.cur
	for root != nil && root.parent != nil {
		root = root.parent
	}
	if root == nil || root.project == p {
		if len(a.projects) == 0 {
			a.projects = []*Project{NewProject(a.t("session.new"))}
		}
		a.newSessionIn(a.projects[0])
	}
	a.view = viewChat
}

// projectFiles lists the files of the project's page; each row opens its
// file in the preview pane.
func (a *App) projectFiles(gtx C) D {
	th, f := a.th, &a.projPage
	switch {
	case f.filesLoading:
		return th.Label(13, a.t("project.files.loading"), th.Muted, font.Normal).Layout(gtx)
	case len(f.files) == 0:
		return th.Label(13, a.t("project.files.none"), th.Muted, font.Normal).Layout(gtx)
	}
	rows := make([]layout.FlexChild, 0, 2*len(f.files))
	for i, file := range f.files {
		if i > 0 {
			rows = append(rows, layout.Rigid(th.HLine))
		}
		rows = append(rows, layout.Rigid(func(gtx C) D {
			return f.fileBtns[i].Layout(gtx, func(gtx C) D {
				return layout.Inset{Top: 8, Bottom: 8}.Layout(gtx, func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							c := th.Text
							if file.Path != "" && f.fileBtns[i].Hovered() {
								c = th.Accent
							}
							return ui.OneLine(gtx, th.Label(14, file.Name, c, font.Medium))
						}),
						layout.Rigid(func(gtx C) D {
							if file.Detail == "" {
								return D{}
							}
							return layout.Inset{Top: 2}.Layout(gtx, th.Label(12, file.Detail, th.Muted, font.Normal).Layout)
						}),
						layout.Rigid(func(gtx C) D {
							if file.Warning == "" {
								return D{}
							}
							return layout.Inset{Top: 2}.Layout(gtx, th.Label(12, file.Warning, th.DiffDel, font.Medium).Layout)
						}),
					)
				})
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

// permissions of the session s: its project's, or the Configuration's.
func (a *App) permissions(s *Session) Permissions {
	for s != nil && s.parent != nil {
		s = s.parent
	}
	if s != nil && s.project != nil && s.project.Permissions != nil {
		return *s.project.Permissions
	}
	return Permissions{AllowCommands: a.setup.allowCmds.Value, AskEdits: a.setup.askEdits.Value}
}

func (a *App) layoutProject(gtx C) D {
	th, f := a.th, &a.projPage
	p := f.p
	if p == nil {
		a.view = viewChat
		return D{}
	}
	if f.save.Clicked(gtx) && !f.busy {
		a.saveProject()
	}
	if f.cancel.Clicked(gtx) && !f.busy {
		a.view = viewChat
	}
	if f.remove.Clicked(gtx) && !f.busy {
		a.removeProject()
	}
	for i := range f.fileBtns {
		if f.fileBtns[i].Clicked(gtx) && f.files[i].Path != "" {
			if d := preview.Open(f.files[i].Path, ""); d != nil {
				a.showPreview(d)
			} else {
				a.notify(a.t("preview.notFound", f.files[i].Path))
			}
		}
	}
	if f.copyDir.Clicked(gtx) {
		gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(p.WorkDir))})
		a.notify(a.t("toast.pathCopied"))
	}
	button := func(btn *widget.Clickable, txt string, kind ui.ButtonKind) layout.Widget {
		return func(gtx C) D {
			if f.busy {
				gtx = gtx.Disabled()
			}
			return th.Button(gtx, btn, txt, kind)
		}
	}
	save := a.t("project.save")
	if f.busy {
		save = a.t("project.saving")
	}
	remove := a.t("project.remove")
	if f.confirmRemove {
		remove = a.t("project.removeConfirm")
	}
	return th.Page(gtx, &f.list, p.Name, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return th.Section(gtx, a.t("project.form.title"), "", func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx C) D { return th.Field(gtx, a.t("project.name"), &f.name, a.t("project.name.hint")) }),
						ui.VSpace(14),
						layout.Rigid(func(gtx C) D {
							gtx.Constraints.Min.Y = gtx.Dp(110)
							return th.Field(gtx, a.t("project.desc"), &f.desc, a.t("project.desc.hint"))
						}),
						layout.Rigid(func(gtx C) D {
							if p.WorkDir == "" {
								return D{}
							}
							return layout.Inset{Top: 14}.Layout(gtx, func(gtx C) D {
								return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, func(gtx C) D {
										return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
											layout.Rigid(th.Label(12, a.t("project.folder"), th.Muted, font.Medium).Layout),
											ui.VSpace(6),
											layout.Rigid(func(gtx C) D { return ui.OneLine(gtx, th.Mono(13, p.WorkDir, th.Text)) }),
										)
									}),
									ui.HSpace(10),
									layout.Rigid(func(gtx C) D { return th.Button(gtx, &f.copyDir, a.t("project.copyFolder"), ui.Secondary) }),
								)
							})
						}),
					)
				})
			}),
			layout.Rigid(func(gtx C) D {
				if p.Permissions == nil {
					return D{}
				}
				return th.Section(gtx, a.t("project.perms.title"), a.t("project.perms.desc"), func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return th.ToggleRow(gtx, &f.allowCmds, a.t("setup.cmds.title"), a.t("setup.cmds.desc"))
						}),
						layout.Rigid(th.HLine),
						layout.Rigid(func(gtx C) D {
							return th.ToggleRow(gtx, &f.askEdits, a.t("setup.edits.title"), a.t("setup.edits.desc"))
						}),
					)
				})
			}),
			layout.Rigid(func(gtx C) D {
				if a.cfg.ProjectFiles == nil {
					return D{}
				}
				return th.Section(gtx, a.t("project.files.title"), a.t("project.files.desc"), a.projectFiles)
			}),
			layout.Rigid(func(gtx C) D {
				if f.err == "" {
					return D{}
				}
				return layout.Inset{Bottom: 16}.Layout(gtx, th.Label(13, f.err, th.DiffDel, font.Normal).Layout)
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						if a.cfg.OnDeleteProject == nil || p.Permanent {
							return D{}
						}
						return button(&f.remove, remove, ui.Danger)(gtx)
					}),
					layout.Flexed(1, ui.Expand),
					layout.Rigid(button(&f.cancel, a.t("project.cancel"), ui.Secondary)),
					ui.HSpace(10),
					layout.Rigid(button(&f.save, save, ui.Primary)),
				)
			}),
		)
	})
}
