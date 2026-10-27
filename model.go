package oriagent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"gioui.org/widget"
)

// Kind is the type of a transcript item.
type Kind int

const (
	KindUser      Kind = iota // user bubble, with optional attachments
	KindAssistant             // assistant text
	KindThinking              // collapsible reasoning
	KindTool                  // tool call with expandable output
	KindTasks                 // task list
	KindCommand               // slash command and its output
	KindNotice                // interruption or error
	KindApproval              // a request the user allows or denies (Turn.Approve)
)

type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskRunning
	TaskDone
)

type Task struct {
	Text   string
	Status TaskStatus
}

// Message is one transcript item. Agents create them through a Turn; demo
// transcripts can build them directly.
type Message struct {
	Kind     Kind
	Text     string
	ToolName string
	ToolArg  string
	Output   string
	Running  bool
	Tasks    []Task
	Secs     int      // thinking duration
	Sub      *Session // subagent behind an Agent tool call: a click opens it
	FilePath string   // file behind a tool call: a click previews it
	Error    bool     // KindNotice: an error rather than an interruption
	Expanded bool     // tool output or reasoning shown
	// With HasDiff, a tool call shows the change from DiffOld to DiffNew
	// instead of its output (ToolCall.Diff).
	HasDiff          bool
	DiffOld, DiffNew string

	diff     diffCache
	approval *approval

	Attachments []*Attachment
	// Queued marks a message the user sent while a turn was running, not yet
	// given to the agent (Turn.TakeMessages).
	Queued bool

	toggle, toggleTail, open widget.Clickable
	rewind                   rewindState
}

// Session is a conversation; Children are the subagents it spawned.
type Session struct {
	// ID identifies the session for the agent (Turn.SessionID), so that it
	// finds its own records again. Applications set it on the sessions
	// they restore; ori-agent gives one to the others.
	ID       string
	Title    string
	Messages []*Message
	Children []*Session
	Busy     bool
	Open     bool // subagents shown in the sidebar
	// ContextUsed and ContextWindow fill the context gauge of the composer
	// (Turn.SetContext); applications restore them with the session.
	ContextUsed, ContextWindow int

	pending   []*Message // sent during the running turn, see interject.go
	waitSince time.Time  // since when the turn shows nothing moving, see chat.go
	rewinding bool       // Config.OnRewind is running: no new turn

	project *Project
	parent  *Session
	cancel  context.CancelFunc
	btn     widget.Clickable
	chev    widget.Clickable
	delBtn  widget.Clickable // delete, on the row in the sidebar
}

func (s *Session) add(m *Message) { s.Messages = append(s.Messages, m) }

// NewSessionID returns a random identifier, as given to new sessions.
func NewSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ensureID gives s an identifier if it has none.
func (s *Session) ensureID() string {
	if s.ID == "" {
		s.ID = NewSessionID()
	}
	return s.ID
}

func (s *Session) Project() *Project { return s.project }
func (s *Session) Parent() *Session  { return s.parent }

// Spawn attaches a subagent session to s (for building demo trees; agents use
// Turn.Subagent).
func (s *Session) Spawn(title string) *Session {
	c := &Session{ID: NewSessionID(), Title: title, project: s.project, parent: s}
	s.Children = append(s.Children, c)
	s.Open = true
	return c
}

// Project groups sessions in the sidebar.
type Project struct {
	Name        string
	Description string
	// WorkDir is the folder of the project: Turn.WorkDir for its sessions.
	// Empty: the folder of the configuration.
	WorkDir string
	// Permissions of the project's sessions; nil: the Configuration's.
	Permissions *Permissions
	// Permanent projects cannot be removed from their page.
	Permanent bool
	// Mode is the working mode of the project's sessions, among
	// Config.Modes (the composer chip); empty: the first one.
	Mode     string
	Sessions []*Session
	Open     bool

	btn, add, manage widget.Clickable
}

// NewProject returns an open project holding sessions (first on top).
func NewProject(name string, sessions ...*Session) *Project {
	p := &Project{Name: name, Open: true}
	for i := len(sessions) - 1; i >= 0; i-- {
		p.Add(sessions[i])
	}
	return p
}

// Add puts s at the top of the project.
func (p *Project) Add(s *Session) {
	var walk func(s *Session)
	walk = func(s *Session) {
		s.project = p
		s.ensureID()
		for _, c := range s.Children {
			walk(c)
		}
	}
	walk(s)
	p.Sessions = append([]*Session{s}, p.Sessions...)
}

// Message constructors, for demo transcripts.

func UserMsg(text string, atts ...*Attachment) *Message {
	return &Message{Kind: KindUser, Text: text, Attachments: atts}
}
func TextMsg(text string) *Message { return &Message{Kind: KindAssistant, Text: text} }
func ThinkingMsg(text string, secs int) *Message {
	return &Message{Kind: KindThinking, Text: text, Secs: secs}
}
func ToolMsg(name, arg, output string) *Message {
	return &Message{Kind: KindTool, ToolName: name, ToolArg: arg, Output: output}
}
func TasksMsg(tasks ...Task) *Message { return &Message{Kind: KindTasks, Tasks: tasks} }

func walkSessions(ps []*Project, f func(*Session)) {
	var walk func(ss []*Session)
	walk = func(ss []*Session) {
		for _, s := range ss {
			f(s)
			walk(s.Children)
		}
	}
	for _, p := range ps {
		walk(p.Sessions)
	}
}
