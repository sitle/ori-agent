package ui

import (
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
