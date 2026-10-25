package preview

import (
	"regexp"
	"strconv"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"

	"github.com/sitle/ori-agent/ui"
)

func itoa(n int) string { return strconv.Itoa(n) }

type blockKind int

const (
	bPara blockKind = iota
	bH1
	bH2
	bH3
	bBullet
	bCode
	bTable
	bBlank
	bRule
)

type block struct {
	kind  blockKind
	text  string
	lines []string
}

var (
	mdLink    = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	mdMarkers = strings.NewReplacer("**", "", "`", "")
)

// inline drops inline markup: links keep their text, emphasis and code marks go.
func inline(s string) string { return mdMarkers.Replace(mdLink.ReplaceAllString(s, "$1")) }

// parseMarkdown handles the block-level subset that READMEs mostly use.
func parseMarkdown(src string) []block {
	var bs []block
	var code *block
	for _, ln := range strings.Split(src, "\n") {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "```") {
			if code != nil {
				bs, code = append(bs, *code), nil
			} else {
				code = &block{kind: bCode}
			}
			continue
		}
		if code != nil {
			code.lines = append(code.lines, ln)
			continue
		}
		last := len(bs) - 1
		switch {
		case t == "":
			bs = append(bs, block{kind: bBlank})
		case strings.HasPrefix(t, "### "):
			bs = append(bs, block{kind: bH3, text: inline(t[4:])})
		case strings.HasPrefix(t, "## "):
			bs = append(bs, block{kind: bH2, text: inline(t[3:])})
		case strings.HasPrefix(t, "# "):
			bs = append(bs, block{kind: bH1, text: inline(t[2:])})
		case strings.HasPrefix(t, "- "), strings.HasPrefix(t, "* "):
			bs = append(bs, block{kind: bBullet, text: inline(t[2:])})
		case t == "---" || t == "***":
			bs = append(bs, block{kind: bRule})
		case strings.HasPrefix(t, "|"):
			if last >= 0 && bs[last].kind == bTable {
				bs[last].lines = append(bs[last].lines, t)
			} else {
				bs = append(bs, block{kind: bTable, lines: []string{t}})
			}
		case last >= 0 && bs[last].kind == bBullet && ln != t:
			bs[last].text += " " + inline(t) // indented continuation of a list item
		case last >= 0 && bs[last].kind == bPara:
			bs[last].text += " " + inline(t)
		default:
			bs = append(bs, block{kind: bPara, text: inline(t)})
		}
	}
	if code != nil {
		bs = append(bs, *code)
	}
	return bs
}

func mdBlock(gtx C, th *ui.Theme, b block) D {
	switch b.kind {
	case bH1:
		return layout.Inset{Top: 6, Bottom: 10}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(th.Label(24, b.text, th.Text, font.SemiBold).Layout),
				ui.VSpace(8),
				layout.Rigid(th.HLine),
			)
		})
	case bH2:
		return layout.Inset{Top: 14, Bottom: 6}.Layout(gtx, th.Label(19, b.text, th.Text, font.SemiBold).Layout)
	case bH3:
		return layout.Inset{Top: 10, Bottom: 4}.Layout(gtx, th.Label(16, b.text, th.Text, font.SemiBold).Layout)
	case bPara:
		return th.Body(14, b.text)(gtx)
	case bBullet:
		return layout.Inset{Top: 2, Bottom: 2, Left: 4}.Layout(gtx, func(gtx C) D {
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return layout.Inset{Top: 8}.Layout(gtx, func(gtx C) D { return ui.Dot(gtx, th.Muted, 5) })
				}),
				ui.HSpace(10),
				layout.Flexed(1, th.Body(14, b.text)),
			)
		})
	case bCode:
		return layout.Inset{Top: 4, Bottom: 4}.Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return ui.Bordered(gtx, th.Code, th.Border, 6, func(gtx C) D {
				return layout.UniformInset(12).Layout(gtx, th.CodeLines(strings.Join(b.lines, "\n")))
			})
		})
	case bTable:
		return layout.Inset{Top: 2, Bottom: 2}.Layout(gtx, th.CodeLines(strings.Join(b.lines, "\n")))
	case bRule:
		return layout.Inset{Top: 10, Bottom: 10}.Layout(gtx, th.HLine)
	case bBlank:
		return layout.Spacer{Height: 8}.Layout(gtx)
	}
	return D{}
}
