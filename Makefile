VERSION ?= dev
LDFLAGS := -s -w -X github.com/aloisdeniel/moth/internal/version.Version=$(VERSION)

.PHONY: build test lint proto proto-dart proto-react run cross clean web web-demo dev dev-server dev-web sdk-test sdk-e2e sdk-goldens preview-goldens sdk-react sdk-react-test sdk-react-e2e website website-screenshots website-check docs-embed docs-proto

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

# Regenerates the TypeScript stubs for the React SDK (sdk/react/src/gen);
# commit the result. Needs sdk/react/node_modules (run `npm ci` in sdk/react
# once).
proto-react:
	buf generate --template buf.gen.react.yaml

# Rebuilds the @moth/react package into sdk/react/dist; commit the result —
# `make build` embeds whatever is there (CI fails when it is stale).
sdk-react:
	cd sdk/react && npm ci && npm run build

# Typechecks and unit-tests the React SDK.
sdk-react-test:
	cd sdk/react && npm run typecheck && npm test

# End-to-end React SDK test against a freshly built moth binary: spawns
# bin/moth, a local Stripe API double and the example app (Vite), then
# drives signup → transparent refresh → sign-out plus the paywall →
# checkout → unlock billing loop in a real browser. Needs node_modules in
# sdk/react and sdk/react/example (`npm ci` once in each) and Playwright's
# chromium (`npx playwright install chromium` in sdk/react once).
sdk-react-e2e: build
	cd sdk/react && npm run e2e

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

# Rebuilds the website's in-browser admin demo (the SPA with a localStorage
# fake backend, see web/admin/src/demo/) into website/public/demo; commit the
# result — the pages deploy publishes whatever is there.
web-demo:
	cd web/admin && npm ci && npm run build:demo

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

# The public website (plan/09): a standalone static Astro+Starlight project
# under website/, separate from the embedded admin SPA. Needs its own
# `npm ci` in website/ once.
website:
	cd website && npm ci && npm run build

# Regenerates the landing-page screenshots from a seeded demo moth instance
# into website/public/screenshots/ (see website/scripts/screenshots.mjs).
# Needs bin/moth (built here) plus Playwright chromium in web/admin and
# sharp in website — run `npm ci` in both once. The script degrades to
# labeled placeholders when the binary/browser are unavailable; CI runs it
# with MOTH_SCREENSHOTS_STRICT=1 to fail instead. Deterministic (fixed seed,
# fixed viewport) and re-runnable. Screenshots are committed.
website-screenshots: build
	cd website && node scripts/screenshots.mjs

# Internal link/asset integrity check over the built site (website/dist).
# Run after `make website`. External links are verified by lychee in CI.
website-check:
	cd website && node scripts/check-links.mjs

# Regenerates the API reference (docs/api/reference.md) from the .proto
# sources with protoc-gen-doc via a buf remote plugin (needs network, no
# local install). Commit the result; never hand-edit it.
docs-proto:
	buf generate --template buf.gen.docs.yaml

# Re-syncs the binary's embedded /docs (internal/docs/content) from the
# public website content and the generated CLI reference. Commit the result —
# `make build` embeds whatever is there, and CI fails when it is stale.
docs-embed:
	node website/scripts/sync-embedded-docs.mjs

clean:
	rm -rf bin
