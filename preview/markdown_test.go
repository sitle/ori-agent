package preview

import "testing"

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
