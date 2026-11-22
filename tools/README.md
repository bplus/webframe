# tools/ui-check

`ui-check.mjs` loads one page of the demo app in Playwright's own Chromium at a
device-style viewport, measures it, and exits non-zero when something is wrong.
It exists because phone width (390px) is a first-class target, also on a machine
with no desktop Chrome; Playwright's Chromium gives a true 390px viewport
(`isMobile`, `hasTouch`, `deviceScaleFactor: 2`) with no 500px floor.

## One-time setup

```bash
npm ci --prefix tools                       # from the repo root: Tailwind CLI + pinned playwright (tools/package.json)
npx --prefix tools playwright install chromium   # ~170 MB into ~/Library/Caches/ms-playwright
```

## Running

Start the demo first (`make dev`, serves on :3333), then:

```bash
make ui-check                     # /items at 390, / at 390 and 1280
node tools/ui-check.mjs --url http://localhost:3333/items --width 390 --shot tools/shots/items-390.png
node tools/ui-check.mjs --url http://localhost:3333/items --width 390 \
  --eval 'document.querySelectorAll("table th").length'
node tools/ui-check.mjs --url http://localhost:3333/items --width 390 \
  --click '[data-row-menu]' --shot tools/shots/items-390-menu.png
```

Arguments:

| flag | meaning |
|------|---------|
| `--url <u>` | page to load (required) |
| `--width <n>` | viewport width in CSS px (default 1280) |
| `--height <n>` | viewport height (default 844 when width <= 500, else 800) |
| `--shot <path>` | full-page PNG; directories are created. `tools/shots/` is git-ignored |
| `--eval <js>` | expression evaluated in the page after load + networkidle; result printed as JSON on its own line |
| `--click <selector>` | click before eval/shot, then wait 300ms |
| `--hover <selector>` | hover before eval/shot, then wait 300ms |
| `--wait <ms>` | extra wait after networkidle |
| `--ignore-console <substring>` | repeatable; a console error or page error whose text (including the resource URL) contains the substring is printed with an `(ignored)` marker and does not fail the run |

Light color scheme always. No requests leave the page other than what the page itself makes.

## What it checks

The last line of stdout is always one JSON object:

```json
{"url":"http://localhost:3333/","width":390,"height":844,"scrollWidth":390,"horizontalOverflow":false}
```

`horizontalOverflow` is `document.documentElement.scrollWidth > width`.

Exit code 1 (with the reason on stderr) when any of:

- the page fails to load (network error, no response, or HTTP status >= 400)
- `horizontalOverflow` is true
- a `console.error` or uncaught page error occurred (each one is printed with its
  resource URL), unless it matches an `--ignore-console` substring

The showcase deliberately points one avatar at `/static/img/missing-avatar.png` to
show the fallback, so `make ui-check` passes `--ignore-console missing-avatar.png`
for the showcase runs only. Use the flag for errors a demo page produces on purpose,
never to quiet a real one.

Exit code 2 for a usage error. `make ui-check` is not part of `make check`
because it needs the dev server running.

## graph-check

`graph-check.mjs` loads a page holding graphwidgets three times. First as is:
every widget must contain a `<canvas>` of non-zero size, be marked
`data-tui-graph-ready`, show no error box, and the page must log no console
errors. Then with `3d-force-graph.min.js` blocked, and then with the graph JSON
answering HTTP 500: each widget must show its error box and a console error
must be logged. Exit 1 on any failure, 2 for a usage error.

```bash
make graph-check   # needs `make dev`; checks /graph against /api/graph
node tools/graph-check.mjs --url http://localhost:3333/graph --data /api/graph
```

## palette-check

`palette-check.mjs` loads a page whose shell shows the command palette
trigger. It fakes the platform three times (Mac, Windows, Linux): the
trigger's hint must read `⌘K` on the Mac and `Ctrl+K` elsewhere, and that
chord must open the palette. Then, at 390px, it replaces
`window.visualViewport` with a stand-in it can resize (headless Chromium has
no on-screen keyboard) and checks that the open sheet's height cap and top
follow it: the whole screen with no keyboard, the visible part and its
offset with one. Exit 1 on any failure, 2 for a usage error.

```bash
make palette-check   # needs `make dev`; checks /items
node tools/palette-check.mjs --url http://localhost:3333/items --shot tools/shots/palette
```

## containers-check

`containers-check.mjs` opens the showcase's long-list dialog and its drawer
with a pinned line at 390x844, 1280x800 and 390x340 (a phone on its side).
In the dialog the panel must fit the screen, the `Body` must scroll, and the
count line and the footer must not move while it does. In the drawer the
`Pinned` line must not move while the `Body` scrolls. On the short screen the
dialog's `Body` must keep its minimum height and the panel must scroll
instead, so that the footer can still be reached. Exit 1 on any failure, 2
for a usage error.

```bash
make containers-check   # needs `make dev`; checks /
node tools/containers-check.mjs --url http://localhost:3333/ --shot tools/shots/containers
```

## splitpane-check

