# Webframe

Webframe is a Go-first component library for building server-rendered web UIs with `templ`, Tailwind v4, HTMX, and Alpine.

## What it contains

- Reusable UI components under `components/`
- Demo app under `cmd/demo` and `demo/`
- Prebuilt static assets under `static/`
- Tailwind and `templ` build steps via `Makefile`

## Development

Requirements:

- Go
- `templ`
- Node/npm for Tailwind CSS

Common commands:

```bash
make build
make dev
make generate
make css
make check-ui
make ui-check   # Playwright measurement of /items and / at phone and desktop width; needs `make dev` running
```

The demo app runs on `http://localhost:3333`. `make ui-check` loads it in
Playwright's own Chromium (one-time `npx playwright install chromium`) at 390
and 1280 px, writes screenshots to `tools/shots/` (git-ignored) and fails on
horizontal overflow or console errors; see `tools/README.md`. It is not part
of `make check` because it needs the dev server.

Run `make check` before pushing. It composes four sub-targets, in order:

- `check-generate` — regenerates templ output and Tailwind's `static/css/output.css`,
  then fails on drift in either
- `check-vet` — `go vet ./...`
- `check-test` — `go test ./...`
- `check-ui` — `go run ./cmd/wflint ./demo`, the UI lint (see Page layer)

The Tailwind CLI version is pinned via `@tailwindcss/cli` in `package.json`/
`package-lock.json` (currently 4.3.3) so `npx @tailwindcss/cli` produces a
byte-stable `output.css` instead of floating to whatever is latest.

## Page layer

The page layer (`appshell`, `pageheader`, `breadcrumbs`, `layout`, `panel`, `action`, `formfield`, `autocomplete`, and the `custompage`/`customblock` fences) lets an app build whole pages by composing components. `assets` embeds `static/` for serving. `wflint` enforces the rule below; `make check-ui` runs it and `make check` includes it. `tabs` takes a `Sizing` (default `SizingTallest`, so switching never moves the page), `layout.Section` and `panel` take a `Level` (h2 to h4) so nesting reads as nesting, and `hovercard` takes a `Placement` (default bottom-start).

Batch 2 (lists and forms) adds the three page types `listpage`, `detailpage` and `formpage`, each with a `Respond(w, r, title, props)`; the pieces they compose, `datatable`, `pagination`, `toolbar`, `descriptionlist`, `stattile`, `banner`, `toggle`, `image`, `figure` and `form.Form` (the `<form>` element itself; a raw one is forbidden); `badge.Status(kind)` so apps never pick status colors; and `listquery` at the module root, the sort/filter/page query-string contract (`sort`, `dir`, `page`, `per`, `q`, filter keys) that table headers, filter bars and pagination emit and handlers parse. A list page is `listquery.Parse` → declare `listpage.Props` → `listpage.Respond`; sort, filter and page links swap only the results region over htmx and push the URL. A form page re-renders with `Values` and `Errors` by field name on a failed POST (422 since batch 3; the shell configures htmx to swap error responses) and redirects on success (`HX-Redirect` for an htmx submit); `formfield.Field.Validate` wires blur validation to a handler whose body is `formfield.RespondFeedback`. The demo's Items app (`/items`) is the acceptance scenario.

Batch 3 (feedback) adds the seven packages `toast`, `flash`, `confirmdialog`, `skeleton`, `spinner`, `errorpage` and `authpage`. A handler answers an htmx action with `toast.Send(w, r, toast.Error(title, body))` (appended out of band to the shell's `#toast-container`; nothing is written for a non-htmx request) and a redirect with `flash.Error(w, r, title, body)` (a one-shot `HttpOnly` cookie the shell renders once in `#wf-flash` on the next page and clears; wiring is `App.Flash = flash.Take`; a fragment render sends it out of band so an htmx action's flash shows at once). Errors first: `Success` constructors exist but are opt-in and discouraged. `action.Confirm` opens the styled confirm dialog (`confirmdialog.Host`, rendered once per page by `appshell.Document`) through `static/js/confirm.js`; without it the native `hx-confirm` box runs. `skeleton.Lines/Card/Table/Tile` hold the height of what replaces them; `spinner.Spinner` is the indeterminate ring (`Indicator` ties it to `htmx-indicator`). `errorpage.Respond`/`NotFound`/`Forbidden`/`Internal` and `authpage.Respond` work with or without an `appshell.App` in the context (`App.RenderBare` and the package-level `appshell.RenderBare` render the document without the shell). `slideover.SideBottom` is a bottom sheet with `Height` tiers; `action.Action` gains `Submit`, `Form`, `FullWidth`, `VariantOutline` and `action.MenuAttrs`; `datatable.StickyHeader` sticks (to a `MaxHeight` scroll box or to the page on md+); list pages get a phone Sort select so a `Hidden` sortable column can still be sorted at 390.

