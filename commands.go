package oriagent

import (
	"fmt"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"

	"github.com/sitle/ori-agent/ui"
)

// Command is a slash command. Desc and Args may be translation keys. Args is
// a usage hint shown in the palette: "<x>" means required (picking the
// command only completes it), "[x]" optional. Run returns the text logged
// in the transcript ("" logs nothing); with Markdown, that text is drawn
// as Markdown (titles, lists, bold, links), else as fixed-width lines,
// which keep their alignment.
type Command struct {
	Name, Args, Desc string
	Run              func(c *Ctl, args string) string
	Markdown         bool
}

func builtinCommands(cfg Config) []*Command {
	cmds := []*Command{
		{Name: "help", Args: "", Desc: "cmd.help.desc", Run: func(c *Ctl, _ string) string {
			var b strings.Builder
			for _, x := range c.Commands() {
				fmt.Fprintf(&b, "%-10s %s\n", "/"+x.Name, c.T(x.Desc))
			}
			return strings.TrimRight(b.String(), "\n")
		}},
		{Name: "new", Args: "", Desc: "cmd.new.desc", Run: func(c *Ctl, _ string) string { c.NewSession(); return "" }},
		{Name: "clear", Args: "", Desc: "cmd.clear.desc", Run: func(c *Ctl, _ string) string { c.Clear(); return c.T("cmd.clear.done") }},
		{Name: "compact", Args: "cmd.compact.args", Desc: "cmd.compact.desc", Run: cmdCompact},
		{Name: "model", Args: "cmd.model.args", Desc: "cmd.model.desc", Run: cmdModel},
		{Name: "open", Args: "cmd.open.args", Desc: "cmd.open.desc", Run: func(c *Ctl, args string) string {
			if args == "" {
				return c.T("cmd.open.usage")
			}
			if err := c.Open(args); err != nil {
				return err.Error()
			}
			return c.T("cmd.open.done", args)
		}},
		{Name: "usage", Args: "", Desc: "cmd.usage.desc", Run: cmdUsage},
		{Name: "status", Args: "", Desc: "cmd.status.desc", Run: cmdStatus},
		{Name: "lang", Args: "cmd.lang.args", Desc: "cmd.lang.desc", Run: cmdLang},
		{Name: "sidebar", Args: "", Desc: "cmd.sidebar.desc", Run: func(c *Ctl, _ string) string { c.ToggleSidebar(); return "" }},
	}
	if !cfg.Disable.Attachments {
		cmds = append(cmds, &Command{Name: "upload", Args: "", Desc: "cmd.upload.desc", Run: func(c *Ctl, _ string) string { c.PickFiles(); return "" }})
	}
	if !cfg.Disable.Voice {
		cmds = append(cmds, &Command{Name: "voice", Args: "", Desc: "cmd.voice.desc", Run: func(c *Ctl, _ string) string { c.ToggleVoice(); return "" }})
	}
	if !cfg.Disable.Pomodoro {
		cmds = append(cmds, &Command{Name: "pomodoro", Args: "[start|pause|reset|skip]", Desc: "cmd.pomodoro.desc", Run: cmdPomodoro})
	}
	if !cfg.Disable.ThemeToggle {
		cmds = append(cmds, &Command{Name: "theme", Args: "", Desc: "cmd.theme.desc", Run: func(c *Ctl, _ string) string {
			c.SetDark(!c.Dark())
			if c.Dark() {
				return c.T("cmd.theme.dark")
			}
			return c.T("cmd.theme.light")
		}})
	}
	if !cfg.Disable.Setup {
		cmds = append(cmds, &Command{Name: "config", Args: "", Desc: "cmd.config.desc", Run: func(c *Ctl, _ string) string { c.Show(viewSetup); return "" }})
	}
	if !cfg.Disable.Profile {
		cmds = append(cmds, &Command{Name: "profile", Args: "", Desc: "cmd.profile.desc", Run: func(c *Ctl, _ string) string { c.Show(viewProfile); return "" }})
	}
	return cmds
}

func cmdCompact(c *Ctl, args string) string {
	s := c.Session()
	n := len(s.Messages)
	if n == 0 {
		return c.T("cmd.compact.empty")
	}
	out := c.T("cmd.compact.done", n)
	if args != "" {
		out += c.T("cmd.compact.with", args)
	}
	s.Messages = nil
	return out
}

