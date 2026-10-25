---
name: ori-agent
description: Build desktop agent-client apps in Go with the ori-agent framework (github.com/sitle/ori-agent), a Gio application shell in the style of the Claude Code desktop "work" view. Use when creating or modifying an app that imports github.com/sitle/ori-agent - writing an Agent (LLM or scripted), slash commands, custom pages, translations, themes, demo data - or when the user asks for a chat/agent desktop UI in Go with Gio.
---

# Building apps with ori-agent

ori-agent gives a complete desktop window: custom title bar, sidebar tree (projects → sessions → subagents), chat transcript (streamed text and reasoning, tool calls, task lists), composer with slash-command palette, attachments, voice dictation, document preview (Markdown, code, PDF, images), Pomodoro, setup and profile pages, French and English UI with live switching. The app author writes the **agent** and configures the rest through one `oriagent.Config`.

Module: `github.com/sitle/ori-agent` (package name `oriagent`). Requires Go 1.26+. API reference: https://pkg.go.dev/github.com/sitle/ori-agent

## Golden rules

1. **One entry point**: `oriagent.Run(cfg)` opens the window and blocks; the process exits when the window closes. Call it from `main`. Never call it twice.
2. **Agents write only through `*oriagent.Turn`.** Never touch `Session`/`Message` fields from an agent goroutine. Turn methods are goroutine-safe; they lock and redraw for you.
3. **Respect cancellation**: the Stop button cancels `ctx`. Pass `ctx` to every blocking call (HTTP, `time.After` in a `select`) and return `ctx.Err()`. Leftover running blocks are closed automatically.
4. **`*oriagent.Ctl` is only valid inside `Command.Run`**, which runs on the UI goroutine with the state locked. Do not keep it, do not block in it: start long work with `c.RunTurn(prompt, fn)` or a goroutine.
5. **No hard-coded UI text if the app is multilingual**: put strings in `Config.Translations` (locale → key → text) and pass keys to `Model.Desc`, `Page.Title`, `Command.Desc`/`Args`, `Placeholder`, `ModeLabel`, `EmptyTitle`. Unknown keys are displayed as is, so plain text also works.
6. **No hard-coded colors in custom pages**: use the theme (`th.Text`, `th.Muted`, `th.Accent`, `th.Surface`…).
7. On Linux the build needs Gio's system packages (see below); on Windows nothing else is needed.

## Minimal app

```go
package main

import (
	"context"

	oriagent "github.com/sitle/ori-agent"
)

func main() {
	oriagent.Run(oriagent.Config{
		AppName: "My assistant",
		Agent: oriagent.AgentFunc(func(ctx context.Context, t *oriagent.Turn) error {
			b := t.Text()
			defer b.Done()
			b.Write("You wrote: " + t.Prompt)
			return nil
		}),
	})
}
```

Setup: `go mod init <name> && go get github.com/sitle/ori-agent@latest && go run .`

## Agent API

```go
type Agent interface{ Run(ctx context.Context, t *Turn) error }
type AgentFunc func(ctx context.Context, t *Turn) error // adapter
type TurnFunc func(ctx context.Context, t *Turn) error  // for Ctl.RunTurn
```

`Run` is called on its own goroutine for each user message; one turn at a time per session. A returned error is shown in red; a canceled turn shows "Interrupted.".

Turn inputs (read-only fields):

| Field | Type | Meaning |
|---|---|---|
| `Prompt` | `string` | user message (may be empty when only files are sent) |
| `Attachments` | `[]*Attachment` | `Path`, `Name`, `Ext` (lower case, no dot), `Size`, `ImageSize`, `IsImage()` |
| `History` | `[]HistoryEntry` | earlier exchanges: `Role` (`"user"`/`"assistant"`), `Text` |
| `Thinking` | `bool` | extended thinking enabled in settings |
| `Model` | `Model` | `Name`, `Desc` chosen by the user |
| `WorkDir` | `string` | working directory |
| `Locale` | `string` | UI language (`"fr"`, `"en"`): answer in it |

Turn outputs:

| Call | Effect |
|---|---|
| `b := t.Text(); b.Write(s); b.Done()` | streamed assistant text |
| `b := t.Think(); b.Write(s); b.Done()` | streamed reasoning, collapsed to "Thought for Ns" on `Done` |
| `c := t.Tool(name, arg); c.File(path); c.Done(output)` | tool call row; `File` makes the arg a link to the preview; output lines starting with `+`/`-` are colored as a diff |
| `l := t.Tasks(items...)` then `l.Advance(i)`, `l.Set(i, oriagent.TaskDone)` | live checklist; item 0 starts running |
| `s := t.Subagent(title, prompt)` … `s.Finish(report)` | child session in the sidebar + clickable "Agent" call; write into `s` like any Turn; `report` becomes the call output |
| `t.Notify(msg)` | toast |
| `t.T(key, args...)` | translate with the app catalogs in the turn's locale |

