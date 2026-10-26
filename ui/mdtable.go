package ui

import (
	"image"
	"regexp"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// Align is the alignment of a table column.
type Align int

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

var mdSeparator = regexp.MustCompile(`^:?-{1,}:?$`)

// SplitRow splits a Markdown table row into its cells; \| stays in a cell.
func SplitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	if strings.HasSuffix(line, "|") && !strings.HasSuffix(line, `\|`) {
		line = line[:len(line)-1]
	}
	var cells []string
	var cur strings.Builder
	for i := 0; i < len(line); i++ {
		switch {
		case line[i] == '\\' && i+1 < len(line) && line[i+1] == '|':
			cur.WriteByte('|')
			i++
		case line[i] == '|':
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(line[i])
		}
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

// ParseTable turns the lines of a Markdown table into its rows (the
// header first) and the column alignments of its separator row. Without a
// separator row, every row is a body row.
func ParseTable(lines []string) (rows [][]string, aligns []Align, header bool) {
	for i, l := range lines {
		cells := SplitRow(l)
		if i == 1 && isSeparator(cells) {
			header = true
			for _, c := range cells {
				switch c = strings.TrimSpace(c); {
				case strings.HasPrefix(c, ":") && strings.HasSuffix(c, ":"):
					aligns = append(aligns, AlignCenter)
				case strings.HasSuffix(c, ":"):
					aligns = append(aligns, AlignRight)
				default:
					aligns = append(aligns, AlignLeft)
				}
			}
			continue
		}
		rows = append(rows, cells)
	}
	return rows, aligns, header
}

func isSeparator(cells []string) bool {
	for _, c := range cells {
		if !mdSeparator.MatchString(strings.TrimSpace(c)) {
			return false
		}
	}
	return len(cells) > 0
}

// mdTable draws a Markdown table: header row on a tinted background, a
// line between rows, columns as wide as their content when it fits, else
// shared out with the text wrapped in its cell.
func (t *Theme) mdTable(gtx C, size unit.Sp, lines []string) D {
	rows, aligns, header := ParseTable(lines)
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r))
	}
	if cols == 0 {
		return D{}
	}
	padX, padY := gtx.Dp(10), gtx.Dp(6)
	line := max(1, gtx.Dp(1))
	weight := func(r int) font.Weight {
		if header && r == 0 {
			return font.SemiBold
		}
		return font.Normal
	}
	cell := func(r, c int) string {
		if c < len(rows[r]) {
			return rows[r][c]
		}
		return ""
	}

	measure := func(r int, s string) int {
		mgtx := gtx
		mgtx.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
		macro := op.Record(gtx.Ops)
		d := t.RichText(size, t.Text, weight(r), ParseInline(s))(mgtx)
		macro.Stop()
		return d.Size.X
	}
	// Each column: its widest cell on one line (natural) and its longest
	// word (least, below which words would be cut).
	natural, least := make([]int, cols), make([]int, cols)
	for r := range rows {
		for c := 0; c < cols; c++ {
			s := cell(r, c)
			natural[c] = max(natural[c], measure(r, s))
			long := ""
			for _, w := range strings.Fields(PlainInline(s)) {
				if len([]rune(w)) > len([]rune(long)) {
					long = w
				}
			}
			if strings.Contains(s, "**") || strings.Contains(s, "__") {
				long = "**" + long + "**" // measured in bold, as drawn
			}
			least[c] = max(least[c], measure(r, long)+1)
		}
	}
	widths := fitColumns(natural, least, gtx.Constraints.Max.X-2*line-cols*2*padX)
	tableW := 2 * line
	for _, w := range widths {
		tableW += w + 2*padX
	}

	// Lay the rows out, then draw backgrounds, lines and cells.
	y := line
	type placed struct {
		call op.CallOp
		x, y int
	}
	var cells []placed
	var lines_ []int // y of the line under each row
	var headerH int
	for r := range rows {
		h := 0
		var rowCells []placed
		x := line
		for c := 0; c < cols; c++ {
			cgtx := gtx
			cgtx.Constraints = layout.Constraints{Min: image.Pt(widths[c], 0), Max: image.Pt(widths[c], 1<<20)}
			macro := op.Record(gtx.Ops)
			w := t.RichText(size, t.Text, weight(r), ParseInline(cell(r, c)))
			var d D
			align := AlignLeft
			if c < len(aligns) {
				align = aligns[c]
			}
			switch align {
			case AlignRight:
				d = layout.E.Layout(cgtx, w)
			case AlignCenter:
				// Center (unlike N) drops the cell's minimum width, so the
				// text keeps its own width and can be centered in the cell.
				d = layout.Center.Layout(cgtx, w)
			default:
				d = w(cgtx)
			}
			call := macro.Stop()
			rowCells = append(rowCells, placed{call, x + padX, y + padY})
			h = max(h, d.Size.Y)
			x += widths[c] + 2*padX
		}
		cells = append(cells, rowCells...)
		y += h + 2*padY
		if r == 0 && header {
			headerH = y
		}
		lines_ = append(lines_, y)
		y += line
	}
	height := y
	size2 := image.Pt(tableW, height)

	rr := clip.UniformRRect(image.Rectangle{Max: size2}, gtx.Dp(6))
	paint.FillShape(gtx.Ops, t.BorderHi, rr.Op(gtx.Ops))
	inner := clip.UniformRRect(image.Rect(line, line, tableW-line, height-line), gtx.Dp(5))
	stack := inner.Push(gtx.Ops)
	paint.Fill(gtx.Ops, t.Bg)
	if headerH > 0 {
		paint.FillShape(gtx.Ops, t.Code, clip.Rect{Max: image.Pt(tableW, headerH)}.Op())
	}
	for i, ly := range lines_ {
		if i == len(lines_)-1 {
			break
		}
		paint.FillShape(gtx.Ops, t.Border, clip.Rect{Min: image.Pt(0, ly), Max: image.Pt(tableW, ly+line)}.Op())
	}
	stack.Pop()
	for _, p := range cells {
		st := op.Offset(image.Pt(p.x, p.y)).Push(gtx.Ops)
		p.call.Add(gtx.Ops)
		st.Pop()
	}
	return D{Size: size2}
}

// fitColumns gives each column its natural width when the table fits in
// avail; otherwise each keeps at least its longest word and the rest of
// the room goes to the columns that need more, in proportion. When even
// the words do not fit, the widths shrink in proportion.
func fitColumns(natural, least []int, avail int) []int {
	widths := make([]int, len(natural))
	total, minTotal := 0, 0
	for c := range natural {
		least[c] = min(least[c], natural[c])
		total += natural[c]
		minTotal += least[c]
	}
	switch {
	case total <= avail:
		copy(widths, natural)
	case minTotal >= avail:
		for c := range widths {
			widths[c] = max(1, avail*least[c]/max(minTotal, 1))
		}
	default:
		extra, want := avail-minTotal, total-minTotal
		for c := range widths {
			widths[c] = least[c] + extra*(natural[c]-least[c])/max(want, 1)
		}
	}
	return widths
}
