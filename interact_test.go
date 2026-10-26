package oriagent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func sessionApp(t *testing.T, cfg Config) (*App, *Session) {
	t.Helper()
	cfg.defaults()
	a := newApp(cfg, nil)
	s := &Session{ID: NewSessionID()}
	NewProject("P").Add(s)
	a.cur = s
	return a, s
}

func waitUntil(t *testing.T, a *App, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		a.mu.Lock()
		ok := cond()
		a.mu.Unlock()
		if ok {
			return
		}
	}
	t.Fatal("timeout")
}

// A message sent during a turn waits, then the agent takes it; one it does
// not take starts a new turn; Stop puts it back in the composer.
func TestQueuedMessages(t *testing.T) {
	release := make(chan struct{})
	var prompts, taken []string
	agent := AgentFunc(func(ctx context.Context, tn *Turn) error {
		prompts = append(prompts, tn.Prompt)
		if tn.Prompt == "premier" {
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
			for _, q := range tn.TakeMessages() {
				taken = append(taken, q.Text)
			}
		}
		return nil
	})
	a, s := sessionApp(t, Config{Agent: agent})
	send := func(text string) {
		a.mu.Lock()
		a.editor.SetText(text)
		a.onSubmit()
		a.mu.Unlock()
	}
	send("premier")
	waitUntil(t, a, func() bool { return s.Busy })
	send("ajout pendant le tour")
	a.mu.Lock()
	last := s.Messages[len(s.Messages)-1]
	queued := last.Queued && last.Text == "ajout pendant le tour" && a.editor.Text() == ""
	a.mu.Unlock()
	if !queued {
		t.Fatalf("not queued: %+v", last)
	}
	close(release)
	waitUntil(t, a, func() bool { return !s.Busy })
	if len(taken) != 1 || taken[0] != "ajout pendant le tour" || last.Queued {
		t.Fatalf("taken %v, still queued %v", taken, last.Queued)
	}

	// Not taken: a new turn, without a second bubble.
	release = make(chan struct{})
	agent2 := AgentFunc(func(ctx context.Context, tn *Turn) error {
		prompts = append(prompts, tn.Prompt)
		if tn.Prompt == "long" {
			<-release
		}
		return nil
	})
	a.cfg.Agent = agent2
	prompts = nil
	send("long")
	waitUntil(t, a, func() bool { return s.Busy })
	send("ensuite")
	n := len(s.Messages)
	close(release)
	waitUntil(t, a, func() bool { return !s.Busy && len(prompts) == 2 })
	if prompts[1] != "ensuite" || len(s.Messages) != n {
		t.Fatalf("prompts %v, %d messages for %d", prompts, len(s.Messages), n)
	}

	// Stopped: the waiting text returns to the composer.
	a.mu.Lock()
	a.cfg.Agent = AgentFunc(func(ctx context.Context, tn *Turn) error { <-ctx.Done(); return ctx.Err() })
	a.mu.Unlock()
	send("bloque")
	waitUntil(t, a, func() bool { return s.Busy })
	send("perdu ?")
	a.mu.Lock()
	s.cancel()
	a.mu.Unlock()
	waitUntil(t, a, func() bool { return !s.Busy })
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.editor.Text() != "perdu ?" {
		t.Fatalf("composer %q", a.editor.Text())
	}
	for _, m := range s.Messages {
		if m.Text == "perdu ?" {
			t.Fatal("the stopped message stays in the transcript")
		}
	}
}

// Going back to a message gives its number to the application, cuts the
// transcript there and puts the text back.
func TestRewind(t *testing.T) {
	var got = -1
	a, s := sessionApp(t, Config{Agent: AgentFunc(func(context.Context, *Turn) error { return nil }),
		OnRewind: func(_ context.Context, rs *Session, n int) error {
			if rs != nil {
				got = n
			}
			if n == 9 {
				return errors.New("non")
			}
			return nil
		}})
	for _, txt := range []string{"un", "deux", "trois"} {
		s.add(UserMsg(txt))
		s.add(&Message{Kind: KindAssistant, Text: "réponse à " + txt})
	}
	a.mu.Lock()
	a.rewindTo(s, s.Messages[2]) // "deux"
	a.mu.Unlock()
	waitUntil(t, a, func() bool { return len(s.Messages) == 2 })
	a.mu.Lock()
	defer a.mu.Unlock()
	if got != 1 || a.editor.Text() != "deux" || s.Messages[1].Text != "réponse à un" {
		t.Fatalf("n=%d composer %q", got, a.editor.Text())
	}
}

// "@" lists the application's suggestions; picking one inserts its path.
func TestMentions(t *testing.T) {
	var queries []string
	a, _ := sessionApp(t, Config{Agent: AgentFunc(func(context.Context, *Turn) error { return nil }),
		Mentions: func(_ *Session, q string) []Mention {
			queries = append(queries, q)
			if strings.HasPrefix("budget.xlsx", q) {
				return []Mention{{Label: "budget.xlsx", Detail: "pieces-jointes", Insert: "pieces-jointes/budget.xlsx"}}
			}
			return nil
		}})
	set := func(text string) {
		a.editor.SetText(text)
		n := len([]rune(text))
		a.editor.SetCaret(n, n)
	}
	set("Résume @bud")
	if res, open := a.mentions(); !open || res[0].Label != "budget.xlsx" {
		t.Fatalf("suggestions %v", res)
	}
	a.mentions()
	if len(queries) != 1 {
		t.Errorf("asked %d times for the same query", len(queries))
	}
	if !a.submitMention() || a.editor.Text() != "Résume @pieces-jointes/budget.xlsx " {
		t.Fatalf("inserted %q", a.editor.Text())
	}
	for _, text := range []string{"mail@exemple.pf", "Résume le budget", "@bud et "} {
		set(text)
		if _, open := a.mentions(); open {
			t.Errorf("%q opens the list", text)
		}
	}
	// Escape closes the list without touching the text; typing reopens it.
	set("Lis @bud")
	a.dismissMention()
	if _, open := a.mentions(); open || a.editor.Text() != "Lis @bud" {
		t.Errorf("dismissed: text %q, still open %v", a.editor.Text(), open)
	}
	set("Lis @budg")
	if _, open := a.mentions(); !open {
		t.Error("typing on does not reopen the list")
	}
}

// A message queued then sent as a new turn is the prompt, not also in the
// history; a turn cannot start while going back.
func TestQueuedPromptNotInHistory(t *testing.T) {
	release := make(chan struct{})
	var hist [][]HistoryEntry
	var prompts []string
	a, s := sessionApp(t, Config{Agent: AgentFunc(func(ctx context.Context, tn *Turn) error {
		hist, prompts = append(hist, tn.History), append(prompts, tn.Prompt)
		if tn.Prompt == "un" {
			<-release
		}
		return nil
	})})
	send := func(text string) {
		a.mu.Lock()
		a.editor.SetText(text)
		a.onSubmit()
		a.mu.Unlock()
	}
	send("un")
	waitUntil(t, a, func() bool { return s.Busy })
	send("deux")
	close(release)
	waitUntil(t, a, func() bool { return !s.Busy && len(prompts) == 2 })
	if prompts[1] != "deux" || len(hist[1]) != 1 || hist[1][0].Text != "un" {
		t.Fatalf("second turn: prompt %q, history %+v", prompts[1], hist[1])
	}

	a.mu.Lock()
	s.rewinding = true
	started := a.startTurn(s, "trois", nil, a.cfg.Agent.Run)
	s.rewinding = false
	a.mu.Unlock()
	if started {
		t.Error("a turn started while going back")
	}
}
