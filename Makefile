.PHONY: build generate css dev clean check

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

check:
	@command -v templ >/dev/null 2>&1 || { echo "error: templ not found on PATH; install with: go install github.com/a-h/templ/cmd/templ@latest"; exit 1; }
	templ generate
	git diff --exit-code -I '^// templ: version' -- '*_templ.go'
	go vet ./...
	go test ./...
