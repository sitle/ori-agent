package oriagent

import (
	"context"
	"image"
	"testing"

	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// The copy actions of an answer put its Markdown, or the text of one of its
// code blocks, in the clipboard.
func TestCopy(t *testing.T) {
	a, s := sessionApp(t, Config{Agent: AgentFunc(func(context.Context, *Turn) error { return nil })})
	m := &Message{Kind: KindAssistant, Text: "Voici :\n\n```sh\necho un\n```\n\nPuis :\n\n```\necho deux\n```"}
	s.add(m)
	var r input.Router
	frame := func() {
		gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Constraints{Max: image.Pt(600, 800)}, Source: r.Source()}
		a.mu.Lock()
		a.assistantMessage(gtx, m)
		a.mu.Unlock()
		r.Frame(gtx.Ops)
	}
	copied := func() string {
		_, data, ok := r.WriteClipboard()
		if !ok {
			return ""
		}
		return string(data)
	}
	frame()
	if len(m.copy.code) != 2 {
		t.Fatalf("%d code buttons, want 2", len(m.copy.code))
	}
	m.copy.code[1].Click()
	frame()
	if got := copied(); got != "echo deux" {
		t.Errorf("code copied %q", got)
	}
	m.copy.msg.Click()
	frame()
	if got := copied(); got != m.Text {
		t.Errorf("answer copied %q", got)
	}
}
