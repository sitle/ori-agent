package oriagent

import (
	"context"
	"strconv"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/ui"
)

// ApprovalRequest is what an agent asks the user to allow: a command to
// run, a file to change… ori-agent shows it and collects the answer; the
// agent decides what needs approval.
type ApprovalRequest struct {
	Title  string // what the agent wants to do, e.g. "Modifier main.go"
	Detail string // the command or other detail, shown as code
	// With Diff, the change from Old to New is shown.
	Diff     bool
	Old, New string
	// AllowSession offers "Allow for this session" besides allow and deny.
	AllowSession bool
}

// Decision is the user's answer to an ApprovalRequest.
type Decision int

const (
	Deny         Decision = iota
	Allow                 // this time
	AllowSession          // this time and the next ones of the session
)

type approval struct {
	req      ApprovalRequest
	from     string // title of the subagent that asks, if any
	done     bool
	canceled bool
	decision Decision
	ch       chan Decision

	diff                diffCache
	allow, always, deny widget.Clickable
}

// Approve shows req in the transcript and waits for the user's answer. A
// subagent's request is shown in its root session, naming the subagent,
// where the user is working. When ctx is canceled (Stop), the request is
// withdrawn and ctx's error returned with Deny.
func (t *Turn) Approve(ctx context.Context, req ApprovalRequest) (Decision, error) {
	ap := &approval{req: req, ch: make(chan Decision, 1)}
	m := &Message{Kind: KindApproval, approval: ap}
	if h := t.app.headless; h != nil {
		// Without a window, the Headless answers at once, outside the lock:
		// its callback may take its time or read the session.
		d := h.approve(req)
		t.app.update(func() {
			ap.done, ap.decision = true, d
			t.s.add(m)
		})
		return d, nil
	}
	t.app.update(func() {
		root := t.s
		for root.parent != nil {
			root = root.parent
		}
		if root != t.s {
			ap.from = t.s.Title
		}
		root.add(m)
		t.created = append(t.created, m)
		if t.app.cur != root {
			t.app.toast.Show(t.app.t("approval.waiting", root.Title))
		}
	})
	select {
	case d := <-ap.ch:
		return d, nil
	case <-ctx.Done():
		t.app.update(func() {
			if !ap.done {
				ap.done, ap.canceled = true, true
			}
		})
		return Deny, ctx.Err()
	}
}

// decide records the answer, once. Called with the lock held.
func (ap *approval) decide(d Decision) {
	if ap.done {
		return
	}
	ap.done, ap.decision = true, d
	ap.ch <- d
}

func (a *App) approvalCard(gtx C, m *Message) D {
	th, ap := a.th, m.approval
	if ap.allow.Clicked(gtx) {
		ap.decide(Allow)
	}
	if ap.always.Clicked(gtx) {
		ap.decide(AllowSession)
	}
	if ap.deny.Clicked(gtx) {
		ap.decide(Deny)
	}
	border := th.Accent
	if ap.done {
		border = th.Border
	}
	return ui.Bordered(gtx, th.Surface, border, 10, func(gtx C) D {
		return layout.UniformInset(14).Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			head := a.t("approval.title")
			if ap.from != "" {
				head = a.t("approval.from", ap.from)
			}
			children := []layout.FlexChild{
				layout.Rigid(th.Label(12, head, th.Muted, font.Medium).Layout),
				ui.VSpace(4),
				layout.Rigid(th.Label(15, ap.req.Title, th.Text, font.SemiBold).Layout),
			}
			if ap.req.Detail != "" {
				children = append(children, ui.VSpace(10), layout.Rigid(func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return ui.Bordered(gtx, th.Code, th.Border, 6, func(gtx C) D {
						return layout.UniformInset(10).Layout(gtx, th.CodeLines(ap.req.Detail))
					})
				}))
			}
			if ap.req.Diff {
				children = append(children, ui.VSpace(10), layout.Rigid(func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return ui.Bordered(gtx, th.Code, th.Border, 6, func(gtx C) D {
						w, _ := ap.diff.view(th, ap.req.Old, ap.req.New, a.t("diff.none"))
						return layout.UniformInset(10).Layout(gtx, w)
					})
				}))
			}
			children = append(children, ui.VSpace(12), layout.Rigid(func(gtx C) D {
				if ap.done {
					label, c := a.t("approval.allowed"), th.Success
					switch {
					case ap.canceled:
						label, c = a.t("approval.canceled"), th.Muted
					case ap.decision == Deny:
						label, c = a.t("approval.denied"), th.DiffDel
					case ap.decision == AllowSession:
						label = a.t("approval.allowedSession")
					}
					return th.Label(13, label, c, font.Medium).Layout(gtx)
				}
				buttons := []layout.FlexChild{
					layout.Rigid(func(gtx C) D { return th.Button(gtx, &ap.allow, a.t("approval.allow"), ui.Primary) }),
				}
				if ap.req.AllowSession {
					buttons = append(buttons, ui.HSpace(8),
						layout.Rigid(func(gtx C) D { return th.Button(gtx, &ap.always, a.t("approval.always"), ui.Secondary) }))
				}
				buttons = append(buttons, ui.HSpace(8),
					layout.Rigid(func(gtx C) D { return th.Button(gtx, &ap.deny, a.t("approval.deny"), ui.Secondary) }))
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx, buttons...)
			}))
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

// diffCache keeps a diff's widget and its "+3 -1" count between frames:
// comparing the texts is quadratic, too slow to redo at each frame.
type diffCache struct {
	old, new, none string
	th             *ui.Theme
	w              layout.Widget
	stat           string
}

func (c *diffCache) view(th *ui.Theme, old, new, none string) (layout.Widget, string) {
	if c.w == nil || c.th != th || c.old != old || c.new != new || c.none != none {
		added, removed := ui.DiffStat(ui.DiffLines(old, new))
		c.old, c.new, c.none, c.th = old, new, none, th
		c.w = th.Diff(old, new, none)
		c.stat = "+" + strconv.Itoa(added) + " -" + strconv.Itoa(removed)
	}
	return c.w, c.stat
}
