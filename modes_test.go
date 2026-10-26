package oriagent

import (
	"context"
	"errors"
	"testing"
	"time"
)

var testModes = []Mode{{ID: "plan", Label: "Plan"}, {ID: "edition", Label: "Édition"}, {ID: "auto", Label: "Auto"}}

// The mode is kept per project; the agent reads it in the turn and may
// change it, and fills the context gauge.
func TestModesAndContext(t *testing.T) {
	var seen string
	h := NewHeadless(Config{Modes: testModes, Agent: AgentFunc(func(ctx context.Context, tn *Turn) error {
		seen = tn.Mode
		tn.SetMode("edition")
		tn.SetContext(42000, 200000)
		return nil
	})})
	p := NewProject("Budget")
	p.Mode = "plan"
	s := &Session{ID: NewSessionID()}
	p.Add(s)
	if _, err := h.Run(context.Background(), s, "prépare un plan", nil, TurnOptions{}); err != nil {
		t.Fatal(err)
	}
	if seen != "plan" || p.Mode != "edition" {
		t.Fatalf("turn mode %q, project mode %q", seen, p.Mode)
	}
	if s.ContextUsed != 42000 || s.ContextWindow != 200000 {
		t.Fatalf("context %d/%d", s.ContextUsed, s.ContextWindow)
	}
	// Out of projects: the first mode, unless the options say otherwise.
	if _, err := h.Run(context.Background(), nil, "x", nil, TurnOptions{}); err != nil || seen != "plan" {
		t.Fatalf("default mode %q %v", seen, err)
	}
	if _, err := h.Run(context.Background(), nil, "x", nil, TurnOptions{Mode: "auto"}); err != nil || seen != "auto" {
		t.Fatalf("options mode %q %v", seen, err)
	}
}

// The user's choice goes to OnMode; an error puts the previous mode back.
func TestChooseMode(t *testing.T) {
	got := make(chan string, 2)
	fail := false
	cfg := Config{Modes: testModes, Agent: AgentFunc(func(context.Context, *Turn) error { return nil }),
		OnMode: func(_ context.Context, p *Project, mode string) error {
			got <- p.Name + "=" + mode
			if fail {
				return errors.New("refusé")
			}
			return nil
		}}
	cfg.defaults()
	a := newApp(cfg, nil)
	p := NewProject("Budget")
	s := &Session{}
	p.Add(s)
	a.cur = s
	if a.sessionMode(s) != "plan" {
		t.Fatalf("default %q", a.sessionMode(s))
	}
	a.chooseMode("auto")
	if p.Mode != "auto" || <-got != "Budget=auto" {
		t.Fatalf("mode %q", p.Mode)
	}
	fail = true
	a.chooseMode("edition")
	<-got
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		m := p.Mode
		a.mu.Unlock()
		if m == "auto" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("failed choice not undone: %q", p.Mode)
}
