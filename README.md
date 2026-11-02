# Webframe

[![CI](https://github.com/bplus/webframe/actions/workflows/ci.yml/badge.svg)](https://github.com/bplus/webframe/actions/workflows/ci.yml)

> **WORKSHEET — not the real README yet.** Each section below tells the maintainer what to
> go read, what question to answer while reading, and then to replace the section
> with his own words. Delete every `TODO(maintainer)` block as you replace it. Work
> top to bottom: each section builds on the one before. When no `TODO(maintainer)`
> is left, this is the README.

---

## 1. What webframe is

> **TODO(maintainer): look before you read.**
> 1. Run the demo: `make dev`, then open http://localhost:3333 (the showcase) and
>    http://localhost:3333/items (a small complete app).
> 2. On `/items`: sort a column, filter, page, create an item with a bad value,
>    delete one. Shrink the browser to phone width and do it again.
> 3. Look at the list of folders in `components/` — that is the whole menu.
>
> **Answer in 3-5 sentences:** who is this for, and what is the one promise it
> makes? (Hint: in the `/items` app, the app author never wrote a single HTML
> tag or CSS class. Everything you clicked is webframe.)
>
> Replace this block with your pitch, plus one screenshot of `/items`
> (desktop) and one at phone width.

## 2. Pages are declarations

> **TODO(maintainer): read, in this order.**
> 1. `demo/items_handlers.go`, `HandleItemsList` — the list page you clicked.
>    Notice it builds a `listpage.Props` value and calls `listpage.Respond`.
>    No markup.
> 2. `components/listpage/listpage.go`, `Respond` (around line 496) — what
>    webframe does with that declaration: a full page on a normal request, only
>    the results region on an htmx request.
> 3. `components/formpage/formpage.go`, `Respond` (around line 233) — the same
>    idea for forms: a failed save re-renders with the values and errors and
>    answers 422; a good save redirects.
>
> **Answer:** what does the app author write, and what does webframe do for
> them? Write the shortest example you can (a list page or a form page) and ask
> Claude to check that it compiles.
>
> Replace this block with that explanation and example.

## Using webframe from templ or from plain Go

An app can be written in templ (`.templ` files run through `templ generate`)
or in plain Go with no `.templ` files at all. Both are supported. Every
component is a Go function that returns a `templ.Component`, and every page
type is a `Props` value passed to a `Respond` function, so most code looks
the same either way.

The one place the two differ is a container: a component such as `card.Card`
or `badge.Badge` that shows content you put inside it. In templ that content
is the block after the call. In plain Go it is passed with
`compose.With(parent, children...)`; `compose.Join` renders components one
after another and `compose.Text` renders a string as escaped text.

A card holding a badge that says "New", in templ:

```templ
templ Status() {
	@card.Card() {
		@badge.Badge() {
			New
		}
	}
}
```

The same in plain Go:

```go
func Status() templ.Component {
	return compose.With(card.Card(),
		compose.With(badge.Badge(), compose.Text("New")),
	)
}
```

Both render the same HTML, and `wflint` passes both.

What works in only one style:

- **Inline text markup is written directly only in templ.** `wflint` allows
  text-level elements (`<strong>`, `<em>`, `<a>`, `<code>`, lists, `<br>`,
  `<time>`) in a `.templ` file. Plain Go has no helper for them; use a
  component (for example `action` for a link) or `prose` with an HTML string.
- **Fences hold hand-written markup only in templ.** In a `.templ` file,
  `wflint` recognizes a `custompage.CustomPage` or `customblock.CustomBlock`
  block, lists it on every run, checks its `Reason` and allows any markup
  inside it. In Go, `compose.With(customblock.CustomBlock(...), ...)` renders,
  but `wflint` neither lists it nor checks its `Reason`, and markup inside it
  (`templ.Raw`, `templ.ComponentFunc`, HTML in a string) is still rejected.

Everything else works the same in both: page types and their `Respond`
functions, `Props` (including the `Attributes` map), `Props` fields that take
a `templ.Component`, and the `Script()` helpers, which are ordinary
components.

## 3. The address bar is the state (`listquery`)

> **TODO(maintainer): read.**
> 1. `listquery/listquery.go` — the package comment at the top, then
>    `ParseValues` and `ParseCurrent`. Note "nothing panics on any input" and
>    the page cap.
> 2. `listquery/url.go`, `Spec.URL` — how a link for "page 2, sorted by name"
>    is built back from the same value.
> 3. `listquery/apply.go`, `Apply` — sorting and paging an in-memory slice.
> 4. Skim `listquery/listquery_test.go` for what it attacks.
>
> **Answer:** why does every list keep its sort/filter/page in the URL instead
> of in the browser or a session? What does a user get from that (reload,
> bookmark, back button, sharing a link)?
>
> All six reviewers called this the best package in the repo. Replace this block
> with a short section on it.

## 4. When things go wrong (`appshell.Middleware`, `errorpage`)

> **TODO(maintainer): read.**
> 1. `components/appshell/app.go`, `Middleware` (around line 83). Follow what
>    happens when a handler panics: `recordingWriter` (around line 207) and
>    `stripRedirectHeaders` (around line 128).
> 2. `components/errorpage/errorpage.go`, `Handler` (around line 104): how a
>    plain "404 page not found" from Go's router becomes a styled page.
>
> **Answer:** what does a user see when the server crashes halfway through a
> page, and why is that harder to get right than it looks? (The reviewers'
> phrase: "most Go developers don't know these details.")
>
> Replace this block with 1-2 paragraphs, or fold it into section 2 if short.

## 5. Apps compose; a linter enforces it (`wflint`)

> **TODO(maintainer): read.**
> 1. `cmd/wflint/main.go` — the command; then `internal/wflint/rules.go` —
>    what it rejects (raw block HTML, `class`/`style`, `templ.Raw`) and how
>    every rejection names the component to use instead.
> 2. `components/custompage` and `components/customblock` — the one escape
>    hatch, which needs a written reason and is reported on every run.
> 3. Run it: `make check-ui`.
>
> **Answer:** why forbid app code from writing HTML at all? What goes wrong in
> a codebase without this rule? (You have lived this: it is the "cards look
> janky" problem.)
>
> Replace this block with the rule, the reason, and how to run the linter.

## 6. How this was built

> **TODO(maintainer): your words only.** Say how it was made: you designed and
> directed it; AI coding agents wrote most of the Go. Name the design calls that
> were yours (components only, no app markup, a linter to enforce it, phone width
> as a hard requirement, errors first). Short and plain is best. Delete this
> section if you would rather not say it here.

## 7. Components

> **TODO(maintainer):** decide whether to list the components here (a table: name,
> one line each) or point readers at the demo showcase. If a table, ask Claude
> to generate the first draft from `components/` and edit it.

---

## Development

Requirements:

- Go
- `templ`
- Node/npm for the Tailwind CSS CLI and Playwright (pinned in `tools/package.json`;
  `make css` runs `npm ci --prefix tools` on first use)

Common commands:

```bash
make build
make dev        # demo app on http://localhost:3333
make generate
make css
make check      # what CI runs: generated-file drift, go vet, go test, wflint
make ui-check   # Playwright measurement at phone and desktop width; needs `make dev` running
```

`make ui-check` loads the demo in Playwright's own Chromium (one-time
`npx --prefix tools playwright install chromium`) at 390 and 1280 px, writes
screenshots to `tools/shots/` (git-ignored) and fails on horizontal overflow or
console errors; see `tools/README.md`.

The Tailwind CLI version is pinned in `tools/package.json` /
`tools/package-lock.json` so `output.css` is byte-stable.

## License

MIT — see [LICENSE](LICENSE). Third-party code included in this repository
(templUI, htmx, Alpine.js, 3d-force-graph and others) is listed with its
licenses in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
