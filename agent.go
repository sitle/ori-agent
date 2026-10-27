package oriagent

import (
	"context"
	"errors"
	"slices"
	"time"
)

// Agent produces the assistant side of a conversation turn: an LLM client,
// a scripted simulation, a rules engine… Run is called on its own goroutine
// and writes to the transcript through t. It should return promptly once ctx
// is canceled (the user pressed Stop).
type Agent interface {
	Run(ctx context.Context, t *Turn) error
}

// AgentFunc adapts a function to the Agent interface.
type AgentFunc func(ctx context.Context, t *Turn) error

func (f AgentFunc) Run(ctx context.Context, t *Turn) error { return f(ctx, t) }

// HistoryEntry is a past exchange, for agents that send context to a model.
type HistoryEntry struct {
	Role string // "user" or "assistant"
	Text string
}

// Turn is the agent's handle on one turn. Its methods are safe to call from
// any goroutine: they lock the UI state and request a redraw.
type Turn struct {
	Prompt        string
	Attachments   []*Attachment
	History       []HistoryEntry // the session before this turn
	Thinking      bool           // extended thinking is enabled in the settings
	AllowCommands bool           // the settings let the agent run commands (the agent enforces it)
	AskEdits      bool           // the settings ask to confirm file edits (the agent enforces it)
	Model         Model
	WorkDir       string
	Locale        string // interface language, so the agent can answer in it
	// SessionID is the Session.ID of the turn's session, stable across
	// turns and, when the application restores sessions with their ID,
	// across launches.
	SessionID string
	// Mode is the working mode chosen in the composer chip when the turn
	// started (Config.Modes); empty without modes.
	Mode string

	app     *App
	s       *Session
	created []*Message // to close blocks left running on cancel
	parent  *ToolCall  // for subagent turns
}

// T translates key in the turn's language, with the framework catalogs and
// Config.Translations: agents can keep their texts there too.
func (t *Turn) T(key string, args ...any) string { return t.app.tr.In(t.Locale, key, args...) }

func (t *Turn) add(m *Message) {
	t.app.update(func() {
		t.s.add(m)
		t.created = append(t.created, m)
	})
}

// Block is a text or reasoning block being streamed.
type Block struct {
	t     *Turn
	m     *Message
	start time.Time
}

// Text starts an assistant text block.
func (t *Turn) Text() *Block {
	b := &Block{t: t, m: &Message{Kind: KindAssistant}, start: time.Now()}
	t.add(b.m)
	return b
}

// Think starts a reasoning block: shown while streaming, then collapsed to
// "Thought for Ns" by Done.
func (t *Turn) Think() *Block {
	b := &Block{t: t, m: &Message{Kind: KindThinking, Running: true}, start: time.Now()}
	t.add(b.m)
	return b
}

// Write appends streamed text.
func (b *Block) Write(s string) {
	b.t.app.update(func() { b.m.Text += s })
}

func (b *Block) Done() {
	b.t.app.update(func() { b.done() })
}

func (b *Block) done() {
	b.m.Running = false
	if b.m.Kind == KindThinking && b.m.Secs == 0 {
		b.m.Secs = max(1, int(time.Since(b.start).Round(time.Second).Seconds()))
	}
}

// ToolCall is a tool invocation shown in the transcript.
type ToolCall struct {
	t *Turn
	m *Message
}

// Tool shows a running tool call, e.g. Tool("Read", "main.go").
func (t *Turn) Tool(name, arg string) *ToolCall {
	c := &ToolCall{t: t, m: &Message{Kind: KindTool, ToolName: name, ToolArg: arg, Running: true}}
	t.add(c.m)
	return c
}

// File makes the tool argument a link that opens path in the preview pane.
func (c *ToolCall) File(path string) *ToolCall {
	c.t.app.update(func() { c.m.FilePath = path })
	return c
}

// Diff shows the change the call made to a file, from old to new, in
// place of its output, unfolded; the header counts the lines added and
// removed.
func (c *ToolCall) Diff(old, new string) *ToolCall {
	c.t.app.update(func() { c.m.HasDiff, c.m.DiffOld, c.m.DiffNew, c.m.Expanded = true, old, new, true })
	return c
}

// Done marks the call finished with its output (shown when expanded).
func (c *ToolCall) Done(output string) {
	c.t.app.update(func() { c.m.Output, c.m.Running = output, false })
}

// TaskList is a live checklist.
type TaskList struct {
	t *Turn
	m *Message
}

// Tasks shows a checklist; the first item starts running.
func (t *Turn) Tasks(items ...string) *TaskList {
	m := &Message{Kind: KindTasks}
	for i, it := range items {
		st := TaskPending
		if i == 0 {
			st = TaskRunning
		}
		m.Tasks = append(m.Tasks, Task{Text: it, Status: st})
	}
	t.add(m)
	return &TaskList{t: t, m: m}
}

func (l *TaskList) Set(i int, st TaskStatus) {
	l.t.app.update(func() {
		if i >= 0 && i < len(l.m.Tasks) {
			l.m.Tasks[i].Status = st
		}
	})
}

