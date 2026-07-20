// Package version exposes the build version of the moth binary.
package version

// Version is the moth build version. Overridden at release time via
// -ldflags "-X github.com/aloisdeniel/moth/internal/version.Version=v1.2.3".
// The "dev" default also gates development-only features such as gRPC
// server reflection.
var Version = "dev"

// IsDev reports whether this is a development (non-release) build.
func IsDev() bool { return Version == "dev" }
