package preview

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A file the pane does not read goes to the converter, in Load.
func TestConverter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.docx")
	os.WriteFile(path, []byte{0x50, 0x4b, 0x03, 0x04, 0xff, 0xfe}, 0o644)
	wait := func(d *Doc) {
		done := make(chan struct{})
		d.Load(func() { close(done) })
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Load never finished")
		}
	}

	d := OpenWith(path, "", func(string) (*Converted, error) {
		return &Converted{Markdown: "# Note\n\nTexte du document.", Label: "Word"}, nil
	})
	if d.Kind != PDF || !d.loading || d.kindLabel() != "DOCX" {
		t.Fatalf("not waiting for the conversion: %v %q", d.Kind, d.kindLabel())
	}
	wait(d)
	if d.Kind != Markdown || len(d.blocks) == 0 || d.kindLabel() != "Word" {
		t.Fatalf("markdown: kind %v, %d blocks, label %q", d.Kind, len(d.blocks), d.kindLabel())
	}

	d = OpenWith(path, "", func(string) (*Converted, error) { return nil, nil })
	wait(d)
	if d.Kind != Code || len(d.lines) != 1 || d.lines[0] != "(binary file)" {
		t.Fatalf("not handled: %v %v", d.Kind, d.lines)
	}

	d = OpenWith(path, "", func(string) (*Converted, error) { return nil, errors.New("Office absent") })
	wait(d)
	if d.err != "Office absent" || d.loading {
		t.Fatalf("error: %q %v", d.err, d.loading)
	}

	// Without a converter, as before.
	if d := Open(path, ""); d.Kind != Code || d.lines[0] != "(binary file)" {
		t.Fatalf("without converter: %v", d.Kind)
	}
	// Text is never converted.
	txt := filepath.Join(t.TempDir(), "a.md")
	os.WriteFile(txt, []byte("# Titre"), 0o644)
	if d := OpenWith(txt, "", func(string) (*Converted, error) { t.Fatal("converted a text file"); return nil, nil }); d.Kind != Markdown {
		t.Fatalf("text: %v", d.Kind)
	}
	// Nor is a long text whose 256 KB cut falls inside an accented letter.
	long := filepath.Join(t.TempDir(), "long.txt")
	os.WriteFile(long, []byte(strings.Repeat("a", 256<<10-1)+"é et la suite"), 0o644)
	if d := OpenWith(long, "", func(string) (*Converted, error) { t.Fatal("converted a long text"); return nil, nil }); d.Kind != Code || d.lines[0] == "(binary file)" {
		t.Fatalf("long text: %v", d.Kind)
	}
}
