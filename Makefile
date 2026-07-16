VERSION ?= dev
LDFLAGS := -s -w -X github.com/aloisdeniel/moth/internal/version.Version=$(VERSION)

.PHONY: build test lint proto run cross clean web dev dev-server dev-web

build:
	go build -ldflags "$(LDFLAGS)" -o bin/moth ./cmd/moth

test:
	go test ./...

lint:
	golangci-lint run
	buf lint

# Regenerates gen/ (Go) and web/admin/src/gen (TypeScript); commit both.
# Needs web/admin/node_modules (run `npm ci` in web/admin once).
proto:
	buf generate

# Rebuilds the embedded admin SPA into internal/server/web/dist; commit the
# result — `make build` embeds whatever is there.
web:
	cd web/admin && npm ci && npm run build

run: build
	./bin/moth serve

# Go server on :8080 + Vite dev server on :5173 (open http://localhost:5173/admin/).
# Frontend edits hot-reload; RPCs are proxied to the Go server.
dev:
	@$(MAKE) -j2 dev-server dev-web

dev-server:
	go run ./cmd/moth serve

dev-web:
	cd web/admin && npm run dev

# Verifies the four release targets cross-compile (CGO-free thanks to
# modernc.org/sqlite).
cross:
	GOOS=linux   GOARCH=amd64 go build -o /dev/null ./cmd/moth
	GOOS=linux   GOARCH=arm64 go build -o /dev/null ./cmd/moth
	GOOS=darwin  GOARCH=arm64 go build -o /dev/null ./cmd/moth
	GOOS=windows GOARCH=amd64 go build -o /dev/null ./cmd/moth

clean:
	rm -rf bin
