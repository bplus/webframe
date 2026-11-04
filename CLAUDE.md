# CLAUDE.md

Guidance for Claude Code and other contributors working in this repository.

## What this is

Webframe (`github.com/bplus/webframe`) is a Go component library for server-rendered web UIs built on templ, Tailwind CSS v4, HTMX and Alpine.js. Applications import it as a module; it ships no server of its own. The demo app (`cmd/demo` + `demo/`) exists only to develop and showcase the components.

## Commands

```bash
make build          # templ generate → tailwind compile → go build ./cmd/demo
make generate       # templ generate (*.templ → *_templ.go)
make css            # compile static/css/input.css → output.css (npm ci --prefix tools on first use)
make dev            # go run ./cmd/demo, serves on :3333
make check          # check-generate, check-vet, check-test, check-ui (run before every commit)
make check-ui       # go run ./cmd/wflint ./demo: app code may only compose components
make ui-check       # Playwright overflow/console check at 390 and 1280; needs `make dev`, not part of check
make graph-check    # Playwright: graphwidget renders a canvas on /graph and shows its error box on failure; needs `make dev`
npx --prefix tools playwright install chromium   # one-time, before the first ui-check
go test ./components/copybutton/...               # a single package
```

Build order matters: templ generate runs before CSS compilation and Go compilation.

## Architecture