func cmdModel(c *Ctl, args string) string {
	ms := c.Models()
	if args != "" {
		for i, m := range ms {
			if strings.HasPrefix(strings.ToLower(m.Name), strings.ToLower(args)) {
				c.SetModel(i)
				return c.T("cmd.model.set", m.Name)
			}
		}
		return c.T("cmd.model.unknown", args)
	}
	var b strings.Builder
	for i, m := range ms {
		mark := "  "
		if i == c.Model() {
			mark = "› "
		}
		fmt.Fprintf(&b, "%s%-11s %s\n", mark, m.Name, c.T(m.Desc))
	}
	return strings.TrimRight(b.String(), "\n")
}

func cmdUsage(c *Ctl, _ string) string {
	s := c.Session()
	chars := 0
	for _, m := range s.Messages {
		chars += len(m.Text) + len(m.Output)
	}
	tokens := chars / 4
	return c.T("cmd.usage.out", s.Title, len(s.Messages), tokens, float64(tokens)/2000)
}

func cmdStatus(c *Ctl, _ string) string {
	busy := 0
	for _, x := range c.Sessions() {
		if x.Busy {
			busy++
		}
	}
	theme := c.T("theme.name.light")
	if c.Dark() {
		theme = c.T("theme.name.dark")
	}
	return c.T("cmd.status.out", c.Models()[c.Model()].Name, c.WorkDir(), len(c.Projects()), len(c.Sessions()), busy, theme, c.LanguageName(c.Locale()))
}

func cmdLang(c *Ctl, args string) string {
	var list []string
	for _, l := range c.Locales() {
		list = append(list, l+" ("+c.LanguageName(l)+")")
	}
	if args == "" {
		return strings.Join(list, "\n")
	}
	if err := c.SetLocale(args); err != nil {
		return c.T("cmd.lang.unknown", args, strings.Join(c.Locales(), ", "))
	}
	return c.T("cmd.lang.set", c.LanguageName(c.Locale()))
}

func cmdPomodoro(c *Ctl, args string) string {
	p := c.Pomodoro()
	now := time.Now()
	switch args {
	case "start", "":
		p.Start(now)
	case "pause":
		p.Pause(now)
	case "reset":
		p.Reset()
	case "skip":
		return p.Next() + "."
	default:
		return c.T("pomo.usage")
	}
	state := c.T("pomo.paused")
	if p.Running() {
		state = c.T("pomo.running")
	}
	return c.T("pomo.state", p.Mode(), state, clock(p.Left(now)))
}

// ---- execution ----

