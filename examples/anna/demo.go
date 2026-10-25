package main

import (
	"image"
	"image/color"

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
func demoProjects(cwdName, doc string) []*oriagent.Project {
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
		expanded(oriagent.ToolMsg("Edit", "cmd/root.go", " func init() {\n \trootCmd.PersistentFlags().StringVar(&cfgFile, \"config\", \"\", \"config file\")\n+\trootCmd.PersistentFlags().BoolVarP(&verbose, \"verbose\", \"v\", false, \"verbose output\")\n }")),
		oriagent.ToolMsg("Edit", "README.md", " ## Flags\n \n-| `--config` | config file |\n+| `--config`  | config file    |\n+| `--verbose` | verbose output |"),
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

	data := oriagent.NewProject("data-pipeline",
		demo("Sauvegarde quotidienne PostgreSQL"),
		demo("Import des données INSEE"),
	)
	data.Open = false

	return []*oriagent.Project{
		oriagent.NewProject(cwdName, verbose),
		oriagent.NewProject("portail-usagers", form, migration),
		data,
	}
}

func demo(title string) *oriagent.Session {
	return &oriagent.Session{Title: title, Messages: []*oriagent.Message{oriagent.UserMsg(title), oriagent.TextMsg("Demo session.")}}
}

func expanded(m *oriagent.Message) *oriagent.Message {
	m.Expanded = true
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
