package ui

import (
	"image"
	"strings"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/sitle/ori-agent/theme"
)

// A cut line is kept: the next frames draw it without measuring again,
// and at the same size.
func TestOneLineKeepsTheCut(t *testing.T) {
	th := NewTheme(theme.OriLight, theme.OriDark)
	long := strings.Repeat("une ligne trop longue pour la largeur ", 20)
	gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(300, 100)}}
	first := OneLine(gtx, th.Label(13, long, th.Text, 0))
	key := oneLineKey{long, 300, 13, th.Label(13, long, th.Text, 0).Font, 1, th.Label(13, "", th.Text, 0).Shaper}
	oneLines.Lock()
	cut, ok := oneLines.m[key]
	oneLines.Unlock()
	if !ok || !strings.HasSuffix(cut, "…") || len(cut) >= len(long) {
		t.Fatalf("cut not kept: %q, %v", cut, ok)
	}
	if again := OneLine(gtx, th.Label(13, long, th.Text, 0)); again.Size != first.Size || first.Size.X > 300 {
		t.Fatalf("sizes %v then %v", first.Size, again.Size)
	}
	short := OneLine(gtx, th.Label(13, "court", th.Text, 0))
	if short.Size.X >= first.Size.X {
		t.Fatalf("short text %v as wide as the cut one %v", short.Size, first.Size)
	}
}
