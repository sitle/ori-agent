package oriagent

import (
	"image"
	"os"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/ui"
)

var setupSteps = []string{"setup.step.welcome", "setup.step.workspace", "setup.step.model", "setup.step.prefs"} // keys

// setupState is the first-run wizard. Choices apply to the running app
// (model, thinking, theme, working directory) but are not persisted.
type setupState struct {
	step       int
	back, next widget.Clickable
	list       widget.List

	dir                 widget.Editor
	browse              widget.Clickable
	allowCmds, askEdits widget.Bool
	thinking, notifyOn  widget.Bool

	model  int
	models []widget.Clickable
	themes [2]widget.Clickable

	langs []widget.Clickable // parallel to the available locales
}

// Device is a row of the profile's "connected devices" section.
type Device struct {
	Name, Detail string
	Current      bool
	revoke       widget.Clickable
}

const (
	pFirst = iota
	pLast
	pEmail
	pService
	pRole
)

type profileState struct {
	fields [5]widget.Editor
	saved  [5]string
	list   widget.List

	emailNotif, weekly widget.Bool
	devices            []*Device

	photo, save, cancel, logout widget.Clickable
}

func (a *App) initSetup() {
	st := &a.setup
	st.dir.SingleLine = true
	st.dir.SetText(a.workDir)
	st.allowCmds.Value, st.askEdits.Value, st.thinking.Value = true, true, true
	st.models = make([]widget.Clickable, len(a.cfg.Models))
}

func (a *App) initProfile() {
	p := &a.profile
	u := a.cfg.User
	p.saved = [5]string{u.First, u.Last, u.Email, u.Service, u.Role}
	for i := range p.fields {
		p.fields[i].SingleLine = true
		p.fields[i].SetText(p.saved[i])
	}
	p.emailNotif.Value = true
	p.devices = a.cfg.Devices
}

// ---- setup wizard ----

func (a *App) layoutSetup(gtx C) D {
	st := &a.setup
	if st.back.Clicked(gtx) && st.step > 0 {
		st.step--
	}
	if st.next.Clicked(gtx) {
		if st.step < len(setupSteps)-1 {
			st.step++
		} else {
			if d := strings.TrimSpace(st.dir.Text()); d != "" {
				if info, err := os.Stat(d); err == nil && info.IsDir() {
					a.workDir = d
				}
			}
			st.step, a.view, a.focus = 0, viewChat, true
			a.notify(a.t("setup.saved"))
		}
	}
	if st.browse.Clicked(gtx) {
		a.pickFolder()
	}
	for i := range st.models {
		if st.models[i].Clicked(gtx) {
			st.model = i
		}
	}
	for i := range st.themes {
		if st.themes[i].Clicked(gtx) {
			a.setDark(i == 1)
		}
	}
	locales := a.tr.Locales()
	if len(st.langs) != len(locales) {
		st.langs = make([]widget.Clickable, len(locales))
	}
	for i := range st.langs {
		if st.langs[i].Clicked(gtx) {
			a.setLocale(locales[i])
		}
	}

	th := a.th
	return th.Page(gtx, &st.list, a.t("view.setup"), func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(a.stepper),
			ui.VSpace(28),
			layout.Rigid(func(gtx C) D { return th.Panel(gtx, a.setupStep) }),
			ui.VSpace(24),
			layout.Rigid(func(gtx C) D {
				next := a.t("setup.next")
				if st.step == len(setupSteps)-1 {
					next = a.t("setup.finish")
				}
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						if st.step == 0 {
							return D{}
						}
						return th.Button(gtx, &st.back, a.t("setup.back"), ui.Secondary)
					}),
					layout.Flexed(1, ui.Expand),
					layout.Rigid(th.Label(12, a.t("setup.stepOf", st.step+1, len(setupSteps)), th.Muted, font.Normal).Layout),
					ui.HSpace(14),
					layout.Rigid(func(gtx C) D { return th.Button(gtx, &st.next, next, ui.Primary) }),
				)
			}),
		)
	})
}

