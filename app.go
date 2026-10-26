package oriagent

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/sitle/ori-agent/i18n"
	"github.com/sitle/ori-agent/preview"
	"github.com/sitle/ori-agent/ui"
)

const (
	viewChat    = "chat"
	viewSetup   = "setup"
	viewProfile = "profile"
)

// App is the running shell. All its state is guarded by mu: the frame
// handler holds it, background work goes through update.
type App struct {
	cfg      Config
	w        *app.Window
	th       *ui.Theme
	logo     *ui.Logo
	tr       *i18n.Bundle
	workDir  string
	commands []*Command

	mu       sync.Mutex
	projects []*Project
	cur      *Session
	view     string
	focus    bool
	toast    ui.Toast

	// chat
	editor     widget.Editor
	send       widget.Clickable
	transcript widget.List

	// sidebar and title bar
	sideOpen             bool
	sideList             widget.List
	newBtn, sideBtn      widget.Clickable
	themeBtn             widget.Clickable
	backBtn, fwdBtn      widget.Clickable
	winBtns              [3]widget.Clickable
	maximized            bool
	navSetup, navProfile widget.Clickable
	pageBtns             []widget.Clickable
	pageLists            map[string]*widget.List
	titleBtn, menuScrim  widget.Clickable
	menuOpen             bool
	menuItems            [5]widget.Clickable
	renaming, renameSeen bool
	renameFrames         int
	renameEd             widget.Editor

	// preview pane
	doc                  *preview.Doc
	previewW             int
	split                int // event tag of the divider
	pressX               float32
	dragging, splitHover bool

	// features
	setup       setupState
	saved       Settings // last values given to Config.OnSettings
	profile     profileState
	newProj     newProjectState
	headless    *Headless // set when running without a window
	projPage    projectPageState
	voice       voiceState
	pomo        Pomodoro
	attachments []*Attachment
	attachBtn   widget.Clickable
	picking     bool
	cmdSel      int
	cmdBtns     []widget.Clickable
}

func newApp(cfg Config, w *app.Window) *App {
	a := &App{cfg: cfg, w: w, workDir: cfg.WorkDir, focus: true, sideOpen: true, view: viewChat}
	a.tr = i18n.New(cfg.FallbackLocale)
	for l, c := range Catalogs {
		a.tr.Add(l, c)
	}
	for l, c := range cfg.Translations {
		a.tr.Add(l, c)
	}
	set := cfg.Settings
	if set != nil && isDir(set.WorkDir) {
		a.workDir = set.WorkDir
	}
	if (set == nil || !a.tr.SetLocale(set.Locale)) && !a.tr.SetLocale(cfg.Locale) {
		a.tr.SetLocale(i18n.Detect()) // keeps the fallback if unavailable
	}
	a.th = ui.NewTheme(cfg.Light, cfg.Dark, cfg.Fonts...)
	a.th.SetDark(!cfg.LightMode)
	if set != nil {
		a.th.SetDark(set.Dark)
	}
	a.logo = ui.NewLogo(cfg.Logo)
	a.editor.Submit = true
	a.renameEd.SingleLine, a.renameEd.Submit = true, true
	a.newProj.name.SingleLine = true
	a.transcript.Axis, a.transcript.ScrollToEnd = layout.Vertical, true
	a.sideList.Axis = layout.Vertical
	a.pageBtns = make([]widget.Clickable, len(cfg.Pages))
	a.pageLists = map[string]*widget.List{}

	if !cfg.Disable.BuiltinCommands {
		a.commands = builtinCommands(cfg)
	}
	a.commands = append(a.commands, cfg.Commands...)
	a.cmdBtns = make([]widget.Clickable, len(a.commands))

	a.projects = cfg.Projects
	if len(a.projects) == 0 {
		a.projects = []*Project{NewProject(filepath.Base(a.workDir), &Session{Title: a.t("session.new")})}
	}
	for _, p := range a.projects {
		if len(p.Sessions) > 0 {
			a.cur = p.Sessions[0]
			break
		}
	}
	if a.cur == nil {
		a.projects[0].Add(&Session{Title: a.t("session.new")})
		a.cur = a.projects[0].Sessions[0]
	}
	if l, ok := cfg.Dictation.(LocaleSetter); ok {
		l.SetLocale(a.tr.Locale())
	}

	a.initSetup()
	a.initProfile()
	a.pomo.init(cfg.Pomodoro, a.tr)
	if cfg.StartView != "" {
		a.view = cfg.StartView
	}
	if a.view == viewSetup && cfg.SetupStep > 0 {
		a.setup.step = min(cfg.SetupStep, len(setupSteps)) - 1
	}
	if len(cfg.Attach) > 0 {
		a.addFiles(cfg.Attach)
	}
	if cfg.Preview != "" {
		a.showPreview(a.openDoc(cfg.Preview, ""))
	}
	a.saved = a.settings()
	return a
}

