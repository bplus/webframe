.PHONY: build generate css dev clean check check-generate check-vet check-test check-ui ui-check

build: generate css
	go build -o bin/demo ./cmd/demo

generate:
	templ generate

css:
	npx @tailwindcss/cli -i static/css/input.css -o static/css/output.css

dev:
	go run ./cmd/demo

clean:
	rm -rf bin/

check: check-generate check-vet check-test check-ui

check-generate:
	@command -v templ >/dev/null 2>&1 || { echo "error: templ not found on PATH; install with: go install github.com/a-h/templ/cmd/templ@latest"; exit 1; }
	@command -v node >/dev/null 2>&1 || { echo "error: node not found on PATH; install Node.js to run the Tailwind CLI"; exit 1; }
	templ generate
	$(MAKE) css
	git diff --exit-code -I '^// templ: version' -- '*_templ.go' static/css/output.css

check-vet:
	go vet ./...

check-test:
	go test ./...

check-ui:
	go run ./cmd/wflint ./demo

# Phone/desktop measurement with Playwright's Chromium. Not part of `check`:
# needs `make dev` running on :3333. One-time: `npx playwright install chromium`.
ui-check:
	node tools/ui-check.mjs --url http://localhost:3333/items --width 390 --shot tools/shots/items-390.png
	node tools/ui-check.mjs --url http://localhost:3333/ --width 390 --shot tools/shots/showcase-390.png
	node tools/ui-check.mjs --url http://localhost:3333/ --width 1280 --shot tools/shots/showcase-1280.png
