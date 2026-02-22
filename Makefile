.PHONY: build generate css dev clean

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
