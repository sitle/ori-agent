package ui

import (
	"image"
	"image/color"
	"regexp"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// MDKind is the kind of a Markdown block.
type MDKind int

const (
	MDPara MDKind = iota
	MDH1
	MDH2
	MDH3
	MDBullet
	MDNumber
	MDQuote
	MDCode
	MDTable
	MDBlank
	MDRule
)

// MDBlock is a block of a Markdown document. Text keeps its inline markup
// (bold, italic, code, links), drawn by RichText.
type MDBlock struct {
	Kind   MDKind
	Text   string
	Lines  []string // code block and table rows
	Level  int      // list nesting, from 0
	Marker string   // "3." for a numbered item
}

var mdNumber = regexp.MustCompile(`^(\d{1,3})[.)] (.*)$`)

// ParseMarkdown handles the subset of Markdown that chat answers and READMEs
// mostly use: headings, paragraphs, bulleted and numbered lists, quotes,
// fenced code, tables and rules. Lines of a paragraph are joined.
func ParseMarkdown(src string) []MDBlock {
	var bs []MDBlock
	var code *MDBlock
	for _, ln := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			if code != nil {
				bs, code = append(bs, *code), nil
			} else {
				code = &MDBlock{Kind: MDCode}
			}
			continue
		}
		if code != nil {
			code.Lines = append(code.Lines, ln)
			continue
		}
		indent := len(ln) - len(strings.TrimLeft(ln, " \t"))
		last := len(bs) - 1
		switch {
		case t == "":
			bs = append(bs, MDBlock{Kind: MDBlank})
		case strings.HasPrefix(t, "#### "), strings.HasPrefix(t, "### "):
			bs = append(bs, MDBlock{Kind: MDH3, Text: strings.TrimSpace(strings.TrimLeft(t, "#"))})
		case strings.HasPrefix(t, "## "):
			bs = append(bs, MDBlock{Kind: MDH2, Text: t[3:]})
		case strings.HasPrefix(t, "# "):
			bs = append(bs, MDBlock{Kind: MDH1, Text: t[2:]})
		case t == "---" || t == "***" || t == "___":
			bs = append(bs, MDBlock{Kind: MDRule})
		case strings.HasPrefix(t, "- "), strings.HasPrefix(t, "* "), strings.HasPrefix(t, "+ "):
			bs = append(bs, MDBlock{Kind: MDBullet, Text: t[2:], Level: min(indent/2, 3)})
		case mdNumber.MatchString(t):
			m := mdNumber.FindStringSubmatch(t)
			bs = append(bs, MDBlock{Kind: MDNumber, Marker: m[1] + ".", Text: m[2], Level: min(indent/2, 3)})
		case strings.HasPrefix(t, ">"):
			q := strings.TrimSpace(strings.TrimPrefix(t, ">"))
			if last >= 0 && bs[last].Kind == MDQuote {
				bs[last].Text += " " + q
			} else {
				bs = append(bs, MDBlock{Kind: MDQuote, Text: q})
			}
		case strings.HasPrefix(t, "|"):
			if last >= 0 && bs[last].Kind == MDTable {
				bs[last].Lines = append(bs[last].Lines, t)
			} else {
				bs = append(bs, MDBlock{Kind: MDTable, Lines: []string{t}})
			}
		case last >= 0 && (bs[last].Kind == MDBullet || bs[last].Kind == MDNumber) && indent > 0:
			bs[last].Text += " " + t // indented continuation of a list item
		case last >= 0 && bs[last].Kind == MDPara:
			bs[last].Text += " " + t
		default:
			bs = append(bs, MDBlock{Kind: MDPara, Text: t})
		}
	}
	if code != nil {
		bs = append(bs, *code) // still streaming, or never closed
	}
	return bs
}

// Span is a run of text with one style.
type Span struct {
	Text                     string
	Bold, Italic, Code, Link bool
}

var mdLinkRe = regexp.MustCompile(`^\[([^\]]+)\]\(([^)\s]+)\)`)