Batch 4 (pages and dashboards) adds the six packages `dashboardpage`, `settingspage`, `streampage` (page types with a `Respond(w, r, title, props)`) and `cardgrid`, `timeline`, `splitpane` (displays inside a page; `splitpane` also has a `Respond`). A dashboard is tiles and titled sections on one `dashboardgrid` (`ColSpan` on a 12-column grid; one column below md since batch 5, 900px before). A settings screen is groups of fields with the `formpage` round trip (422 re-render with `Values`/`Errors`, redirect on success) and an always-present sticky save bar; `formpage.ResolveField`, `ErrorBanner` and `Submit` are exported so both pages resolve fields, report errors and render the submit button the same way. A feed wraps a `trafficstream` panel and renders its script once. A catalog is a grid of cards (4→2→1 columns, `Badges` as `cardgrid.Badge{Label, Props}`), a history a `timeline` of events grouped by day. A mail-style browser is a `splitpane`: two routes build the same props, one with `Detail` nil and one with it set, list links spread `splitpane.LinkAttrs(p, href)`, and from md up a click swaps only the detail region and pushes the URL while on a phone the `splitpane.js` leaves the click alone and the plain `href` navigates to the detail view with a Back link; `Placeholder` fills the empty detail region on wide screens. `timeband` is not built (the spec says to prototype it with the maintainer first). Its deferred items went into batch 5.

Batch 5 (content widgets) adds the eleven packages `codeblock`, `logviewer`, `jsonviewer`, `diffview`, `prose` (display) and `tagsinput`, `keyvalueeditor`, `filedrop`, `datetimepicker`, `rating`, `commandpalette` (input), all inline Alpine or no script at all. `codeblock` is escaped monospace text with copy, CSS line numbers, wrap and a height cap (no highlighter); `logviewer` is a fixed-height `role="log"` box that polls `Src` (`hx-get` every N seconds, appended) and whose handler answers `logviewer.Fragment(newLines)` or 204 when nothing is new, following the bottom only while the user is there; `jsonviewer` renders any Go value as a native `<details>` tree; `diffview` renders `[]File` or `diffview.Parse(gitDiff)` as tables, inline below md and side by side from md up when asked. `prose` is the one route for content HTML: `prose.HTML` is the library's only sanctioned `templ.Raw` (the app owns sanitization), `prose.Text` is escaped paragraphs for untrusted text, both inside `.wf-prose` scoped typography. The form widgets submit plain fields: tags comma-joined in one field, key/value rows as `{Name}.key`/`{Name}.value` pairs that `keyvalueeditor.Parse(r.Form, name)` zips back, a rating as an integer, dates and times as the native control writes them (`datetimepicker.Parse`), files as a multipart part (`Multipart` on the form). `formfield.Field.Type` gains `TypeTags`, `TypeRating` and `TypeTime`, and `TypeDate`/`TypeDateTime`/`TypeFile` now render `datetimepicker` and `filedrop`, so `formpage` and `settingspage` declare them like any field. The command palette is `App.Palette = &commandpalette.Props{Items: …}`: the shell shows the trigger, `Document` renders the dialog, mod+k opens it, Alpine filters the server-rendered links. Also in the batch: `appshell.Middleware` recovers panics (`App.OnPanic`, `appshell.RequestID`), `errorpage.Handler(mux)` and `errorpage.PanicHandler`, the theme-following `dark:` variant, `htmx-errors.js` sniffing unlabeled HTML, and `dashboardgrid` collapsing at md. Deferred items are tracked as tickets, which replaced the batch 6 inputs document on 2026-11-01.

**Changed defaults in batch 5:** `dark:` utilities follow `data-theme` (an `@custom-variant dark` in `input.css`) instead of the OS preference, and the theme sets `color-scheme` (light on `:root`, dark under `[data-theme="dark"]`) so native controls and pickers draw in the theme's scheme; `appshell.Middleware` recovers panics, logs them with a request id (`X-Request-Id` or generated; not echoed on the response) and answers a bare 500 page with the id unless `App.OnPanic = errorpage.PanicHandler` gives the styled one; `errorpage.Handler(mux)` replaces the documented `ServeMux` wrapper snippet (it asks the mux for its own 404/405 verdict and copies `Allow`); `htmx-errors.js` swaps an error body that has no or a non-HTML `Content-Type` when the body starts with `<`; `dashboardgrid` collapses at md (was 900px); `Document` renders the palette when `App.Palette` is set and the shell shows its trigger; `formfield` `TypeDate`/`TypeDateTime` render `datetimepicker` (color-scheme classes on the input, `min`/`max` from `Field.MinValue`/`MaxValue`, `aria-invalid` instead of `HasError`, no placeholder) and `TypeFile` renders `filedrop` (a dashed `<label>` zone around an sr-only file input whose id is the field's control id plus `-input`, plus a hidden refusal message and file list; `Field.Accept`/`Multiple`/`Preview`); `tagsinput` at `MaxTags` is readonly ("Limit reached"), not disabled, so focus and the phone keyboard stay; `rating` shows a destructive ring when its fieldset is `aria-invalid`.

