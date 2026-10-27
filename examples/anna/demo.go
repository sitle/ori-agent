package main

import (
	"context"
	"image"
	"image/color"
	"path/filepath"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	oriagent "github.com/sitle/ori-agent"
	"github.com/sitle/ori-agent/i18n"
	"github.com/sitle/ori-agent/ui"
)

// demoProjects is a pre-filled sidebar so every element is visible at launch.
func demoProjects(cwd, doc string) []*oriagent.Project {
	verbose := &oriagent.Session{Title: "Add a --verbose flag", Messages: []*oriagent.Message{
		oriagent.UserMsg("Add a --verbose flag to the CLI and document it in the README."),
		oriagent.ThinkingMsg("The user wants a --verbose flag on the CLI and matching docs. Flags are probably defined on the root command, "+
			"so I'll read cmd/root.go first to see how the existing --config flag is registered. A persistent BoolVarP with a -v shorthand is the idiomatic choice with cobra. "+
			"Then I'll add a row to the README's flag table and run the tests to make sure nothing broke.", 4),
		oriagent.TextMsg("I'll look at how flags are wired up in the root command first, then add the new one and update the docs."),
		oriagent.TasksMsg(
			oriagent.Task{Text: "Look at how flags are set up", Status: oriagent.TaskDone},
			oriagent.Task{Text: "Add the --verbose flag", Status: oriagent.TaskDone},
			oriagent.Task{Text: "Document it in the README", Status: oriagent.TaskDone},
			oriagent.Task{Text: "Run the tests", Status: oriagent.TaskDone},
		),
		oriagent.ToolMsg("Read", "cmd/root.go", "   1  package cmd\n   2  \n   3  var rootCmd = &cobra.Command{\n   4  \tUse:   \"mycli\",\n   5  \tShort: \"A tiny example CLI\",\n   6  }\n   7  \n   8  func init() {\n   9  \trootCmd.PersistentFlags().StringVar(&cfgFile, \"config\", \"\", \"config file\")\n  10  }"),
		expanded(edit("cmd/root.go", rootGo, strings.Replace(rootGo, "\")\n}", "\")\n\trootCmd.PersistentFlags().BoolVarP(&verbose, \"verbose\", \"v\", false, \"verbose output\")\n}", 1))),
		edit("README.md", "## Flags\n\n| Flag | Effect |\n|---|---|\n| `--config` | config file |\n",
			"## Flags\n\n| Flag | Effect |\n|---|---|\n| `--config`  | config file    |\n| `--verbose` | verbose output |\n"),
		oriagent.ToolMsg("Bash", "go test ./...", "ok  \texample.com/mycli/cmd\t0.412s\nok  \texample.com/mycli/internal/log\t0.108s"),
		oriagent.ToolMsg("Read", doc, "PDF document - click the file name to open it in the preview pane."),
		oriagent.TextMsg("Added a persistent `--verbose` / `-v` flag in cmd/root.go and a row for it in the README's flag table. Tests pass.\n\n" +
			"This transcript is canned demo content - type a message below to start a simulated run in the current directory."),
	}}
	for _, c := range []struct{ title, task, tool, arg, out string }{
		{"Explore flag wiring in cmd/", "Find where CLI flags are registered and how --config is wired.", "Read", "cmd/root.go", "   1  package cmd\n   2  \n   3  var rootCmd = &cobra.Command{"},
		{"Update the README flag table", "Add a --verbose row to the flags table in README.md.", "Edit", "README.md", "+| `--verbose` | verbose output |"},
		{"Run the test suite", "Run go test ./... and report failures.", "Bash", "go test ./...", "ok  \texample.com/mycli/cmd\t0.412s"},
	} {
		s := verbose.Spawn(c.title)
		s.Messages = []*oriagent.Message{
			oriagent.UserMsg(c.task),
			oriagent.ToolMsg(c.tool, c.arg, c.out),
			oriagent.TextMsg("Done - reported back to the main session."),
		}
	}

	form := &oriagent.Session{Title: "Formulaire de demande en ligne", Messages: []*oriagent.Message{
		oriagent.UserMsg("Construis le formulaire de demande en ligne avec validation côté API."),
		oriagent.TextMsg("I'll split this across three subagents: API validation, Ori form components, and end-to-end tests."),
	}}
	for i, t := range []string{"Validation des champs côté API", "Composants Ori pour le formulaire", "Tests end-to-end du parcours"} {
		s := form.Spawn(t)
		s.Busy = i < 2 // shown as running, to show the status dots
		s.Messages = []*oriagent.Message{oriagent.UserMsg(t), oriagent.TextMsg("Demo subagent - still working.")}
	}
	migration := &oriagent.Session{Title: "Migration vers Ori 0.4", Messages: []*oriagent.Message{
		oriagent.UserMsg("Migre les composants vers Ori 0.4."),
		oriagent.TextMsg("Demo session - no transcript was recorded."),
	}}

	// The folder the app runs in: its sessions work there, and it cannot be
	// removed from its page.
	here := oriagent.NewProject(filepath.Base(cwd), verbose)
	here.WorkDir, here.Permanent = cwd, true
	here.Description = "Le dossier dans lequel Anna a été lancée."

	// A project with permissions of its own: the agent may not run commands
	// in its sessions, whatever the Configuration says.
	portal := oriagent.NewProject("portail-usagers", form, migration)
	portal.Description = "Démarches en ligne des usagers du service public."
	portal.Permissions = &oriagent.Permissions{AllowCommands: false, AskEdits: true}

	data := oriagent.NewProject("data-pipeline",
		demo("Sauvegarde quotidienne PostgreSQL"),
		demo("Import des données INSEE"),
	)
	data.Open = false

	return []*oriagent.Project{here, portal, data}
}

