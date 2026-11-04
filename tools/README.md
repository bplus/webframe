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
