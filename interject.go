package oriagent

// Messages sent during a turn: while the agent works, the composer stays
// usable; what the user sends is shown at once, marked as waiting, and
// given to the agent when it asks for it (Turn.TakeMessages), or as a new
// turn when this one ends without taking it.

// QueuedMessage is a message the user sent while a turn was running.
type QueuedMessage struct {
	Text        string
	Attachments []*Attachment
}

// queue adds a message to the running turn of s. Called with the lock held.
func (a *App) queue(s *Session, text string, atts []*Attachment) {
	m := UserMsg(text, atts...)
	m.Queued = true
	s.add(m)
	s.pending = append(s.pending, m)
}

// takePending returns the waiting messages of s, now delivered. Called with
// the lock held.
func (s *Session) takePending() []QueuedMessage {
	out := make([]QueuedMessage, 0, len(s.pending))
	for _, m := range s.pending {
		m.Queued = false
		out = append(out, QueuedMessage{Text: m.Text, Attachments: m.Attachments})
	}
	s.pending = nil
	return out
}

// TakeMessages returns the messages the user sent since the turn started,
// or since the last call; the agent adds them to its work, typically before
// its next model call. Those it does not take start a new turn when this one
// ends. A subagent's turn has none.
func (t *Turn) TakeMessages() []QueuedMessage {
	if t.s.parent != nil {
		return nil
	}
	var out []QueuedMessage
	t.app.update(func() { out = t.s.takePending() })
	return out
}

// afterTurn runs the messages left waiting at the end of a turn as a new
// one, or, when the user stopped the turn, puts their text back in the
// composer. Called with the lock held.
func (a *App) afterTurn(s *Session, canceled bool, fn TurnFunc) {
	if len(s.pending) == 0 {
		return
	}
	if canceled {
		if a.cur != s {
			// Another session's composer is shown: the messages stay in the
			// transcript, unanswered, rather than being lost.
			s.takePending()
			return
		}
		// Back in the composer, after what it already holds, attachments
		// included.
		texts := []string{a.editor.Text()}
		for _, m := range s.pending {
			texts = append(texts, m.Text)
			a.attachments = append(a.attachments, m.Attachments...)
			s.remove(m)
		}
		s.pending = nil
		a.editor.SetText(joinTexts(texts))
		return
	}
	sent := s.pending
	var texts []string
	var atts []*Attachment
	for _, q := range s.takePending() {
		texts = append(texts, q.Text)
		atts = append(atts, q.Attachments...)
	}
	a.runTurn(s, joinTexts(texts), atts, fn, sent)
}

func joinTexts(texts []string) string {
	out := ""
	for _, t := range texts {
		if t == "" {
			continue
		}
		if out != "" {
			out += "\n\n"
		}
		out += t
	}
	return out
}

// remove takes m out of the transcript.
func (s *Session) remove(m *Message) {
	for i, x := range s.Messages {
		if x == m {
			s.Messages = append(s.Messages[:i], s.Messages[i+1:]...)
			return
		}
	}
}
