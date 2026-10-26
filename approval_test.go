package oriagent

import (
	"context"
	"errors"
	"testing"
)

// pending returns the open approval of s, if any.
func pending(s *Session) *approval {
	for _, m := range s.Messages {
		if m.Kind == KindApproval && !m.approval.done {
			return m.approval
		}
	}
	return nil
}

func TestApprove(t *testing.T) {
	a := testApp(Config{})
	s := a.cur
	type answer struct {
		d   Decision
		err error
	}
	got := make(chan answer, 3)
	var sub *Session
	locked(a, func() {
		a.startTurn(s, "go", nil, func(ctx context.Context, tn *Turn) error {
			if tn.SessionID == "" || tn.SessionID != s.ID {
				t.Errorf("session id %q, session %q", tn.SessionID, s.ID)
			}
			d, err := tn.Approve(ctx, ApprovalRequest{Title: "Modifier note.md", Diff: true, Old: "a\n", New: "b\n", AllowSession: true})
			got <- answer{d, err}
			// A subagent's request is shown in the root session.
			child := tn.Subagent("Recherche", "cherche")
			a.update(func() { sub = child.s })
			d, err = child.Approve(ctx, ApprovalRequest{Title: "Lancer go test", Detail: "go test ./..."})
			got <- answer{d, err}
			child.Finish("fini")
			// Stop withdraws a pending request.
			_, err = tn.Approve(ctx, ApprovalRequest{Title: "Encore"})
			got <- answer{Deny, err}
			return nil
		})
	})
	waitFor(t, a, func() bool { return pending(s) != nil })
	locked(a, func() { pending(s).decide(AllowSession) })
	if r := <-got; r.d != AllowSession || r.err != nil {
		t.Fatalf("first answer: %+v", r)
	}
	waitFor(t, a, func() bool { return pending(s) != nil })
	locked(a, func() {
		ap := pending(s)
		if ap.from != "Recherche" || pending(sub) != nil {
			t.Errorf("subagent request: from %q", ap.from)
		}
		if sub.ID == "" || sub.ID == s.ID {
			t.Errorf("subagent session id %q", sub.ID)
		}
		ap.decide(Deny)
		ap.decide(Allow) // answered once only
	})
	if r := <-got; r.d != Deny || r.err != nil {
		t.Fatalf("subagent answer: %+v", r)
	}
	waitFor(t, a, func() bool { return pending(s) != nil })
	locked(a, func() { s.cancel() })
	if r := <-got; !errors.Is(r.err, context.Canceled) {
		t.Fatalf("stop: %+v", r)
	}
	waitFor(t, a, func() bool { return !s.Busy })
	var withdrawn bool
	for _, m := range s.Messages {
		withdrawn = withdrawn || (m.Kind == KindApproval && m.approval.canceled)
	}
	if !withdrawn {
		t.Error("request not marked withdrawn")
	}
}