// demoModels are the models offered before /login; descriptions are keys.
var demoModels = []oriagent.Model{
	{Name: "Opus 5.5", Desc: "model.opus"},
	{Name: "Sonnet 5.5", Desc: "model.sonnet"},
	{Name: "Haiku 5.5", Desc: "model.haiku"},
}

// loginCommand simulates signing in to an identity provider: it runs a turn
// that replaces the profile and the list of models (Turn.SetUser,
// Turn.SetModels), as a real application would after an authentication.
var loginCommand = &oriagent.Command{
	Name: "login",
	Desc: "login.desc",
	Run: func(c *oriagent.Ctl, _ string) string {
		u := c.User() // Ctl is only valid here, not in the turn's goroutine
		err := c.RunTurn("", func(ctx context.Context, t *oriagent.Turn) error {
			call := t.Tool("Auth", "sso.example.org")
			if err := pause(ctx, 900); err != nil {
				return err
			}
			call.Done(t.T("login.tool"))
			u.Email = "teva.tane@example.org"
			u.Role = t.T("login.role")
			t.SetUser(u)
			t.SetModels(append(demoModels, oriagent.Model{Name: "Opus 5.5 Entreprise", Desc: "model.enterprise"}))
			b := t.Text()
			b.Write(t.T("login.done"))
			b.Done()
			return nil
		})
		if err != nil {
			return err.Error()
		}
		return ""
	},
}

func demo(title string) *oriagent.Session {
	return &oriagent.Session{Title: title, Messages: []*oriagent.Message{oriagent.UserMsg(title), oriagent.TextMsg("Demo session.")}}
}

func expanded(m *oriagent.Message) *oriagent.Message {
	m.Expanded = true
	return m
}

// rootGo is the demo's cmd/root.go before the --verbose flag.
const rootGo = "package cmd\n\nvar rootCmd = &cobra.Command{\n\tUse:   \"mycli\",\n\tShort: \"A tiny example CLI\",\n}\n\nfunc init() {\n\trootCmd.PersistentFlags().StringVar(&cfgFile, \"config\", \"\", \"config file\")\n}\n"

// edit is an Edit tool call that shows its change as a diff.
func edit(path, old, new string) *oriagent.Message {
	m := oriagent.ToolMsg("Edit", path, "")
	m.HasDiff, m.DiffOld, m.DiffNew = true, old, new
	return m
}

func demoDevices() []*oriagent.Device {
	return []*oriagent.Device{
		{Name: "Windows 11", Detail: "Cet ordinateur · actif maintenant", Current: true},
		{Name: "macOS", Detail: "MacBook Air · il y a 2 jours"},
		{Name: "Linux", Detail: "Poste de travail · il y a 1 semaine"},
	}
}

