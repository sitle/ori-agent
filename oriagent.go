// Package oriagent is an application shell for agent clients, in the style
// of the Claude Code desktop "work" view: custom title bar, sidebar tree of
// projects → sessions → subagents, chat transcript with streamed reasoning,
// tool calls and task lists, composer with slash-command palette, voice
// dictation and attachments, resizable document preview, Pomodoro timer,
// setup and profile pages. Colors follow the Ori design system.
//
// A project plugs its own behavior through Config: an Agent that writes each
// turn, extra slash Commands, extra Pages, models, user, demo data.
//
//	oriagent.Run(oriagent.Config{
//		AppName: "Mon assistant",
//		Agent:   myAgent,
//	})
//
// It runs on Windows, macOS and Linux without CGO on Windows; macOS and
// Linux need the usual Gio toolchain.
package oriagent

import (
	"context"
	"image/color"
	"log"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/sitle/ori-agent/i18n"
	"github.com/sitle/ori-agent/theme"
	"github.com/sitle/ori-agent/ui"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// Interface texts given in Config (Model.Desc, Page.Title, Command.Desc,
// Placeholder…) may be translation keys: they are looked up in the
// catalogs, and shown as is when no translation exists.

type Model struct {
	Name, Desc string
}

// User fills the profile page and the sidebar footer.
type User struct {
	First, Last, Email, Service, Role string
}

// Page is an extra screen reached from the sidebar. Layout draws the content;
// the shell adds the title and the centered, scrollable column. tr translates
// in the current interface language.
type Page struct {
	ID, Title string
	Icon      func(gtx layout.Context, c color.NRGBA) layout.Dimensions // optional
	Layout    func(gtx layout.Context, th *ui.Theme, tr i18n.Translator) layout.Dimensions
}

// Features turns built-in parts off; the zero value enables everything.
type Features struct {
	Voice, Pomodoro, Attachments, Preview bool
	Setup, Profile, ThemeToggle           bool
	BuiltinCommands                       bool
}

type PomodoroConfig struct {
	Focus, Short, Long time.Duration // defaults 25, 5 and 15 minutes
}

type Config struct {
	AppName     string // title bar; default "Assistant"
	WindowTitle string // OS window title; default AppName
	Logo        []byte // SVG, shown in the title bar, empty states and setup

	Light, Dark theme.Palette // default: Ori
	Fonts       []font.FontFace
	LightMode   bool // start in light mode

	// Locale is the interface language ("fr", "en", "en-US"…); empty means
	// the OS language when a catalog exists for it. Translations add or
	// override catalogs (locale → key → text) on top of Catalogs; keys
	// missing from a locale fall back to FallbackLocale (default "fr").
	Locale         string
	FallbackLocale string
	Translations   map[string]map[string]string
	OnLocale       func(locale string) // called after the language changes

	Agent    Agent // default: a placeholder that explains no agent is set
	WorkDir  string
	Models   []Model
	User     User
	Devices  []*Device  // profile's connected devices; empty hides the section
	Projects []*Project // demo data; default: one empty session
	Commands []*Command // added to the built-in ones
	Pages    []Page

	Dictation Dictation // default: simulated
	Pomodoro  PomodoroConfig
	Disable   Features

	Placeholder   string // composer hint
	ModeLabel     string // chip in the composer ("-" hides it)
	EmptyTitle    string // empty session
	EmptySubtitle string // default: the working directory

	// Startup state.
	StartView string   // "chat" (default), "setup", "profile" or a Page ID
	SetupStep int      // 1-based, with StartView "setup"
	Attach    []string // files attached to the composer at startup
	Preview   string   // file opened in the preview pane at startup

	Width, Height unit.Dp // default 1280×820
}

func (c *Config) defaults() {
	if c.AppName == "" {
		c.AppName = "Assistant"
	}
	if c.WindowTitle == "" {
		c.WindowTitle = c.AppName
	}
	if c.Light == (theme.Palette{}) {
		c.Light = theme.OriLight
	}
	if c.Dark == (theme.Palette{}) {
		c.Dark = theme.OriDark
	}
	if c.Agent == nil {
		c.Agent = AgentFunc(func(ctx context.Context, t *Turn) error {
			b := t.Text()
			b.Write(t.T("agent.none"))
			b.Done()
			return nil
		})
	}
	if c.WorkDir == "" {
		c.WorkDir, _ = os.Getwd()
	}
	if len(c.Models) == 0 {
		c.Models = []Model{{Name: "Default"}}
	}
	if c.FallbackLocale == "" {
		c.FallbackLocale = "fr"
	}
	if c.Dictation == nil {
		c.Dictation = &SimDictation{}
	}
	if c.Pomodoro.Focus == 0 {
		c.Pomodoro = PomodoroConfig{25 * time.Minute, 5 * time.Minute, 15 * time.Minute}
	}
	if c.Placeholder == "" {
		c.Placeholder = "composer.placeholder"
	}
	if c.ModeLabel == "" {
		c.ModeLabel = "composer.mode"
	}
	if c.EmptyTitle == "" {
		c.EmptyTitle = "empty.title"
	}
	if c.Width == 0 {
		c.Width, c.Height = 1280, 820
	}
}

// Run opens the window and blocks; the process exits when it is closed.
func Run(cfg Config) {
	cfg.defaults()
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title(cfg.WindowTitle),
			app.Size(cfg.Width, cfg.Height),
			app.MinSize(unit.Dp(760), unit.Dp(480)),
			app.Decorated(false), // the shell draws its own title bar
		)
		if err := newApp(cfg, w).loop(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