// settings are the current values of the Configuration page.
func (a *App) settings() Settings {
	st := &a.setup
	return Settings{
		WorkDir:       a.workDir,
		AllowCommands: st.allowCmds.Value,
		AskEdits:      st.askEdits.Value,
		Thinking:      st.thinking.Value,
		Notifications: st.notifyOn.Value,
		Model:         a.model().Name,
		Dark:          a.th.Dark,
		Locale:        a.tr.Locale(),
	}
}

// checkSettings calls Config.OnSettings when a value changed since the last
// call; the frame ends with it, so every source of change is covered.
func (a *App) checkSettings() {
	if s := a.settings(); s != a.saved {
		a.saved = s
		if a.cfg.OnSettings != nil {
			a.cfg.OnSettings(s)
		}
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return path != "" && err == nil && info.IsDir()
}

func (a *App) loop() error {
	var ops op.Ops
	for {
		switch e := a.w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.ConfigEvent:
			a.maximized = e.Config.Mode == app.Maximized
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			a.mu.Lock()
			a.frame(gtx)
			a.mu.Unlock()
			e.Frame(gtx.Ops)
		}
	}
}

// update mutates state from a background goroutine and requests a redraw.
func (a *App) update(f func()) {
	a.mu.Lock()
	f()
	a.mu.Unlock()
	if a.w != nil { // nil in tests
		a.w.Invalidate()
	}
}

func (a *App) notify(msg string) { a.toast.Show(msg) }

// t translates key in the interface language.
func (a *App) t(key string, args ...any) string { return a.tr.T(key, args...) }

// LocaleSetter is implemented by parts that follow the interface language,
// such as a Dictation engine.
type LocaleSetter interface {
	SetLocale(locale string)
}

// setLocale switches the interface language; everything is redrawn from
// the catalogs on the next frame.
func (a *App) setLocale(locale string) bool {
	if !a.tr.SetLocale(locale) {
		return false
	}
	l := a.tr.Locale()
	if s, ok := a.cfg.Dictation.(LocaleSetter); ok {
		s.SetLocale(l)
	}
	if a.cfg.OnLocale != nil {
		a.cfg.OnLocale(l)
	}
	return true
}

// languageName is a locale's own name, from its catalog.
func (a *App) languageName(locale string) string { return a.tr.In(locale, "lang.name") }

func (a *App) model() Model { return a.cfg.Models[a.setup.model] }

// setModels replaces the model list and keeps the selected model when its
// name is still there, else selects the first one.
func (a *App) setModels(ms []Model) {
	if len(ms) == 0 {
		ms = []Model{{Name: "Default"}}
	}
	cur, sel := a.model().Name, 0
	for i, m := range ms {
		if m.Name == cur {
			sel = i
			break
		}
	}
	a.cfg.Models = append([]Model(nil), ms...)
	a.setup.model = sel
	a.setup.models = make([]widget.Clickable, len(ms))
}

func (a *App) setDark(dark bool) { a.th.SetDark(dark) }

func (a *App) frame(gtx C) {
	a.handlePalette(gtx)
	for {
		ev, ok := a.editor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			a.onSubmit()
		}
	}
	if a.send.Clicked(gtx) {
		if a.cur.Busy && a.cur.cancel != nil {
			a.cur.cancel() // the send button is a stop button while busy
		} else {
			a.onSubmit()
		}
	}
	if a.newBtn.Clicked(gtx) {
		if a.cfg.OnNewProject != nil {
			a.openNewProject()
		} else {
			a.newSession()
		}
	}
	for _, p := range a.projects {
		if p.add.Clicked(gtx) {
			a.newSessionIn(p)
		}
		if p.manage.Clicked(gtx) {
			a.openProject(p)
		}
	}
	if a.navSetup.Clicked(gtx) {
		a.view = viewSetup
	}
	if a.navProfile.Clicked(gtx) {
		a.view = viewProfile
	}
	for i := range a.pageBtns {
		if a.pageBtns[i].Clicked(gtx) {
			a.view = a.cfg.Pages[i].ID
		}
	}
	a.handleTitleBar(gtx)
	// Ctrl+B (Cmd+B on macOS) toggles the sidebar, as in the desktop client.
	for {
		ev, ok := gtx.Event(key.Filter{Name: "B", Required: key.ModShortcut})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			a.sideOpen = !a.sideOpen
		}
	}
	a.handleTree(gtx)
	a.handleAttachments(gtx)
	for _, m := range a.cur.Messages {
		if m.toggle.Clicked(gtx) || m.toggleTail.Clicked(gtx) {
			m.Expanded = !m.Expanded
		}
		if m.open.Clicked(gtx) {
			a.openMessage(m)
		}
	}
	if a.doc != nil && a.doc.Close.Clicked(gtx) {
		a.doc = nil
	}
	a.splitDrag(gtx)
	if a.focus {
		if a.view == viewChat {
			gtx.Execute(key.FocusCmd{Tag: &a.editor})
		}
		a.focus = false
	}

	paint.Fill(gtx.Ops, a.th.Bg)
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(a.titleBar),
		layout.Flexed(1, a.layoutBody),
	)
	a.toast.Layout(gtx, a.th)
	a.checkSettings()
}