// translations are this app's own texts, merged with the framework's.
var translations = map[string]map[string]string{
	"fr": {
		"model.opus":         "Le plus capable, pour les tâches complexes et longues.",
		"model.sonnet":       "Bon équilibre entre vitesse et qualité.",
		"model.haiku":        "Le plus rapide, pour les tâches simples.",
		"shortcuts.title":    "Raccourcis",
		"shortcuts.keyboard": "Clavier",
		"shortcuts.desc":     "Les raccourcis disponibles dans la conversation.",
		"key.enter":          "Entrée",
		"key.shiftEnter":     "Maj + Entrée",
		"key.palette":        "↑ ↓ · Tab · Échap",
		"key.send":           "Envoyer le message",
		"key.newline":        "Nouvelle ligne",
		"key.slash":          "Ouvrir la palette de commandes",
		"key.paletteNav":     "Naviguer, compléter, fermer la palette",
		"key.sidebar":        "Afficher ou masquer la barre latérale",
		"model.enterprise":   "Réservé aux comptes de l'administration, après connexion.",
		"login.desc":         "Se connecter avec le compte de l'administration (simulé)",
		"login.tool":         "Authentification réussie.",
		"login.role":         "Responsable de projet (compte vérifié)",
		"login.done":         "Connexion simulée : le profil vient de l'annuaire et un modèle réservé est disponible dans la Configuration.",
		"delete.info":        "Messages retirés : %d. Sous-agents retirés : %d. Anna ne garde aucun fichier de session : rien d'autre n'est effacé.",
	},
	"en": {
		"model.opus":         "The most capable, for complex and long tasks.",
		"model.sonnet":       "A good balance of speed and quality.",
		"model.haiku":        "The fastest, for simple tasks.",
		"shortcuts.title":    "Shortcuts",
		"shortcuts.keyboard": "Keyboard",
		"shortcuts.desc":     "Shortcuts available in the conversation.",
		"key.enter":          "Enter",
		"key.shiftEnter":     "Shift + Enter",
		"key.palette":        "↑ ↓ · Tab · Esc",
		"key.send":           "Send the message",
		"key.newline":        "New line",
		"key.slash":          "Open the command palette",
		"key.paletteNav":     "Navigate, complete, close the palette",
		"key.sidebar":        "Show or hide the sidebar",
		"model.enterprise":   "For administration accounts, after signing in.",
		"login.desc":         "Sign in with the administration account (simulated)",
		"login.tool":         "Authentication succeeded.",
		"login.role":         "Project manager (verified account)",
		"login.done":         "Simulated sign-in: the profile now comes from the directory and a reserved model is available in the settings.",
		"delete.info":        "Messages removed: %d. Subagents removed: %d. Anna keeps no session files: nothing else is erased.",
	},
}

// shortcutsPage shows how a project adds its own screen, built with the ui
// kit and translated through tr.
var shortcutsPage = oriagent.Page{
	ID:    "shortcuts",
	Title: "shortcuts.title",
	Icon:  keyboardIcon,
	Layout: func(gtx layout.Context, th *ui.Theme, tr i18n.Translator) layout.Dimensions {
		rows := [][2]string{
			{tr.T("key.enter"), tr.T("key.send")},
			{tr.T("key.shiftEnter"), tr.T("key.newline")},
			{"/", tr.T("key.slash")},
			{tr.T("key.palette"), tr.T("key.paletteNav")},
			{"Ctrl + B", tr.T("key.sidebar")},
		}
		return th.Section(gtx, tr.T("shortcuts.keyboard"), tr.T("shortcuts.desc"), func(gtx layout.Context) layout.Dimensions {
			var children []layout.FlexChild
			for i, r := range rows {
				if i > 0 {
					children = append(children, layout.Rigid(th.HLine))
				}
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: 10, Bottom: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								gtx.Constraints.Min.X = gtx.Dp(180)
								return ui.Bordered(gtx, th.Code, th.Border, 6, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: 3, Bottom: 3, Left: 8, Right: 8}.Layout(gtx, th.Mono(13, r[0], th.Text).Layout)
								})
							}),
							ui.HSpace(16),
							layout.Flexed(1, th.Label(14, r[1], th.Text, font.Normal).Layout),
						)
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	},
}

func keyboardIcon(gtx layout.Context, c color.NRGBA) layout.Dimensions {
	w, h := gtx.Dp(16), gtx.Dp(12)
	sw := gtx.Dp(1.5)
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: clip.UniformRRect(image.Rect(1, 1, w-1, h-1), gtx.Dp(2)).Path(gtx.Ops), Width: float32(sw)}.Op())
	k := gtx.Dp(2)
	for _, x := range []unit.Dp{4, 7, 10} {
		paint.FillShape(gtx.Ops, c, clip.Rect{Min: image.Pt(gtx.Dp(x), gtx.Dp(4)), Max: image.Pt(gtx.Dp(x)+k, gtx.Dp(4)+k)}.Op())
	}
	paint.FillShape(gtx.Ops, c, clip.Rect{Min: image.Pt(gtx.Dp(4), gtx.Dp(7.5)), Max: image.Pt(gtx.Dp(12), gtx.Dp(7.5)+k)}.Op())
	return layout.Dimensions{Size: image.Pt(w, h)}
}
