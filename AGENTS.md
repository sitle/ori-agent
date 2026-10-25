# AGENTS.md

Instructions for coding agents working **in this repository** (the ori-agent framework itself). To build an application that **uses** ori-agent, read [`skills/ori-agent/SKILL.md`](skills/ori-agent/SKILL.md) instead.

## Project

ori-agent (`github.com/sitle/ori-agent`) is a Go framework on [Gio](https://gioui.org) that provides a desktop application shell for agent clients, in the style of the Claude Code desktop "work" view, with the Ori design system colors. Licence EUPL-1.2. Go 1.26+.

## Commands

```bash
gofmt -l .          # must print nothing
go vet ./...
go test ./...       # unit tests + compiled examples (example_test.go)
go build ./examples/...
go run ./examples/anna          # full demo; flags: -lang en, -light, -view setup -step 3, -view profile, -attach "a,b"
go run ./examples/minimal       # smallest app
```

Always run `gofmt -l .`, `go vet ./...` and `go test ./...` before finishing. UI changes should also be checked by running `examples/anna`.

Linux needs Gio's system packages (`gcc pkg-config libwayland-dev libx11-dev libx11-xcb-dev libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev libxcursor-dev libvulkan-dev`).

## Layout

| Path | Content |
|---|---|
| `oriagent.go` | `Config`, `Run`, public types (`Model`, `User`, `Page`, `Features`) |
| `app.go` | `App` state, frame loop, view routing, locale switching |
| `agent.go` | `Agent`, `Turn` API, `startTurn` (goroutine, cancellation, cleanup) |
| `model.go` | `Project`, `Session`, `Message`, constructors |
| `chat.go`, `sidebar.go`, `titlebar.go`, `pages.go` | UI areas |
| `commands.go`, `ctl.go` | slash commands, palette, `Ctl` |
| `voice.go`, `pomodoro.go`, `upload.go`, `previewpane.go` | features |
| `locales.go` | **all** built-in UI strings, `fr` and `en` |
| `ui/` | widget kit (must not import the root package) |
| `theme/` | palettes; `i18n/` translations; `preview/` document preview; `sim/` simulated agent |
| `docs/` | human documentation, in French |
| `examples/` | `minimal`, `anna` |

## Architecture rules

- **Single lock**: all app state is guarded by `App.mu`. The frame handler holds it; background work must go through `App.update(func())`. `Turn` methods wrap `update`. `Command.Run` runs inside the frame (lock held).
- **Immediate mode**: the screen is rebuilt every frame from state. Widget state (`widget.Clickable`, …) lives in structs, never in locals of layout functions.
- **Dependencies**: `ui`, `theme`, `i18n`, `preview` never import the root package. `sim` uses only the public API.
- **No hard-coded UI text**: every user-visible string goes through `a.t("key")` (or `c.T`, `t.T`, `tr.T`) with entries in **both** `catalogFR` and `catalogEN` in `locales.go` (and `sim/locales.go` for the simulated agent). Tests fail if keys or `fmt` verbs differ between languages.
- **No hard-coded colors**: widgets read the active `theme.Palette` through `ui.Theme`.
- **Public API changes**: update `CHANGELOG.md` (section "Non publié"), the docs in `docs/`, `skills/ori-agent/SKILL.md` and, when relevant, `example_test.go`.

## Writing conventions

- Code comments in English, matching the existing density.
- Human documentation (`README.md`, `docs/`, `CHANGELOG.md`, `CONTRIBUTING.md`) is in French; agent-facing files (`AGENTS.md`, `llms.txt`, `skills/`) are in English.
- French text must have correct accents (é, è, à, ç…), including UI strings and test data.
- Use a plain hyphen `-`, never long dashes (em dash U+2014, en dash U+2013), in code, strings and docs.
- Do not add AI attribution trailers (such as `Co-Authored-By`) to commits.
