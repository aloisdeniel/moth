VERSION ?= dev
LDFLAGS := -s -w -X github.com/aloisdeniel/moth/internal/version.Version=$(VERSION)

.PHONY: build test lint proto proto-dart run cross clean web dev dev-server dev-web sdk-test sdk-e2e sdk-goldens preview-goldens

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

# Regenerates the Dart stubs for the Flutter SDK (sdk/flutter/lib/src/gen);
# commit the result. Separate from `make proto` because CI's proto job has no
# Dart toolchain. Needs protoc-gen-dart (`dart pub global activate
# protoc_plugin`) and the Dart SDK.
proto-dart:
	PATH="$(HOME)/.pub-cache/bin:$(PATH)" buf generate --template buf.gen.dart.yaml
	cd sdk/flutter && dart format lib/src/gen >/dev/null

# Analyzes and tests the Flutter SDK and its example app.
sdk-test:
	cd sdk/flutter && flutter analyze && flutter test
	cd sdk/flutter/example && flutter analyze && flutter test

# End-to-end SDK test against a freshly built moth binary: spawns bin/moth,
# creates an admin + project over the connect endpoints, then drives
# MothClient through signup → sign-in → transparent refresh → sign-out.
sdk-e2e: build
	cd sdk/flutter && flutter test --run-skipped --tags integration test/integration

# Golden tests for the themed login screen (3 reference themes × light/
# dark). Rasterization is platform-dependent, so they are excluded from the
# default `flutter test` run and CI — run locally on the machine flavor
# that generated the committed images; `make sdk-goldens UPDATE=1`
# regenerates them.
sdk-goldens:
	cd sdk/flutter && flutter test --run-skipped --tags golden $(if $(UPDATE),--update-goldens) test/golden

# Preview honesty (plan/06): captures the admin live preview for the same
# three reference themes as the Flutter golden suite (light/dark) into
# web/admin/e2e/preview/, for a side-by-side review against
# sdk/flutter/test/golden/goldens whenever either rendering changes. Runs
# the whole Playwright suite (the capture scenario needs the setup flow).
preview-goldens: build
	cd web/admin && npx playwright test
	@echo "Compare web/admin/e2e/preview/ against sdk/flutter/test/golden/goldens/"

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