**Changed defaults in batch 4:** `static/js/htmx-errors.js` (first-party, in `appshell.Scripts` after `confirm.js`) refuses to swap an error response whose body is not `text/html` or whose target (or an ancestor) carries `data-wf-swallow-errors`, so a plain Go 500 never lands as text and the shell's nav badge and live dot never show an error body, while 422 forms, 409 toasts and 404 error pages still swap; dismissed toasts remove themselves from the DOM 300 ms after dismissal; `graphwidget` loads with the shared 32px `spinner` (was a 24px inline SVG); `authpage` keeps a Provider's own `Variant`; phone Sort option labels past 32 runes are cut with the full label in `title`; `errorpage` has 405 wording and `MethodNotAllowed` (no `Allow` header; the package doc has the `ServeMux` wrapper); controls in a form with a save bar get a 5rem bottom scroll margin; `theme-toggle.js` and `autocomplete.js` have re-run guards; `dashboardgrid` collapses under 900px, not md, so between 768 and 900 a dashboard is one column while `TwoColumn` and `cardgrid` are two (aligned to md in batch 5).

**Changed defaults in batch 3:** colors in `selectbox`, `dialog`, `slideover`, `button`, `badge`, `nodepill`, `trafficstream` and `lifecyclestepper` come from theme tokens (backdrops stay `bg-black/50`, a scrim darkens in both themes; destructive text `text-destructive-foreground`, nodepill dot hues from `info`/`success`/`warning`/`destructive`/`primary`/`muted-foreground`); slideover panels at z-index 60 over a 55 backdrop and full width below `sm`; toasts at the top at every width at z-index 70 (above an open sheet), `Timeout` 0 → 4 s, `Sticky` never; `listquery.Spec.URL` omits the default order; the datatable selection column renders at every width; `skeleton.Lines` at a 20px pitch; the document title is `Title` alone when `AppName` is empty; `viewport-fit=cover`; `statcomparison` and `stattile` share arrows; `action.Button` with both `Href` and `Submit` renders the submit button.

**Changed defaults in batch 2:** `layout.Inline` wraps below `md` by default; a bare `?sort=key` without `dir` parses ascending; `appshell` adds `HX-Target` to `Vary`; `statcomparison`, `copybutton` and `form.Message` colors come from the theme tokens `text-success`/`text-destructive`/`text-info`; `avatar.Image` sanitizes `src`; new tokens `--color-success`, `--color-warning`, `--color-info`. wflint no longer allows a raw `<img>` (use `image`) and now allows the `download` attribute; the `html-from-go-string` rule only fires on a quoted attribute value or a tag with markup around it.

**Changed defaults in batch 1b:** `tabs` are now as tall as their tallest panel by default (set `Sizing: tabs.SizingFit` for the old grow-and-shrink behavior), hover cards open below by default (set `Placement` to choose), `layout.Section` Level 2 headings are `text-2xl` and the page title is `text-3xl`. Consumers pinned to v0.1.0 (one app uses tabs and hovercard) will see these changes when they upgrade.

App code never contains anything below the component boundary (no div, no class, no layout decision); composition means choosing and ordering components. The only place that boundary may be crossed is inside a CustomPage or CustomBlock, which must carry a written reason and is printed by every build. A missing component is never a reason to use the fence: stop and file it against webframe.

```go
var App = &appshell.App{Name: "Demo", HomeHref: "/", Flash: flash.Take}
App.OnPanic = errorpage.PanicHandler              // styled 500 for a recovered panic
App.Palette = &commandpalette.Props{Items: items} // mod+k command palette on every page
http.ListenAndServe(addr, App.Middleware(errorpage.Handler(mux)))
// in a handler:
err := App.Render(w, r, "Showcase", pages.Showcase())
// after a failed htmx action / before a redirect:
_ = toast.Send(w, r, toast.Error("Could not delete", "Item 1 is referenced."))
flash.Error(w, r, "Could not save", "Try again.")
```

Consumers: several server-rendered Go apps.

## Conventions

- Keep components additive and avoid breaking existing component contracts
- Each component should live in `components/<name>/`
- Prefer a small `Props` struct and a `Script()` templ when JS is required
- Add focused render tests for new components
- Commit generated `*_templ.go` output when templ files change

## New component: `trafficstream`

`trafficstream` is intended for operator/admin surfaces that need to watch live request and response activity.

Features:

- SSE-backed live feed
- Client-side filtering by text, provider, event, and direction
- Pause/resume and clear controls
- Bounded in-browser row retention
- Works well for `/admin/traffic` style pages

Minimal usage:

```go
@trafficstream.TrafficStream(trafficstream.Props{
    StreamURL: "/api/traffic/stream",
    Height: 560,
    ProviderOptions: []string{"anthropic", "openai", "gemini"},
})
@trafficstream.Script()
```