// Advance marks item i done and starts the next one.
func (l *TaskList) Advance(i int) {
	l.Set(i, TaskDone)
	l.Set(i+1, TaskRunning)
}

// Subagent spawns a child session under the current one, shown in the
// sidebar tree and as an "Agent" tool call here. Write the subagent's work
// through the returned turn, then call its Finish.
func (t *Turn) Subagent(title, prompt string) *Turn {
	call := &ToolCall{t: t, m: &Message{Kind: KindTool, ToolName: "Agent", ToolArg: title, Running: true}}
	var child *Turn
	t.app.update(func() {
		sub := t.s.Spawn(title)
		sub.Busy = true
		call.m.Sub = sub
		t.s.add(call.m)
		t.created = append(t.created, call.m)
		sub.add(UserMsg(prompt))
		child = &Turn{Prompt: prompt, Thinking: t.Thinking, AllowCommands: t.AllowCommands, AskEdits: t.AskEdits, Model: t.Model, WorkDir: t.WorkDir, Locale: t.Locale, SessionID: sub.ID, app: t.app, s: sub, parent: call}
	})
	return child
}

// SetUser replaces the profile (Profile page and sidebar footer), for
// example after a sign-in turn.
func (t *Turn) SetUser(u User) {
	t.app.update(func() { t.app.setUser(u) })
}

// SetModels replaces the model list, as Ctl.SetModels. The running turn
// keeps its Model.
func (t *Turn) SetModels(ms []Model) {
	t.app.update(func() { t.app.setModels(ms) })
}

// Finish ends a subagent turn: report becomes the parent's tool output.
func (t *Turn) Finish(report string) {
	t.app.update(func() {
		t.s.Busy = false
		t.closeOpen()
		if t.parent != nil {
			t.parent.m.Output, t.parent.m.Running = report, false
		}
	})
}

// Notify shows a transient message at the bottom of the window.
func (t *Turn) Notify(msg string) {
	t.app.update(func() { t.app.toast.Show(msg) })
}

// closeOpen stops anything the turn left running (on cancel or error).
// Called with the lock held.
func (t *Turn) closeOpen() {
	for _, m := range t.created {
		if ap := m.approval; ap != nil && !ap.done {
			ap.done, ap.canceled = true, true // left unanswered: no live buttons
		}
		if m.Running {
			if m.Kind == KindThinking && m.Secs == 0 {
				m.Secs = 1
			}
			m.Running = false
		}
		if m.Sub != nil && m.Sub.Busy {
			m.Sub.Busy = false
		}
		for i := range m.Tasks {
			if m.Tasks[i].Status == TaskRunning {
				m.Tasks[i].Status = TaskPending
			}
		}
	}
}

// TurnFunc runs a turn; commands use it to start custom turns.
type TurnFunc func(ctx context.Context, t *Turn) error

// history is what the agent is given of s's earlier exchanges, but skip.
func (s *Session) history(skip ...*Message) []HistoryEntry {
	var hist []HistoryEntry
	for _, m := range s.Messages {
		if slices.Contains(skip, m) {
			continue
		}
		switch m.Kind {
		case KindUser:
			hist = append(hist, HistoryEntry{Role: "user", Text: m.Text})
		case KindAssistant:
			hist = append(hist, HistoryEntry{Role: "assistant", Text: m.Text})
		}
	}
	return hist
}

// startTurn runs fn for session s. Called with the lock held.
func (a *App) startTurn(s *Session, prompt string, atts []*Attachment, fn TurnFunc) bool {
	return a.runTurn(s, prompt, atts, fn, nil)
}

// runTurn is startTurn; with sent, the prompt is already in the transcript
// as these messages (sent during the previous turn), which the history
// leaves out since they are the prompt.
func (a *App) runTurn(s *Session, prompt string, atts []*Attachment, fn TurnFunc, sent []*Message) bool {
	if s.Busy || a.locked(s) {
		return false
	}
	hist := s.history(sent...)
	if sent == nil && (prompt != "" || len(atts) > 0) {
		s.add(UserMsg(prompt, atts...))
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.Busy, s.cancel = true, cancel
	perms := a.permissions(s)
	t := &Turn{Prompt: prompt, Attachments: atts, History: hist, Thinking: a.setup.thinking.Value,
		AllowCommands: perms.AllowCommands, AskEdits: perms.AskEdits, Model: a.model(), WorkDir: a.sessionWorkDir(s), Locale: a.tr.Locale(), SessionID: s.ensureID(), Mode: a.sessionMode(s), app: a, s: s}
	go func() {
		err := fn(ctx, t)
		canceled := ctx.Err() != nil
		cancel()
		a.update(func() {
			s.Busy, s.cancel = false, nil
			t.closeOpen()
			switch {
			case canceled:
				s.add(&Message{Kind: KindNotice, Text: "notice.interrupted"})
			case err != nil && !errors.Is(err, context.Canceled):
				s.add(&Message{Kind: KindNotice, Text: err.Error(), Error: true})
			}
			a.afterTurn(s, canceled, a.cfg.Agent.Run)
		})
	}()
	return true
}
