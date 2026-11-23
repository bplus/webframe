.PHONY: build generate css dev clean check check-generate check-vet check-test check-ui ui-check containers-check graph-check palette-check splitpane-check fetch-errors-check filterbar-check

build: generate css
	go build -o bin/demo ./cmd/demo

generate:
	templ generate -include-version=false

# Node tooling (Tailwind CLI, Playwright) is pinned in tools/package.json.
# TAILWIND can be overridden from the environment.
TAILWIND ?= tools/node_modules/.bin/tailwindcss

tools/node_modules/.bin/tailwindcss: tools/package.json tools/package-lock.json
	npm ci --prefix tools --no-audit --no-fund
	touch $@

# input.css imports "tailwindcss", and the resolver looks for it in the
# node_modules of static/css and of each directory above it, never in tools/.
# NODE_PATH adds tools/node_modules to that search (issue 83).
css: $(TAILWIND)
	NODE_PATH="$(CURDIR)/tools/node_modules" $(TAILWIND) -i static/css/input.css -o static/css/output.css

dev:
	go run ./cmd/demo

clean:
	rm -rf bin/

check: check-generate check-vet check-test check-ui

check-generate:
	@command -v templ >/dev/null 2>&1 || { echo "error: templ not found on PATH; install with: go install github.com/a-h/templ/cmd/templ@latest"; exit 1; }
	@command -v node >/dev/null 2>&1 || { echo "error: node not found on PATH; install Node.js to run the Tailwind CLI"; exit 1; }
	templ generate -include-version=false
	$(MAKE) css
	git diff --exit-code -I '^// templ: version' -- '*_templ.go' static/css/output.css

check-vet:
	go vet ./...

check-test:
	go test ./...

check-ui:
	go run ./cmd/wflint ./demo ./examples

# Phone/desktop measurement with Playwright's Chromium. Not part of `check`:
# needs `make dev` running on :3333. One-time: `npx --prefix tools playwright install chromium`.
# The showcase deliberately 404s /static/img/missing-avatar.png (avatar fallback demo).
# /longtext holds 300-character unbroken tokens (pageheader, its status
# badge, banner; issues 84 and 85) and a banner Body of three lines; LONGTEXT_LINES is true when the three lines
# show as three.
LONGTEXT_LINES = (() => { const p = document.querySelector("\#longtext-lines p.text-sm"); return p.getBoundingClientRect().height >= 2.5 * parseFloat(getComputedStyle(p).lineHeight); })()
ui-check: tools/node_modules/.bin/tailwindcss
	node tools/ui-check.mjs --url http://localhost:3333/items --width 390 --shot tools/shots/items-390.png
	node tools/ui-check.mjs --url http://localhost:3333/ --width 390 --shot tools/shots/showcase-390.png --ignore-console missing-avatar.png
	node tools/ui-check.mjs --url http://localhost:3333/ --width 1280 --shot tools/shots/showcase-1280.png --ignore-console missing-avatar.png
	node tools/ui-check.mjs --url http://localhost:3333/longtext --width 390 --shot tools/shots/longtext-390.png --assert '$(LONGTEXT_LINES)'
	node tools/ui-check.mjs --url http://localhost:3333/longtext --width 1280 --shot tools/shots/longtext-1280.png --assert '$(LONGTEXT_LINES)'

# dialog and slideover in a real browser: the showcase's long-list dialog and
# its drawer with a pinned line, at 390x844, 1280x800 and 390x340 (a phone on
# its side). The Body scrolls while the lines around it stay in view, and on
# the short screen the Body keeps its minimum height. Needs `make dev`
# running, like ui-check.
containers-check: tools/node_modules/.bin/tailwindcss
	node tools/containers-check.mjs --url http://localhost:3333/

# graphwidget in a real browser: a non-empty <canvas> in every widget on
# /graph with no console errors, and a visible error box when the library or
# the graph JSON is missing. Needs `make dev` running, like ui-check.
graph-check: tools/node_modules/.bin/tailwindcss
	node tools/graph-check.mjs --url http://localhost:3333/graph --data /api/graph

# commandpalette in a real browser: the trigger's shortcut hint per platform
# (⌘K on a Mac, Ctrl+K elsewhere) and the phone sheet following
# window.visualViewport. Needs `make dev` running, like ui-check.
palette-check: tools/node_modules/.bin/tailwindcss
	node tools/palette-check.mjs --url http://localhost:3333/items

# splitpane in a real browser: a list click on a deleted item (404), on an
# item answering 500 and with no answer shows a toast and leaves the detail
# and the URL alone; on a phone the click shows the 404 page. Deletes one
# item from the running demo. Needs `make dev` running, like ui-check.
splitpane-check: tools/node_modules/.bin/tailwindcss
	node tools/splitpane-check.mjs --url http://localhost:3333/browse

# Failed htmx requests in a real browser: a self-targeting GET answered 404
# shows the "Could not load that" toast and leaves the button alone, a
# plain-text 500 shows the same toast, and an action error on the sign-in
# page lands in its <main id="wf-main">. Needs `make dev` running, like ui-check.
fetch-errors-check: tools/node_modules/.bin/tailwindcss
	node tools/fetch-errors-check.mjs --url http://localhost:3333

# filterbar in a real browser: the search box's change event sends no second
# request, a pill click after a search sends one request with both values,
# Enter sends at once, and with answers held back only the answer to the
# newest request lands; a single-choice group sends one value, and 16
# buttons at 390 stay on one line with the selected one in view; counts
# follow the search while the search box keeps the focus. Needs `make dev`
# running, like ui-check.
filterbar-check: tools/node_modules/.bin/tailwindcss
	node tools/filterbar-check.mjs --url http://localhost:3333
