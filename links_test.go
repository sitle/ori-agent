package oriagent

import (
	"context"
	"image"
	"testing"
	"time"

	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// A link opens only once confirmed, and only a web or mail address.
func TestLinkConfirmed(t *testing.T) {
	opened := make(chan string, 4)
	a, _ := sessionApp(t, Config{Agent: AgentFunc(func(context.Context, *Turn) error { return nil }),
		OnLink: func(url string) error { opened <- url; return nil }})
	var r input.Router
	frame := func() {
		gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Constraints{Max: image.Pt(800, 600)}, Source: r.Source()}
		a.mu.Lock()
		a.handleLink(gtx)
		a.layoutLink(gtx)
		a.mu.Unlock()
		r.Frame(gtx.Ops)
	}
	for _, bad := range []string{"javascript:alert(1)", "file:///C:/Windows/system32/cmd.exe", "ms-settings:", "https://"} {
		if a.openLink(bad); a.link.url != "" {
			t.Errorf("%s asks to be opened", bad)
		}
	}
	if a.openLink("mailto:hina@example.pf?subject=Note"); a.link.host != "hina@example.pf" || !a.link.mail {
		t.Errorf("mail dialog %+v", a.link)
	}
	a.openLink("https://www.tauturu.gov.pf/front/knowbaseitem.form.php?id=185")
	if a.link.host != "www.tauturu.gov.pf" {
		t.Fatalf("dialog %+v", a.link)
	}
	frame()
	a.link.cancel.Click()
	frame()
	if a.link.url != "" || len(opened) != 0 {
		t.Fatal("cancelled link still asked or opened")
	}
	a.openLink("https://wiki/org")
	frame()
	a.link.ok.Click()
	frame()
	select {
	case got := <-opened:
		if got != "https://wiki/org" || a.link.url != "" {
			t.Errorf("opened %q, dialog %q", got, a.link.url)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("confirmed link not opened")
	}
}