`splitpane-check.mjs` loads the demo's `/browse` split pane at 1280px. It opens
one item, deletes another behind the page's back (`DELETE /items/{id}`) and
clicks it: the "That item is no longer here" toast must show, a second click
must not stack a second toast, and the detail region and the address bar must
still show the first item. A click answered 500 and a click that gets no answer
must show the "Could not load that item" toast the same way, and it must be
the only toast on the page (`htmx-errors.js` leaves list links to
`splitpane.js`). At 390px the click
on the deleted item is a plain page load and must show the 404 page. It deletes
one item from the running demo (restart the demo to get it back). Exit 1 on any
failure, 2 for a usage error.

```bash
make splitpane-check   # needs `make dev`; checks /browse
node tools/splitpane-check.mjs --url http://localhost:3333/browse --shot tools/shots/splitpane
```

## fetch-errors-check

`fetch-errors-check.mjs` loads the demo's showcase at 1280px and at 390px and
clicks "Refresh stats", a button with `hx-get` and no `hx-target` that the
server answers with `errorpage.NotFound`: the "Could not load that" toast must
show, the button must keep its label and the page its content. It then clicks
"Request a plain 500" (a text/plain error that `htmx-errors.js` refuses): the
same toast must show, once even after a second click, and the sentence beside
the button must not change. Then it clicks "Refresh stats" while Playwright
aborts the request (no answer, as from a stopped server: htmx fires
`htmx:sendError`) and again while the request hangs past `htmx.config.timeout`
(`htmx:timeout`): each time the same toast must show, and only one. It then
refuses "Delete account" on `/settings` twice (a 409 answered by
`toast.Respond`) and clicks the showcase's Error toast button twice: each pair
must leave one toast, and the Warning button must then add a second. Last, on
the sign-in page it adds an htmx POST
button with no `hx-target` aimed at an address the demo does not serve: the
404 page must replace the page's `<main id="wf-main">`. Exit 1 on any failure,
2 for a usage error.

```bash
make fetch-errors-check   # needs `make dev`
node tools/fetch-errors-check.mjs --url http://localhost:3333 --shot tools/shots/fetch-errors
```

## filterbar-check

`filterbar-check.mjs` drives the filter bar of the demo's `/items` list at
1280px and counts the htmx requests the page sends. It types in Search, waits
for the answer and clicks elsewhere: the search box's `change` event must
send no second request. It types, waits and clicks a toggle pill: exactly one
more request must go out, with the search text and the pill. It types and
presses Enter before the debounce time is over: one request must go out at
once and none when the timer would have fired. Then it holds answers back
(route interception): with the search request held, a pill click must go out
at once, and after every answer has landed the results must be the rows a
fresh load of the address bar's URL shows; the same with a held request
sent after the first answer. No toast may show. On the list's single-choice
Pack group a click must send one request with that value alone, a click on
another button that one alone, and a click on the selected button or on
"Clear filters" no value. On the showcase at 390px the group of 16 School
buttons must stay on one line with no horizontal overflow of the page, mark
its cut-off edge (`data-tui-filterbar-more`), and bring the selected button
into view after a click and after a resize. The Pack counts must follow the
search (each answer renders the group again out of band) and equal those of
a fresh load, while the bar and the search box stay the same elements and
the box keeps the focus, its text and its caret as a held answer lands;
"Clear filters" must clear the group and bring the first counts back. A
select that has the focus must not be replaced by an answer, one without the
focus must be, and a control whose incoming copy is the same (after a sort
from the table header) must stay the same element; a clicked choice button must have the focus again after its
group was replaced; a scrolled group must keep its scroll position. Last,
the showcase's bar must be bare (no padding, border or background) and its
search box must fill its line beside its label at 1280px and at 390px, 192px
wide at least and as high as before (2rem). The long line must have a thin
scrollbar where there is a mouse and none on a touch screen, and Tab must
reach all 17 buttons with each clear of the faded edges. With the script
loaded a second time, one click on a choice button must still send one
request and select it. The bar aborts a superseded request and htmx logs two
console errors for an abort (`htmx:afterRequest`, `htmx:sendAbort`): exactly
those two are expected, the held scenario checks that they appear, and any
other console error in any scenario fails the run. Exit 1 on any failure, 2 for a usage
error.

```bash
make filterbar-check   # needs `make dev`
node tools/filterbar-check.mjs --url http://localhost:3333
```

## htmx-errors-check

`htmx-errors-check.mjs` runs `static/js/htmx-errors.js` in a bare Node
context (no browser, no npm packages) and hands its `htmx:beforeSwap`
listener a list of error answers: an XML body, Go's `<nil>` as text/plain,
an HTML fragment Go sniffed as text/plain, a full document, a JSON error and
more. Each must be swapped or refused as the script's header comment says,
and each refused answer must put one "Could not load that" toast in
`#toast-container` (none for a `data-wf-swallow-errors` target).
`go test ./components/appshell/` runs it when `node` is on PATH.

```bash
node tools/htmx-errors-check.mjs
```
