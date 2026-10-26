package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	oriagent "github.com/sitle/ori-agent"
	"github.com/sitle/ori-agent/i18n"
)

// store keeps what ori-agent leaves to the application: the Configuration
// settings, the profile, and the projects created from the sidebar. Each
// project is a folder under dir/projects, marked by a project.json file;
// only such folders are ever removed.
type store struct {
	dir string
	tr  *i18n.Bundle // follows the interface language (Config.OnLocale), for file sizes
}

const projectFile = "project.json"

type projectMeta struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Permissions *oriagent.Permissions `json:"permissions,omitempty"`
	Mode        string                `json:"mode,omitempty"` // Config.Modes
}

func newStore(dir string) *store {
	tr := i18n.New("fr")
	for l, c := range oriagent.Catalogs {
		tr.Add(l, c)
	}
	return &store{dir: dir, tr: tr}
}

// setLocale is Config.OnLocale.
func (s *store) setLocale(locale string) { s.tr.SetLocale(locale) }

func (s *store) path(name string) string { return filepath.Join(s.dir, name) }

func (s *store) readJSON(name string, v any) {
	if data, err := os.ReadFile(s.path(name)); err == nil {
		_ = json.Unmarshal(data, v) // a broken file keeps the defaults
	}
}

func (s *store) writeJSON(name string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path(name)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.path(name), data, 0o600)
}

// settings returns the saved settings over the defaults.
func (s *store) settings() oriagent.Settings {
	set := oriagent.DefaultSettings()
	s.readJSON("settings.json", &set)
	return set
}

// saveSettings is Config.OnSettings: called on the UI goroutine, so the
// file is written on a goroutine of its own.
func (s *store) saveSettings(set oriagent.Settings) {
	go func() { _ = s.writeJSON("settings.json", set) }()
}

// profile returns the saved profile, or def.
func (s *store) profile(def oriagent.User) oriagent.User {
	u := def
	s.readJSON("profile.json", &u)
	return u
}

// saveProfile is Config.OnProfile: it normalizes the fields like a real
// directory would, then saves them. The e-mail is locked (ProfileLock).
func (s *store) saveProfile(ctx context.Context, u oriagent.User) (oriagent.User, error) {
	if err := pause(ctx, 600); err != nil {
		return u, err
	}
	u.First = titleCase(u.First)
	u.Last = strings.ToUpper(strings.TrimSpace(u.Last))
	u.Service, u.Role = strings.TrimSpace(u.Service), strings.TrimSpace(u.Role)
	if u.First == "" || u.Last == "" {
		return u, errors.New("le prénom et le nom sont obligatoires")
	}
	return u, s.writeJSON("profile.json", u)
}

// logout is Config.OnLogout: there is no server, it only takes its time.
func (s *store) logout(ctx context.Context) error { return pause(ctx, 500) }

// projects loads the projects created in earlier runs.
func (s *store) projects() []*oriagent.Project {
	entries, _ := os.ReadDir(s.path("projects"))
	var ps []*oriagent.Project
	for _, e := range entries {
		dir := filepath.Join(s.path("projects"), e.Name())
		var m projectMeta
		data, err := os.ReadFile(filepath.Join(dir, projectFile))
		if !e.IsDir() || err != nil || json.Unmarshal(data, &m) != nil {
			continue
		}
		p := oriagent.NewProject(m.Name)
		p.Description, p.Permissions, p.WorkDir, p.Open, p.Mode = m.Description, m.Permissions, dir, false, m.Mode
		ps = append(ps, p)
	}
	return ps
}

