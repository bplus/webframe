FROM golang:1.24-alpine

RUN go install github.com/a-h/templ/cmd/templ@latest
RUN apk add --no-cache nodejs npm
# Tailwind CLI at the version pinned in tools/package-lock.json, installed
# outside /app because docker-compose bind-mounts the checkout over it.
COPY tools/package.json tools/package-lock.json /opt/webframe-tools/
RUN npm ci --prefix /opt/webframe-tools --omit=dev --no-audit --no-fund
ENV TAILWIND=/opt/webframe-tools/node_modules/.bin/tailwindcss

WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .

EXPOSE 3333
CMD ["go", "run", "./cmd/demo"]
