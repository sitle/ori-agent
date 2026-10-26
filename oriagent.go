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
	"github.com/sitle/ori-agent/preview"
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

// ProfileLock makes Profile page fields read-only, for values the
// application does not let the user change (from a directory, an identity
// provider…).
type ProfileLock struct {
	First, Last, Email, Service, Role bool
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

// Settings are the values of the Configuration page, so that an application
// can restore them at startup (Config.Settings) and keep them when the user
// changes them (Config.OnSettings). The JSON tags make them easy to save in
// a file; ori-agent itself stores nothing.
type Settings struct {
	WorkDir       string `json:"workDir,omitempty"` // ignored at startup if it is not a folder
	AllowCommands bool   `json:"allowCommands"`
	AskEdits      bool   `json:"askEdits"`
	Thinking      bool   `json:"thinking"` // extended thinking
	Notifications bool   `json:"notifications"`
	Model         string `json:"model,omitempty"` // a Model.Name; empty or unknown means the first model
	Dark          bool   `json:"dark"`
	Locale        string `json:"locale,omitempty"` // ignored at startup if no catalog matches
}

// DefaultSettings are the values used when Config.Settings is nil. Decode a
// saved file over them, so that missing fields keep their default.
func DefaultSettings() Settings {
	return Settings{AllowCommands: true, AskEdits: true, Thinking: true, Dark: true}
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

	// Settings, when set, are the initial values of the Configuration page
	// and take precedence over WorkDir, LightMode and Locale (a WorkDir or
	// Locale that cannot be used keeps the Config one). OnSettings receives
	// the new values whenever one of them changes: a setup toggle, model,
	// theme or language, the title bar theme button, /model, /theme, /lang,
	// or the working directory applied by the setup's Finish. It is called
	// at the end of the frame, on the UI goroutine with the state locked:
	// it must return quickly (save in a goroutine of its own if needed).
	Settings   *Settings
	OnSettings func(Settings)

	Agent   Agent // default: a placeholder that explains no agent is set
	WorkDir string
	Models  []Model
	User    User
	// OnProfile is called with the edited fields when the user saves the
	// Profile page, and OnLogout when they sign out from it. Both run on a
	// goroutine of their own while the page buttons wait; the User returned
	// by OnProfile (which may normalize it) replaces the fields, an error is
	// shown and keeps the edits. Nil keeps the profile in memory and only
	// simulates the sign-out.
	OnProfile   func(ctx context.Context, u User) (User, error)
	OnLogout    func(ctx context.Context) error
	ProfileLock ProfileLock // read-only Profile page fields
	// OnNewProject creates a project, such as a workspace folder, from the
	// name and description the user typed. When it is set, the sidebar's
	// main button opens that form instead of starting a session (the + of
	// each project starts one). It runs on a goroutine of its own; an error
	// is shown in the form, which keeps the input.
	OnNewProject func(ctx context.Context, name, description string) (*Project, error)
	// OnUpdateProject saves the edits made on a project's page (opened by
	// the gear of its sidebar row) and returns them as kept, and
	// OnDeleteProject removes a project that is not Permanent. Both run on
	// a goroutine of their own; an error stays on the page. Without
	// OnUpdateProject the edits only apply in memory; without
	// OnDeleteProject the page has no remove button.
	OnUpdateProject func(ctx context.Context, p *Project, edit ProjectEdit) (ProjectEdit, error)
	OnDeleteProject func(ctx context.Context, p *Project) error
	// ProjectFiles lists the files shown in a Files section of a project's
	// page, such as the files attached in its sessions; nil hides the
	// section. It is called on a goroutine of its own when the page opens.
	ProjectFiles func(p *Project) []ProjectFile
	Devices      []*Device  // profile's connected devices; empty hides the section
	Projects     []*Project // demo data; default: one empty session
	Commands     []*Command // added to the built-in ones
	Pages        []Page

	Dictation Dictation // default: simulated
	Pomodoro  PomodoroConfig
	Disable   Features

	Placeholder string // composer hint
	ModeLabel   string // chip in the composer ("-" hides it); unused with Modes

	// Modes turn the composer chip into a menu of working modes, kept per
	// project (Project.Mode); OnMode keeps the user's choice (p nil: the
	// sessions out of projects). An error puts the previous mode back.
	Modes  []Mode
	OnMode func(ctx context.Context, p *Project, mode string) error

	// OnRewind brings the session back to the state before its n-th user
	// message (0 for the first): the application restores its files and its
	// own record; then the transcript is cut there and the message's text
	// returns to the composer. Nil: no "Go back here" action.
	OnRewind func(ctx context.Context, s *Session, n int) error
	// Mentions suggests what "@query" in the composer may refer to, files
	// most often; called on the UI goroutine when the query changes, so it
	// must answer quickly. Nil: no suggestions.
	Mentions      func(s *Session, query string) []Mention
	EmptyTitle    string // empty session
	EmptySubtitle string // default: the working directory

	// Startup state.
	StartView string   // "chat" (default), "setup", "profile" or a Page ID
	SetupStep int      // 1-based, with StartView "setup"
	Attach    []string // files attached to the composer at startup
	Preview   string   // file opened in the preview pane at startup

	// Convert renders in the preview pane the files it does not read itself
	// (Word, Excel, PowerPoint…): as a PDF or as Markdown, in the
	// background. nil, or a nil result: shown as a binary file.
	Convert preview.Converter

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
