package ui

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParseMarkdown(t *testing.T) {
	src := "# Title\n\nSome **bold** text\ncontinued [link](http://x).\n\n- item one\n  continued\n* item two\n  - nested\n1. first\n2) second\n> quoted\n> more\n\n```go\nfmt.Println()\n```\n| a | b |\n| 1 | 2 |\n---\n## Sub\n### Subsub"
	got := ParseMarkdown(src)
	want := []struct {
		kind MDKind
		text string
	}{
		{MDH1, "Title"},
		{MDBlank, ""},
		{MDPara, "Some bold text continued link."},
		{MDBlank, ""},
		{MDBullet, "item one continued"},
		{MDBullet, "item two"},
		{MDBullet, "nested"},
		{MDNumber, "first"},
		{MDNumber, "second"},
		{MDQuote, "quoted more"},
		{MDBlank, ""},
		{MDCode, ""},
		{MDTable, ""},
		{MDRule, ""},
		{MDH2, "Sub"},
		{MDH3, "Subsub"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d blocks, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Kind != w.kind || PlainInline(got[i].Text) != w.text {
			t.Errorf("block %d = {%v %q}, want {%v %q}", i, got[i].Kind, got[i].Text, w.kind, w.text)
		}
	}
	if got[6].Level != 1 || got[8].Marker != "2." {
		t.Errorf("nesting %d, marker %q", got[6].Level, got[8].Marker)
	}
	if len(got[11].Lines) != 1 || got[11].Lines[0] != "fmt.Println()" {
		t.Errorf("code block lines = %q", got[11].Lines)
	}
	if len(got[12].Lines) != 2 {
		t.Errorf("table lines = %q", got[12].Lines)
	}
}

func TestParseInline(t *testing.T) {
	cases := map[string][]Span{
		"**Je peux** : lire": {{Text: "Je peux", Bold: true}, {Text: " : lire"}},
		"a *b* `c` [d](https://x.pf)": {
			{Text: "a "}, {Text: "b", Italic: true}, {Text: " "}, {Text: "c", Code: true}, {Text: " "}, {Text: "d", Link: true},
		},
		// Still streaming: the closing marker has not arrived yet.
		"**gras pas fini":  {{Text: "**gras pas fini"}},
		"2 * 3 = 6":        {{Text: "2 * 3 = 6"}},
		"snake_case_name":  {{Text: "snake_case_name"}},
		`\*pas italique\*`: {{Text: "*pas italique*"}},
	}
	for in, want := range cases {
		if got := ParseInline(in); !reflect.DeepEqual(got, want) {
			t.Errorf("ParseInline(%q) = %+v, want %+v", in, got, want)
		}
	}
}

func TestParseTable(t *testing.T) {
	rows, aligns, header := ParseTable([]string{
		"| Phase | Objet | Coût |",
		"|:---|:---:|---:|",
		`| T3 | Réseau \| fibre | 12 |`,
		"| T5 | Vérifications |",
	})
	if !header || !reflect.DeepEqual(aligns, []Align{AlignLeft, AlignCenter, AlignRight}) {
		t.Fatalf("header %v, aligns %v", header, aligns)
	}
	want := [][]string{{"Phase", "Objet", "Coût"}, {"T3", "Réseau | fibre", "12"}, {"T5", "Vérifications"}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %q", rows)
	}
	// Without a separator row, every row is a body row.
	if rows, _, header := ParseTable([]string{"| a | b |", "| c | d |"}); header || len(rows) != 2 {
		t.Errorf("no header: %v %q", header, rows)
	}
}

func TestFitColumns(t *testing.T) {
	for _, c := range []struct {
		natural, least []int
		avail          int
		want           []int
	}{
		{[]int{100, 200}, []int{50, 80}, 400, []int{100, 200}},  // fits
		{[]int{100, 500}, []int{60, 100}, 300, []int{72, 227}},  // words kept, rest shared
		{[]int{100, 500}, []int{100, 200}, 150, []int{50, 100}}, // even the words are too wide
	} {
		if got := fitColumns(c.natural, c.least, c.avail); !reflect.DeepEqual(got, c.want) {
			t.Errorf("fitColumns(%v, %v, %d) = %v, want %v", c.natural, c.least, c.avail, got, c.want)
		}
	}
}

func TestDiffLines(t *testing.T) {
	old := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\n"
	new := "a\nb\nC\nd\ne\nf\ng\nh\ni\nj\nk\n"
	lines := DiffLines(old, new)
	if a, r := DiffStat(lines); a != 2 || r != 1 {
		t.Fatalf("stat +%d -%d: %+v", a, r, lines)
	}
	var changed []string
	for _, l := range lines {
		if l.Op != ' ' {
			changed = append(changed, fmt.Sprintf("%c%d/%d %s", l.Op, l.Old, l.New, l.Text))
		}
	}
	if want := []string{"-3/0 c", "+0/3 C", "+0/11 k"}; !reflect.DeepEqual(changed, want) {
		t.Errorf("changes %q, want %q", changed, want)
	}
	// Two changes far apart: two hunks, each with its context.
	h := Hunks(lines, 3)
	if len(h) != 2 || h[0][0].Text != "a" || h[1][len(h[1])-1].Text != "k" {
		t.Errorf("hunks: %+v", h)
	}
	if Hunks(DiffLines("x\n", "x\n"), 3) != nil {
		t.Error("no change, yet hunks")
	}
	// A new file: everything added.
	if a, r := DiffStat(DiffLines("", "un\ndeux")); a != 2 || r != 0 {
		t.Errorf("new file +%d -%d", a, r)
	}
}
