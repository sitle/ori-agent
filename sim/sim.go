// Package sim is a scripted agent for demos and UI work: no model is called,
// but its tool calls really run against the working directory (listing,
// reading files). It answers in the turn's language. It only uses the
// public Turn API, so it doubles as the reference for writing a real agent.
package sim

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	oriagent "github.com/sitle/ori-agent"
	"github.com/sitle/ori-agent/i18n"
)

type Agent struct {
	// DocName is the file /init pretends to write. Default "AGENT.md".
	DocName string

	mu      sync.Mutex
	allowed map[string]bool // Turn.SessionID of the sessions where edits were allowed for good
}

func New() *Agent { return &Agent{DocName: "AGENT.md"} }

// tr translates the agent's own texts in the turn's language, falling back
// to English.
func tr(t *oriagent.Turn, key string, args ...any) string {
	s, ok := Translations[i18n.Normalize(t.Locale)][key]
	if !ok {
		s = Translations["en"][key]
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// Commands returns /review and /init, backed by this agent. Their
// descriptions are keys of Translations.
func (s *Agent) Commands() []*oriagent.Command {
	return []*oriagent.Command{
		{Name: "review", Desc: "sim.review.desc", Run: func(c *oriagent.Ctl, _ string) string {
			if err := c.RunTurn("", s.Review); err != nil {
				return err.Error()
			}
			return c.T("sim.review.start", c.WorkDir())
		}},
		{Name: "init", Desc: "sim.init.desc", Run: func(c *oriagent.Ctl, _ string) string {
			if err := c.RunTurn("", s.Init); err != nil {
				return err.Error()
			}
			return c.T("sim.init.start", s.DocName)
		}},
	}
}

// ---- streaming helpers ----

func sleep(ctx context.Context, ms int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(ms) * time.Millisecond):
		return nil
	}
}

// stream writes text word by word into a block, like a model would.
func stream(ctx context.Context, b *oriagent.Block, text string, minMs, spread int) error {
	defer b.Done()
	for _, w := range strings.SplitAfter(text, " ") {
		b.Write(w)
		if err := sleep(ctx, minMs+rand.IntN(spread)); err != nil {
			return err
		}
	}
	return nil
}

func say(ctx context.Context, t *oriagent.Turn, text string) error {
	if err := stream(ctx, t.Text(), text, 20, 35); err != nil {
		return err
	}
	return sleep(ctx, 250)
}

func think(ctx context.Context, t *oriagent.Turn, text string) error {
	b := t.Think()
	if err := sleep(ctx, 500); err != nil {
		return err
	}
	if err := stream(ctx, b, text, 25, 45); err != nil {
		return err
	}
	return sleep(ctx, 200)
}

func tool(ctx context.Context, t *oriagent.Turn, name, arg string, ms int, run func() string) error {
	c := t.Tool(name, arg)
	if err := sleep(ctx, ms); err != nil {
		return err
	}
	c.Done(run())
	return sleep(ctx, 200)
}

// ---- turns ----

