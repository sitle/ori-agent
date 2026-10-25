package oriagent

import (
	"context"
	"errors"
	"testing"
)

func TestNewSessionIn(t *testing.T) {
	busy := &Session{Title: "En cours", Messages: []*Message{UserMsg("bonjour")}}
	p := NewProject("Budget", busy)
	a := testApp(Config{Projects: []*Project{NewProject("Default", &Session{Title: "x"}), p}})
	a.newSessionIn(p)
	if a.cur == busy || a.cur.project != p || len(p.Sessions) != 2 || p.Sessions[0] != a.cur {
		t.Fatalf("no new session on top of the project: %+v", p.Sessions)
	}
	// An empty session is reused rather than piling up.
	a.newSessionIn(p)
	if len(p.Sessions) != 2 {
		t.Fatalf("empty session not reused: %d sessions", len(p.Sessions))
	}
}

func TestSessionWorkDir(t *testing.T) {
	s := &Session{Title: "s"}
	p := NewProject("Budget", s)
	a := testApp(Config{WorkDir: "/conf", Projects: []*Project{p}})
	if got := a.sessionWorkDir(s); got != "/conf" {
		t.Errorf("without a project folder: %q", got)
	}
	p.WorkDir = "/ws/budget"
	child := &Session{parent: s}
	if got := a.sessionWorkDir(child); got != "/ws/budget" {
		t.Errorf("subagent session: %q", got)
	}
}

func TestCreateProject(t *testing.T) {
	var gotName, gotDesc string
	a := testApp(Config{OnNewProject: func(_ context.Context, name, desc string) (*Project, error) {
		if name == "Doublon" {
			return nil, errors.New("existe déjà")
		}
		gotName, gotDesc = name, desc
		return &Project{Name: name, Description: desc, WorkDir: "/ws/" + name}, nil
	}})
	n := len(a.projects)
	locked(a, func() {
		a.openNewProject()
		a.createProject() // no name
		if a.newProj.err == "" || a.newProj.busy {
			t.Error("empty name accepted")
		}
		a.newProj.name.SetText("Doublon")
		a.createProject()
	})
	waitFor(t, a, func() bool { return !a.newProj.busy })
	if a.newProj.err != "existe déjà" || a.view != viewNewProject || len(a.projects) != n {
		t.Fatalf("error not kept in the form: %q, view %s", a.newProj.err, a.view)
	}
	locked(a, func() {
		a.newProj.name.SetText("  Budget 2027 ")
		a.newProj.desc.SetText("Préparation du budget")
		a.createProject()
	})
	waitFor(t, a, func() bool { return !a.newProj.busy })
	p := a.projects[len(a.projects)-1]
	if gotName != "Budget 2027" || gotDesc != "Préparation du budget" || p.Name != "Budget 2027" {
		t.Fatalf("project %+v from %q %q", p, gotName, gotDesc)
	}
	if a.cur.project != p || a.view != viewChat || a.sessionWorkDir(a.cur) != "/ws/Budget 2027" {
		t.Fatalf("the new project's session is not selected")
	}
}

func TestProjectPage(t *testing.T) {
	s := &Session{Title: "s", Messages: []*Message{UserMsg("bonjour")}}
	keep := NewProject("Default", &Session{Title: "d"})
	keep.Permanent = true
	p := NewProject("Budget", s)
	p.Permissions = &Permissions{AllowCommands: false, AskEdits: true}
	var removed *Project
	a := testApp(Config{
		Projects: []*Project{keep, p},
		OnUpdateProject: func(_ context.Context, _ *Project, e ProjectEdit) (ProjectEdit, error) {
			if e.Name == "Doublon" {
				return e, errors.New("existe déjà")
			}
			e.Name += " (vu)"
			return e, nil
		},
		OnDeleteProject: func(_ context.Context, x *Project) error { removed = x; return nil },
	})
	if got := a.permissions(s); got.AllowCommands || !got.AskEdits {
		t.Fatalf("project permissions not used: %+v", got)
	}
	locked(a, func() {
		a.cur = s
		a.openProject(p)
		a.projPage.name.SetText("Doublon")
		a.saveProject()
	})
	waitFor(t, a, func() bool { return !a.projPage.busy })
	if a.projPage.err == "" || p.Name != "Budget" {
		t.Fatal("refused edit applied")
	}
	locked(a, func() {
		a.projPage.name.SetText("Budget 2027")
		a.projPage.desc.SetText("Préparation")
		a.projPage.allowCmds.Value = true
		a.saveProject()
	})
	waitFor(t, a, func() bool { return !a.projPage.busy })
	if p.Name != "Budget 2027 (vu)" || p.Description != "Préparation" || !p.Permissions.AllowCommands {
		t.Fatalf("edit not applied as returned: %+v %+v", p, p.Permissions)
	}
	locked(a, func() {
		a.openProject(p)
		a.removeProject()
		if !a.projPage.confirmRemove || a.projPage.busy {
			t.Error("removed without confirmation")
		}
		a.removeProject()
	})
	waitFor(t, a, func() bool { return !a.projPage.busy })
	if removed != p || len(a.projects) != 1 || a.cur.project != keep {
		t.Fatalf("project not removed, or the shown session still in it")
	}
}
