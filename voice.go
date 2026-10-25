package oriagent

import (
	"fmt"
	"image/color"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/ui"
)

// Dictation turns speech into text for the composer. Start begins capture;
// Stop ends it and returns the transcript (it may block: it runs on its own
// goroutine). Plug a real speech-to-text engine here.
type Dictation interface {
	Start() error
	Stop() (string, error)
}

// Leveler is optionally implemented by a Dictation to drive the level meter
// with the real input level, in [0,1]. Without it the meter is simulated.
type Leveler interface {
	Level() float32
}

// SimDictation captures nothing and returns canned phrases in turn, in the
// interface language (Phrases maps a locale to its phrases).
type SimDictation struct {
	Phrases map[string][]string
	locale  string
	n       int
}

var simPhrases = map[string][]string{
	"fr": {
		"Explique-moi la structure du projet.",
		"Ajoute des tests pour le formulaire de demande en ligne.",
		"Peux-tu résumer les changements de la session en cours ?",
	},
	"en": {
		"Explain the structure of this project.",
		"Add tests for the online request form.",
		"Can you summarize the changes in this session?",
	},
}

func (d *SimDictation) SetLocale(locale string) { d.locale = locale }

func (d *SimDictation) Start() error { return nil }

func (d *SimDictation) Stop() (string, error) {
	time.Sleep(900 * time.Millisecond) // "transcribing"
	src := d.Phrases
	if src == nil {
		src = simPhrases
	}
	ps := src[d.locale]
	if len(ps) == 0 {
		ps = src["en"]
	}
	if len(ps) == 0 {
		return "", nil
	}
	p := ps[d.n%len(ps)]
	d.n++
	return p, nil
}

type voiceState struct {
	listening, transcribing bool
	start                   time.Time
	btn                     widget.Clickable
	levels                  []float32 // sampled from a Leveler
	lastSample              time.Time
}

func (v *voiceState) active() bool { return v.listening || v.transcribing }

func (a *App) handleVoice(gtx C) {
	if a.voice.btn.Clicked(gtx) {
		a.toggleVoice()
	}
}

func (a *App) toggleVoice() {
	v := &a.voice
	d := a.cfg.Dictation
	switch {
	case a.cfg.Disable.Voice || v.transcribing:
		return
	case v.listening:
		v.listening, v.transcribing = false, true
		go func() {
			text, err := d.Stop()
			a.update(func() { a.voice.transcribing = false })
			if err != nil {
				a.update(func() { a.notify(a.t("voice.error", err.Error())) })
				return
			}
			a.typeIntoComposer(text)
		}()
	default:
		if err := d.Start(); err != nil {
			a.notify(a.t("voice.unavailable", err.Error()))
			return
		}
		v.listening, v.start, v.levels = true, time.Now(), nil
		a.view = viewChat
	}
}

// typeIntoComposer appends text to the composer letter by letter.
func (a *App) typeIntoComposer(text string) {
	prefix := ""
	a.update(func() {
		if t := a.editor.Text(); t != "" {
			prefix = t + " "
		}
	})
	r := []rune(text)
	for i := range r {
		a.update(func() {
			t := prefix + string(r[:i+1])
			a.editor.SetText(t)
			n := len([]rune(t))
			a.editor.SetCaret(n, n)
		})
		time.Sleep(18 * time.Millisecond)
	}
	a.update(func() { a.focus = true })
}

func elapsed(d time.Duration) string {
	return fmt.Sprintf("%d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}

// voiceMeter replaces the working directory in the composer while active.
func (a *App) voiceMeter(gtx C) D {
	th, v := a.th, &a.voice
	gtx.Execute(op.InvalidateCmd{})
	if v.transcribing {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D { return ui.Dot(gtx, ui.Pulse(gtx, th.Accent), 8) }),
			ui.HSpace(8),
			layout.Rigid(th.Label(13, a.t("voice.transcribing"), th.Muted, font.Medium).Layout),
		)
	}
	level := ui.FakeLevel
	if l, ok := a.cfg.Dictation.(Leveler); ok {
		// Sample the real level into a ring buffer; bar i shows sample i.
		if gtx.Now.Sub(v.lastSample) >= 45*time.Millisecond {
			v.levels, v.lastSample = append(v.levels, l.Level()), gtx.Now
			if len(v.levels) > 400 {
				v.levels = v.levels[len(v.levels)-400:]
			}
		}
		now := float64(gtx.Now.UnixNano()) / 1e9
		level = func(t float64) float32 {
			i := len(v.levels) - 1 - int((now-t)/0.045+0.5)
			if i < 0 || i >= len(v.levels) {
				return 0
			}
			return v.levels[i]
		}
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx C) D { return ui.Dot(gtx, ui.Pulse(gtx, th.DiffDel), 8) }),
		ui.HSpace(8),
		layout.Rigid(th.Label(13, a.t("voice.listening"), th.Text, font.Medium).Layout),
		ui.HSpace(8),
		layout.Rigid(th.Mono(12, elapsed(time.Since(v.start)), th.Muted).Layout),
		ui.HSpace(12),
		layout.Flexed(1, func(gtx C) D { return ui.Waveform(gtx, th.Accent, 22, level) }),
	)
}

// micButton toggles dictation; it turns red with a stop square while listening.
func (a *App) micButton(gtx C) D {
	th, v := a.th, &a.voice
	bg, fg := th.Surface, th.Muted
	icon := ui.Mic
	if v.listening {
		bg, fg, icon = th.DiffDel, th.OnStrong, ui.Stop
	}
	return th.RoundButton(gtx, &v.btn, bg, fg, func(gtx C, c color.NRGBA) D { return icon(gtx, c) })
}

// voicePill is the title bar indicator shown while the microphone is on.
func (a *App) voicePill(gtx C) D {
	th := a.th
	if !a.voice.listening {
		return D{}
	}
	return layout.Inset{Right: 6}.Layout(gtx, func(gtx C) D {
		return ui.Bordered(gtx, th.Bg, th.DiffDel, 10, func(gtx C) D {
			return layout.Inset{Top: 3, Bottom: 3, Left: 8, Right: 9}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D { return ui.Dot(gtx, ui.Pulse(gtx, th.DiffDel), 7) }),
					ui.HSpace(6),
					layout.Rigid(th.Label(12, a.t("voice.pill", elapsed(time.Since(a.voice.start))), th.DiffDel, font.Medium).Layout),
				)
			})
		})
	})
}
