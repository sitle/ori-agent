package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/sitle/ori-agent/theme"
)

// BenchmarkDiff measures a frame of a diff of about 250 changed lines,
// each cut to the width of the transcript.
func BenchmarkDiff(b *testing.B) {
	var o, n []string
	for i := 0; i < 129; i++ {
		o = append(o, fmt.Sprintf("- Ligne %d de l'ancienne note de sécurité, avec un texte assez long pour remplir la largeur du fil", i))
	}
	for i := 0; i < 117; i++ {
		n = append(n, fmt.Sprintf("+ Ligne %d de la nouvelle note, en Markdown, avec un texte assez long lui aussi pour la largeur", i))
	}
	th := NewTheme(theme.OriLight, theme.OriDark)
	w := th.Diff(strings.Join(o, "\n"), strings.Join(n, "\n"), "aucun")
	ops := new(op.Ops)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ops.Reset()
		gtx := layout.Context{Ops: ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(760, 820))}
		w(gtx)
	}
}
