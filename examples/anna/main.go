// Command anna is "Anna Assistant", the demo application of the ori-agent
// framework: it configures the shell, provides demo data, and keeps what
// ori-agent leaves to the application (settings, profile, projects) in a
// folder of its own (store.go).
package main

import (
	"context"
	_ "embed"
	"flag"
	"os"
	"path/filepath"
	"strings"

	oriagent "github.com/sitle/ori-agent"
	"github.com/sitle/ori-agent/i18n"
	"github.com/sitle/ori-agent/sim"
)

// Logo of the Polynésie française, from https://ori.gov.pf/assets/logo-pf.svg.
//
//go:embed assets/logo-pf.svg
var logo []byte

// demoDoc finds the demo PDF relative to the working directory, whether the
// example runs from the repository root or from its own folder.
func demoDoc(cwd string) string {
	for _, p := range []string{filepath.Join("examples", "anna", "docs", "demo.pdf"), filepath.Join("docs", "demo.pdf")} {
		if _, err := os.Stat(filepath.Join(cwd, p)); err == nil {
			return filepath.ToSlash(p)
		}
	}
	return "docs/demo.pdf"
}

// dataDir is where Anna keeps its files by default: the user's
// configuration folder (%AppData% on Windows, ~/.config on Linux,
// ~/Library/Application Support on macOS).
func dataDir() string {
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "anna-assistant")
	}
	return ".anna-assistant"
}

func main() {
	view := flag.String("view", "chat", "initial view: chat, setup, profile or shortcuts")
	step := flag.Int("step", 1, "initial setup step (1-4), with -view setup")
	attach := flag.String("attach", "", "comma-separated files to attach at startup")
	lang := flag.String("lang", "", "interface language (fr, en); default: the saved one, else the OS language")
	light := flag.Bool("light", false, "start in light mode")
	data := flag.String("data", dataDir(), "folder for the saved settings, profile and projects")
	flag.Parse()

	cwd, _ := os.Getwd()
	doc := demoDoc(cwd)
	var files []string
	if *attach != "" {
		files = strings.Split(*attach, ",")
	}

	// Saved settings win over the defaults; flags given on the command
	// line win over the saved settings.
	st := newStore(*data)
	settings := st.settings()
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "lang":
			settings.Locale = *lang
		case "light":
			settings.Dark = !*light
		}
	})
	locale := settings.Locale
	if locale == "" {
		locale = i18n.Detect()
	}
	st.setLocale(locale)

	agent := sim.New()
	agent.DocName = "ANNA.md"

	oriagent.Run(oriagent.Config{
		AppName:  "Anna Assistant",
		Logo:     logo,
		Agent:    agent,
		Commands: append(agent.Commands(), loginCommand), // /review, /init, /think, /login
		Pages:    []oriagent.Page{shortcutsPage},
		Panels:   []oriagent.Panel{supervisionPanel}, // supervision.go

		// Interface language: the framework's catalogs, plus the simulated
		// agent's and this app's own texts (demo.go).
		Translations: i18n.Merge(sim.Translations, translations, supervisionTranslations),
		OnLocale:     st.setLocale,

		// Configuration page values, restored and saved by the app.
		Settings:   &settings,
		OnSettings: st.saveSettings,

		Models: demoModels,
		User: st.profile(oriagent.User{
			First: "Teva", Last: "TANE", Email: "teva.tane@example.org",
			Service: "Direction du numérique", Role: "Responsable de projet",
		}),
		ProfileLock: oriagent.ProfileLock{Email: true}, // as if it came from a directory
		OnProfile:   st.saveProfile,
		OnLogout:    st.logout,
		Devices:     demoDevices(),

		// Demo projects first, then the ones created from the sidebar.
		Projects:        append(demoProjects(cwd, doc), st.projects()...),
		OnNewProject:    st.newProject,
		OnUpdateProject: st.updateProject,
		OnDeleteProject: st.deleteProject,
		ProjectFiles:    st.projectFiles,

		// Deleting a session, with what it takes away (store.go).
		OnDeleteSession:   st.deleteSession,
		DeleteSessionInfo: st.deleteInfo,

		// The simulated agent's working modes, kept per project.
		Modes:  sim.Modes,
		OnMode: st.saveMode,

		// "@" suggests the files of the session's folder (mentions.go).
		Mentions: mentions,
		// Going back to a message: the simulated agent never writes a file
		// and keeps no record, so there is nothing to restore.
		OnRewind: func(context.Context, *oriagent.Session, int) error { return nil },

		// Word documents in the preview pane (convert.go).
		Convert: convert,

		StartView: *view,
		SetupStep: *step,
		Attach:    files,
		Preview:   filepath.Join(cwd, doc),
	})
}
