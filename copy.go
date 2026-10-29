package oriagent

import (
	"io"
	"strings"

	"gioui.org/font"
	"gioui.org/io/clipboard"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/ui"
)

// Copying from the transcript: the text drawn there cannot be selected, so
// each answer and each message of the user has a "Copy" action, shown on
// hover, and each code block of an answer a "Copy" button. What is copied
// is the Markdown source, which pastes well in a document or an editor.

// copyState is the copy actions of a message.
type copyState struct {
	hover, msg widget.Clickable
	code       []*widget.Clickable // pointers: their address is their identity
}

// copyText puts text in the clipboard and says so.
func (a *App) copyText(gtx C, text string) {
	gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(text))})
	a.notify(a.t("toast.copied"))
}

// assistantMessage is an answer, with a copy button on its code blocks and
// the action to copy it under it.
func (a *App) assistantMessage(gtx C, m *Message) D {
	th, c := a.th, &m.copy
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return c.hover.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(th.MarkdownWith(15, m.Text, ui.MDOptions{CodeAction: func(gtx C, i int, code string) D {
				for len(c.code) <= i {
					c.code = append(c.code, new(widget.Clickable))
				}
				return a.codeCopy(gtx, c.code[i], code)
			}})),
			layout.Rigid(func(gtx C) D { return a.copyAction(gtx, m) }),
		)
	})
}

// codeCopy is the button of a code block.
func (a *App) codeCopy(gtx C, btn *widget.Clickable, code string) D {
	th := a.th
	if btn.Clicked(gtx) {
		a.copyText(gtx, code)
	}
	return btn.Layout(gtx, func(gtx C) D {
		col := th.Muted
		if btn.Hovered() {
			col = th.Text
		}
		return ui.Card(gtx, th.Bg, 6, layout.Inset{Top: 2, Bottom: 2, Left: 8, Right: 8}, th.Label(12, a.t("copy.action"), col, font.Medium).Layout)
	})
}

// copyAction is drawn under a message: the action on hover, else a blank
// line of the same height, so that the transcript does not move.
func (a *App) copyAction(gtx C, m *Message) D {
	th, c := a.th, &m.copy
	if c.msg.Clicked(gtx) {
		a.copyText(gtx, m.Text)
	}
	hidden := m.Text == "" || m.Queued || m.rewind.confirming || m.rewind.running
	if hidden || !(c.hover.Hovered() || c.msg.Hovered() || m.rewind.hover.Hovered()) {
		return layout.Inset{Top: 4}.Layout(gtx, th.Label(12, " ", th.Muted, font.Medium).Layout)
	}
	return layout.Inset{Top: 4}.Layout(gtx, func(gtx C) D {
		return c.msg.Layout(gtx, func(gtx C) D {
			col := th.Muted
			if c.msg.Hovered() {
				col = th.Text
			}
			return th.Label(12, a.t("copy.message"), col, font.Medium).Layout(gtx)
		})
	})
}
