package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	oriagent "github.com/sitle/ori-agent"
)

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"Budget 2027":           "budget-2027",
		"  Démarches en ligne!": "démarches-en-ligne",
		"***":                   "projet",
	} {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStoreProjects(t *testing.T) {
	s := newStore(t.TempDir())
	ctx := context.Background()

	p, err := s.newProject(ctx, "Budget 2027", "Préparation du budget")
	if err != nil {
		t.Fatal(err)
	}
	if !s.owned(p) {
		t.Fatal("a created project should be owned")
	}
	if _, err := s.newProject(ctx, "Budget 2027", ""); err == nil {
		t.Error("creating the same project twice should fail")
	}

	perms := &oriagent.Permissions{AskEdits: true}
	if _, err := s.updateProject(ctx, p, oriagent.ProjectEdit{Name: "Budget", Description: "Nouveau", Permissions: perms}); err != nil {
		t.Fatal(err)
	}
	ps := s.projects()
	if len(ps) != 1 || ps[0].Name != "Budget" || ps[0].Permissions == nil || !ps[0].Permissions.AskEdits {
		t.Fatalf("reloaded projects = %+v", ps)
	}

	if err := s.deleteProject(ctx, p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p.WorkDir); !os.IsNotExist(err) {
		t.Error("the project folder should be removed")
	}
}

func TestStoreProjectFiles(t *testing.T) {
	s := newStore(t.TempDir())
	p, err := s.newProject(context.Background(), "Pièces", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.WorkDir, "note.md"), []byte("# Note"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(p.WorkDir, "sous-dossier"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := s.projectFiles(p)
	if len(files) != 1 || files[0].Name != "note.md" || files[0].Path != filepath.Join(p.WorkDir, "note.md") || files[0].Warning != "" {
		t.Fatalf("files = %+v (project.json and folders must be left out)", files)
	}
	if got := s.projectFiles(oriagent.NewProject("sans dossier")); got != nil {
		t.Errorf("a project without folder should list nothing, got %+v", got)
	}
}

// A project the store did not create, such as the folder Anna runs in, must
// never be removed from disk.
func TestStoreKeepsForeignFolders(t *testing.T) {
	s := newStore(t.TempDir())
	foreign := t.TempDir()
	keep := filepath.Join(foreign, "important.txt")
	if err := os.WriteFile(keep, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	p := oriagent.NewProject("ici")
	p.WorkDir = foreign
	if s.owned(p) {
		t.Fatal("a foreign folder must not be owned")
	}
	if err := s.deleteProject(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Error("a foreign folder was touched")
	}
}

func TestStoreSettingsAndProfile(t *testing.T) {
	s := newStore(t.TempDir())
	if got := s.settings(); got != oriagent.DefaultSettings() {
		t.Errorf("no file should give the defaults, got %+v", got)
	}
	set := oriagent.DefaultSettings()
	set.Locale, set.Dark = "en", false
	if err := s.writeJSON("settings.json", set); err != nil {
		t.Fatal(err)
	}
	if got := s.settings(); got != set {
		t.Errorf("settings = %+v, want %+v", got, set)
	}

	u, err := s.saveProfile(context.Background(), oriagent.User{First: "  hina  ", Last: "teriitahi", Email: "hina@example.pf"})
	if err != nil {
		t.Fatal(err)
	}
	if u.First != "Hina" || u.Last != "TERIITAHI" {
		t.Errorf("normalized profile = %+v", u)
	}
	if got := s.profile(oriagent.User{}); got != u {
		t.Errorf("reloaded profile = %+v, want %+v", got, u)
	}
	if _, err := s.saveProfile(context.Background(), oriagent.User{First: "Hina"}); err == nil {
		t.Error("a missing last name should be refused")
	}
}
