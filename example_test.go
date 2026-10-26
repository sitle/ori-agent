package oriagent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"

	oriagent "github.com/sitle/ori-agent"
	"github.com/sitle/ori-agent/i18n"
	"github.com/sitle/ori-agent/ui"
)

// The smallest application: an agent that answers with one streamed block.
func ExampleRun() {
	oriagent.Run(oriagent.Config{
		AppName: "Mon assistant",
		Agent: oriagent.AgentFunc(func(ctx context.Context, t *oriagent.Turn) error {
			b := t.Text()
			defer b.Done()
			b.Write("Vous avez écrit : " + t.Prompt)
			return nil
		}),
	})
}

// An agent uses every part of the transcript: reasoning, a task list, tool
// calls (with a file link) and a final answer. It stops when ctx is canceled.
func ExampleAgentFunc() {
	agent := oriagent.AgentFunc(func(ctx context.Context, t *oriagent.Turn) error {
		if t.Thinking { // extended thinking enabled in the settings
			th := t.Think()
			th.Write("I will list the folder first.")
			th.Done()
		}
		tasks := t.Tasks("List files", "Answer")

		entries, err := os.ReadDir(t.WorkDir)
		if err != nil {
			return err // shown in red in the transcript
		}
		call := t.Tool("Bash", "ls")
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		call.Done(strings.Join(names, "\n"))
		tasks.Advance(0)

		if err := ctx.Err(); err != nil { // the user pressed Stop
			return err
		}
		t.Tool("Read", "go.mod").File(filepath.Join(t.WorkDir, "go.mod")).Done("module …")

		b := t.Text()
		b.Write("The folder has " + itoa(len(entries)) + " entries.")
		b.Done()
		tasks.Set(1, oriagent.TaskDone)
		return nil
	})
	oriagent.Run(oriagent.Config{Agent: agent})
}

// A subagent appears under the session in the sidebar tree, and as an
// "Agent" tool call that opens it.
func ExampleTurn_Subagent() {
	_ = oriagent.AgentFunc(func(ctx context.Context, t *oriagent.Turn) error {
		sub := t.Subagent("Explore the docs", "Read every Markdown file and summarize.")
		sub.Tool("Glob", "**/*.md").Done("README.md\ndocs/guide.md")
		sub.Finish("2 documents found.") // becomes the parent's tool output

		b := t.Text()
		b.Write("The subagent found 2 documents.")
		b.Done()
		return nil
	})
}

// An agent runs without a window through the same Turn, for tests and
// evaluation benches; the transcript is read from the returned session.
func ExampleNewHeadless() {
	h := oriagent.NewHeadless(oriagent.Config{Agent: oriagent.AgentFunc(func(ctx context.Context, t *oriagent.Turn) error {
		d, err := t.Approve(ctx, oriagent.ApprovalRequest{Title: "Run go test", Detail: "go test ./..."})
		if err != nil || d == oriagent.Deny {
			return err
		}
		b := t.Text()
		b.Write("Tests pass in " + t.WorkDir)
		b.Done()
		return nil
	})})
	h.Approve = func(r oriagent.ApprovalRequest) oriagent.Decision { return oriagent.Allow }

	s, err := h.Run(context.Background(), nil, "test it", nil, oriagent.TurnOptions{WorkDir: "/ws"})
	if err != nil {
		panic(err)
	}
	fmt.Println(s.Messages[len(s.Messages)-1].Text)
	// Output: Tests pass in /ws
}

// A slash command acts on the application through Ctl. Desc can be a
// translation key or plain text.
func ExampleCommand() {
	hello := &oriagent.Command{
		Name: "hello",
		Args: "[name]",
		Desc: "Say hello",
		Run: func(c *oriagent.Ctl, args string) string {
			if args == "" {
				args = "world"
			}
			c.Notify("Hello sent")
			return "Hello, " + args + "!" // logged in the transcript
		},
	}
	oriagent.Run(oriagent.Config{Commands: []*oriagent.Command{hello}})
}

// A page is an extra screen in the sidebar, drawn with the ui kit and
// translated through tr.
func ExamplePage() {
	about := oriagent.Page{
		ID:    "about",
		Title: "about.title", // translation key
		Layout: func(gtx layout.Context, th *ui.Theme, tr i18n.Translator) layout.Dimensions {
			return th.Section(gtx, tr.T("about.title"), "", th.Label(14, tr.T("about.body"), th.Text, font.Normal).Layout)
		},
	}
	oriagent.Run(oriagent.Config{
		Pages: []oriagent.Page{about},
		Translations: map[string]map[string]string{
			"fr": {"about.title": "À propos", "about.body": "Version de démonstration."},
			"en": {"about.title": "About", "about.body": "Demo version."},
		},
	})
}

// The application keeps the Configuration page values in a file of its own:
// it restores them at startup and saves them whenever the user changes one.
func ExampleSettings() {
	path := filepath.Join(os.TempDir(), "settings.json")
	set := oriagent.DefaultSettings() // missing fields keep their default
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &set)
	}
	oriagent.Run(oriagent.Config{
		Models:   []oriagent.Model{{Name: "Fast"}, {Name: "Smart"}},
		Settings: &set,
		OnSettings: func(s oriagent.Settings) { // UI goroutine: do not block
			data, _ := json.MarshalIndent(s, "", "  ")
			go os.WriteFile(path, data, 0o600)
		},
	})
}

// The application saves the Profile page and handles the sign-out; a /login
// command refreshes the profile and the models from its own turn.
func ExampleConfig_profile() {
	login := &oriagent.Command{Name: "login", Desc: "Sign in", Run: func(c *oriagent.Ctl, _ string) string {
		err := c.RunTurn("", func(ctx context.Context, t *oriagent.Turn) error {
			// … sign in, then:
			t.SetUser(oriagent.User{First: "Hina", Last: "Teriitahi", Email: "hina@example.pf"})
			t.SetModels([]oriagent.Model{{Name: "Fast"}, {Name: "Smart"}})
			return nil
		})
		if err != nil {
			return err.Error()
		}
		return ""
	}}
	oriagent.Run(oriagent.Config{
		Commands:    []*oriagent.Command{login},
		ProfileLock: oriagent.ProfileLock{Email: true}, // from the identity provider
		OnProfile: func(ctx context.Context, u oriagent.User) (oriagent.User, error) {
			// Runs on its own goroutine: save u, return it as saved.
			return u, nil
		},
		OnLogout: func(ctx context.Context) error { return nil },
	})
}

// Demo data fills the sidebar tree and transcripts at startup.
func ExampleNewProject() {
	s := &oriagent.Session{Title: "Fix the login page", Messages: []*oriagent.Message{
		oriagent.UserMsg("The login button does nothing."),
		oriagent.ThinkingMsg("The click handler is probably not bound.", 3),
		oriagent.ToolMsg("Read", "login.go", "   1  package main"),
		oriagent.TextMsg("Fixed: the handler is now registered."),
	}}
	s.Spawn("Run the tests") // a subagent under the session
	oriagent.Run(oriagent.Config{Projects: []*oriagent.Project{oriagent.NewProject("web", s)}})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}
