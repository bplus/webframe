# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Webframe is a Go module (`github.com/bplus/webframe`) — a component library for server-rendered web UIs. It's consumed as a dependency by other Go projects. It is **not** a standalone application; the demo app exists only for development/showcase purposes.

## Commands

```bash
make build          # templ generate → tailwind compile → go build ./cmd/demo
make generate       # templ generate (produces *_templ.go from *.templ)
make css            # npx @tailwindcss/cli -i static/css/input.css -o static/css/output.css
make dev            # go run ./cmd/demo (serves on :3333)
make check          # check-generate → check-vet → check-test
make check-generate # templ generate → css rebuild → drift check on *_templ.go and output.css
make check-vet      # go vet ./...
make check-test     # go test ./...
go test ./...       # run all tests
go test ./components/copybutton/...  # single package
```

Build order matters: templ generate must run before CSS compilation or Go compilation.

## Architecture

**Component pattern**: each component lives in `components/<name>/` containing:
- `<name>.templ` — template with a `Props` struct and templ component function
- `*_templ.go` — generated Go (committed to git, do not hand-edit)
- Optional: `helpers.go`, `*_test.go`

**Props convention**: components accept a Props struct with `ID`, `Class`, `Attributes templ.Attributes` fields plus component-specific fields. Variadic `...Props` with zero-value defaults is common.

**Key utilities** (`utils/`):
- `TwMerge(classes...)` — always use this to combine Tailwind classes, never concatenate
- `EscapeJS(string)` — use for values injected into JS string literals
- `SanitizeColor(string)` — use for user-provided CSS colors
- `If[T]` / `IfElse[T]` — conditional helpers for templates
- `ScriptURL(path)` — appends cache-busting `?v=` timestamp

**JS integration**: Alpine.js for client-side reactivity (inline `x-data`/`x-on`), HTMX for server-driven interactions (`hx-*` attributes). A global `htmx:afterSwap` listener reinitializes Alpine on dynamic content.

**CSS**: Tailwind v4 with custom theme tokens in `static/css/input.css`. Light (Observatory) is the default palette; dark is an opt-in stored preference.

**Demo app** (`cmd/demo/` + `demo/`): HTTP server on :3333 with showcase gallery and traffic stream page. Routes include HTMX interaction demos (click counter, search, validation, toasts).

## Conventions

- Always commit `*_templ.go` when `.templ` files change — reviewers need to see generated HTML
- Keep components additive; don't break existing Props contracts
- Tests are render-focused: render to `bytes.Buffer`, assert on HTML string content
- Chart components (barchart, piechart, areachart, sparkline, metricring) are SVG-based, accept `float64` values, and sanitize color inputs
