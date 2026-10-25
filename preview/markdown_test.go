package preview

import "testing"

func TestParseMarkdown(t *testing.T) {
	src := "# Title\n\nSome **bold** text\ncontinued [link](http://x).\n\n- item one\n  continued\n* item two\n\n```go\nfmt.Println()\n```\n| a | b |\n| 1 | 2 |\n---\n## Sub\n### Subsub"
	got := parseMarkdown(src)
	want := []struct {
		kind blockKind
		text string
	}{
		{bH1, "Title"},
		{bBlank, ""},
		{bPara, "Some bold text continued link."},
		{bBlank, ""},
		{bBullet, "item one continued"},
		{bBullet, "item two"},
		{bBlank, ""},
		{bCode, ""},
		{bTable, ""},
		{bRule, ""},
		{bH2, "Sub"},
		{bH3, "Subsub"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d blocks, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].kind != w.kind || got[i].text != w.text {
			t.Errorf("block %d = {%v %q}, want {%v %q}", i, got[i].kind, got[i].text, w.kind, w.text)
		}
	}
	if len(got[7].lines) != 1 || got[7].lines[0] != "fmt.Println()" {
		t.Errorf("code block lines = %q", got[7].lines)
	}
	if len(got[8].lines) != 2 {
		t.Errorf("table lines = %q", got[8].lines)
	}
}

func TestStripLineNumbers(t *testing.T) {
	in := "   1  package main\n  12  func main() {}\nno number"
	want := "package main\nfunc main() {}\nno number"
	if got := StripLineNumbers(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIsImage(t *testing.T) {
	for ext, want := range map[string]bool{"png": true, "jpeg": true, "gif": true, "pdf": false, "": false} {
		if IsImage(ext) != want {
			t.Errorf("IsImage(%q) != %v", ext, want)
		}
	}
}
