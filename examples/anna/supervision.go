package main

import (
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"

	oriagent "github.com/sitle/ori-agent"
	"github.com/sitle/ori-agent/i18n"
	"github.com/sitle/ori-agent/sim"
	"github.com/sitle/ori-agent/ui"
)

// supervisionPanel shows how an application adds a panel next to the chat
// (Config.Panels): the state of the session shown, read from the public
// API only, redrawn while the agent works.
var supervisionPanel = oriagent.Panel{
	ID:     "supervision",
	Title:  "sup.title",
	Layout: supervision,
}

type (
	C = layout.Context
	D = layout.Dimensions
)

func supervision(gtx C, th *ui.Theme, tr i18n.Translator, s *oriagent.Session) D {
	for s.Parent() != nil {
		s = s.Parent() // a subagent's view shows its session's state
	}
	var tasks []oriagent.Task
	tools := 0
	for _, m := range s.Messages {
		switch m.Kind {
		case oriagent.KindTasks:
			tasks = m.Tasks // the last list is the current one
		case oriagent.KindTool:
			tools++
		}
	}
	state, c := tr.T("sup.idle"), th.Muted
	if s.Busy {
		state, c = tr.T("sup.busy"), th.Accent
	}
	rows := [][2]string{
		{tr.T("sup.state"), state},
		{tr.T("sup.mode"), modeLabel(s, tr)},
		{tr.T("sup.tools"), tr.T("sup.count", tools)},
	}
	if s.ContextWindow > 0 {
		rows = append(rows, [2]string{tr.T("sup.context"), tr.T("sup.tokens", s.ContextUsed/1000, s.ContextWindow/1000)})
	}
	section := func(title string, w layout.Widget) layout.FlexChild {
		return layout.Rigid(func(gtx C) D {
			return layout.Inset{Bottom: 18}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(th.Label(11, title, th.Muted, font.SemiBold).Layout),
					ui.VSpace(8),
					layout.Rigid(w),
				)
			})
		})
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		section(tr.T("sup.session"), func(gtx C) D {
			var children []layout.FlexChild
			for i, r := range rows {
				valueColor := th.Text
				if i == 0 {
					valueColor = c
				}
				children = append(children, layout.Rigid(func(gtx C) D {
					return layout.Inset{Bottom: 6}.Layout(gtx, func(gtx C) D {
						return layout.Flex{}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								gtx.Constraints.Min.X = gtx.Dp(110)
								return th.Label(13, r[0], th.Muted, font.Normal).Layout(gtx)
							}),
							layout.Flexed(1, th.Label(13, r[1], valueColor, font.Medium).Layout),
						)
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
		section(tr.T("sup.tasks"), func(gtx C) D {
			if len(tasks) == 0 {
				return th.Label(13, tr.T("sup.none"), th.Muted, font.Normal).Layout(gtx)
			}
			var children []layout.FlexChild
			for _, t := range tasks {
				dot, text := th.Border, th.Muted
				switch t.Status {
				case oriagent.TaskRunning:
					dot, text = ui.Pulse(gtx, th.Accent), th.Text
				case oriagent.TaskDone:
					dot = th.Success
				}
				children = append(children, statusRow(th, dot, t.Text, "", text))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
		section(tr.T("sup.agents"), func(gtx C) D {
			if len(s.Children) == 0 {
				return th.Label(13, tr.T("sup.none"), th.Muted, font.Normal).Layout(gtx)
			}
			var children []layout.FlexChild
			for _, sub := range s.Children {
				dot, state := th.Success, tr.T("sup.done")
				if sub.Busy {
					dot, state = ui.Pulse(gtx, th.Accent), tr.T("sup.busy")
				}
				children = append(children, statusRow(th, dot, sub.Title, state, th.Text))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)
}

// statusRow is a dot, a text and an optional state on the right.
func statusRow(th *ui.Theme, dot color.NRGBA, text, state string, c color.NRGBA) layout.FlexChild {
	return layout.Rigid(func(gtx C) D {
		return layout.Inset{Bottom: 7}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D { return ui.Dot(gtx, dot, 8) }),
				ui.HSpace(10),
				layout.Flexed(1, func(gtx C) D { return ui.OneLine(gtx, th.Label(13, text, c, font.Normal)) }),
				layout.Rigid(th.Label(12, state, th.Muted, font.Normal).Layout),
			)
		})
	})
}

// modeLabel is the working mode of s's project, translated; the first of
// sim.Modes when none is chosen yet.
func modeLabel(s *oriagent.Session, tr i18n.Translator) string {
	id := ""
	if p := s.Project(); p != nil {
		id = p.Mode
	}
	for _, m := range sim.Modes {
		if m.ID == id || id == "" {
			return tr.T(m.Label)
		}
	}
	return id
}

var supervisionTranslations = map[string]map[string]string{
	"fr": {
		"sup.title":   "Supervision",
		"sup.session": "SESSION",
		"sup.state":   "État",
		"sup.idle":    "Au repos",
		"sup.busy":    "En cours",
		"sup.done":    "Terminé",
		"sup.mode":    "Mode",
		"sup.tools":   "Outils",
		"sup.count":   "%d appels",
		"sup.context": "Contexte",
		"sup.tokens":  "%d k / %d k jetons",
		"sup.tasks":   "TÂCHES",
		"sup.agents":  "SOUS-AGENTS",
		"sup.none":    "Aucun pour l'instant.",
	},
	"en": {
		"sup.title":   "Supervision",
		"sup.session": "SESSION",
		"sup.state":   "State",
		"sup.idle":    "Idle",
		"sup.busy":    "Running",
		"sup.done":    "Done",
		"sup.mode":    "Mode",
		"sup.tools":   "Tools",
		"sup.count":   "%d calls",
		"sup.context": "Context",
		"sup.tokens":  "%d k / %d k tokens",
		"sup.tasks":   "TASKS",
		"sup.agents":  "SUBAGENTS",
		"sup.none":    "None yet.",
	},
}