// ParseInline splits text into styled spans: **bold**, *italic*, `code`
// and [links](url), which keep their text. A marker without its closing
// one stays as written, as it does while an answer is still streaming.
func ParseInline(s string) []Span {
	var spans []Span
	var cur strings.Builder
	bold, italic := false, false
	flush := func() {
		if cur.Len() > 0 {
			spans = append(spans, Span{Text: cur.String(), Bold: bold, Italic: italic})
			cur.Reset()
		}
	}
	for i := 0; i < len(s); {
		rest := s[i:]
		switch {
		case strings.HasPrefix(rest, `\`) && len(rest) > 1 && strings.ContainsRune("*_`[]\\", rune(rest[1])):
			cur.WriteByte(rest[1])
			i += 2
			continue
		case rest[0] == '`':
			if end := strings.IndexByte(rest[1:], '`'); end >= 0 {
				flush()
				spans = append(spans, Span{Text: rest[1 : 1+end], Code: true, Bold: bold})
				i += end + 2
				continue
			}
		case strings.HasPrefix(rest, "**") || strings.HasPrefix(rest, "__"):
			if bold || strings.Contains(rest[2:], rest[:2]) {
				flush()
				bold = !bold
				i += 2
				continue
			}
		case rest[0] == '*' && len(rest) > 1 && rest[1] != ' ':
			if italic || strings.Contains(rest[1:], "*") {
				flush()
				italic = !italic
				i++
				continue
			}
		case rest[0] == '*' && italic:
			flush()
			italic = false
			i++
			continue
		case rest[0] == '[':
			if m := mdLinkRe.FindStringSubmatch(rest); m != nil {
				flush()
				spans = append(spans, Span{Text: m[1], Link: true, Bold: bold, Italic: italic})
				i += len(m[0])
				continue
			}
		}
		cur.WriteByte(s[i])
		i++
	}
	flush()
	return spans
}

// PlainInline is text without its inline markup.
func PlainInline(s string) string {
	var b strings.Builder
	for _, sp := range ParseInline(s) {
		b.WriteString(sp.Text)
	}
	return b.String()
}

// RichText draws styled spans as a paragraph that wraps at word
// boundaries. Non-breaking spaces keep their words together.
func (t *Theme) RichText(size unit.Sp, c color.NRGBA, w font.Weight, spans []Span) layout.Widget {
	return func(gtx C) D {
		maxW := gtx.Constraints.Max.X
		space := gtx.Sp(size) * 28 / 100
		type item struct {
			call op.CallOp
			dims D
			x    int
			code bool
		}
		var line []item
		x, y, width := 0, 0, 0
		flushLine := func() {
			asc, desc := 0, 0
			for _, it := range line {
				asc = max(asc, it.dims.Size.Y-it.dims.Baseline)
				desc = max(desc, it.dims.Baseline)
			}
			for _, it := range line {
				off := image.Pt(it.x, y+asc-(it.dims.Size.Y-it.dims.Baseline))
				st := op.Offset(off).Push(gtx.Ops)
				if it.code {
					pad := gtx.Dp(3)
					r := image.Rect(-pad, gtx.Dp(2), it.dims.Size.X+pad, it.dims.Size.Y-gtx.Dp(2))
					paint.FillShape(gtx.Ops, t.Code, clip.UniformRRect(r, gtx.Dp(4)).Op(gtx.Ops))
				}
				it.call.Add(gtx.Ops)
				st.Pop()
			}
			if len(line) > 0 {
				last := line[len(line)-1]
				width = max(width, last.x+last.dims.Size.X)
				y += asc + desc
			}
			line, x = line[:0], 0
		}
		spaceBefore := false
		for _, sp := range spans {
			words := strings.FieldsFunc(sp.Text, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' })
			lead := len(sp.Text) > 0 && strings.ContainsRune(" \t\n", rune(sp.Text[0]))
			for i, word := range words {
				l := t.Label(size, word, c, w)
				l.LineHeightScale = 1.45
				switch {
				case sp.Code:
					l.Font.Typeface = t.MonoFace
					l.TextSize = size * 92 / 100
				case sp.Link:
					l.Color = t.Accent
				}
				if sp.Bold {
					l.Font.Weight = font.SemiBold
				}
				if sp.Italic {
					l.Font.Style = font.Italic
				}
				macro := op.Record(gtx.Ops)
				cgtx := gtx
				cgtx.Constraints = layout.Constraints{Max: image.Pt(maxW, gtx.Constraints.Max.Y)}
				dims := l.Layout(cgtx)
				call := macro.Stop()
				gap := 0
				if x > 0 && (i > 0 || lead || spaceBefore) {
					gap = space
				}
				if x > 0 && x+gap+dims.Size.X > maxW {
					flushLine()
					gap = 0
				}
				line = append(line, item{call: call, dims: dims, x: x + gap, code: sp.Code})
				x += gap + dims.Size.X
			}
			spaceBefore = len(sp.Text) > 0 && strings.ContainsRune(" \t\n", rune(sp.Text[len(sp.Text)-1]))
			if len(words) == 0 && len(sp.Text) > 0 {
				spaceBefore = true
			}
		}
		flushLine()
		return D{Size: image.Pt(max(width, gtx.Constraints.Min.X), y)}
	}
}

// Markdown draws a Markdown text, such as a chat answer, at body size.
func (t *Theme) Markdown(size unit.Sp, src string) layout.Widget {
	return func(gtx C) D {
		blocks := ParseMarkdown(src)
		// Blank lines only separate blocks: none at the ends, one at most
		// in a row.
		var kept []MDBlock
		for _, b := range blocks {
			if b.Kind == MDBlank && (len(kept) == 0 || kept[len(kept)-1].Kind == MDBlank) {
				continue
			}
			kept = append(kept, b)
		}
		for len(kept) > 0 && kept[len(kept)-1].Kind == MDBlank {
			kept = kept[:len(kept)-1]
		}
		children := make([]layout.FlexChild, len(kept))
		for i, b := range kept {
			children[i] = layout.Rigid(func(gtx C) D { return t.MarkdownBlock(gtx, size, b) })
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// MarkdownBlock draws one block; size is the body text size.
func (t *Theme) MarkdownBlock(gtx C, size unit.Sp, b MDBlock) D {
	body := func(c color.NRGBA) layout.Widget { return t.RichText(size, c, font.Normal, ParseInline(b.Text)) }
	heading := func(scale float32) layout.Widget {
		return t.RichText(unit.Sp(float32(size)*scale), t.Text, font.SemiBold, ParseInline(b.Text))
	}
	switch b.Kind {
	case MDH1:
		return layout.Inset{Top: 6, Bottom: 10}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(heading(1.6)),
				VSpace(8),
				layout.Rigid(t.HLine),
			)
		})
	case MDH2:
		return layout.Inset{Top: 12, Bottom: 4}.Layout(gtx, heading(1.3))
	case MDH3:
		return layout.Inset{Top: 8, Bottom: 2}.Layout(gtx, heading(1.12))
	case MDPara:
		return body(t.Text)(gtx)
	case MDBullet, MDNumber:
		return layout.Inset{Top: 2, Bottom: 2, Left: unit.Dp(4 + 18*b.Level)}.Layout(gtx, func(gtx C) D {
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					if b.Kind == MDNumber {
						l := t.Label(size, b.Marker, t.Muted, font.Normal)
						l.LineHeightScale = 1.45
						return l.Layout(gtx)
					}
					return layout.Inset{Top: unit.Dp(float32(size) * 0.58)}.Layout(gtx, func(gtx C) D { return Dot(gtx, t.Muted, 5) })
				}),
				HSpace(10),
				layout.Flexed(1, body(t.Text)),
			)
		})
	case MDQuote:
		return layout.Inset{Top: 2, Bottom: 2}.Layout(gtx, func(gtx C) D { return Ruled(gtx, t.BorderHi, body(t.Muted)) })
	case MDCode:
		return layout.Inset{Top: 4, Bottom: 4}.Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return Bordered(gtx, t.Code, t.Border, 6, func(gtx C) D {
				return layout.UniformInset(12).Layout(gtx, t.CodeLines(strings.Join(b.Lines, "\n")))
			})
		})
	case MDTable:
		return layout.Inset{Top: 4, Bottom: 6}.Layout(gtx, func(gtx C) D { return t.mdTable(gtx, size*93/100, b.Lines) })
	case MDRule:
		return layout.Inset{Top: 10, Bottom: 10}.Layout(gtx, t.HLine)
	case MDBlank:
		return layout.Spacer{Height: 8}.Layout(gtx)
	}
	return D{}
}
