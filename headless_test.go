package oriagent

import (
	"context"
	"testing"
)

func TestHeadless(t *testing.T) {
	var seen *Turn
	h := NewHeadless(Config{Agent: AgentFunc(func(ctx context.Context, tn *Turn) error {
		seen = tn
		d, err := tn.Approve(ctx, ApprovalRequest{Title: "Modifier note.md"})
		if err != nil {
			return err
		}
		b := tn.Text()
		if d == Allow {
			b.Write("autorisé : " + tn.Prompt)
		} else {
			b.Write("refusé")
		}
		b.Done()
		tn.Tool("Read", "note.md").Done("contenu")
		return nil
	})})
	h.Approve = func(r ApprovalRequest) Decision {
		if r.Title == "Modifier note.md" {
			return Allow
		}
		return Deny
	}
	opts := TurnOptions{WorkDir: "/ws", AllowCommands: true, Model: Model{Name: "qwen"}}
	s, err := h.Run(context.Background(), nil, "bonjour", nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if seen.WorkDir != "/ws" || !seen.AllowCommands || seen.Model.Name != "qwen" || seen.SessionID != s.ID || s.ID == "" {
		t.Fatalf("turn %+v", seen)
	}
	kinds := []Kind{}
	for _, m := range s.Messages {
		kinds = append(kinds, m.Kind)
	}
	if len(s.Messages) != 4 || s.Messages[2].Text != "autorisé : bonjour" || s.Messages[3].Output != "contenu" || s.Busy {
		t.Fatalf("transcript %v %+v", kinds, s.Messages)
	}
	// A second turn in the same session sees the history.
	h.Approve = nil
	if _, err := h.Run(context.Background(), s, "encore", nil, opts); err != nil {
		t.Fatal(err)
	}
	if len(seen.History) != 2 || s.Messages[len(s.Messages)-2].Text != "refusé" {
		t.Fatalf("second turn: history %v", seen.History)
	}
}

// The approval callback runs outside the lock: it may read the session,
// and a session runs one turn at a time.
func TestHeadlessApproveAndBusy(t *testing.T) {
	var h *Headless
	h = NewHeadless(Config{Agent: AgentFunc(func(ctx context.Context, tn *Turn) error {
		_, err := tn.Approve(ctx, ApprovalRequest{Title: "Lancer go test"})
		return err
	})})
	var s0 *Session
	var nested error
	h.Approve = func(ApprovalRequest) Decision {
		h.app.update(func() {}) // would deadlock if called with the lock held
		_, nested = h.Run(context.Background(), s0, "encore", nil, TurnOptions{})
		return Allow
	}
	s0 = &Session{Title: "existante"}
	if _, err := h.Run(context.Background(), s0, "go", nil, TurnOptions{}); err != nil {
		t.Fatal(err)
	}
	if nested != ErrSessionBusy {
		t.Errorf("a second turn in a busy session: %v", nested)
	}
}
