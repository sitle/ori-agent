package oriagent

import (
	"errors"
	"path/filepath"

	"github.com/sitle/ori-agent/preview"
)

// Ctl is what a slash command can do to the application. It is only valid
// during Command.Run, which runs on the UI goroutine with the state locked:
// long work must go through RunTurn (or a goroutine of its own).
type Ctl struct{ a *App }

func (c *Ctl) Session() *Session    { return c.a.cur }
func (c *Ctl) Sessions() []*Session { return c.a.allSessions() }
func (c *Ctl) Projects() []*Project { return c.a.projects }
func (c *Ctl) Commands() []*Command { return c.a.commands }
func (c *Ctl) WorkDir() string      { return c.a.sessionWorkDir(c.a.cur) }
func (c *Ctl) Models() []Model      { return c.a.cfg.Models }
func (c *Ctl) Model() int           { return c.a.setup.model }
func (c *Ctl) SetModel(i int)       { c.a.setup.model = max(0, min(i, len(c.a.cfg.Models)-1)) }
func (c *Ctl) Dark() bool           { return c.a.th.Dark }
func (c *Ctl) SetDark(dark bool)    { c.a.setDark(dark) }
func (c *Ctl) ToggleSidebar()       { c.a.sideOpen = !c.a.sideOpen }
func (c *Ctl) Notify(msg string)    { c.a.notify(msg) }
func (c *Ctl) NewSession()          { c.a.newSession() }

// NewSessionIn selects an empty session of p, or starts one in it.
func (c *Ctl) NewSessionIn(p *Project) { c.a.newSessionIn(p) }
func (c *Ctl) Pomodoro() *Pomodoro     { return &c.a.pomo }
func (c *Ctl) ToggleVoice()            { c.a.toggleVoice() }
func (c *Ctl) PickFiles()              { c.a.pickFiles() }
func (c *Ctl) Clear()                  { c.a.cur.Messages = nil }

// User is the saved profile; SetUser replaces it (and the Profile fields).
func (c *Ctl) User() User     { return c.a.user() }
func (c *Ctl) SetUser(u User) { c.a.setUser(u) }

// SetModels replaces the model list; the selected model stays when its name
// is still in the list, else the first one is selected.
func (c *Ctl) SetModels(ms []Model) { c.a.setModels(ms) }

// Settings are the current values of the Configuration page.
func (c *Ctl) Settings() Settings { return c.a.settings() }

// T translates key in the interface language.
func (c *Ctl) T(key string, args ...any) string { return c.a.t(key, args...) }

func (c *Ctl) Locale() string { return c.a.tr.Locale() }

// Locales lists the available interface languages.
func (c *Ctl) Locales() []string { return c.a.tr.Locales() }

// LanguageName is a locale's own name ("Français", "English").
func (c *Ctl) LanguageName(locale string) string { return c.a.languageName(locale) }

// SetLocale switches the interface language ("en", "fr-FR"…).
func (c *Ctl) SetLocale(locale string) error {
	if !c.a.setLocale(locale) {
		return errors.New(locale)
	}
	return nil
}

// Show switches the main area to "chat", "setup", "profile" or a Page ID.
func (c *Ctl) Show(view string) { c.a.view = view }

// Open shows a file in the preview pane; relative paths are resolved
// against the working directory of the current session.
func (c *Ctl) Open(path string) error {
	if !filepath.IsAbs(path) {
		path = filepath.Join(c.a.sessionWorkDir(c.a.cur), path)
	}
	d := preview.Open(path, "")
	if d == nil {
		return errors.New(c.a.t("preview.notFound", path))
	}
	c.a.showPreview(d)
	return nil
}

// RunTurn starts a turn in the current session; fn nil means the configured
// Agent. The prompt (if any) is shown as the user's message. It fails when
// the session is already busy.
func (c *Ctl) RunTurn(prompt string, fn TurnFunc) error {
	if fn == nil {
		fn = c.a.cfg.Agent.Run
	}
	if !c.a.startTurn(c.a.cur, prompt, nil, fn) {
		return errors.New(c.a.t("cmd.busy"))
	}
	return nil
}