// Run plays a full turn: reasoning, a task list, a subagent exploring the
// working directory, then an answer. Reasoning is skipped when extended
// thinking is off in the settings.
func (s *Agent) Run(ctx context.Context, t *oriagent.Turn) error {
	if len(t.Attachments) > 0 {
		return s.readAttachments(ctx, t)
	}
	base := filepath.Base(t.WorkDir)
	if t.Thinking {
		if err := think(ctx, t, tr(t, "sim.think1", t.Prompt, base)); err != nil {
			return err
		}
	}
	if err := say(ctx, t, tr(t, "sim.start", base)); err != nil {
		return err
	}
	tasks := t.Tasks(tr(t, "sim.task1"), tr(t, "sim.task2"), tr(t, "sim.task3"))
	if err := sleep(ctx, 400); err != nil {
		return err
	}

	// The exploration runs in a subagent: it shows up under this session in
	// the sidebar tree, with its own transcript.
	sub := t.Subagent(tr(t, "sim.sub.title", base), tr(t, "sim.sub.prompt"))
	if err := sleep(ctx, 500); err != nil {
		return err
	}
	entries, listing := listDir(t.WorkDir)
	if err := tool(ctx, sub, "Bash", "ls -la", 800, func() string { return listing }); err != nil {
		return err
	}
	tasks.Advance(0)
	f := pickFile(entries)
	if f != "" {
		err := tool(ctx, sub, "Read", f, 600, func() string { return Head(filepath.Join(t.WorkDir, f), 24) })
		if err != nil {
			return err
		}
	}
	report := tr(t, "sim.report", len(entries), base, f)
	if err := say(ctx, sub, report+tr(t, "sim.reportTail")); err != nil {
		return err
	}
	sub.Finish(report)
	tasks.Advance(1)
	if f != "" {
		if err := s.proposeEdit(ctx, t, f); err != nil {
			return err
		}
	}

	dirs := 0
	for _, e := range entries {
		if e.IsDir() {
			dirs++
		}
	}
	if t.Thinking {
		if err := think(ctx, t, tr(t, "sim.think2", len(entries), dirs, f)); err != nil {
			return err
		}
	}
	err := say(ctx, t, tr(t, "sim.done", t.WorkDir, t.Prompt, len(entries), dirs, len(entries)-dirs))
	tasks.Set(2, oriagent.TaskDone)
	return err
}

// proposeEdit shows a change to the top of f as a diff, without writing it.
// With Turn.AskEdits, the user approves it first, unless edits were allowed
// for the whole session before.
func (s *Agent) proposeEdit(ctx context.Context, t *oriagent.Turn, f string) error {
	path := filepath.Join(t.WorkDir, f)
	old := firstLines(path, 8)
	if old == "" {
		return nil
	}
	new := tr(t, "sim.edit.note") + "\n" + old
	s.mu.Lock()
	ask := t.AskEdits && !s.allowed[t.SessionID]
	s.mu.Unlock()
	if ask {
		d, err := t.Approve(ctx, oriagent.ApprovalRequest{Title: tr(t, "sim.edit.title", f), Diff: true, Old: old, New: new, AllowSession: true})
		if err != nil {
			return err
		}
		switch d {
		case oriagent.Deny:
			return say(ctx, t, tr(t, "sim.edit.denied", f))
		case oriagent.AllowSession:
			s.mu.Lock()
			if s.allowed == nil {
				s.allowed = map[string]bool{}
			}
			s.allowed[t.SessionID] = true
			s.mu.Unlock()
		}
	}
	c := t.Tool("Edit", f).File(path).Diff(old, new)
	if err := sleep(ctx, 500); err != nil {
		return err
	}
	c.Done(tr(t, "sim.edit.output", f))
	return sleep(ctx, 200)
}

// firstLines returns the first n lines of a text file, or "" for a binary
// or unreadable one.
func firstLines(path string, n int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, 16<<10)
	k, _ := f.Read(buf)
	if k == 0 || !utf8.Valid(buf[:k]) {
		return ""
	}
	lines := strings.Split(strings.TrimSuffix(strings.ReplaceAll(string(buf[:k]), "\r\n", "\n"), "\n"), "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n") + "\n"
}

// readAttachments reads each attached file, then lists what was received.
func (s *Agent) readAttachments(ctx context.Context, t *oriagent.Turn) error {
	names := make([]string, len(t.Attachments))
	for i, a := range t.Attachments {
		names[i] = a.Name
	}
	if t.Thinking {
		if err := think(ctx, t, tr(t, "sim.att.think", len(t.Attachments), strings.Join(names, ", "))); err != nil {
			return err
		}
	}
	var b strings.Builder
	for _, a := range t.Attachments {
		c := t.Tool("Read", a.Name).File(a.Path)
		if err := sleep(ctx, 500+rand.IntN(400)); err != nil {
			return err
		}
		c.Done(summary(t, a))
		fmt.Fprintf(&b, "\n- %s (%s)", a.Name, oriagent.HumanSize(a.Size, t))
	}
	ask := ""
	if t.Prompt != "" {
		ask = tr(t, "sim.att.ask", t.Prompt)
	}
	return say(ctx, t, tr(t, "sim.att.read")+b.String()+ask+tr(t, "sim.att.tail"))
}