`Block.Write` appends; call `Done` exactly once (use `defer`). Starting a new `Text()` after a `Think()` is the normal pattern.

### LLM streaming adapter (pattern)

```go
func (a *Agent) Run(ctx context.Context, t *oriagent.Turn) error {
	msgs := convert(t.History, t.Prompt, t.Attachments)
	for {
		stream, err := a.client.Stream(ctx, req(t.Model.Name, msgs, t.Thinking))
		if err != nil {
			return err
		}
		var text, thought *oriagent.Block
		var calls []ToolUse
		for ev := range stream.Events() {
			switch ev.Kind {
			case ThinkingDelta:
				if thought == nil {
					thought = t.Think()
				}
				thought.Write(ev.Text)
			case TextDelta:
				if thought != nil {
					thought.Done()
					thought = nil
				}
				if text == nil {
					text = t.Text()
				}
				text.Write(ev.Text)
			case ToolUseEvent:
				calls = append(calls, ev.Tool)
			}
		}
		if thought != nil {
			thought.Done()
		}
		if text != nil {
			text.Done()
		}
		if err := stream.Err(); err != nil {
			return err
		}
		if len(calls) == 0 {
			return nil
		}
		for _, c := range calls {
			row := t.Tool(c.Name, c.Summary())
			out := a.exec(ctx, c)
			row.Done(out)
			msgs = append(msgs, toolResult(c, out))
		}
	}
}
```

Replace the imaginary `client`/event types with the provider's SDK. Keep `ctx` flowing into the client.

## Config reference

All fields optional (`oriagent.Config`):

| Field | Type | Default / notes |
|---|---|---|
| `AppName`, `WindowTitle` | `string` | `"Assistant"`, `AppName` |
| `Logo` | `[]byte` | SVG, typically `//go:embed` |
| `Light`, `Dark` | `theme.Palette` | `theme.OriLight`, `theme.OriDark` |
| `Fonts` | `[]font.FontFace` | Go fonts |
| `LightMode` | `bool` | start in light mode |
| `Locale`, `FallbackLocale` | `string` | OS language if available; fallback `"fr"` |
| `Translations` | `map[string]map[string]string` | merged over `oriagent.Catalogs`; combine sets with `i18n.Merge` |
| `OnLocale` | `func(string)` | after a language change |
| `Agent` | `Agent` | placeholder that says none is set |
| `WorkDir` | `string` | current directory |
| `Models` | `[]Model` | `{Name, Desc}`; one "Default" model |
| `User` | `User` | `First, Last, Email, Service, Role` for the profile |
| `Devices` | `[]*Device` | `{Name, Detail, Current}`; empty hides the section |
| `Projects` | `[]*Project` | initial tree; default one empty session |
| `Commands` | `[]*Command` | added to built-ins |
| `Pages` | `[]Page` | sidebar pages |
| `Dictation` | `Dictation` | simulated |
| `Pomodoro` | `PomodoroConfig` | `{Focus, Short, Long time.Duration}`, 25/5/15 min |
| `Disable` | `Features` | `Voice, Pomodoro, Attachments, Preview, Setup, Profile, ThemeToggle, BuiltinCommands bool` |
| `Placeholder`, `ModeLabel`, `EmptyTitle`, `EmptySubtitle` | `string` | text or translation keys; `ModeLabel: "-"` hides the chip |
| `StartView`, `SetupStep`, `Attach`, `Preview` | | startup state: `"chat"`, `"setup"`, `"profile"` or a page ID; step 1-4; files; file to preview |
| `Width`, `Height` | `unit.Dp` | 1280×820 |

## Slash commands

```go
&oriagent.Command{
	Name: "export",           // typed as /export
	Args: "<file>",           // "<x>" required: picking completes instead of running; "[x]" optional
	Desc: "cmd.export.desc",  // key or plain text
	Run: func(c *oriagent.Ctl, args string) string {
		// return text to log in the transcript, "" for nothing
		return c.T("cmd.export.done", args)
	},
}
```

`Ctl` methods: `Session()`, `Sessions()`, `Projects()`, `Commands()`, `WorkDir()`, `NewSession()`, `Clear()`, `RunTurn(prompt string, fn TurnFunc) error` (nil fn = configured agent; error if the session is busy), `Open(path) error`, `PickFiles()`, `Notify(msg)`, `Show(view)`, `Models()`, `Model()`, `SetModel(i)`, `Dark()`, `SetDark(bool)`, `Locale()`, `Locales()`, `SetLocale(l) error`, `LanguageName(l)`, `T(key, args...)`, `ToggleSidebar()`, `ToggleVoice()`, `Pomodoro()`.

