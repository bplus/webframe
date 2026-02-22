FROM golang:1.24-alpine

RUN go install github.com/a-h/templ/cmd/templ@latest
RUN apk add --no-cache nodejs npm
RUN npm install -g @tailwindcss/cli

WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .

EXPOSE 3333
CMD ["go", "run", "./cmd/demo"]
