package sim

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	oriagent "github.com/sitle/ori-agent"
)

// edits returns the Edit tool calls of s that show a diff.
func edits(s *oriagent.Session) []*oriagent.Message {
	var out []*oriagent.Message
	for _, m := range s.Messages {
		if m.Kind == oriagent.KindTool && m.ToolName == "Edit" && m.HasDiff {
			out = append(out, m)
		}
	}
	return out
}

// The simulated agent run without a window: it asks before its edit, is
// not asked again in the session once allowed for good, and never writes.
func TestRunHeadless(t *testing.T) {
	dir := t.TempDir()
	note := filepath.Join(dir, "note.md")
	if err := os.WriteFile(note, []byte("# Note\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := oriagent.NewHeadless(oriagent.Config{Agent: New()})
	var asked []string
	h.Approve = func(r oriagent.ApprovalRequest) oriagent.Decision {
		asked = append(asked, r.Title)
		return oriagent.AllowSession
	}
	opts := oriagent.TurnOptions{AskEdits: true, WorkDir: dir, Locale: "fr"}
	s, err := h.Run(context.Background(), nil, "relis", nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || asked[0] != "Ajouter une note en tête de note.md" {
		t.Fatalf("approvals asked: %q", asked)
	}
	e := edits(s)
	if len(e) != 1 || !strings.HasPrefix(e[0].DiffNew, "Note : relu") || e[0].DiffOld != "# Note\n" {
		t.Fatalf("edits: %+v", e)
	}
	if _, err := h.Run(context.Background(), s, "encore", nil, opts); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || len(edits(s)) != 2 {
		t.Errorf("second turn: %d approvals, %d edits", len(asked), len(edits(s)))
	}
	if data, _ := os.ReadFile(note); string(data) != "# Note\n" {
		t.Errorf("the file was written: %q", data)
	}
	if s.ContextWindow == 0 || s.ContextUsed == 0 {
		t.Errorf("context gauge not filled: %d/%d", s.ContextUsed, s.ContextWindow)
	}
}

// In plan mode, the agent asks to run a plan, then edits without asking
// again and switches to edit mode.
func TestRunPlanMode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "note.md"), []byte("# Note\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := oriagent.NewHeadless(oriagent.Config{Agent: New(), Modes: Modes})
	var asked []oriagent.ApprovalRequest
	h.Approve = func(r oriagent.ApprovalRequest) oriagent.Decision {
		asked = append(asked, r)
		return oriagent.Allow
	}
	p := oriagent.NewProject("Notes")
	p.Mode = ModePlan
	s := &oriagent.Session{}
	p.Add(s)
	if _, err := h.Run(context.Background(), s, "relis", nil, oriagent.TurnOptions{AskEdits: true, WorkDir: dir, Locale: "fr"}); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || !asked[0].Markdown || asked[0].AllowLabel != "Exécuter le plan" {
		t.Fatalf("approvals: %+v", asked)
	}
	if len(edits(s)) != 1 || p.Mode != ModeEdit {
		t.Errorf("%d edits, mode %q", len(edits(s)), p.Mode)
	}
}