func (a *App) layoutBody(gtx C) D {
	var children []layout.FlexChild
	if a.sideOpen {
		children = append(children, layout.Rigid(a.layoutSidebar))
	}
	children = append(children, layout.Flexed(1, a.layoutView))
	// The preview sits next to the chat, and next to a project's page so its
	// files can be read without leaving it.
	if a.doc != nil && (a.view == viewChat || a.view == viewProject) && !a.cfg.Disable.Preview {
		avail := gtx.Constraints.Max.X
		if a.sideOpen {
			avail -= gtx.Dp(sidebarWidth)
		}
		pw := a.clampPreview(gtx, avail)
		children = append(children,
			layout.Rigid(a.splitHandle),
			layout.Rigid(func(gtx C) D {
				gtx.Constraints.Min.X, gtx.Constraints.Max.X = pw, pw
				return a.doc.Layout(gtx, a.th)
			}),
		)
	}
	return layout.Flex{}.Layout(gtx, children...)
}

func (a *App) layoutView(gtx C) D {
	switch a.view {
	case viewChat:
		return a.layoutChat(gtx)
	case viewSetup:
		return a.layoutSetup(gtx)
	case viewProfile:
		return a.layoutProfile(gtx)
	case viewNewProject:
		return a.layoutNewProject(gtx)
	case viewProject:
		return a.layoutProject(gtx)
	}
	for _, p := range a.cfg.Pages {
		if p.ID == a.view {
			l := a.pageLists[p.ID]
			if l == nil {
				l = new(widget.List)
				a.pageLists[p.ID] = l
			}
			return a.th.Page(gtx, l, a.t(p.Title), func(gtx C) D { return p.Layout(gtx, a.th, a.tr) })
		}
	}
	return a.layoutChat(gtx)
}

func (a *App) viewTitle() string {
	switch a.view {
	case viewChat:
		title := a.cur.Title
		if p := a.cur.parent; p != nil {
			title = p.Title + " / " + title
		}
		if p := a.cur.project; p != nil {
			title = p.Name + " / " + title
		}
		return title
	case viewSetup:
		return a.t("view.setup")
	case viewProfile:
		return a.t("view.profile")
	case viewNewProject:
		return a.t("project.new")
	case viewProject:
		if p := a.projPage.p; p != nil {
			return p.Name
		}
	}
	for _, p := range a.cfg.Pages {
		if p.ID == a.view {
			return a.t(p.Title)
		}
	}
	return ""
}

// onSubmit routes Enter: palette pick, slash command, or a chat message.
func (a *App) onSubmit() {
	if a.submitPalette() {
		return
	}
	if t := strings.TrimSpace(a.editor.Text()); strings.HasPrefix(t, "/") {
		a.editor.SetText("")
		a.runCommand(t)
		return
	}
	s := a.cur
	txt := strings.TrimSpace(a.editor.Text())
	atts := a.attachments
	if (txt == "" && len(atts) == 0) || s.Busy {
		return
	}
	for _, t := range atts {
		if t.uploading() {
			a.notify(a.t("composer.uploading"))
			return
		}
	}
	a.editor.SetText("")
	a.attachments = nil
	if len(s.Messages) == 0 {
		title := txt
		if title == "" {
			title = atts[0].Name
		}
		s.Title = truncate(title, 40)
	}
	a.startTurn(s, txt, atts, a.cfg.Agent.Run)
}

func (a *App) newSession() {
	if len(a.cur.Messages) > 0 {
		p := a.cur.project
		s := &Session{Title: a.t("session.new")}
		p.Add(s)
		p.Open, a.cur = true, s
	}
	a.focus, a.view = true, viewChat
}

func (a *App) allSessions() []*Session {
	var all []*Session
	walkSessions(a.projects, func(s *Session) { all = append(all, s) })
	return all
}

// archive removes s from the tree and selects another session.
func (a *App) archive(s *Session) {
	drop := func(ss []*Session) []*Session {
		for i, x := range ss {
			if x == s {
				return append(ss[:i:i], ss[i+1:]...)
			}
		}
		return ss
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.parent != nil {
		s.parent.Children = drop(s.parent.Children)
	} else {
		s.project.Sessions = drop(s.project.Sessions)
	}
	a.notify(a.t("toast.archived", s.Title))
	if all := a.allSessions(); len(all) > 0 {
		a.cur = all[0]
		return
	}
	n := &Session{Title: a.t("session.new")}
	s.project.Add(n)
	a.cur = n
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}
