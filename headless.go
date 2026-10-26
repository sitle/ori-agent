package oriagent

import (
	"context"
	"errors"
)

// ErrSessionBusy is returned by Headless.Run when a turn is already
// running in the session.
var ErrSessionBusy = errors.New("oriagent: a turn is already running in this session")

// Headless runs an agent without a window, through the same Turn as the
// application: for tests, evaluation benches and scripts. The transcript
// of each session can be read afterwards (Session.Messages).
type Headless struct {
	app *App
	// Approve answers the approval requests of the agent (Turn.Approve), on
	// the agent's goroutine; nil denies them.
	Approve func(ApprovalRequest) Decision
}

// TurnOptions are the settings of a headless turn, which the window takes
// from the Configuration page and the selected model.
type TurnOptions struct {
	Thinking      bool
	AllowCommands bool
	AskEdits      bool
	Model         Model
	WorkDir       string
	Locale        string // default: the configuration's
	Mode          string // default: the session's (Config.Modes)
}

// NewHeadless prepares cfg.Agent to run without a window; the translations
// of cfg are used for the texts the agent writes through the turn.
func NewHeadless(cfg Config) *Headless {
	cfg.defaults()
	h := &Headless{}
	h.app = newApp(cfg, nil)
	h.app.headless = h
	return h
}

// Run runs one turn of the agent in session s, or in a new session when s
// is nil, and returns once the agent is done, with the session.
func (h *Headless) Run(ctx context.Context, s *Session, prompt string, atts []*Attachment, o TurnOptions) (*Session, error) {
	a := h.app
	locale := o.Locale
	if locale == "" {
		locale = a.tr.Locale()
	}
	var t *Turn
	busy := false
	a.update(func() {
		if s == nil {
			s = &Session{ID: NewSessionID(), Title: prompt}
		}
		if busy = s.Busy; busy {
			return
		}
		hist := s.history()
		if prompt != "" || len(atts) > 0 {
			s.add(UserMsg(prompt, atts...))
		}
		s.Busy = true
		t = &Turn{Prompt: prompt, Attachments: atts, History: hist, Thinking: o.Thinking,
			AllowCommands: o.AllowCommands, AskEdits: o.AskEdits, Model: o.Model, WorkDir: o.WorkDir,
			Locale: locale, SessionID: s.ensureID(), Mode: a.sessionMode(s), app: a, s: s}
		if o.Mode != "" {
			t.Mode = o.Mode
		}
	})
	if busy {
		return s, ErrSessionBusy
	}
	err := a.cfg.Agent.Run(ctx, t)
	a.update(func() {
		s.Busy = false
		t.closeOpen()
	})
	return s, err
}

// approve answers req at once, for a headless turn. Called without the
// lock.
func (h *Headless) approve(req ApprovalRequest) Decision {
	if h.Approve == nil {
		return Deny
	}
	return h.Approve(req)
}
