// Command anna is "Anna Assistant", the demo application of the ori-agent
// framework: it only configures the shell and provides demo data.
package main

import (
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

func main() {
	view := flag.String("view", "chat", "initial view: chat, setup, profile or shortcuts")
	step := flag.Int("step", 1, "initial setup step (1-4), with -view setup")
	attach := flag.String("attach", "", "comma-separated files to attach at startup")
	lang := flag.String("lang", "", "interface language (fr, en); default: the OS language")
	light := flag.Bool("light", false, "start in light mode")
	flag.Parse()

	cwd, _ := os.Getwd()
	doc := demoDoc(cwd)
	var files []string
	if *attach != "" {
		files = strings.Split(*attach, ",")
	}

	agent := sim.New()
	agent.DocName = "ANNA.md"

	oriagent.Run(oriagent.Config{
		AppName:  "Anna Assistant",
		Logo:     logo,
		Agent:    agent,
		Commands: agent.Commands(), // /review and /init
		Pages:    []oriagent.Page{shortcutsPage},

		// Interface language: the framework's catalogs, plus the simulated
		// agent's and this app's own texts (demo.go).
		Locale:       *lang,
		LightMode:    *light,
		Translations: i18n.Merge(sim.Translations, translations),

		Models: []oriagent.Model{ // descriptions are translation keys
			{Name: "Opus 5.5", Desc: "model.opus"},
			{Name: "Sonnet 5.5", Desc: "model.sonnet"},
			{Name: "Haiku 5.5", Desc: "model.haiku"},
		},
		User: oriagent.User{
			First: "Teva", Last: "Tane", Email: "teva.tane@example.org",
			Service: "Direction du numérique", Role: "Responsable de projet",
		},
		Devices:  demoDevices(),
		Projects: demoProjects(filepath.Base(cwd), doc),

		StartView: *view,
		SetupStep: *step,
		Attach:    files,
		Preview:   filepath.Join(cwd, doc),
	})
}
