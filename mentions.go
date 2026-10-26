package oriagent

import (
	"strconv"
	"unicode"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/ui"
)

// Mentions (Config.Mentions): typing "@" and a few letters in the composer
// lists what the application suggests, typically the files of the working
// directory; the arrows choose, Tab or Enter inserts "@path ".

// Mention is a suggestion for "@".
type Mention struct {
	Label  string // shown, e.g. the file name
	Detail string // shown after, muted, e.g. the folder
	Insert string // inserted after "@"; empty: Label
}

type mentionState struct {
	query     string // the query the results are for
	results   []Mention
	sel       int
	btns      []widget.Clickable
	dismissed string // "@query" closed with Escape, at its position
}

// mentionKey identifies an "@query" by its position and text.
func mentionKey(at int, q string) string { return strconv.Itoa(at) + "@" + q }

// dismissMention closes the list for the "@query" being typed, leaving the
// text as is; typing on opens it again.
func (a *App) dismissMention() {
	if q, at, ok := a.mentionQuery(); ok {
		a.mention.dismissed = mentionKey(at, q)
	}
}

// mentionQuery finds "@word" right before the caret: the word, and the
// position of "@" in runes.
func (a *App) mentionQuery() (string, int, bool) {
	if a.cfg.Mentions == nil {
		return "", 0, false
	}
	runes := []rune(a.editor.Text())
	_, caret := a.editor.Selection()
	if caret > len(runes) {
		caret = len(runes)
	}
	for i := caret - 1; i >= 0; i-- {
		r := runes[i]
		if unicode.IsSpace(r) {
			return "", 0, false
		}
		if r == '@' && (i == 0 || unicode.IsSpace(runes[i-1])) {
			return string(runes[i+1 : caret]), i, true
		}
	}
	return "", 0, false
}

// mentions are the suggestions for the current query, asked again only
// when it changes.
func (a *App) mentions() ([]Mention, bool) {
	q, at, ok := a.mentionQuery()
	if !ok || mentionKey(at, q) == a.mention.dismissed {
		a.mention.query, a.mention.results = "\x00", nil
		return nil, false
	}
	st := &a.mention
	if q != st.query {
		st.query, st.results, st.sel = q, a.cfg.Mentions(a.cur, q), 0
		if len(st.results) > 8 {
			st.results = st.results[:8]
		}
		st.btns = make([]widget.Clickable, len(st.results))
	}
	return st.results, len(st.results) > 0
}

// pickMention replaces "@query" with "@insert ".
func (a *App) pickMention(m Mention) {
	_, at, ok := a.mentionQuery()
	if !ok {
		return
	}
	runes := []rune(a.editor.Text())
	_, caret := a.editor.Selection()
	ins := m.Insert
	if ins == "" {
		ins = m.Label
	}
	text := string(runes[:at]) + "@" + ins + " " + string(runes[min(caret, len(runes)):])
	a.editor.SetText(text)
	pos := at + 1 + len([]rune(ins)) + 1
	a.editor.SetCaret(pos, pos)
	a.mention.query = "\x00"
}

// handleMentions runs before the editor sees the frame's key events, so the
// arrows, Tab and Escape drive the list while it is open.
func (a *App) handleMentions(gtx C) {
	results, open := a.mentions()
	if !open {
		return
	}
	st := &a.mention
	for {
		ev, ok := gtx.Event(
			key.Filter{Focus: &a.editor, Name: key.NameUpArrow},
			key.Filter{Focus: &a.editor, Name: key.NameDownArrow},
			key.Filter{Focus: &a.editor, Name: key.NameTab},
			key.Filter{Focus: &a.editor, Name: key.NameEscape},
		)
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		switch e.Name {
		case key.NameUpArrow:
			st.sel--
		case key.NameDownArrow:
			st.sel++
		case key.NameTab:
			a.pickMention(results[wrap(st.sel, len(results))])
			return
		case key.NameEscape:
			a.dismissMention()
			return
		}
	}
	for i := range st.btns {
		if st.btns[i].Clicked(gtx) {
			a.pickMention(results[i])
			a.focus = true
			return
		}
	}
	st.sel = wrap(st.sel, len(results))
}

// submitMention handles Enter while the list is open.
func (a *App) submitMention() bool {
	results, open := a.mentions()
	if !open {
		return false
	}
	a.pickMention(results[wrap(a.mention.sel, len(results))])
	return true
}

func (a *App) layoutMentions(gtx C) D {
	th, st := a.th, &a.mention
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return ui.Bordered(gtx, th.Surface, th.BorderHi, 12, func(gtx C) D {
		return layout.UniformInset(6).Layout(gtx, func(gtx C) D {
			children := []layout.FlexChild{
				layout.Rigid(func(gtx C) D {
					return layout.Inset{Top: 4, Bottom: 6, Left: 10}.Layout(gtx, th.Label(11, a.t("mention.header"), th.Muted, font.SemiBold).Layout)
				}),
			}
			for i, m := range st.results {
				selected := i == st.sel
				btn := &st.btns[i]
				children = append(children, layout.Rigid(func(gtx C) D {
					return btn.Layout(gtx, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						bg := th.Surface
						switch {
						case selected:
							bg = th.Selected
						case btn.Hovered():
							bg = th.Hover
						}
						return ui.Card(gtx, bg, 8, layout.Inset{Top: 7, Bottom: 7, Left: 10, Right: 10}, func(gtx C) D {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(th.Mono(13, "@"+m.Label, th.Accent).Layout),
								ui.HSpace(10),
								layout.Flexed(1, func(gtx C) D { return ui.OneLine(gtx, th.Label(12, m.Detail, th.Muted, font.Normal)) }),
							)
						})
					})
				}))
			}
			children = append(children, layout.Rigid(func(gtx C) D {
				return layout.Inset{Top: 6, Bottom: 2, Left: 10}.Layout(gtx, th.Label(11, a.t("mention.hint"), th.Muted, font.Normal).Layout)
			}))
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}