func (a *App) stepper(gtx C) D {
	th := a.th
	var children []layout.FlexChild
	for i, name := range setupSteps {
		if i > 0 {
			done := i <= a.setup.step
			children = append(children, layout.Flexed(1, func(gtx C) D {
				c := th.Border
				if done {
					c = th.Accent
				}
				return layout.Inset{Left: 10, Right: 10}.Layout(gtx, func(gtx C) D {
					w := gtx.Constraints.Max.X
					paint.FillShape(gtx.Ops, c, clip.Rect{Max: image.Pt(w, gtx.Dp(2))}.Op())
					return D{Size: image.Pt(w, gtx.Dp(2))}
				})
			}))
		}
		children = append(children, layout.Rigid(func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D { return a.stepDot(gtx, i) }),
				ui.HSpace(8),
				layout.Rigid(func(gtx C) D {
					c, w := th.Muted, font.Normal
					if i == a.setup.step {
						c, w = th.Text, font.SemiBold
					}
					return th.Label(13, a.t(name), c, w).Layout(gtx)
				}),
			)
		}))
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
}

func (a *App) stepDot(gtx C, i int) D {
	th := a.th
	s := gtx.Dp(26)
	gtx.Constraints = layout.Exact(image.Pt(s, s))
	c := th.Muted
	switch {
	case i < a.setup.step:
		paint.FillShape(gtx.Ops, th.Accent, clip.Ellipse(image.Rect(0, 0, s, s)).Op(gtx.Ops))
		c = th.OnAccent
	case i == a.setup.step:
		paint.FillShape(gtx.Ops, th.UserBubble, clip.Ellipse(image.Rect(0, 0, s, s)).Op(gtx.Ops))
		ui.Ring(gtx, th.Accent, 26)
		c = th.Accent
	default:
		ui.Ring(gtx, th.BorderHi, 26)
	}
	return layout.Center.Layout(gtx, th.Label(12, itoa(i+1), c, font.SemiBold).Layout)
}

func (a *App) stepTitle(gtx C, title, desc string) D {
	th := a.th
	return layout.Inset{Bottom: 16}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(th.Label(20, title, th.Text, font.SemiBold).Layout),
			ui.VSpace(4),
			layout.Rigid(th.Label(13, desc, th.Muted, font.Normal).Layout),
		)
	})
}