// Review reads a source file and gives canned findings.
func (s *Agent) Review(ctx context.Context, t *oriagent.Turn) error {
	if err := think(ctx, t, tr(t, "sim.review.think")); err != nil {
		return err
	}
	entries, _ := listDir(t.WorkDir)
	f := pickFile(entries)
	if f == "" {
		return say(ctx, t, tr(t, "sim.review.none"))
	}
	if err := tool(ctx, t, "Read", f, 700, func() string { return Head(filepath.Join(t.WorkDir, f), 30) }); err != nil {
		return err
	}
	return say(ctx, t, tr(t, "sim.review.out", f))
}

// Init drafts DocName from the directory listing; nothing is written.
func (s *Agent) Init(ctx context.Context, t *oriagent.Turn) error {
	base := filepath.Base(t.WorkDir)
	if err := think(ctx, t, tr(t, "sim.init.think", s.DocName)); err != nil {
		return err
	}
	entries, _ := listDir(t.WorkDir)
	var names []string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() {
			n += "/"
		}
		names = append(names, n)
	}
	if err := tool(ctx, t, "Glob", "*", 500, func() string { return strings.Join(names, "\n") }); err != nil {
		return err
	}
	draft := []string{"+# " + base, "+", "+## " + tr(t, "sim.init.build"), "+", "+    go run .", "+", "+## " + tr(t, "sim.init.layout"), "+"}
	for _, n := range names {
		if strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "/") {
			draft = append(draft, "+- "+n)
		}
	}
	if err := tool(ctx, t, "Write", s.DocName, 700, func() string { return strings.Join(draft, "\n") }); err != nil {
		return err
	}
	return say(ctx, t, tr(t, "sim.init.done", s.DocName))
}

// ---- file helpers ----

func listDir(dir string) ([]os.DirEntry, string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err.Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "total %d\n", len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		fmt.Fprintf(&b, "%s %9d %s %s\n", info.Mode(), info.Size(), info.ModTime().Format("Jan _2 15:04"), name)
	}
	return entries, strings.TrimRight(b.String(), "\n")
}

func pickFile(entries []os.DirEntry) string {
	preferred := map[string]bool{".go": true, ".md": true, ".rs": true, ".py": true, ".js": true, ".ts": true, ".toml": true, ".json": true, ".mod": true, ".txt": true}
	fallback := ""
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		if preferred[filepath.Ext(e.Name())] {
			return e.Name()
		}
		if fallback == "" {
			fallback = e.Name()
		}
	}
	return fallback
}

// Head returns the first n lines of a text file, numbered like a Read tool.
func Head(path string, n int) string {
	f, err := os.Open(path)
	if err != nil {
		return err.Error()
	}
	defer f.Close()
	buf := make([]byte, 64<<10)
	k, _ := f.Read(buf)
	if !utf8.Valid(buf[:k]) {
		return "(binary file)"
	}
	lines := strings.Split(strings.ReplaceAll(string(buf[:k]), "\r\n", "\n"), "\n")
	var b strings.Builder
	for i, l := range lines {
		if i == n {
			fmt.Fprintf(&b, "… %d more lines", len(lines)-n)
			break
		}
		fmt.Fprintf(&b, "%4d  %s\n", i+1, l)
	}
	return strings.TrimRight(b.String(), "\n")
}

// summary is what the simulated Read tool reports for an attachment.
func summary(t *oriagent.Turn, a *oriagent.Attachment) string {
	switch {
	case a.Ext == "pdf":
		return tr(t, "sim.att.pdf", oriagent.HumanSize(a.Size, t))
	case a.IsImage():
		return tr(t, "sim.att.image", a.ImageSize.X, a.ImageSize.Y, oriagent.HumanSize(a.Size, t))
	}
	return Head(a.Path, 20)
}