- **Components** live in `components/<name>/`: `<name>.templ` (a `Props` struct plus the templ function), the generated `*_templ.go` (committed, never hand-edited), optional `helpers.go` and tests. Props carry `ID`, `Class` and `Attributes templ.Attributes` plus component fields; most components take variadic `...Props` with usable zero values.
- **utils/**: `TwMerge` (always use it to combine classes; later arguments win), `If`/`IfElse`, `MergeAttributes`, `EscapeJS` (values inside JS string literals), `SanitizeColor` (user-supplied CSS colors), `ScriptURL` (cache-busted script URLs; overridable in `init`).
- **listquery** (module root): the sort/filter/page query-string contract (`Parse`, `Spec`, `Spec.URL` and its `With*` helpers, `Apply[T]`). Renders nothing.
- **compose** (module root): `With(parent, children...)`, `Join`, `Text` fill container components (`{ children... }`) from plain Go, with no `.templ` file. `components/compose_children_test.go` must have a row for every package whose templates render `{ children... }`.
- **assets** embeds `static/`: `assets.Mount(mux)`, `assets.Handler()`, `assets.URL(path)`.
- **JS**: Alpine for local state (inline `x-data`), HTMX for server interactions. First-party scripts in `static/js/` (`confirm.js`, `htmx-errors.js`, `splitpane.js`, `autocomplete.js`, `theme-toggle.js`) are plain files; the `*.min.js` component scripts come from templUI and are not edited here.
- **CSS**: theme tokens in `static/css/input.css`. Light is the default; dark is an opt-in via `data-theme="dark"`, and `dark:` utilities follow that attribute, not the OS.

## The page layer

Apps are built by composing components, never by writing markup. Configure one `appshell.App`, wrap the mux in its middleware, and render pages through it:

```go
var App = &appshell.App{Name: "Demo", HomeHref: "/", Flash: flash.Take}
App.OnPanic = errorpage.PanicHandler                 // styled 500 for a recovered panic
App.Palette = &commandpalette.Props{Items: items}    // mod+k command palette on every page
http.ListenAndServe(addr, App.Middleware(errorpage.Handler(mux))) // mux 404/405 become error pages
err := App.Render(w, r, "Showcase", pages.Showcase()) // body only for a plain htmx request, full document otherwise
```

Each page type is one `Props` declaration and one `Respond(w, r, title, p)` call:

| Package | Page |
|---|---|
| `listpage` | header, filter bar, toolbar, `datatable`, `pagination`; driven by a `listquery.Spec` |
| `detailpage` | header, summary `descriptionlist`, tabs (inline or lazy), aside |
| `formpage` | sections of `formfield.Field` with `Values`/`Errors` by name and an error banner |
| `settingspage` | grouped fields with an always-visible save bar |
| `dashboardpage` | `stattile` tiles and sections on a 12-column `dashboardgrid` |
| `splitpane` | list beside detail from md up, one or the other on a phone; a failed list click keeps the detail and shows a toast (`Gone` for 404/410, `Failed` otherwise) |
| `authpage`, `errorpage` | sign-in / signed-out / forbidden cards; 403/404/405/500 pages |

Building blocks: `action`, `pageheader`, `breadcrumbs`, `layout` (`Stack`, `Inline`, `Section`, `TwoColumn`, ...), `panel`, `form.Form`, `formfield`, `toolbar`, `datatable`, `pagination`, `descriptionlist`, `stattile`, `banner`, `toast`, `flash`, `confirmdialog`, `skeleton`, `spinner`, `cardgrid`, `timeline`, `image`, `figure`, `toggle`, `autocomplete`, `commandpalette`. Content widgets: `prose` (the only sanctioned `templ.Raw`, for trusted HTML), `codeblock`, `logviewer`, `jsonviewer`, `diffview`. Form widgets, also reachable through `formfield.Field.Type`: `tagsinput`, `keyvalueeditor`, `rating`, `datetimepicker`, `filedrop`. Each package's doc comment is the reference for its props.

Behavior worth knowing before you change a handler or a page type:

- **Partial responses.** `Respond` serves only the named region when `HX-Target` names it (for example `listpage.Results`, id `ID+"-results"`), the page body for other htmx requests and the full document otherwise. Handlers acting from inside a list use `listquery.ParseCurrent` (the `HX-Current-URL` query) and `listpage.RespondResults`.
- **Errors swap.** `appshell.Document` configures htmx to swap 4xx/5xx responses like 2xx, so a failed form POST re-renders with status 422, and `errorpage` or `toast.Respond` answer other errors. `htmx-errors.js` refuses a non-HTML error body (a plain `http.Error` 500), showing the `appshell.LoadFailed` toast ("Could not load that", copied from a `<template data-wf-load-failed>` that `Document` renders), shows the same toast when a request gets no answer at all (`htmx:sendError` for a network error or a stopped server, `htmx:timeout`), replaces an earlier copy instead of stacking a second, and silently drops anything targeting an element marked `data-wf-swallow-errors`; a split pane list link's failures get `splitpane.js`'s own toast instead, so each failure shows one toast. When `errorpage.Respond` answers an htmx request whose target is the element that sent it (no `HX-Target`, as a button with `hx-get` or `hx-post` and no `hx-target` sends, or `HX-Target` equal to `HX-Trigger`): a GET gets the `appshell.LoadFailed` toast alone through `toast.Respond` (`HX-Reswap: none`, the error status), so the element and the page stay as they were; an action (any other method) gets `HX-Retarget: #wf-main` (the page's `<main>`, `appshell.MainID`, which `RenderBare` pages also have, unstyled) and `HX-Reswap: innerHTML show:window:top` through `App.RenderContentStatus`, so the error page replaces the page content instead of landing inside the button. A request aimed at a named region (a list's results, a split pane's detail, a lazy tab's panel) keeps its target. An app rewords the toast by assigning `appshell.LoadFailed`.
- **Feedback.** `toast.Send` for htmx responses (out of band into `#toast-container`); `flash.Set`/`flash.Success` before a redirect (rendered once in `#wf-flash`); `appshell.Redirect(w, r, url)` for that redirect (303, or `HX-Redirect` with an empty 200 for an htmx submit; never hand-write it); `action.Confirm` for a styled confirmation dialog handled by `confirm.js`.
- **Action checks.** `action.Validate(a)` (and `ValidateAll`) returns an error wrapping `action.ErrInvalid` for two shapes that render without complaint: `Submit` with an `Hx*` request field, and `Form` on a plain link. The page types' `Respond` (and `listpage.RespondResults`) call it on every action they declare and return the error before writing; the demo panics on it (`demo/render_failed.go`).
- **Action variants.** `action.VariantDefault` (`""`) is the zero value and means "not chosen": each component picks its own default (a page Submit and a list's Create primary, an auth Provider outline, a settings Cancel, a table row action and a card footer action ghost, anything else secondary). `action.VariantSecondary` is `"secondary"` and always renders secondary. Before v0.3.0 `VariantSecondary` was the zero value, so a Submit or Provider set to it rendered its default instead.
- **Field validation.** `Field.Validate` is a URL; the handler's whole body is `formfield.RespondFeedback(w, r, field, msg)`, status 200 either way.
- **Request id.** `appshell.Middleware` stores a request id (`appshell.RequestID(ctx)`: the request's `X-Request-Id` when it is at most 64 bytes of `[A-Za-z0-9._-]`, else generated) and echoes the id it used in the response's `X-Request-Id` header.
- **Panics.** `appshell.Middleware` recovers, logs with the request id and answers through `App.OnPanic` when nothing has been written yet; the default is a bare 500 (`appshell.PanicPage(requestID, homeHref)`) with a plain link to `App.HomeHref`.
- **No JavaScript.** `appshell.Document` (so every page, `RenderBare` included) opens `<body>` with a warning `banner` inside `<noscript>`: pages, links and most forms work; in-place buttons such as Delete, filters, tabs, menus, drop-down pickers, search and live updates need JavaScript. The wording is fixed and general, so it is true for any app.

## wflint and the rule for app code

App code never contains anything below the component boundary: no `div`, no `class`, no layout decisions. `internal/wflint` (run via `cmd/wflint`, part of `make check`) enforces this on `.templ`, Go and CSS: it rejects raw block HTML, `class`/`style` attributes, `Class:` in Props literals, `templ.Raw`, HTML markup in Go strings, and Alpine class/style bindings. Only text-level elements (`p, a, span, strong, em, b, i, code, kbd, br, ul, ol, li, time, small, sup, sub`) and component calls are allowed; images go through `image.Image`. The one exemption is a `custompage.CustomPage` or `customblock.CustomBlock` fence with a written reason, and every run prints every fence. A missing component is never a reason to use a fence: add the component to webframe instead. The demo passes with zero fences.

Known gaps: `p.Class = ...` and `attrs["class"] = ...` assignments are not caught; Go expressions in `.templ` control-flow conditions are not scanned; a fence `Reason` must be a double-quoted literal; `*_test.go` files are exempt from the HTML-in-Go-strings rule.

## CSS sources

`input.css` imports Tailwind with `source(none)` and scans only `components/**/*.templ`, `components/**/*.go` (minus tests and the icon tables) and `demo/**/*.templ`, so markdown and JS never change `output.css`. Classes written only by first-party JS, and utilities that consumer pages written before `appshell` still use directly, are listed in `@source inline(...)` lines. Class-less text inside `.wf-content` gets element typography from `input.css`.

## Conventions

- Commit `*_templ.go` together with the `.templ` change; reviewers read the generated HTML.
- Keep Props changes additive where possible.
- Tests are render-focused: render to a `bytes.Buffer` and assert on the HTML.
- `components/render_smoke_test.go` has one row per `components/` directory; a new package without a row fails the test.
- Chart components (barchart, piechart, areachart, sparkline, metricring) are SVG, take `float64` values and sanitize colors through `SanitizeColor`. barchart and areachart scale down to their container (`max-width:100%`); areachart's default series colors are theme tokens. barchart keeps bar order (`MaxBars` folds only the smallest into "Other") and `ShowValues` reserves room for the labels.
- Tone and color are never class names in Props: `progressbar.Props.Variant` (`VariantDefault`/`Success`/`Warning`/`Destructive`/`Info`) picks the fill; `statcomparison.ToneClass(dir, lowerIsBetter)` is the delta color rule shared by `statcomparison.Props.LowerIsBetter` and `stattile.Delta.LowerIsBetter` (the arrow always follows the direction).
- Doc comments follow Go convention: start with the identifier, say what it does and what is non-obvious, briefly.