func (a *App) setupStep(gtx C) D {
	th, st := a.th, &a.setup
	switch st.step {
	case 0:
		feature := func(title, desc string) layout.FlexChild {
			return layout.Rigid(func(gtx C) D {
				return layout.Inset{Top: 8, Bottom: 8}.Layout(gtx, func(gtx C) D {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return layout.Inset{Top: 5}.Layout(gtx, func(gtx C) D { return ui.Dot(gtx, th.Accent, 10) })
						}),
						ui.HSpace(14),
						layout.Flexed(1, func(gtx C) D {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(th.Label(14, title, th.Text, font.SemiBold).Layout),
								ui.VSpace(2),
								layout.Rigid(th.Label(13, desc, th.Muted, font.Normal).Layout),
							)
						}),
					)
				})
			})
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layout.Center.Layout(gtx, func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D { return a.logo.Layout(gtx, 72, th.Accent) }),
						ui.VSpace(16),
						layout.Rigid(th.Label(22, a.t("setup.welcome.title", a.cfg.AppName), th.Text, font.SemiBold).Layout),
						ui.VSpace(6),
						layout.Rigid(th.Label(14, a.t("setup.welcome.sub"), th.Muted, font.Normal).Layout),
					)
				})
			}),
			ui.VSpace(22),
			feature(a.t("setup.feat1.title"), a.t("setup.feat1.desc")),
			feature(a.t("setup.feat2.title"), a.t("setup.feat2.desc")),
			feature(a.t("setup.feat3.title"), a.t("setup.feat3.desc")),
		)
	case 1:
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return a.stepTitle(gtx, a.t("setup.step.workspace"), a.t("setup.workspace.desc"))
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Flex{Alignment: layout.End}.Layout(gtx,
					layout.Flexed(1, func(gtx C) D { return th.Field(gtx, a.t("setup.dir.label"), &st.dir, a.t("setup.dir.hint")) }),
					ui.HSpace(10),
					layout.Rigid(func(gtx C) D { return th.Button(gtx, &st.browse, a.t("setup.browse"), ui.Secondary) }),
				)
			}),
			ui.VSpace(14),
			layout.Rigid(func(gtx C) D {
				return th.ToggleRow(gtx, &st.allowCmds, a.t("setup.cmds.title"), a.t("setup.cmds.desc"))
			}),
			layout.Rigid(th.HLine),
			layout.Rigid(func(gtx C) D {
				return th.ToggleRow(gtx, &st.askEdits, a.t("setup.edits.title"), a.t("setup.edits.desc"))
			}),
		)
	case 2:
		children := []layout.FlexChild{
			layout.Rigid(func(gtx C) D {
				return a.stepTitle(gtx, a.t("setup.step.model"), a.t("setup.model.desc"))
			}),
		}
		for i, m := range a.cfg.Models {
			children = append(children, layout.Rigid(func(gtx C) D {
				return layout.Inset{Bottom: 10}.Layout(gtx, func(gtx C) D {
					return th.ChoiceCard(gtx, &st.models[i], st.model == i, m.Name, a.t(m.Desc))
				})
			}))
		}
		children = append(children, ui.VSpace(6), layout.Rigid(func(gtx C) D {
			return th.ToggleRow(gtx, &st.thinking, a.t("setup.thinking.title"), a.t("setup.thinking.desc"))
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	default:
		pair := func(btns *[2]widget.Clickable, sel int, a0, a1 string) layout.Widget {
			return func(gtx C) D {
				return layout.Flex{}.Layout(gtx,
					layout.Flexed(1, func(gtx C) D { return th.ChoiceCard(gtx, &btns[0], sel == 0, a0, "") }),
					ui.HSpace(12),
					layout.Flexed(1, func(gtx C) D { return th.ChoiceCard(gtx, &btns[1], sel == 1, a1, "") }),
				)
			}
		}
		dark := 0
		if th.Dark {
			dark = 1
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return a.stepTitle(gtx, a.t("setup.step.prefs"), a.t("setup.prefs.desc"))
			}),
			layout.Rigid(th.Label(12, a.t("setup.theme"), th.Muted, font.Medium).Layout),
			ui.VSpace(6),
			layout.Rigid(pair(&st.themes, dark, a.t("setup.theme.light"), a.t("setup.theme.dark"))),
			ui.VSpace(18),
			layout.Rigid(th.Label(12, a.t("setup.lang"), th.Muted, font.Medium).Layout),
			ui.VSpace(6),
			layout.Rigid(a.languageChoices),
			ui.VSpace(14),
			layout.Rigid(func(gtx C) D {
				return th.ToggleRow(gtx, &st.notifyOn, a.t("setup.notify.title"), a.t("setup.notify.desc"))
			}),
		)
	}
}

