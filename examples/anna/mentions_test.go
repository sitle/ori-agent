package main

import (
	"os"
	"path/filepath"
	"testing"

	oriagent "github.com/sitle/ori-agent"
)

func TestMentions(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"budget.xlsx", "docs/budget-2027.md", "docs/notes.md", ".git/budget-objet", "a/b/c/budget-profond.md"} {
		path := filepath.Join(dir, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	p := oriagent.NewProject("Budget")
	p.WorkDir = dir
	s := &oriagent.Session{}
	p.Add(s)

	got := mentions(s, "BUD")
	var inserts []string
	for _, m := range got {
		inserts = append(inserts, m.Insert)
	}
	// .git and the folders three levels down are left out.
	if len(got) != 2 || inserts[0] != "budget.xlsx" || inserts[1] != "docs/budget-2027.md" || got[1].Detail != "docs" {
		t.Fatalf("mentions: %+v", got)
	}
	if got := mentions(s, "otes"); len(got) != 1 || got[0].Label != "notes.md" {
		t.Errorf("contained in the name: %+v", got)
	}
}
