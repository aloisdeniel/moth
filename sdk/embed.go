// Package sdk embeds the moth_auth, moth_billing and moth_push Flutter
// package sources and the built @moth/react npm package so the server can
// serve them as a pub hosted repository under /pub and an npm registry under
// /npm. Only the files a published package tarball ships are embedded — no
// tests, no example app, no node_modules, no TypeScript sources, and no
// pubspec_overrides.yaml (the local-dev path override must never be served).
package sdk

import "embed"

// FS holds the publishable subset of sdk/flutter, sdk/flutter_billing,
// sdk/flutter_push and sdk/react.
//
//go:embed flutter/lib flutter/pubspec.yaml flutter/analysis_options.yaml
//go:embed flutter/README.md flutter/CHANGELOG.md flutter/LICENSE
//go:embed flutter_billing/lib
//go:embed flutter_billing/ios/Classes flutter_billing/ios/moth_billing.podspec
//go:embed flutter_billing/android/build.gradle flutter_billing/android/settings.gradle
//go:embed flutter_billing/android/src
//go:embed flutter_billing/pubspec.yaml flutter_billing/analysis_options.yaml
//go:embed flutter_billing/README.md flutter_billing/CHANGELOG.md flutter_billing/LICENSE
//go:embed flutter_push/lib
//go:embed flutter_push/ios/Classes flutter_push/ios/moth_push.podspec
//go:embed flutter_push/android/build.gradle flutter_push/android/settings.gradle
//go:embed flutter_push/android/src
//go:embed flutter_push/pubspec.yaml flutter_push/analysis_options.yaml
//go:embed flutter_push/README.md flutter_push/CHANGELOG.md flutter_push/LICENSE
//go:embed react/package.json react/dist
//go:embed react/README.md react/CHANGELOG.md react/LICENSE
var FS embed.FS
