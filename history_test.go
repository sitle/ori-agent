package oriagent

import (
	"context"
	"image"
	"testing"
	"time"

	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// Up recalls what was sent, newest first, and Down comes back to an empty
// composer; the arrows are left to the caret in a text being written.
func TestHistory(t *testing.T) {
	a, _ := sessionApp(t, Config{Agent: AgentFunc(func(context.Context, *Turn) error { return nil })})
	var r input.Router
	frame := func() {
		gtx := layout.Context{Ops: new(op.Ops), Now: time.Now(), Constraints: layout.Exact(image.Pt(1200, 800)),
			Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Source: r.Source()}
		gtx.Execute(key.FocusCmd{Tag: &a.editor})
		a.mu.Lock()
		a.frame(gtx)
		a.mu.Unlock()
		r.Frame(gtx.Ops)
	}
	press := func(name key.Name) string {
		r.Queue(key.Event{Name: name, State: key.Press})
		frame()
		return a.editor.Text()
	}
	frame()
	a.runCommand("/help")
	a.remember("Bonjour")
	a.remember("Bonjour") // a repeat is kept once
	a.runCommand("/status")
	frame()

	for i, want := range []string{"/status", "Bonjour", "/help", "/help"} {
		if got := press(key.NameUpArrow); got != want {
			t.Fatalf("Up %d: %q, want %q", i+1, got, want)
		}
	}
	for i, want := range []string{"Bonjour", "/status", ""} {
		if got := press(key.NameDownArrow); got != want {
			t.Fatalf("Down %d: %q, want %q", i+1, got, want)
		}
	}
	// A text being written keeps its arrows.
	a.editor.SetText("une ligne\nune autre")
	frame()
	if got := press(key.NameUpArrow); got != "une ligne\nune autre" {
		t.Errorf("Up replaced the draft: %q", got)
	}
}

// Enter on a full command name runs that command, even when another one
// starts the same way and comes first.
func TestExactCommandFirst(t *testing.T) {
	a, _ := sessionApp(t, Config{Commands: []*Command{{Name: "models", Run: func(*Ctl, string) string { return "" }}}})
	if m := a.matchCommands("model"); len(m) < 2 || m[0].Name != "model" {
		t.Fatalf("matches %v", m)
	}
}

// A command's Markdown output is marked so, to be drawn as Markdown.
func TestCommandMarkdown(t *testing.T) {
	a, s := sessionApp(t, Config{Commands: []*Command{
		{Name: "liste", Markdown: true, Run: func(*Ctl, string) string { return "- **un**" }},
	}})
	a.runCommand("/liste")
	if m := s.Messages[len(s.Messages)-1]; !m.Markdown || m.Output != "- **un**" {
		t.Fatalf("message %+v", m)
	}
}