Built-ins: `/help /new /clear /compact /model /open /usage /status /lang /sidebar /upload /voice /pomodoro /theme /config /profile`.

## Custom pages

```go
oriagent.Page{
	ID:    "stats",
	Title: "stats.title",
	Icon:  nil, // optional func(gtx layout.Context, c color.NRGBA) layout.Dimensions
	Layout: func(gtx layout.Context, th *ui.Theme, tr i18n.Translator) layout.Dimensions {
		return th.Section(gtx, tr.T("stats.title"), tr.T("stats.desc"), func(gtx layout.Context) layout.Dimensions {
			return th.Label(14, tr.T("stats.body"), th.Text, font.Normal).Layout(gtx)
		})
	},
}
```

The shell adds the page title and a centered scrolling column; return only the content. Keep widget state (`widget.Clickable`, `widget.Bool`, `widget.Editor`) in package-level or struct variables, never as locals inside `Layout` (they would reset every frame).

Useful `ui` helpers (import `github.com/sitle/ori-agent/ui`): `th.Section`, `th.Panel`, `th.Label`, `th.Mono`, `th.Body`, `th.Button(gtx, &btn, text, ui.Primary|ui.Secondary|ui.Danger)`, `th.Field(gtx, label, &editor, hint)`, `th.ToggleRow(gtx, &boolWidget, title, desc)`, `th.ChoiceCard`, `th.Chip`, `th.Avatar`, `th.HLine`, `ui.Card`, `ui.Bordered`, `ui.HSpace`, `ui.VSpace`, `ui.Expand`, `ui.OneLine`, `ui.Flow`, icons `ui.Chevron`, `ui.Close`, `ui.Gear`. Handle clicks with `btn.Clicked(gtx)` at the start of `Layout`.

## Demo data

```go
s := &oriagent.Session{Title: "Fix login", Messages: []*oriagent.Message{
	oriagent.UserMsg("The login button does nothing."),
	oriagent.ThinkingMsg("Probably an unbound handler.", 3),
	oriagent.TasksMsg(oriagent.Task{Text: "Read the code", Status: oriagent.TaskDone}),
	oriagent.ToolMsg("Read", "login.go", "   1  package main"),
	oriagent.TextMsg("Fixed."),
}}
s.Spawn("Run the tests")                       // subagent child
p := oriagent.NewProject("web", s)             // first session on top
// Message.Expanded, Session.Busy, Project.Open adjust the initial display.
```

## Translations

```go
Translations: i18n.Merge(sim.Translations, map[string]map[string]string{
	"fr": {"cmd.export.desc": "Exporter la conversation", "cmd.export.done": "Exporté vers %s."},
	"en": {"cmd.export.desc": "Export the conversation", "cmd.export.done": "Exported to %s."},
}),
```

Texts may contain `fmt` verbs filled by `T` arguments; keep the same verbs in every language. A new language needs a catalog with the keys of `oriagent.Catalogs["fr"]` plus `lang.name`. `oriagent.HumanSize(n, tr)` formats sizes per locale (`tr` can be a `*Turn` or `*Ctl`).

## Voice dictation

Implement `oriagent.Dictation` (`Start() error`, `Stop() (string, error)` - may block) for a real speech-to-text engine. Optional: `Level() float32` (`Leveler`) for the live meter, `SetLocale(string)` (`LocaleSetter`) to follow the UI language.

## Simulated agent

`github.com/sitle/ori-agent/sim`: `a := sim.New(); a.DocName = "AGENT.md"`; use `a` as `Config.Agent`, `a.Commands()` for `/review` and `/init`, and merge `sim.Translations`. Good for UI work without a model and as a structural reference (`sim/sim.go`).

## Common mistakes

- Writing to the transcript after `Run` returned (blocks are closed; start a new turn instead).
- Forgetting `Done()` on a block: the reasoning stays "Thinking…" until the turn ends.
- Blocking in `Command.Run` (freezes the UI): use `c.RunTurn`.
- Declaring `widget.Clickable` inside a page `Layout` function: clicks are lost.
- Writing `Page.Layout` as `func(gtx, th)`: the signature is `func(gtx layout.Context, th *ui.Theme, tr i18n.Translator) layout.Dimensions`.
- Hard-coding French or English strings in a multilingual app instead of keys.

## Verify your work

```bash
go vet ./...
go build ./...
go run .            # look at the window: send a message, try /help, switch /lang en
```

Linux system packages (Debian/Ubuntu): `gcc pkg-config libwayland-dev libx11-dev libx11-xcb-dev libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev libxcursor-dev libvulkan-dev`. macOS: Xcode Command Line Tools.

Runnable references in the repository: `examples/minimal` (smallest app), `examples/anna` (everything: demo data, page, commands, translations), `example_test.go` (compiled snippets).
