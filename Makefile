VERSION ?= dev
LDFLAGS := -s -w -X github.com/aloisdeniel/moth/internal/version.Version=$(VERSION)

.PHONY: build test lint proto run cross clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/moth ./cmd/moth

test:
	go test ./...

lint:
	golangci-lint run
	buf lint

proto:
	buf generate

run: build
	./bin/moth serve

# Verifies the four release targets cross-compile (CGO-free thanks to
# modernc.org/sqlite).
cross:
	GOOS=linux   GOARCH=amd64 go build -o /dev/null ./cmd/moth
	GOOS=linux   GOARCH=arm64 go build -o /dev/null ./cmd/moth
	GOOS=darwin  GOARCH=arm64 go build -o /dev/null ./cmd/moth
	GOOS=windows GOARCH=amd64 go build -o /dev/null ./cmd/moth

clean:
	rm -rf bin
