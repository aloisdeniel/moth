// Package sdk embeds the moth_auth Flutter package sources and the built
// @moth/react npm package so the server can serve them as a pub hosted
// repository under /pub and an npm registry under /npm. Only the files a
// published package tarball ships are embedded — no tests, no example app,
// no node_modules, no TypeScript sources.
package sdk

import "embed"

// FS holds the publishable subset of sdk/flutter and sdk/react.
//
//go:embed flutter/lib flutter/pubspec.yaml flutter/analysis_options.yaml
//go:embed flutter/README.md flutter/CHANGELOG.md flutter/LICENSE
//go:embed react/package.json react/dist
//go:embed react/README.md react/CHANGELOG.md react/LICENSE
var FS embed.FS
