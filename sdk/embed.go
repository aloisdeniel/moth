// Package sdk embeds the moth_auth Flutter package sources so the server
// can serve them as a pub hosted repository under /pub. Only the files a
// published package tarball ships are embedded — no tests, no example app,
// no build artifacts.
package sdk

import "embed"

// FS holds the publishable subset of sdk/flutter.
//
//go:embed flutter/lib flutter/pubspec.yaml flutter/analysis_options.yaml
//go:embed flutter/README.md flutter/CHANGELOG.md flutter/LICENSE
var FS embed.FS
