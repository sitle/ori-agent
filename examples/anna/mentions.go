package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	oriagent "github.com/sitle/ori-agent"
)

// skipDirs are never listed for "@": tools' folders, too large or of no use.
var skipDirs = []string{".git", "node_modules", "vendor", ".idea", ".vscode"}

// mentions is Config.Mentions: the files of the session's folder whose name
// starts with the query (then those that contain it), three levels deep at
// most. It runs on the UI goroutine, so the walk stops after a few thousand
// entries.
func mentions(s *oriagent.Session, query string) []oriagent.Mention {
	dir := ""
	if p := s.Project(); p != nil {
		dir = p.WorkDir
	}
	if dir == "" {
		dir, _ = os.Getwd()
	}
	q := strings.ToLower(query)
	var first, then []oriagent.Mention
	seen := 0
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // an unreadable folder is skipped
		}
		if seen++; seen > 5000 || len(first) >= 8 {
			return filepath.SkipAll
		}
		rel, _ := filepath.Rel(dir, path)
		if d.IsDir() {
			if rel != "." && (slices.Contains(skipDirs, d.Name()) || strings.Count(rel, string(filepath.Separator)) >= 2) {
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(d.Name())
		if !strings.Contains(name, q) {
			return nil
		}
		m := oriagent.Mention{Label: d.Name(), Detail: filepath.ToSlash(filepath.Dir(rel)), Insert: filepath.ToSlash(rel)}
		if m.Detail == "." {
			m.Detail = ""
		}
		if strings.HasPrefix(name, q) {
			first = append(first, m)
		} else {
			then = append(then, m)
		}
		return nil
	})
	out := append(first, then...)
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}