// languageChoices lists the available languages, each under its own name,
// two per row; picking one switches the interface at once.
func (a *App) languageChoices(gtx C) D {
	th, st := a.th, &a.setup
	locales := a.tr.Locales()
	cur := a.tr.Locale()
	var rows []layout.FlexChild
	for i := 0; i < len(locales) && i < len(st.langs); i += 2 {
		var cells []layout.FlexChild
		for j := i; j < i+2; j++ {
			if j > i {
				cells = append(cells, ui.HSpace(12))
			}
			if j >= len(locales) {
				cells = append(cells, layout.Flexed(1, ui.Expand))
				continue
			}
			l := locales[j]
			cells = append(cells, layout.Flexed(1, func(gtx C) D {
				return th.ChoiceCard(gtx, &st.langs[j], l == cur, a.languageName(l), l)
			}))
		}
		if i > 0 {
			rows = append(rows, ui.VSpace(10))
		}
		rows = append(rows, layout.Rigid(func(gtx C) D { return layout.Flex{}.Layout(gtx, cells...) }))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

// ---- profile ----

func (a *App) layoutProfile(gtx C) D {
	th, p := a.th, &a.profile
	if p.photo.Clicked(gtx) {
		a.notify(a.t("profile.photoPicker"))
	}
	if p.save.Clicked(gtx) {
		for i := range p.fields {
			p.saved[i] = p.fields[i].Text()
		}
		a.notify(a.t("profile.saved"))
	}
	if p.cancel.Clicked(gtx) {
		for i := range p.fields {
			p.fields[i].SetText(p.saved[i])
		}
		a.notify(a.t("profile.canceled"))
	}
	if p.logout.Clicked(gtx) {
		a.notify(a.t("profile.loggedOut"))
	}
	for i := 0; i < len(p.devices); i++ {
		if d := p.devices[i]; d.revoke.Clicked(gtx) {
			p.devices = append(p.devices[:i:i], p.devices[i+1:]...)
			a.notify(a.t("profile.devices.revoked", d.Name))
			i--
		}
	}

	return th.Page(gtx, &p.list, a.t("view.profile"), func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D { return layout.Inset{Bottom: 28}.Layout(gtx, a.profileCard) }),
			layout.Rigid(func(gtx C) D {
				return th.Section(gtx, a.t("profile.info.title"), a.t("profile.info.desc"), a.profileFields)
			}),
			layout.Rigid(func(gtx C) D {
				return th.Section(gtx, a.t("profile.notif.title"), "", func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return th.ToggleRow(gtx, &p.emailNotif, a.t("profile.emailNotif.title"), a.t("profile.emailNotif.desc"))
						}),
						layout.Rigid(th.HLine),
						layout.Rigid(func(gtx C) D {
							return th.ToggleRow(gtx, &p.weekly, a.t("profile.weekly.title"), a.t("profile.weekly.desc"))
						}),
					)
				})
			}),
			layout.Rigid(func(gtx C) D { return layout.Inset{Bottom: 28}.Layout(gtx, a.usageTiles) }),
			layout.Rigid(func(gtx C) D {
				if len(p.devices) == 0 {
					return D{}
				}
				return th.Section(gtx, a.t("profile.devices.title"), a.t("profile.devices.desc"), a.deviceList)
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D { return th.Button(gtx, &p.logout, a.t("profile.logout"), ui.Danger) }),
					layout.Flexed(1, ui.Expand),
					layout.Rigid(func(gtx C) D { return th.Button(gtx, &p.cancel, a.t("profile.cancel"), ui.Secondary) }),
					ui.HSpace(10),
					layout.Rigid(func(gtx C) D { return th.Button(gtx, &p.save, a.t("profile.save"), ui.Primary) }),
				)
			}),
		)
	})
}

func (a *App) profileCard(gtx C) D {
	th, p := a.th, &a.profile
	return th.Panel(gtx, func(gtx C) D {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D { return th.Avatar(gtx, a.initials(), 64) }),
			ui.HSpace(18),
			layout.Flexed(1, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(th.Label(20, a.fullName(), th.Text, font.SemiBold).Layout),
					ui.VSpace(2),
					layout.Rigid(th.Label(13, p.fields[pEmail].Text(), th.Muted, font.Normal).Layout),
					ui.VSpace(8),
					layout.Rigid(func(gtx C) D {
						var chips []layout.FlexChild
						if s := p.fields[pService].Text(); s != "" {
							chips = append(chips, layout.Rigid(func(gtx C) D { return th.Chip(gtx, s, th.Accent) }), ui.HSpace(6))
						}
						if r := p.fields[pRole].Text(); r != "" {
							chips = append(chips, layout.Rigid(func(gtx C) D { return th.Chip(gtx, r, th.Muted) }))
						}
						return layout.Flex{}.Layout(gtx, chips...)
					}),
				)
			}),
			layout.Rigid(func(gtx C) D { return th.Button(gtx, &p.photo, a.t("profile.photo"), ui.Secondary) }),
		)
	})
}

