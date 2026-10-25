package oriagent

import (
	"context"
	"image"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// testApp builds an App without a window, as Run would.
func testApp(cfg Config) *App {
	cfg.defaults()
	return newApp(cfg, nil)
}

func testModels() []Model { return []Model{{Name: "Fast"}, {Name: "Smart"}} }

func TestDefaultSettings(t *testing.T) {
	a := testApp(Config{Models: testModels(), WorkDir: t.TempDir(), Locale: "fr"})
	got := a.settings()
	want := DefaultSettings()
	want.WorkDir, want.Model, want.Locale = a.workDir, "Fast", "fr"
	if got != want {
		t.Errorf("settings = %+v, want %+v", got, want)
	}
}

func TestInitialSettings(t *testing.T) {
	dir := t.TempDir()
	want := Settings{WorkDir: dir, AllowCommands: false, AskEdits: true, Thinking: false,
		Notifications: true, Model: "Smart", Dark: false, Locale: "en"}
	s := want
	a := testApp(Config{Models: testModels(), Locale: "fr", Settings: &s})
	if got := a.settings(); got != want {
		t.Errorf("settings = %+v, want %+v", got, want)
	}
	if a.model().Name != "Smart" {
		t.Errorf("model = %q", a.model().Name)
	}
}

// Values that cannot be used keep the Config ones.
func TestInitialSettingsFallback(t *testing.T) {
	dir := t.TempDir()
	s := Settings{WorkDir: dir + "/missing", Model: "Gone", Locale: "xx", Dark: true}
	a := testApp(Config{Models: testModels(), WorkDir: dir, Locale: "en", Settings: &s})
	got := a.settings()
	if got.WorkDir != dir || got.Model != "Fast" || got.Locale != "en" {
		t.Errorf("settings = %+v", got)
	}
}

func TestOnSettings(t *testing.T) {
	var calls []Settings
	a := testApp(Config{Models: testModels(), WorkDir: t.TempDir(), Locale: "fr",
		OnSettings: func(s Settings) { calls = append(calls, s) }})
	c := &Ctl{a}
	a.checkSettings()
	if len(calls) != 0 {
		t.Fatalf("called without a change: %+v", calls)
	}

	c.SetModel(1)
	c.SetDark(false)
	if err := c.SetLocale("en"); err != nil {
		t.Fatal(err)
	}
	a.checkSettings()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	if s := calls[0]; s.Model != "Smart" || s.Dark || s.Locale != "en" || s != c.Settings() {
		t.Errorf("settings = %+v", s)
	}

	// A setup toggle, seen at the end of a real frame.
	a.setup.thinking.Value = false
	gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(1280, 820)), Now: time.Now()}
	a.frame(gtx)
	if len(calls) != 2 || calls[1].Thinking {
		t.Fatalf("after frame: %+v", calls)
	}
	a.frame(gtx)
	if len(calls) != 2 {
		t.Errorf("called again without a change: %d calls", len(calls))
	}
}

func TestTurnSettings(t *testing.T) {
	s := DefaultSettings()
	s.AllowCommands, s.AskEdits = false, true
	a := testApp(Config{Models: testModels(), WorkDir: t.TempDir(), Settings: &s})
	got := make(chan *Turn, 1)
	a.startTurn(a.cur, "", nil, func(ctx context.Context, t *Turn) error {
		got <- t
		<-ctx.Done() // never canceled: the end of a turn needs a window
		return nil
	})
	turn := <-got
	if turn.AllowCommands || !turn.AskEdits || !turn.Thinking {
		t.Errorf("turn = %+v", turn)
	}
}
