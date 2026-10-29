package ui

import (
	"image"
	"testing"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/sitle/ori-agent/theme"
)

func TestBareURL(t *testing.T) {
	for word, want := range map[string]string{
		"https://wiki/onboarding":             "https://wiki/onboarding",
		"(https://tauturu.gov.pf/x?id=1)":     "https://tauturu.gov.pf/x?id=1",
		"https://fr.wikipedia.org/wiki/A_(b)": "https://fr.wikipedia.org/wiki/A_(b)",
		"https://tauturu.gov.pf/x?id=1).":     "https://tauturu.gov.pf/x?id=1",
		"http://a.pf,":                        "http://a.pf",
		"https://":                            "",
		"ftp://a.pf":                          "",
		"wiki":                                "",
	} {
		if got := BareURL(word); got != want {
			t.Errorf("BareURL(%q) = %q, want %q", word, got, want)
		}
	}
	if sp := ParseInline("voir [la page](https://wiki/p) ici"); len(sp) != 3 || sp[1].URL != "https://wiki/p" || !sp[1].Link {
		t.Errorf("spans %+v", sp)
	}
}

// A click on a link calls OpenLink with its address.
func TestLinkClick(t *testing.T) {
	th := NewTheme(theme.OriLight, theme.OriDark)
	var opened []string
	th.OpenLink = func(url string) { opened = append(opened, url) }
	var r input.Router
	frame := func() {
		gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Constraints{Max: image.Pt(600, 200)}, Source: r.Source()}
		th.RichText(15, th.Text, font.Normal, ParseInline("[Organisation](https://wiki/org) de la DSI"))(gtx)
		r.Frame(gtx.Ops)
	}
	frame()
	at := f32.Pt(10, 10) // inside the first word, the link
	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: at},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: at})
	frame()
	if len(opened) != 1 || opened[0] != "https://wiki/org" {
		t.Fatalf("opened %q", opened)
	}
}