// newProject is Config.OnNewProject: a workspace folder named after the
// project, which becomes Turn.WorkDir for its sessions.
func (s *store) newProject(ctx context.Context, name, desc string) (*oriagent.Project, error) {
	if err := pause(ctx, 400); err != nil {
		return nil, err
	}
	dir := filepath.Join(s.path("projects"), slug(name))
	if _, err := os.Stat(dir); err == nil {
		return nil, fmt.Errorf("un projet « %s » existe déjà", name)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	m := projectMeta{Name: name, Description: desc}
	if err := s.writeJSON(filepath.Join("projects", filepath.Base(dir), projectFile), m); err != nil {
		return nil, err
	}
	p := oriagent.NewProject(name)
	p.Description, p.WorkDir = desc, dir
	return p, nil
}

// owned reports whether p is a project folder created by this store.
func (s *store) owned(p *oriagent.Project) bool {
	root, err := filepath.Abs(s.path("projects"))
	if err != nil || p.WorkDir == "" {
		return false
	}
	dir, err := filepath.Abs(p.WorkDir)
	if err != nil || filepath.Dir(dir) != root {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, projectFile))
	return err == nil
}

// updateProject is Config.OnUpdateProject: created projects keep their
// edits in their project.json; the demo ones only in memory.
func (s *store) updateProject(ctx context.Context, p *oriagent.Project, e oriagent.ProjectEdit) (oriagent.ProjectEdit, error) {
	if err := pause(ctx, 300); err != nil {
		return e, err
	}
	e.Name = strings.TrimSpace(e.Name)
	if e.Name == "" {
		return e, errors.New("le nom du projet est obligatoire")
	}
	if s.owned(p) {
		name := filepath.Join("projects", filepath.Base(p.WorkDir), projectFile)
		var m projectMeta
		s.readJSON(name, &m) // keeps the saved mode
		m.Name, m.Description, m.Permissions = e.Name, e.Description, e.Permissions
		if err := s.writeJSON(name, m); err != nil {
			return e, err
		}
	}
	return e, nil
}

// saveMode is Config.OnMode: created projects keep their mode in their
// project.json; the demo ones and the sessions out of projects only in
// memory.
func (s *store) saveMode(ctx context.Context, p *oriagent.Project, mode string) error {
	if p == nil || !s.owned(p) {
		return nil
	}
	name := filepath.Join("projects", filepath.Base(p.WorkDir), projectFile)
	var m projectMeta
	s.readJSON(name, &m)
	m.Mode = mode
	return s.writeJSON(name, m)
}

// deleteProject is Config.OnDeleteProject: a created project's folder is
// removed with its content; demo projects only leave the sidebar.
func (s *store) deleteProject(ctx context.Context, p *oriagent.Project) error {
	if err := pause(ctx, 300); err != nil {
		return err
	}
	if s.owned(p) {
		return os.RemoveAll(p.WorkDir)
	}
	return nil
}

// projectFiles is Config.ProjectFiles: the files at the top of the project's
// folder, with their size and date. A file too large to be attached gets a
// warning. Folders and Anna's own project.json are left out.
func (s *store) projectFiles(p *oriagent.Project) []oriagent.ProjectFile {
	if p.WorkDir == "" {
		return nil
	}
	entries, err := os.ReadDir(p.WorkDir)
	if err != nil {
		return []oriagent.ProjectFile{{Name: p.WorkDir, Warning: err.Error()}}
	}
	var files []oriagent.ProjectFile
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() || e.Name() == projectFile {
			continue
		}
		f := oriagent.ProjectFile{
			Name:   e.Name(),
			Detail: oriagent.HumanSize(info.Size(), s.tr) + " · " + info.ModTime().Format("2006-01-02 15:04"),
			Path:   filepath.Join(p.WorkDir, e.Name()),
		}
		if info.Size() > oriagent.MaxUpload {
			f.Warning = "Trop volumineux pour être joint (20 Mo au plus)"
		}
		files = append(files, f)
	}
	return files
}

func pause(ctx context.Context, ms int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(ms) * time.Millisecond):
		return nil
	}
}

// slug turns a project name into a folder name: "Budget 2027" → "budget-2027".
func slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.TrimSuffix(b.String(), "-")
	if s == "" {
		s = "projet"
	}
	return s
}

func titleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}