func (a *App) findCommand(name string) *Command {
	for _, c := range a.commands {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// matchCommands returns prefix matches first, then other substring matches.
func (a *App) matchCommands(filter string) []*Command {
	var exact, pre, sub []*Command
	for _, c := range a.commands {
		switch {
		case c.Name == filter:
			// First: Enter on a full name (typed or recalled) runs it.
			exact = append(exact, c)
		case strings.HasPrefix(c.Name, filter):
			pre = append(pre, c)
		case strings.Contains(c.Name, filter) || strings.Contains(strings.ToLower(a.t(c.Desc)), filter):
			sub = append(sub, c)
		}
	}
	return append(append(exact, pre...), sub...)
}

func (a *App) runCommand(line string) {
	name, args, _ := strings.Cut(strings.TrimPrefix(strings.TrimSpace(line), "/"), " ")
	args = strings.TrimSpace(args)
	shown := "/" + name
	if args != "" {
		shown += " " + args
	}
	a.remember(shown)
	c := a.findCommand(name)
	if c == nil {
		a.cur.add(&Message{Kind: KindCommand, Text: shown, Output: a.t("cmd.unknown")})
		return
	}
	if out := c.Run(&Ctl{a}, args); out != "" {
		a.cur.add(&Message{Kind: KindCommand, Text: shown, Output: out, Markdown: c.Markdown})
	}
}

// ---- palette ----

// paletteFilter reports whether the composer holds a command name being
// typed ("/mo"), and returns it. Once a space is typed, arguments follow.
func (a *App) paletteFilter() (string, bool) {
	t := a.editor.Text()
	if !strings.HasPrefix(t, "/") || strings.ContainsAny(t, " \n") {
		return "", false
	}
	return strings.ToLower(t[1:]), true
}

// handlePalette runs before the editor sees the frame's key events, so the
// arrows, Tab and Escape drive the palette while it is open.
func (a *App) handlePalette(gtx C) {
	filter, open := a.paletteFilter()
	if !open {
		a.cmdSel = 0
		return
	}
	matches := a.matchCommands(filter)
	for {
		ev, ok := gtx.Event(
			key.Filter{Focus: &a.editor, Name: key.NameUpArrow},
			key.Filter{Focus: &a.editor, Name: key.NameDownArrow},
			key.Filter{Focus: &a.editor, Name: key.NameTab},
			key.Filter{Focus: &a.editor, Name: key.NameEscape},
		)
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		switch e.Name {
		case key.NameUpArrow:
			a.cmdSel--
		case key.NameDownArrow:
			a.cmdSel++
		case key.NameTab:
			if len(matches) > 0 {
				a.complete(matches[wrap(a.cmdSel, len(matches))])
			}
		case key.NameEscape:
			a.editor.SetText("")
		}
	}
	for i, c := range a.commands {
		if a.cmdBtns[i].Clicked(gtx) {
			a.pick(c)
			a.focus = true
		}
	}
	a.cmdSel = wrap(a.cmdSel, len(matches))
}

func wrap(i, n int) int {
	if n == 0 {
		return 0
	}
	return (i%n + n) % n
}

// complete fills the composer with "/name " so arguments can follow.
func (a *App) complete(c *Command) {
	t := "/" + c.Name + " "
	a.editor.SetText(t)
	n := len([]rune(t))
	a.editor.SetCaret(n, n)
}

// pick runs a command right away, or completes it when it requires arguments.
func (a *App) pick(c *Command) {
	if strings.HasPrefix(a.t(c.Args), "<") {
		a.complete(c)
		return
	}
	a.editor.SetText("")
	a.runCommand("/" + c.Name)
}

// submitPalette handles Enter while the palette is open.
func (a *App) submitPalette() bool {
	filter, open := a.paletteFilter()
	if !open {
		return false
	}
	if matches := a.matchCommands(filter); len(matches) > 0 {
		a.pick(matches[wrap(a.cmdSel, len(matches))])
	} else {
		a.runCommand(a.editor.Text())
		a.editor.SetText("")
	}
	return true
}

func (a *App) layoutPalette(gtx C) D {
	th := a.th
	filter, _ := a.paletteFilter()
	matches := a.matchCommands(filter)
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return ui.Bordered(gtx, th.Surface, th.BorderHi, 12, func(gtx C) D {
		return layout.UniformInset(6).Layout(gtx, func(gtx C) D {
			if len(matches) == 0 {
				return layout.UniformInset(10).Layout(gtx, th.Label(13, a.t("palette.none", filter), th.Muted, font.Normal).Layout)
			}
			children := []layout.FlexChild{
				layout.Rigid(func(gtx C) D {
					return layout.Inset{Top: 4, Bottom: 6, Left: 10}.Layout(gtx, th.Label(11, a.t("palette.header"), th.Muted, font.SemiBold).Layout)
				}),
			}
			const maxRows = 8
			for i, c := range matches {
				if i == maxRows {
					break
				}
				btn := &a.cmdBtns[a.commandIndex(c)]
				selected := i == a.cmdSel
				children = append(children, layout.Rigid(func(gtx C) D {
					return btn.Layout(gtx, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						bg := th.Surface
						switch {
						case selected:
							bg = th.Selected
						case btn.Hovered():
							bg = th.Hover
						}
						return ui.Card(gtx, bg, 8, layout.Inset{Top: 7, Bottom: 7, Left: 10, Right: 10}, func(gtx C) D {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx C) D {
									gtx.Constraints.Min.X = gtx.Dp(110)
									return th.Mono(13, "/"+c.Name, th.Accent).Layout(gtx)
								}),
								ui.HSpace(10),
								layout.Flexed(1, func(gtx C) D { return ui.OneLine(gtx, th.Label(13, a.t(c.Desc), th.Text, font.Normal)) }),
								layout.Rigid(func(gtx C) D {
									if c.Args == "" {
										return D{}
									}
									return layout.Inset{Left: 10}.Layout(gtx, th.Mono(12, a.t(c.Args), th.Muted).Layout)
								}),
							)
						})
					})
				}))
			}
			children = append(children, layout.Rigid(func(gtx C) D {
				return layout.Inset{Top: 6, Bottom: 2, Left: 10}.Layout(gtx,
					th.Label(11, a.t("palette.hint"), th.Muted, font.Normal).Layout)
			}))
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}

func (a *App) commandIndex(c *Command) int {
	for i, x := range a.commands {
		if x == c {
			return i
		}
	}
	return 0
}