func (a *App) profileFields(gtx C) D {
	th, f := a.th, &a.profile.fields
	row := func(l0 string, e0 *widget.Editor, l1 string, e1 *widget.Editor) layout.Widget {
		return func(gtx C) D {
			return layout.Flex{}.Layout(gtx,
				layout.Flexed(1, func(gtx C) D { return th.Field(gtx, l0, e0, "") }),
				ui.HSpace(16),
				layout.Flexed(1, func(gtx C) D { return th.Field(gtx, l1, e1, "") }),
			)
		}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(row(a.t("profile.first"), &f[pFirst], a.t("profile.last"), &f[pLast])),
		ui.VSpace(14),
		layout.Rigid(func(gtx C) D { return th.Field(gtx, a.t("profile.email"), &f[pEmail], "") }),
		ui.VSpace(14),
		layout.Rigid(row(a.t("profile.service"), &f[pService], a.t("profile.role"), &f[pRole])),
	)
}

// usageTiles shows live counts from the sessions.
func (a *App) usageTiles(gtx C) D {
	th := a.th
	all := a.allSessions()
	msgs, reads := 0, 0
	for _, s := range all {
		for _, m := range s.Messages {
			switch {
			case m.Kind == KindUser:
				msgs++
			case m.Kind == KindTool && m.ToolName == "Read":
				reads++
			}
		}
	}
	tile := func(n int, label string) layout.FlexChild {
		return layout.Flexed(1, func(gtx C) D {
			return th.Panel(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(th.Label(28, itoa(n), th.Text, font.SemiBold).Layout),
					layout.Rigid(th.Label(13, label, th.Muted, font.Normal).Layout),
				)
			})
		})
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(th.Label(16, a.t("profile.usage"), th.Text, font.SemiBold).Layout),
		ui.VSpace(12),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{}.Layout(gtx,
				tile(len(all), a.t("profile.usage.sessions")), ui.HSpace(12),
				tile(msgs, a.t("profile.usage.messages")), ui.HSpace(12),
				tile(reads, a.t("profile.usage.reads")),
			)
		}),
	)
}

func (a *App) deviceList(gtx C) D {
	th := a.th
	var children []layout.FlexChild
	for i, d := range a.profile.devices {
		if i > 0 {
			children = append(children, layout.Rigid(th.HLine))
		}
		children = append(children, layout.Rigid(func(gtx C) D {
			return layout.Inset{Top: 10, Bottom: 10}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						s := gtx.Dp(34)
						gtx.Constraints = layout.Exact(image.Pt(s, s))
						paint.FillShape(gtx.Ops, th.Selected, clip.UniformRRect(image.Rect(0, 0, s, s), gtx.Dp(8)).Op(gtx.Ops))
						initial := "?"
						if d.Name != "" {
							initial = string([]rune(d.Name)[:1])
						}
						return layout.Center.Layout(gtx, th.Label(14, initial, th.Text, font.SemiBold).Layout)
					}),
					ui.HSpace(12),
					layout.Flexed(1, func(gtx C) D {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(th.Label(14, d.Name, th.Text, font.Medium).Layout),
							layout.Rigid(th.Label(12, d.Detail, th.Muted, font.Normal).Layout),
						)
					}),
					layout.Rigid(func(gtx C) D {
						if d.Current {
							return th.Chip(gtx, a.t("profile.devices.current"), th.Accent)
						}
						return d.revoke.Layout(gtx, func(gtx C) D {
							c := th.DiffDel
							if d.revoke.Hovered() {
								c = th.Text
							}
							return th.Label(13, a.t("profile.devices.revoke"), c, font.Medium).Layout(gtx)
						})
					}),
				)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}
