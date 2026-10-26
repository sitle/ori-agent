package oriagent

import (
	"context"
	"slices"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/ui"
)

// Going back to a message (Config.OnRewind): under each message of the
// user, a "Go back here" action asks for confirmation, then the application
// restores what it keeps (files, its own record of the conversation), and
// the transcript is cut before the message, whose text returns to the
// composer.

// rewindState is the action under a user message.
type rewindState struct {
	hover, ask, ok, cancel widget.Clickable
	confirming, running    bool
}

// userIndex is the number of user messages of s before message i.
func (s *Session) userIndex(i int) int {
	n := 0
	for _, m := range s.Messages[:i] {
		if m.Kind == KindUser && !m.Queued {
			n++
		}
	}
	return n
}

// rewindTo asks the application to go back to m, then cuts the transcript.
// Called with the lock held.
func (a *App) rewindTo(s *Session, m *Message) {
	idx := -1
	for i, x := range s.Messages {
		if x == m {
			idx = i
			break
		}
	}
	if idx < 0 || s.Busy || s.rewinding || a.cfg.OnRewind == nil {
		return
	}
	n := s.userIndex(idx)
	m.rewind.running, s.rewinding = true, true // no turn starts meanwhile
	go func() {
		err := a.cfg.OnRewind(context.Background(), s, n)
		a.update(func() {
			m.rewind.running, m.rewind.confirming, s.rewinding = false, false, false
			if err != nil {
				a.notify(a.t("rewind.failed", err.Error()))
				return
			}
			for i, x := range s.Messages {
				if x == m {
					s.cut(i)
					break
				}
			}
			s.ContextUsed = 0
			if a.cur == s && a.editor.Text() == "" && len(a.attachments) == 0 {
				a.editor.SetText(m.Text)
				a.attachments = m.Attachments
				a.focus = true
			}
			a.notify(a.t("rewind.done"))
		})
	}()
}

// cut drops the messages of s from i on, with the subagents they spawned.
func (s *Session) cut(i int) {
	for _, m := range s.Messages[i:] {
		if m.Sub != nil {
			s.Children = slices.DeleteFunc(s.Children, func(c *Session) bool { return c == m.Sub })
		}
	}
	s.Messages = s.Messages[:i]
}

// rewindAction is drawn under a user message: nothing, the action on
// hover, or the confirmation.
func (a *App) rewindAction(gtx C, s *Session, m *Message) D {
	th, r := a.th, &m.rewind
	if a.cfg.OnRewind == nil || s.Busy || m.Queued || s.parent != nil || (s.rewinding && !r.running) {
		return D{}
	}
	if r.ask.Clicked(gtx) {
		r.confirming = true
	}
	if r.cancel.Clicked(gtx) {
		r.confirming = false
	}
	if r.ok.Clicked(gtx) && !r.running {
		a.rewindTo(s, m)
	}
	switch {
	case r.running:
		return layout.Inset{Top: 6}.Layout(gtx, th.Label(12, a.t("rewind.running"), th.Muted, font.Normal).Layout)
	case r.confirming:
		// The question above, the buttons under it: both on the right.
		return layout.Inset{Top: 6}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.End}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					gtx.Constraints.Max.X = gtx.Constraints.Max.X * 4 / 5
					return th.Label(12, a.t("rewind.confirm"), th.Text, font.Normal).Layout(gtx)
				}),
				ui.VSpace(6),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D { return th.Button(gtx, &r.cancel, a.t("rewind.cancel"), ui.Secondary) }),
						ui.HSpace(6),
						layout.Rigid(func(gtx C) D { return th.Button(gtx, &r.ok, a.t("rewind.ok"), ui.Danger) }),
					)
				}),
			)
		})
	case r.hover.Hovered() || r.ask.Hovered():
		return layout.Inset{Top: 4}.Layout(gtx, func(gtx C) D {
			return r.ask.Layout(gtx, func(gtx C) D {
				c := th.Muted
				if r.ask.Hovered() {
					c = th.Text
				}
				return th.Label(12, "↶ "+a.t("rewind.here"), c, font.Medium).Layout(gtx)
			})
		})
	}
	// Keeps the height, so that the transcript does not move on hover.
	return layout.Inset{Top: 4}.Layout(gtx, th.Label(12, " ", th.Muted, font.Medium).Layout)
}

// userMessage is the user's bubble, its attachments, and under them the
// "waiting" mark of a message sent during a turn, or the action to go back
// to it.
func (a *App) userMessage(gtx C, m *Message) D {
	th := a.th
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.End}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			if m.Text == "" {
				return D{}
			}
			return layout.E.Layout(gtx, func(gtx C) D {
				gtx.Constraints.Max.X = gtx.Constraints.Max.X * 4 / 5
				bg := th.UserBubble
				if m.Queued {
					bg = th.Hover
				}
				return ui.Card(gtx, bg, 14, layout.Inset{Top: 10, Bottom: 10, Left: 14, Right: 14}, th.Body(15, m.Text))
			})
		}),
		layout.Rigid(func(gtx C) D {
			if len(m.Attachments) == 0 {
				return D{}
			}
			return layout.Inset{Top: 6}.Layout(gtx, func(gtx C) D {
				return layout.E.Layout(gtx, func(gtx C) D { return a.attachmentFlow(gtx, m.Attachments, false) })
			})
		}),
		layout.Rigid(func(gtx C) D {
			if m.Queued {
				return layout.E.Layout(gtx, func(gtx C) D {
					return layout.Inset{Top: 4}.Layout(gtx, th.Label(12, a.t("composer.queued"), th.Muted, font.Normal).Layout)
				})
			}
			return layout.E.Layout(gtx, func(gtx C) D { return a.rewindAction(gtx, a.cur, m) })
		}),
	)
}
