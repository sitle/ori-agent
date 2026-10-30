package oriagent

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
)

// The composer recalls what was sent, as a shell does: Up shows the
// previous messages and commands, Down comes back, past the newest to an
// empty composer. The arrows recall only while the composer is empty or
// shows a recalled entry, so that they still move the caret in a text
// being written.

// maxHistory bounds the entries recalled.
const maxHistory = 100

type inputHistory struct {
	entries []string
	pos     int // entry shown; len(entries): none
}

// remember adds text sent from the composer, unless it repeats the last.
func (a *App) remember(text string) {
	h := &a.history
	if text == "" || (len(h.entries) > 0 && h.entries[len(h.entries)-1] == text) {
		h.pos = len(h.entries)
		return
	}
	h.entries = append(h.entries, text)
	if len(h.entries) > maxHistory {
		h.entries = h.entries[len(h.entries)-maxHistory:]
	}
	h.pos = len(h.entries)
}

// handleHistory runs before the palette and the mentions see the frame's
// key events: a recalled command opens the palette, whose arrows would
// otherwise take over.
func (a *App) handleHistory(gtx C) {
	h := &a.history
	text := a.editor.Text()
	browsing := h.pos < len(h.entries) && text == h.entries[h.pos]
	if !browsing {
		h.pos = len(h.entries)
		if text != "" || len(h.entries) == 0 {
			return
		}
	}
	filters := []event.Filter{key.Filter{Focus: &a.editor, Name: key.NameUpArrow}}
	if browsing {
		filters = append(filters, key.Filter{Focus: &a.editor, Name: key.NameDownArrow})
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		switch e.Name {
		case key.NameUpArrow:
			if h.pos > 0 {
				h.pos--
			}
		case key.NameDownArrow:
			h.pos++
		}
		if h.pos < len(h.entries) {
			a.recall(h.entries[h.pos])
		} else {
			a.editor.SetText("")
			return
		}
	}
}

// recall puts text in the composer, the caret at its end.
func (a *App) recall(text string) {
	a.editor.SetText(text)
	n := len([]rune(text))
	a.editor.SetCaret(n, n)
}
