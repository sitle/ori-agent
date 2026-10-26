package ui

import (
	"image"
	"image/color"
	"strconv"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// DiffLine is a line of a line-based diff: Op is ' ' (kept), '-' (removed)
// or '+' (added); Old and New are 1-based line numbers, 0 when the line is
// not on that side.
type DiffLine struct {
	Op       byte
	Old, New int
	Text     string
}

// maxDiffCells bounds the comparison table of the changed middle part;
// beyond, the whole middle is shown as removed then added.
const maxDiffCells = 4_000_000

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// DiffLines compares two texts line by line (longest common subsequence,
// after the common start and end are set aside).
func DiffLines(old, new string) []DiffLine {
	a, b := splitLines(old), splitLines(new)
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	var out []DiffLine
	for i := 0; i < pre; i++ {
		out = append(out, DiffLine{Op: ' ', Old: i + 1, New: i + 1, Text: a[i]})
	}
	ma, mb := a[pre:len(a)-suf], b[pre:len(b)-suf]
	n, m := len(ma), len(mb)
	if n*m > maxDiffCells {
		for i, l := range ma {
			out = append(out, DiffLine{Op: '-', Old: pre + i + 1, Text: l})
		}
		for j, l := range mb {
			out = append(out, DiffLine{Op: '+', New: pre + j + 1, Text: l})
		}
	} else {
		// lcs[i][j]: length of the common subsequence of ma[i:] and mb[j:].
		lcs := make([][]int32, n+1)
		for i := range lcs {
			lcs[i] = make([]int32, m+1)
		}
		for i := n - 1; i >= 0; i-- {
			for j := m - 1; j >= 0; j-- {
				if ma[i] == mb[j] {
					lcs[i][j] = lcs[i+1][j+1] + 1
				} else {
					lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
				}
			}
		}
		i, j := 0, 0
		for i < n || j < m {
			switch {
			case i < n && j < m && ma[i] == mb[j]:
				out = append(out, DiffLine{Op: ' ', Old: pre + i + 1, New: pre + j + 1, Text: ma[i]})
				i++
				j++
			case i < n && (j == m || lcs[i+1][j] >= lcs[i][j+1]):
				// Removed lines come before the added ones that replace them.
				out = append(out, DiffLine{Op: '-', Old: pre + i + 1, Text: ma[i]})
				i++
			default:
				out = append(out, DiffLine{Op: '+', New: pre + j + 1, Text: mb[j]})
				j++
			}
		}
	}
	for k := 0; k < suf; k++ {
		out = append(out, DiffLine{Op: ' ', Old: len(a) - suf + k + 1, New: len(b) - suf + k + 1, Text: a[len(a)-suf+k]})
	}
	return out
}

// DiffStat counts the added and removed lines.
func DiffStat(lines []DiffLine) (added, removed int) {
	for _, l := range lines {
		switch l.Op {
		case '+':
			added++
		case '-':
			removed++
		}
	}
	return
}

// Hunks keeps the changed lines with context lines around them; nil when
// nothing changed.
func Hunks(lines []DiffLine, context int) [][]DiffLine {
	var hunks [][]DiffLine
	keep := make([]bool, len(lines))
	for i, l := range lines {
		if l.Op != ' ' {
			for k := max(0, i-context); k <= min(len(lines)-1, i+context); k++ {
				keep[k] = true
			}
		}
	}
	var cur []DiffLine
	for i, l := range lines {
		if keep[i] {
			cur = append(cur, l)
		} else if cur != nil {
			hunks, cur = append(hunks, cur), nil
		}
	}
	if cur != nil {
		hunks = append(hunks, cur)
	}
	return hunks
}

// maxDiffRows is the number of diff rows drawn; the rest is counted.
const maxDiffRows = 300

// Diff draws the changes from old to new: hunks of changed lines with
// three lines of context, old and new line numbers, removed lines on red,
// added lines on green. A new file is all green; an empty text on both
// sides says so.
func (t *Theme) Diff(old, new string, noChange string) layout.Widget {
	hunks := Hunks(DiffLines(old, new), 3)
	return func(gtx C) D {
		if len(hunks) == 0 {
			return t.Label(13, noChange, t.Muted, font.Normal).Layout(gtx)
		}
		// Width of the line numbers: the largest one.
		maxNo := 0
		for _, h := range hunks {
			for _, l := range h {
				maxNo = max(maxNo, l.Old, l.New)
			}
		}
		numW := len(strconv.Itoa(maxNo))
		var rows []layout.FlexChild
		shown, total := 0, 0
		for hi, h := range hunks {
			total += len(h)
			if shown >= maxDiffRows {
				continue
			}
			if hi > 0 {
				rows = append(rows, layout.Rigid(func(gtx C) D {
					return layout.Inset{Top: 2, Bottom: 2}.Layout(gtx, t.Mono(12, strings.Repeat(" ", 2*numW+1)+"⋯", t.Muted).Layout)
				}))
			}
			for _, l := range h {
				if shown >= maxDiffRows {
					break
				}
				shown++
				rows = append(rows, layout.Rigid(t.diffRow(l, numW)))
			}
		}
		if total > shown {
			more := total - shown
			rows = append(rows, layout.Rigid(func(gtx C) D {
				return layout.Inset{Top: 4}.Layout(gtx, t.Label(12, "… +"+strconv.Itoa(more), t.Muted, font.Normal).Layout)
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
	}
}

func (t *Theme) diffRow(l DiffLine, numW int) layout.Widget {
	num := func(n int) string {
		if n == 0 {
			return strings.Repeat(" ", numW)
		}
		s := strconv.Itoa(n)
		return strings.Repeat(" ", numW-len(s)) + s
	}
	fg := t.CodeText
	var bg color.NRGBA
	switch l.Op {
	case '+':
		fg, bg = t.DiffAdd, t.DiffAdd
	case '-':
		fg, bg = t.DiffDel, t.DiffDel
	}
	bg.A = 0x24
	text := strings.ReplaceAll(l.Text, "\t", "    ")
	return func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx C) D {
				if l.Op != ' ' {
					paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(gtx.Constraints.Min.X, gtx.Constraints.Min.Y)}.Op())
				}
				return D{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx C) D {
				return layout.Flex{}.Layout(gtx,
					layout.Rigid(t.Mono(12, num(l.Old)+" "+num(l.New)+" ", t.Muted).Layout),
					layout.Rigid(t.Mono(12, string(l.Op)+" ", fg).Layout),
					layout.Flexed(1, func(gtx C) D { return OneLine(gtx, t.Mono(12, text, fg)) }),
				)
			}),
		)
	}
}
